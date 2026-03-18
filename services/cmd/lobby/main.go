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

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/lobby/v1/lobbyv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/lobby"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const (
	serviceName    = "lobby-service"
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
	migrationsFS, _ := fs.Sub(lobby.MigrationsFS, "migrations")
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
	compensation.Handle(platform.TopicLobbyFailed, func(ctx context.Context, event platform.Event) error {
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
		agg := lobby.NewAggregate(streamID)
		if err := platform.LoadAggregate(ctx, eventStore, agg); err != nil {
			return err
		}
		agg.Compensate("saga compensation")
		if err := platform.SaveAggregate(ctx, eventStore, outbox, agg, lobby.LobbyTopicMapper); err != nil {
			return err
		}
		slog.Info("compensation: aggregate compensated via ES", "stream_id", streamID)
		return nil
	})
	go func() {
		if err := compensation.Run(ctx, brokers, "lobby-compensation"); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("compensation router error", "error", err)
		}
	}()

	// --- Auth & Idempotency ---
	verifier := platform.NewJWTVerifier(os.Getenv("JWKS_URL"))
	idempotencyStore := platform.NewIdempotencyStore(dbPool)
	platform.StartIdempotencyCleanup(ctx, idempotencyStore)

	// --- Service ---
	svc := lobby.NewService(eventStore, outbox)

	// --- Connect-RPC Handler with interceptors ---
	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
	if err != nil {
		return err
	}
	path, handler := lobbyv1connect.NewLobbyServiceHandler(
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
		port = "8089"
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
