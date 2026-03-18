//go:build integration

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	authv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/auth/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const createTablesSQL = `
CREATE TABLE IF NOT EXISTS event_store (
    id BIGSERIAL,
    stream_id TEXT NOT NULL,
    stream_type TEXT NOT NULL,
    version INTEGER NOT NULL,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    data JSONB NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (stream_id, version)
);

CREATE INDEX IF NOT EXISTS idx_event_store_stream ON event_store (stream_id, version);
CREATE INDEX IF NOT EXISTS idx_event_store_type ON event_store (event_type);
CREATE INDEX IF NOT EXISTS idx_event_store_created ON event_store (created_at);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    response BYTEA,
    status_code INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires ON idempotency_keys(expires_at);

CREATE TABLE IF NOT EXISTS outbox_events (
    id            UUID PRIMARY KEY,
    event_type    TEXT NOT NULL,
    topic         TEXT NOT NULL,
    payload       JSONB NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published     BOOLEAN NOT NULL DEFAULT FALSE,
    published_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS snapshots (
    stream_id TEXT PRIMARY KEY,
    stream_type TEXT NOT NULL,
    version INTEGER NOT NULL,
    state JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY,
    email varchar UNIQUE NOT NULL,
    password_hash text NOT NULL,
    role varchar DEFAULT 'user',
    created_at timestamptz NOT NULL,
    last_login_at timestamptz,
    updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS user_pokemon (
    user_id uuid NOT NULL,
    pokemon_id varchar NOT NULL,
    caught_at timestamptz NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, pokemon_id)
);

CREATE INDEX IF NOT EXISTS idx_user_pokemon_user_id ON user_pokemon(user_id);
`

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

	if _, err := pool.Exec(ctx, createTablesSQL); err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	return pool
}

func setupRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

func TestRegisterUser_Integration(t *testing.T) {
	pool := setupPostgres(t)
	privateKey, publicKey := setupRSAKeys(t)
	ctx := context.Background()

	eventStore := platform.NewEventStore(pool)
	outbox := platform.NewOutboxStore(pool, nil)
	svc := NewService(eventStore, outbox, pool, privateKey, publicKey, "test-kid")

	req := connect.NewRequest(&authv1.RegisterUserRequest{
		Email:    "integration@example.com",
		Password: "password123",
	})

	resp, err := svc.RegisterUser(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.User.Email != "integration@example.com" {
		t.Errorf("got email %q, want %q", resp.Msg.User.Email, "integration@example.com")
	}
	if resp.Msg.User.Role != "user" {
		t.Errorf("got role %q, want %q", resp.Msg.User.Role, "user")
	}
	if resp.Msg.User.Id == "" {
		t.Error("expected non-empty user ID")
	}

	var email, passwordHash, role string
	err = pool.QueryRow(ctx, "SELECT email, password_hash, role FROM users WHERE id = $1", resp.Msg.User.Id).
		Scan(&email, &passwordHash, &role)
	if err != nil {
		t.Fatalf("failed to query users table: %v", err)
	}
	if email != "integration@example.com" {
		t.Errorf("users table: got email %q, want %q", email, "integration@example.com")
	}
	if passwordHash == "" {
		t.Error("users table: password_hash should not be empty")
	}
	if role != "user" {
		t.Errorf("users table: got role %q, want %q", role, "user")
	}

	var eventCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM event_store WHERE stream_id = $1", resp.Msg.User.Id).Scan(&eventCount)
	if err != nil {
		t.Fatalf("failed to query event_store: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("expected 1 event in event_store, got %d", eventCount)
	}

	var outboxCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE event_type = $1", "user.registered").Scan(&outboxCount)
	if err != nil {
		t.Fatalf("failed to query outbox_events: %v", err)
	}
	if outboxCount != 1 {
		t.Errorf("expected 1 outbox event, got %d", outboxCount)
	}
}

func TestLoginUser_Integration(t *testing.T) {
	pool := setupPostgres(t)
	privateKey, publicKey := setupRSAKeys(t)
	ctx := context.Background()

	eventStore := platform.NewEventStore(pool)
	outbox := platform.NewOutboxStore(pool, nil)
	svc := NewService(eventStore, outbox, pool, privateKey, publicKey, "test-kid")

	registerReq := connect.NewRequest(&authv1.RegisterUserRequest{
		Email:    "login@example.com",
		Password: "password123",
	})
	registerResp, err := svc.RegisterUser(ctx, registerReq)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	loginReq := connect.NewRequest(&authv1.LoginUserRequest{
		Email:    "login@example.com",
		Password: "password123",
	})
	loginResp, err := svc.LoginUser(ctx, loginReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loginResp.Msg.Token == "" {
		t.Error("expected non-empty token")
	}
	if loginResp.Msg.User.Email != "login@example.com" {
		t.Errorf("got email %q, want %q", loginResp.Msg.User.Email, "login@example.com")
	}
	if loginResp.Msg.User.LastLoginAt == nil {
		t.Error("expected last_login_at to be set")
	}

	var eventCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM event_store WHERE stream_id = $1 AND event_type = $2",
		registerResp.Msg.User.Id, "user.logged_in").Scan(&eventCount)
	if err != nil {
		t.Fatalf("failed to query event_store: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("expected 1 login event, got %d", eventCount)
	}
}

func TestLoginUser_InvalidPassword_Integration(t *testing.T) {
	pool := setupPostgres(t)
	privateKey, publicKey := setupRSAKeys(t)
	ctx := context.Background()

	eventStore := platform.NewEventStore(pool)
	outbox := platform.NewOutboxStore(pool, nil)
	svc := NewService(eventStore, outbox, pool, privateKey, publicKey, "test-kid")

	registerReq := connect.NewRequest(&authv1.RegisterUserRequest{
		Email:    "wrong@example.com",
		Password: "correctpassword",
	})
	_, err := svc.RegisterUser(ctx, registerReq)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	loginReq := connect.NewRequest(&authv1.LoginUserRequest{
		Email:    "wrong@example.com",
		Password: "wrongpassword",
	})
	_, err = svc.LoginUser(ctx, loginReq)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected Unauthenticated error, got %v", err)
	}
}

func TestGetUserProfile_Integration(t *testing.T) {
	pool := setupPostgres(t)
	privateKey, publicKey := setupRSAKeys(t)
	ctx := context.Background()

	eventStore := platform.NewEventStore(pool)
	outbox := platform.NewOutboxStore(pool, nil)
	svc := NewService(eventStore, outbox, pool, privateKey, publicKey, "test-kid")

	registerReq := connect.NewRequest(&authv1.RegisterUserRequest{
		Email:    "profile@example.com",
		Password: "password123",
	})
	registerResp, err := svc.RegisterUser(ctx, registerReq)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	profileReq := connect.NewRequest(&authv1.GetUserProfileRequest{
		UserId: registerResp.Msg.User.Id,
	})
	profileResp, err := svc.GetUserProfile(ctx, profileReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profileResp.Msg.User.Email != "profile@example.com" {
		t.Errorf("got email %q, want %q", profileResp.Msg.User.Email, "profile@example.com")
	}
	if profileResp.Msg.User.Role != "user" {
		t.Errorf("got role %q, want %q", profileResp.Msg.User.Role, "user")
	}
}

func TestGetUserProfile_NotFound_Integration(t *testing.T) {
	pool := setupPostgres(t)
	privateKey, publicKey := setupRSAKeys(t)
	ctx := context.Background()

	eventStore := platform.NewEventStore(pool)
	outbox := platform.NewOutboxStore(pool, nil)
	svc := NewService(eventStore, outbox, pool, privateKey, publicKey, "test-kid")

	profileReq := connect.NewRequest(&authv1.GetUserProfileRequest{
		UserId: "00000000-0000-0000-0000-000000000000",
	})
	_, err := svc.GetUserProfile(ctx, profileReq)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestRegisterPokemon_Integration(t *testing.T) {
	pool := setupPostgres(t)
	privateKey, publicKey := setupRSAKeys(t)
	ctx := context.Background()

	eventStore := platform.NewEventStore(pool)
	outbox := platform.NewOutboxStore(pool, nil)
	svc := NewService(eventStore, outbox, pool, privateKey, publicKey, "test-kid")

	registerReq := connect.NewRequest(&authv1.RegisterUserRequest{
		Email:    "trainer@example.com",
		Password: "password123",
	})
	registerResp, err := svc.RegisterUser(ctx, registerReq)
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	userID := registerResp.Msg.User.Id

	err = svc.RegisterPokemon(ctx, userID, "pikachu")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pokemonID string
	err = pool.QueryRow(ctx, "SELECT pokemon_id FROM user_pokemon WHERE user_id = $1", userID).Scan(&pokemonID)
	if err != nil {
		t.Fatalf("failed to query user_pokemon: %v", err)
	}
	if pokemonID != "pikachu" {
		t.Errorf("got pokemon_id %q, want %q", pokemonID, "pikachu")
	}

	err = svc.RegisterPokemon(ctx, userID, "pikachu")
	if err != nil {
		t.Errorf("unexpected error on duplicate: %v", err)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_pokemon WHERE user_id = $1", userID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count user_pokemon: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 pokemon, got %d", count)
	}
}
