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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/gateway/v1/gatewayv1connect"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/gateway"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const (
	serviceName    = "gateway-service"
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

	customBaseURL := os.Getenv("CUSTOM_LANG_BASE_URL")
	if customBaseURL == "" {
		customBaseURL = "http://custom-lang-service.microservices:3000"
	}

	migrationsFS, _ := fs.Sub(gateway.MigrationsFS, "migrations")
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

	// Compensation handler for invocation.failed
	compensation := platform.NewCompensationRouter()
	compensation.Handle(platform.TopicInvocationFailed, func(ctx context.Context, event platform.Event) error {
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
		agg := gateway.NewInvocationAggregate(streamID)
		if err := platform.LoadAggregate(ctx, eventStore, agg); err != nil {
			return err
		}
		agg.Compensate("downstream service failure")
		return platform.SaveAggregate(ctx, eventStore, outbox, agg, gateway.InvocationTopicMapper)
	})
	go func() {
		if err := compensation.Run(ctx, brokers, "gateway-compensation"); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("compensation router error", "error", err)
		}
	}()

	verifier := platform.NewJWTVerifier(os.Getenv("JWKS_URL"))
	idempotencyStore := platform.NewIdempotencyStore(dbPool)
	platform.StartIdempotencyCleanup(ctx, idempotencyStore)

	svc := gateway.NewService(&http.Client{
		Timeout:   2 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}, customBaseURL, time.Second, dbPool, outbox, eventStore)
	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTrustRemote())
	if err != nil {
		return err
	}
	path, handler := gatewayv1connect.NewGatewayServiceHandler(
		svc,
		connect.WithInterceptors(
			otelInterceptor,
			platform.NewAuthInterceptor(verifier),
			platform.NewIdempotencyInterceptor(idempotencyStore),
			platform.NewLoggingInterceptor(logger),
		),
	)

	mux := http.NewServeMux()
	mux.Handle(path, handler)

	// Raid allocate & join endpoints (REST, not connect-rpc)
	allocationStore := gateway.NewAllocationStore()
	allocateHandler, allocErr := gateway.NewAllocateHandler("default", allocationStore)
	if allocErr != nil {
		slog.Warn("agones allocator not available (not running in k8s?)", "error", allocErr)
	} else {
		mux.Handle("/api/raid/allocate", allocateHandler)
	}
	mux.Handle("/api/raid/join", gateway.NewJoinHandler("default", allocationStore))
	mux.Handle("/api/raid/active", gateway.NewActiveHandler("default", allocationStore))
	mux.Handle("/api/raid/ws", gateway.NewWSProxyHandler("default", allocationStore))

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
		port = "8082"
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
		logger.InfoContext(ctx, "starting gateway service", "port", port)
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
