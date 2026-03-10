package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/projector"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	brokers := platform.ParseKafkaBrokers(os.Getenv("KAFKA_BROKERS"))

	// Ensure Kafka topics exist
	ensureClient, err := platform.NewKafkaProducer(ctx, brokers)
	if err != nil {
		slog.Warn("failed to create kafka client for topic setup", "error", err)
	}
	if ensureClient != nil {
		if err := platform.EnsureTopics(ctx, ensureClient, platform.DefaultTopics()); err != nil {
			slog.Warn("failed to ensure topics", "error", err)
		}
		ensureClient.Close()
	}

	databaseURL := os.Getenv("DATABASE_URL")
	pool, _ := platform.NewDBPool(ctx, databaseURL)

	// Run migrations
	if pool != nil && databaseURL != "" {
		migrationsFS, _ := fs.Sub(projector.MigrationsFS, "migrations")
		if err := platform.RunMigrations(databaseURL, migrationsFS); err != nil {
			slog.Warn("projector migration failed", "error", err)
		} else {
			slog.Info("projector migrations applied")
		}
	}

	publisher, _ := platform.NewEventPublisher(brokers)
	defer publisher.Close()

	projection := projector.NewProjectionHandler(pool)

	// Event store connection for rebuilds (connects to greeter DB).
	greeterDBURL := os.Getenv("GREETER_DATABASE_URL")
	var eventStore *platform.EventStore
	if greeterDBURL != "" {
		greeterPool, _ := platform.NewDBPool(ctx, greeterDBURL)
		if greeterPool != nil {
			eventStore = platform.NewEventStore(greeterPool)
			defer greeterPool.Close()
		}
	}

	rebuilder := projector.NewRebuilder(pool, eventStore, projection)

	p, err := projector.New(brokers, pool, publisher, projection)
	if err != nil {
		return err
	}
	defer p.Close()

	// Health check HTTP server
	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8083"
	}
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if pool != nil {
			if err := pool.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("db unhealthy\n"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	healthMux.HandleFunc("/rebuild", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("projection")
		if name == "" {
			name = projector.ProjectionGreetingsView
		}
		if rebuilder == nil {
			http.Error(w, "rebuilder not configured", http.StatusServiceUnavailable)
			return
		}
		if err := rebuilder.Rebuild(r.Context(), name); err != nil {
			slog.Error("rebuild failed", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("rebuild complete\n"))
	})
	go func() {
		slog.Info("projector health check listening", "port", healthPort)
		if err := http.ListenAndServe(":"+healthPort, healthMux); err != nil {
			slog.Error("health check server error", "error", err)
		}
	}()

	slog.Info("projector started")
	return p.Run(ctx)
}
