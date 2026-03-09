//go:build integration

package greeter

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	callerv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1"
	greeterv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const sagaSetupSQL = `
CREATE TABLE IF NOT EXISTS greetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    external_status INT,
    status TEXT NOT NULL DEFAULT 'completed',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS outbox_events (
    id            UUID PRIMARY KEY,
    event_type    TEXT NOT NULL,
    topic         TEXT NOT NULL,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published     BOOLEAN NOT NULL DEFAULT FALSE,
    published_at  TIMESTAMPTZ
);
`

func setupSagaPostgres(t *testing.T) *pgxpool.Pool {
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

	if _, err := pool.Exec(ctx, sagaSetupSQL); err != nil {
		t.Fatalf("failed to setup schema: %v", err)
	}

	return pool
}

func TestSaga_CallerFailure_PublishesGreetingFailed(t *testing.T) {
	pool := setupSagaPostgres(t)
	ctx := context.Background()

	outbox := platform.NewOutboxStore(pool, nil)
	mock := &mockCallerClient{err: errors.New("caller is down")}
	svc := NewService(mock, "http://example.com", 5*time.Second, pool, outbox)

	_, err := svc.Greet(ctx, connect.NewRequest(&greeterv1.GreetRequest{Name: "SagaUser"}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var outboxCount int
	if qErr := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'greeting.failed'").Scan(&outboxCount); qErr != nil {
		t.Fatalf("query outbox: %v", qErr)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 greeting.failed outbox event, got %d", outboxCount)
	}

	var greetingCount int
	if qErr := pool.QueryRow(ctx, "SELECT COUNT(*) FROM greetings").Scan(&greetingCount); qErr != nil {
		t.Fatalf("query greetings: %v", qErr)
	}
	if greetingCount != 0 {
		t.Errorf("expected 0 greetings on failure, got %d", greetingCount)
	}
}

func TestSaga_CallerSuccess_AtomicOutboxWrite(t *testing.T) {
	pool := setupSagaPostgres(t)
	ctx := context.Background()

	outbox := platform.NewOutboxStore(pool, nil)
	mock := &mockCallerClient{
		resp: connect.NewResponse(&callerv1.CallExternalResponse{StatusCode: 200, BodyLength: 42}),
	}
	svc := NewService(mock, "http://example.com", 5*time.Second, pool, outbox)

	resp, err := svc.Greet(ctx, connect.NewRequest(&greeterv1.GreetRequest{Name: "AtomicUser"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetMessage() != "Hello AtomicUser from greeter-service!" {
		t.Errorf("unexpected message: %s", resp.Msg.GetMessage())
	}

	var status string
	if qErr := pool.QueryRow(ctx, "SELECT status FROM greetings WHERE name = 'AtomicUser'").Scan(&status); qErr != nil {
		t.Fatalf("query greeting: %v", qErr)
	}
	if status != "completed" {
		t.Errorf("expected status 'completed', got %q", status)
	}

	var outboxCount int
	if qErr := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = 'greeting.created'").Scan(&outboxCount); qErr != nil {
		t.Fatalf("query outbox: %v", qErr)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 greeting.created outbox event, got %d", outboxCount)
	}
}
