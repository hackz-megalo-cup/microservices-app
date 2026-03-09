package main

import (
	"context"
	"io/fs"
	"log/slog"
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

	p, err := projector.New(brokers, pool, publisher)
	if err != nil {
		return err
	}
	defer p.Close()

	slog.Info("projector started")
	return p.Run(ctx)
}
