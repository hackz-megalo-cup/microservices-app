package main

import (
	"context"
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
	pool, _ := platform.NewDBPool(ctx, os.Getenv("DATABASE_URL"))

	p, err := projector.New(brokers, pool)
	if err != nil {
		return err
	}
	defer p.Close()

	slog.Info("projector started")
	return p.Run(ctx)
}
