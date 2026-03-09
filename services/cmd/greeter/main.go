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

	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1/callerv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v1/greeterv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v2/greeterv2connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/greeter"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const (
	serviceName    = "greeter-service"
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

	logger := platform.NewLogger()
	shutdownOTel, err := platform.SetupOTelSDK(ctx, serviceName, serviceVersion)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, shutdownOTel(context.Background()))
	}()

	callerBaseURL := os.Getenv("CALLER_BASE_URL")
	if callerBaseURL == "" {
		callerBaseURL = "http://caller-service.microservices:8081"
	}
	externalURL := os.Getenv("EXTERNAL_API_URL")
	if externalURL == "" {
		externalURL = "https://httpbin.org/get"
	}

	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return err
	}
	clientHTTP := &http.Client{Timeout: 3 * time.Second}
	callerClient := callerv1connect.NewCallerServiceClient(
		clientHTTP,
		callerBaseURL,
		connect.WithInterceptors(otelInterceptor),
	)
	migrationsFS, _ := fs.Sub(greeter.MigrationsFS, "migrations")
	dbPool := platform.InitDB(ctx, os.Getenv("DATABASE_URL"), migrationsFS, serviceName)
	if dbPool != nil {
		defer dbPool.Close()
	}

	brokers := platform.ParseKafkaBrokers(os.Getenv("KAFKA_BROKERS"))
	platform.TryEnsureTopics(ctx, brokers)

	publisher, _ := platform.NewEventPublisher(brokers)
	defer publisher.Close()

	outbox := platform.NewOutboxStore(dbPool, publisher)
	outbox.StartPoller(ctx, 500*time.Millisecond)

	eventStore := platform.NewEventStore(dbPool)

	// Compensation handler for greeting.failed
	compensation := platform.NewCompensationRouter()
	compensation.Handle(platform.TopicGreetingFailed, func(ctx context.Context, event platform.Event) error {
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
		agg := greeter.NewGreetingAggregate(streamID)
		if err := platform.LoadAggregate(ctx, eventStore, agg); err != nil {
			return err
		}
		agg.Compensate("saga compensation")
		if err := platform.SaveAggregate(ctx, eventStore, outbox, agg, greeter.GreetingTopicMapper); err != nil {
			return err
		}
		slog.Info("compensation: greeting compensated via ES", "stream_id", streamID)
		return nil
	})
	go func() {
		if err := compensation.Run(ctx, brokers, "greeter-compensation"); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("compensation router error", "error", err)
		}
	}()

	verifier := platform.NewJWTVerifier(os.Getenv("JWKS_URL"))
	idempotencyStore := platform.NewIdempotencyStore(dbPool)
	platform.StartIdempotencyCleanup(ctx, idempotencyStore)

	greeterSvc := greeter.NewService(callerClient, externalURL, 2*time.Second, dbPool, outbox, eventStore)

	connectOpts := connect.WithInterceptors(
		otelInterceptor,
		platform.NewAuthInterceptor(verifier),
		platform.NewIdempotencyInterceptor(idempotencyStore),
		platform.NewLoggingInterceptor(logger),
	)

	pathV1, handlerV1 := greeterv1connect.NewGreeterServiceHandler(greeterSvc, connectOpts)

	greeterSvcV2 := greeter.NewServiceV2(greeterSvc)
	pathV2, handlerV2 := greeterv2connect.NewGreeterServiceHandler(greeterSvcV2, connectOpts)

	mux := http.NewServeMux()
	mux.Handle(pathV1, handlerV1)
	mux.Handle(pathV2, handlerV2)
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
		port = "8080"
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
		logger.InfoContext(ctx, "starting greeter service", "port", port)
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
