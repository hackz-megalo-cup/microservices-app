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

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/thirdlf03/microservice-app/services/gen/go/gateway/v1/gatewayv1connect"
	"github.com/thirdlf03/microservice-app/services/internal/gateway"
	"github.com/thirdlf03/microservice-app/services/internal/platform"
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

	// DB init (optional: サービスは DB なしでも起動する)
	databaseURL := os.Getenv("DATABASE_URL")
	var dbPool *pgxpool.Pool
	if databaseURL != "" {
		dbPool, err = platform.NewDBPool(ctx, databaseURL)
		if err != nil {
			logger.WarnContext(ctx, "database unavailable, running without DB", "error", err)
			dbPool = nil
		} else {
			defer dbPool.Close()
			migrationsFS, _ := fs.Sub(gateway.MigrationsFS, "migrations")
			if err := platform.RunMigrations(databaseURL, migrationsFS); err != nil {
				logger.WarnContext(ctx, "migration failed, running without DB", "error", err)
				dbPool.Close()
				dbPool = nil
			} else {
				logger.InfoContext(ctx, "database ready", "service", serviceName)
			}
		}
	}

	svc := gateway.NewService(&http.Client{
		Timeout:   2 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}, customBaseURL, time.Second, dbPool)
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return err
	}
	path, handler := gatewayv1connect.NewGatewayServiceHandler(
		svc,
		connect.WithInterceptors(
			otelInterceptor,
			platform.NewLoggingInterceptor(logger),
		),
	)

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
