//go:build integration

package greeter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	callerv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1"
	greeterv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v1"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS greetings (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    message TEXT NOT NULL,
    external_status INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

func setupPostgres(t *testing.T) *pgxpool.Pool {
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
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if _, err := pool.Exec(ctx, createTableSQL); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return pool
}

func TestGreet_Integration_DBInsert(t *testing.T) {
	pool := setupPostgres(t)

	mock := &mockCallerClient{
		resp: connect.NewResponse(&callerv1.CallExternalResponse{
			StatusCode: 200,
			BodyLength: 99,
		}),
	}
	svc := NewService(mock, "http://example.com", 5*time.Second, pool, nil)

	ctx := context.Background()
	resp, err := svc.Greet(ctx, connect.NewRequest(&greeterv1.GreetRequest{Name: "IntegrationUser"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetMessage() != "Hello IntegrationUser from greeter-service!" {
		t.Errorf("got message %q", resp.Msg.GetMessage())
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM greetings WHERE name = $1", "IntegrationUser").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query greetings: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	var dbName, dbMessage string
	var dbStatus int32
	err = pool.QueryRow(ctx, "SELECT name, message, external_status FROM greetings WHERE name = $1", "IntegrationUser").Scan(&dbName, &dbMessage, &dbStatus)
	if err != nil {
		t.Fatalf("failed to scan row: %v", err)
	}
	if dbName != "IntegrationUser" {
		t.Errorf("got name %q", dbName)
	}
	if dbMessage != "Hello IntegrationUser from greeter-service!" {
		t.Errorf("got message %q", dbMessage)
	}
	if dbStatus != 200 {
		t.Errorf("got status %d, want 200", dbStatus)
	}
}

func TestGreet_Integration_MultipleInserts(t *testing.T) {
	pool := setupPostgres(t)

	mock := &mockCallerClient{
		resp: connect.NewResponse(&callerv1.CallExternalResponse{
			StatusCode: 200,
			BodyLength: 10,
		}),
	}
	svc := NewService(mock, "http://example.com", 5*time.Second, pool, nil)
	ctx := context.Background()

	for i := range 3 {
		name := fmt.Sprintf("User%d", i)
		_, err := svc.Greet(ctx, connect.NewRequest(&greeterv1.GreetRequest{Name: name}))
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", name, err)
		}
	}

	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM greetings").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}
