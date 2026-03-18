package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/capture/v1/capturev1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1/itemv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1/masterdatav1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/capture"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const (
	serviceName    = "capture-service"
	serviceVersion = "0.1.0"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Logging & Observability ---
	logger := platform.NewLogger()
	shutdownOTel, err := platform.SetupOTelSDK(ctx, serviceName, serviceVersion)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, shutdownOTel(context.Background()))
	}()

	// --- Database ---
	migrationsFS, _ := fs.Sub(capture.MigrationsFS, "migrations")
	dbPool := platform.InitDB(ctx, os.Getenv("DATABASE_URL"), migrationsFS, serviceName)
	if dbPool != nil {
		defer dbPool.Close()
	}

	// --- Kafka ---
	brokers := platform.ParseKafkaBrokers(os.Getenv("KAFKA_BROKERS"))
	platform.TryEnsureTopics(ctx, brokers)

	publisher, _ := platform.NewEventPublisher(brokers)
	defer publisher.Close()

	// --- Outbox (transactional event publishing) ---
	outbox := platform.NewOutboxStore(dbPool, publisher)
	outbox.StartPoller(ctx, 500*time.Millisecond)

	// --- Event Store ---
	eventStore := platform.NewEventStore(dbPool)

	// --- Compensation (saga rollback handler) ---
	compensation := platform.NewCompensationRouter()
	compensation.Handle(platform.TopicCaptureFailed, func(ctx context.Context, event platform.Event) error {
		if eventStore == nil {
			return nil
		}
		data, ok := event.Data.(map[string]any)
		if !ok {
			slog.Warn("compensation: unexpected data type", "event_id", event.ID)
			return nil
		}
		streamID, _ := data["stream_id"].(string)
		if streamID == "" {
			slog.Warn("compensation: missing stream_id", "event_id", event.ID)
			return nil
		}
		agg := capture.NewCaptureAggregate(streamID)
		if err := platform.LoadAggregate(ctx, eventStore, agg); err != nil {
			return err
		}
		agg.Compensate("saga compensation")
		if err := platform.SaveAggregate(ctx, eventStore, outbox, agg, capture.CaptureTopicMapper); err != nil {
			return err
		}
		slog.Info("compensation: aggregate compensated via ES", "stream_id", streamID)
		return nil
	})
	go func() {
		if err := compensation.Run(ctx, brokers, "capture-compensation"); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("compensation router error", "error", err)
		}
	}()

	// --- Masterdata client ---
	var masterdataClient masterdatav1connect.MasterdataServiceClient
	if masterdataURL := os.Getenv("MASTERDATA_URL"); masterdataURL != "" {
		masterdataClient = masterdatav1connect.NewMasterdataServiceClient(
			platform.NewInstrumentedHTTPClient(3*time.Second),
			masterdataURL,
		)
	}

	// --- Item client ---
	var itemClient itemv1connect.ItemServiceClient
	if itemURL := os.Getenv("ITEM_URL"); itemURL != "" {
		itemClient = itemv1connect.NewItemServiceClient(
			platform.NewInstrumentedHTTPClient(3*time.Second),
			itemURL,
		)
	}

	// --- Auth & Idempotency ---
	verifier := platform.NewJWTVerifier(os.Getenv("JWKS_URL"))
	idempotencyStore := platform.NewIdempotencyStore(dbPool)
	platform.StartIdempotencyCleanup(ctx, idempotencyStore)

	// --- Service ---
	svc := capture.NewService(eventStore, outbox, dbPool, masterdataClient, itemClient)

	// --- Battle.finished Consumer ---
	go runBattleFinishedConsumer(ctx, brokers, svc)

	// --- Connect-RPC Handler with interceptors ---
	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
	if err != nil {
		return err
	}
	path, handler := capturev1connect.NewCaptureServiceHandler(
		svc,
		connect.WithInterceptors(
			otelInterceptor,
			platform.NewAuthInterceptor(verifier),
			platform.NewIdempotencyInterceptor(idempotencyStore),
			platform.NewLoggingInterceptor(logger),
		),
	)

	// --- HTTP Server ---
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if dbPool != nil {
			if err := dbPool.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("db unhealthy\n"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
	}

	srvErr := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "starting "+serviceName, "port", port)
		srvErr <- srv.ListenAndServe()
	}()

	select {
	case err = <-srvErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func runBattleFinishedConsumer(ctx context.Context, brokers []string, svc *capture.Service) {
	sub, err := platform.NewEventSubscriber(brokers, "capture-battle-consumer")
	if err != nil {
		slog.Error("failed to create battle.finished subscriber", "error", err)
		return
	}
	if sub == nil {
		return
	}
	defer sub.Close()

	ch, err := sub.Subscribe(ctx, platform.TopicBattleFinished)
	if err != nil {
		slog.Error("failed to subscribe to battle.finished", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			handleBattleMessage(ctx, msg, svc)
		}
	}
}

func handleBattleMessage(ctx context.Context, msg *message.Message, svc *capture.Service) {
	event, err := platform.ParseEvent(msg)
	if err != nil {
		slog.Error("capture battle consumer: failed to parse event", "error", err)
		msg.Nack()
		return
	}
	data, ok := event.Data.(map[string]any)
	if !ok {
		slog.Warn("capture battle consumer: unexpected data type", "event_id", event.ID)
		msg.Ack()
		return
	}

	result, _ := data["result"].(string)
	if result != "win" {
		msg.Ack()
		return
	}

	battleSessionID, _ := data["session_id"].(string)
	bossPokemonID, _ := data["boss_pokemon_id"].(string)
	participantUserIDs := extractParticipantIDs(data)

	if err := svc.HandleBattleFinished(ctx, battleSessionID, bossPokemonID, participantUserIDs); err != nil {
		slog.Error("capture battle consumer: failed to handle event", "error", err)
		msg.Nack()
		return
	}
	msg.Ack()
}

func extractParticipantIDs(data map[string]any) []string {
	var ids []string
	rawIDs, ok := data["participant_user_ids"].([]any)
	if !ok {
		return ids
	}
	for _, id := range rawIDs {
		if s, ok := id.(string); ok {
			ids = append(ids, s)
		}
	}
	return ids
}
