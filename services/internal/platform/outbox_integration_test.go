//go:build integration

package platform

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const outboxSetupSQL = `
CREATE TABLE outbox_events (
    id            UUID PRIMARY KEY,
    event_type    TEXT NOT NULL,
    topic         TEXT NOT NULL,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published     BOOLEAN NOT NULL DEFAULT FALSE,
    published_at  TIMESTAMPTZ
);
CREATE INDEX idx_outbox_unpublished ON outbox_events (created_at) WHERE published = FALSE;

CREATE TABLE greetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    external_status INT,
    status TEXT NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func setupTestPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if _, err := pool.Exec(ctx, outboxSetupSQL); err != nil {
		t.Fatalf("failed to setup schema: %v", err)
	}

	return pool
}

func TestOutbox_Integration_AtomicWrite(t *testing.T) {
	pool := setupTestPostgres(t)
	ctx := context.Background()

	outbox := NewOutboxStore(pool, nil)

	tx, err := outbox.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO greetings (name, message, external_status, status) VALUES ($1, $2, $3, $4)",
		"TestUser", "Hello TestUser", 200, "completed",
	)
	if err != nil {
		t.Fatalf("insert greeting: %v", err)
	}

	event := NewEvent("greeting.created", "greeter-service", map[string]any{"name": "TestUser"})
	if err := outbox.InsertEvent(ctx, tx, "greeting.created", event); err != nil {
		t.Fatalf("insert outbox event: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var greetingCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM greetings WHERE name = 'TestUser'").Scan(&greetingCount); err != nil {
		t.Fatalf("query greetings: %v", err)
	}
	if greetingCount != 1 {
		t.Errorf("expected 1 greeting, got %d", greetingCount)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'greeting.created'").Scan(&outboxCount); err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 outbox event, got %d", outboxCount)
	}
}

func TestOutbox_Integration_RollbackRemovesBoth(t *testing.T) {
	pool := setupTestPostgres(t)
	ctx := context.Background()

	outbox := NewOutboxStore(pool, nil)

	tx, err := outbox.BeginTx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO greetings (name, message, external_status, status) VALUES ($1, $2, $3, $4)",
		"RollbackUser", "Hello", 200, "completed",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	event := NewEvent("greeting.created", "greeter-service", map[string]any{"name": "RollbackUser"})
	if err := outbox.InsertEvent(ctx, tx, "greeting.created", event); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var greetingCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM greetings WHERE name = 'RollbackUser'").Scan(&greetingCount); err != nil {
		t.Fatalf("query: %v", err)
	}
	if greetingCount != 0 {
		t.Errorf("expected 0 greetings after rollback, got %d", greetingCount)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events").Scan(&outboxCount); err != nil {
		t.Fatalf("query: %v", err)
	}
	if outboxCount != 0 {
		t.Errorf("expected 0 outbox events after rollback, got %d", outboxCount)
	}
}

func TestOutbox_Integration_Cleanup(t *testing.T) {
	pool := setupTestPostgres(t)
	ctx := context.Background()

	outbox := NewOutboxStore(pool, nil)

	event := NewEvent("test", "test", map[string]any{})
	payload, _ := json.Marshal(event)
	_, err := pool.Exec(ctx,
		`INSERT INTO outbox_events (id, event_type, topic, payload, created_at, published, published_at)
		 VALUES ($1, $2, $3, $4, $5, TRUE, $6)`,
		event.ID, event.Type, "test", payload, event.Timestamp,
		time.Now().UTC().Add(-48*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := outbox.Cleanup(ctx, 24*time.Hour); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 events after cleanup, got %d", count)
	}
}

func TestOutbox_Integration_UnpublishedSurvivesCleanup(t *testing.T) {
	pool := setupTestPostgres(t)
	ctx := context.Background()

	outbox := NewOutboxStore(pool, nil)

	event := NewEvent("test", "test", map[string]any{})
	payload, _ := json.Marshal(event)
	_, err := pool.Exec(ctx,
		`INSERT INTO outbox_events (id, event_type, topic, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		event.ID, event.Type, "test", payload, event.Timestamp,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := outbox.Cleanup(ctx, 24*time.Hour); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 unpublished event to survive cleanup, got %d", count)
	}
}
