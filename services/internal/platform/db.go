package platform

import (
	"context"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDBPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

// InitDB creates a connection pool and runs migrations. Returns nil pool on failure (graceful degradation).
func InitDB(ctx context.Context, databaseURL string, migrationsFS fs.FS, serviceName string) *pgxpool.Pool {
	if databaseURL == "" {
		return nil
	}
	pool, err := NewDBPool(ctx, databaseURL)
	if err != nil {
		slog.WarnContext(ctx, "database unavailable, running without DB", "error", err)
		return nil
	}
	if err := RunMigrations(databaseURL, migrationsFS); err != nil {
		slog.WarnContext(ctx, "migration failed, running without DB", "error", err)
		pool.Close()
		return nil
	}
	slog.InfoContext(ctx, "database ready", "service", serviceName)
	return pool
}
