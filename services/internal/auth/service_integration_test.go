//go:build integration

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io/fs"
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

	migrationsFS, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		t.Fatalf("failed to prepare migrations fs: %v", err)
	}

	if err := platform.RunMigrations(connStr, migrationsFS); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
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
