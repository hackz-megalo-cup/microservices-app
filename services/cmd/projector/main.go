package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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
	ensureKafkaTopics(ctx, brokers)

	pool := initDB(ctx, os.Getenv("DATABASE_URL"), func() (fs.FS, error) {
		return fs.Sub(projector.MigrationsFS, "migrations")
	})

	publisher, _ := platform.NewEventPublisher(brokers)
	defer publisher.Close()

	itemEventStore, itemOutbox, itemPool := initItemEventSourcing(ctx, publisher)
	if itemPool != nil {
		defer itemPool.Close()
	}
	projection := projector.NewProjectionHandler(pool, itemEventStore, itemOutbox)

	rebuilder := projector.NewRebuilder(pool, nil, projection)

	p, err := projector.New(brokers, pool, publisher, projection)
	if err != nil {
		return err
	}
	defer p.Close()

	startHealthServer(pool, rebuilder)

	slog.Info("projector started")
	return p.Run(ctx)
}

func ensureKafkaTopics(ctx context.Context, brokers []string) {
	ensureClient, err := platform.NewKafkaProducer(ctx, brokers)
	if err != nil {
		slog.Warn("failed to create kafka client for topic setup", "error", err)
	}
	if ensureClient == nil {
		return
	}
	defer ensureClient.Close()

	if err := platform.EnsureTopics(ctx, ensureClient, platform.DefaultTopics()); err != nil {
		slog.Warn("failed to ensure topics", "error", err)
	}
}

func initDB(ctx context.Context, databaseURL string, migrations func() (fs.FS, error)) *pgxpool.Pool {
	pool, _ := platform.NewDBPool(ctx, databaseURL)
	if pool == nil || databaseURL == "" {
		return pool
	}

	migrationsFS, err := migrations()
	if err != nil {
		slog.Warn("failed to load migrations fs", "error", err)
		return pool
	}
	if err := platform.RunMigrations(databaseURL, migrationsFS); err != nil {
		slog.Warn("projector migration failed", "error", err)
	} else {
		slog.Info("projector migrations applied")
	}
	return pool
}

func initItemEventSourcing(ctx context.Context, publisher *platform.EventPublisher) (*platform.EventStore, *platform.OutboxStore, *pgxpool.Pool) {
	itemDBURL := os.Getenv("ITEM_DATABASE_URL")
	if itemDBURL == "" {
		slog.Warn("ITEM_DATABASE_URL not set; login bonus will not be granted")
		return nil, nil, nil
	}

	itemPool, _ := platform.NewDBPool(ctx, itemDBURL)
	if itemPool == nil {
		return nil, nil, nil
	}

	itemEventStore := platform.NewEventStore(itemPool)
	itemOutbox := platform.NewOutboxStore(itemPool, publisher)
	itemOutbox.StartPoller(ctx, 500*time.Millisecond)

	return itemEventStore, itemOutbox, itemPool
}

func startHealthServer(pool *pgxpool.Pool, rebuilder *projector.Rebuilder) {
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
			http.Error(w, "projection query param required", http.StatusBadRequest)
			return
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
}
