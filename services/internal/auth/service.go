package auth

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/auth/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// Service handles authentication business logic
type Service struct {
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
	pool       *pgxpool.Pool
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
}

// NewService creates a new authentication service
func NewService(
	eventStore *platform.EventStore,
	outbox *platform.OutboxStore,
	pool *pgxpool.Pool,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	keyID string,
) *Service {
	return &Service{
		eventStore: eventStore,
		outbox:     outbox,
		pool:       pool,
		privateKey: privateKey,
		publicKey:  publicKey,
		keyID:      keyID,
	}
}

// RegisterUser creates a new user
func (s *Service) RegisterUser(ctx context.Context, req *connect.Request[authv1.RegisterUserRequest]) (*connect.Response[authv1.RegisterUserResponse], error) {
	email := req.Msg.GetEmail()
	password := req.Msg.GetPassword()
	name := req.Msg.GetName()

	if email == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email and password are required"))
	}
	if err := s.requireWriteDependencies(); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	now := time.Now().UTC()

	// Create aggregate
	agg := NewUserAggregate(uuid.NewString())
	agg.RegisterUser(email, name, string(passwordHash), now)

	tx, err := s.eventStore.BeginTx(ctx)
	if err != nil {
		slog.Error("failed to begin register transaction", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		agg.AggregateID(), email, name, passwordHash, agg.Role, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("email already exists"))
		}
		slog.Error("failed to write user projection", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	newVersion, err := s.eventStore.AppendToStream(ctx, tx, agg.AggregateID(), agg.StreamType(), agg.Version(), agg.Changes())
	if err != nil {
		slog.Error("failed to append register events", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	if err := s.insertOutboxEvents(ctx, tx, agg); err != nil {
		slog.Error("failed to insert register outbox events", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("failed to commit register transaction", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	agg.SetVersion(newVersion)
	agg.ClearChanges()

	return connect.NewResponse(&authv1.RegisterUserResponse{
		User: &authv1.User{
			Id:        agg.AggregateID(),
			Email:     agg.Email,
			Name:      agg.Name,
			Role:      agg.Role,
			CreatedAt: timestampFromTime(agg.CreatedAt),
		},
	}), nil
}

// LoginUser authenticates a user and returns JWT token
func (s *Service) LoginUser(ctx context.Context, req *connect.Request[authv1.LoginUserRequest]) (*connect.Response[authv1.LoginUserResponse], error) {
	email := req.Msg.GetEmail()
	password := req.Msg.GetPassword()

	if email == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email and password are required"))
	}
	if err := s.requireWriteDependencies(); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}

	// Lookup user by email from projection table
	var userID, passwordHash, role, name string
	err := s.pool.QueryRow(ctx,
		"SELECT id, password_hash, role, name FROM users WHERE email = $1",
		email,
	).Scan(&userID, &passwordHash, &role, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid email or password"))
		}
		slog.Error("failed to lookup user by email", "email", email, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		slog.Warn("invalid password", "email", email)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid email or password"))
	}

	// Load aggregate to trigger login event
	agg := NewUserAggregate(userID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Error("failed to load user aggregate", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	now := time.Now().UTC()

	// Record login
	agg.LoggedIn(now)

	tx, err := s.eventStore.BeginTx(ctx)
	if err != nil {
		slog.Error("failed to begin login transaction", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx,
		`UPDATE users SET last_login_at = $1, updated_at = $1 WHERE id = $2`,
		now, userID,
	); err != nil {
		slog.Error("failed to update last_login_at projection", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	newVersion, err := s.eventStore.AppendToStream(ctx, tx, agg.AggregateID(), agg.StreamType(), agg.Version(), agg.Changes())
	if err != nil {
		slog.Error("failed to append login events", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	if err := s.insertOutboxEvents(ctx, tx, agg); err != nil {
		slog.Error("failed to insert login outbox events", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("failed to commit login transaction", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	agg.SetVersion(newVersion)
	agg.ClearChanges()

	// Generate JWT token (24 hour expiry)
	token, err := s.issueJWT(userID, email, role, 24*time.Hour)
	if err != nil {
		slog.Error("failed to generate token", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

	return connect.NewResponse(&authv1.LoginUserResponse{
		User: &authv1.User{
			Id:          userID,
			Email:       email,
			Name:        name,
			Role:        role,
			CreatedAt:   timestampFromTime(agg.CreatedAt),
			LastLoginAt: timestampFromTime(*agg.LastLoginAt),
		},
		Token: token,
	}), nil
}

// GetUserProfile retrieves user profile by ID
func (s *Service) GetUserProfile(ctx context.Context, req *connect.Request[authv1.GetUserProfileRequest]) (*connect.Response[authv1.GetUserProfileResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	// Load aggregate from EventStore
	agg := NewUserAggregate(userID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Error("failed to load user", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load user profile"))
	}
	if agg.Version() == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found"))
	}

	user := &authv1.User{
		Id:        agg.AggregateID(),
		Email:     agg.Email,
		Name:      agg.Name,
		Role:      agg.Role,
		CreatedAt: timestampFromTime(agg.CreatedAt),
	}
	if agg.LastLoginAt != nil {
		user.LastLoginAt = timestampFromTime(*agg.LastLoginAt)
	}

	return connect.NewResponse(&authv1.GetUserProfileResponse{
		User: user,
	}), nil
}

// Helper functions

// timestampFromTime converts time.Time to protobuf Timestamp
func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// issueJWT generates a JWT token for the user
func (s *Service) issueJWT(userID, email, role string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"iss":   "auth-service",
		"exp":   time.Now().Add(expiresIn).Unix(),
	})
	token.Header["kid"] = s.keyID

	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}

// GetUserPokemon returns all pokemon IDs owned by the user.
func (s *Service) GetUserPokemon(ctx context.Context, req *connect.Request[authv1.GetUserPokemonRequest]) (*connect.Response[authv1.GetUserPokemonResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}
	if s.pool == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("database not configured"))
	}

	rows, err := s.pool.Query(ctx, `SELECT pokemon_id FROM user_pokemon WHERE user_id = $1`, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user pokemon: %w", err))
	}
	defer rows.Close()

	var pokemonIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan pokemon_id: %w", err))
		}
		pokemonIDs = append(pokemonIDs, pid)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to iterate pokemon_ids: %w", err))
	}

	return connect.NewResponse(&authv1.GetUserPokemonResponse{PokemonIds: pokemonIDs}), nil
}

// RegisterPokemon implements the pokemonRegistrar interface for Kafka consumer
func (s *Service) RegisterPokemon(ctx context.Context, userID, pokemonID string) error {
	if userID == "" || pokemonID == "" {
		return fmt.Errorf("user_id and pokemon_id are required")
	}
	if s.pool == nil {
		return fmt.Errorf("database not configured")
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_pokemon (user_id, pokemon_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, pokemon_id) DO NOTHING`,
		userID, pokemonID,
	)
	return err
}

func (s *Service) insertOutboxEvents(ctx context.Context, tx pgx.Tx, agg *UserAggregate) error {
	if s.outbox == nil {
		return nil
	}

	for _, change := range agg.Changes() {
		topic := UserTopicMapper(change.Type)
		if topic == "" {
			continue
		}

		event := platform.NewEvent(change.Type, "auth-service", change.Data)
		enrichedData := map[string]any{
			"stream_id": agg.AggregateID(),
		}
		if raw, marshalErr := json.Marshal(change.Data); marshalErr == nil {
			var payload map[string]any
			if unmarshalErr := json.Unmarshal(raw, &payload); unmarshalErr == nil {
				for key, value := range payload {
					enrichedData[key] = value
				}
			}
		}
		event.Data = enrichedData

		if err := s.outbox.InsertEvent(ctx, tx, topic, event); err != nil {
			return err
		}
	}

	return nil
}

// starterPokemonIDs is the set of allowed starter Pokémon.
var starterPokemonIDs = map[string]bool{
	"00000000-0000-0000-0000-000000000001": true, // Go
	"00000000-0000-0000-0000-000000000002": true, // Python
	"00000000-0000-0000-0000-000000000009": true, // Whitespace
}

// ChooseStarter registers a starter Pokémon for a new user.
func (s *Service) ChooseStarter(ctx context.Context, req *connect.Request[authv1.ChooseStarterRequest]) (*connect.Response[authv1.ChooseStarterResponse], error) {
	userID := req.Msg.GetUserId()
	pokemonID := req.Msg.GetPokemonId()

	if userID == "" || pokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and pokemon_id are required"))
	}
	if !starterPokemonIDs[pokemonID] {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid starter pokemon_id: %s", pokemonID))
	}

	// Guard: user must not already own any pokemon (idempotent for same starter)
	if s.pool != nil {
		var count int
		if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_pokemon WHERE user_id = $1`, userID).Scan(&count); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing pokemon: %w", err))
		}
		if count > 0 {
			var exists bool
			if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_pokemon WHERE user_id = $1 AND pokemon_id = $2)`, userID, pokemonID).Scan(&exists); err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check starter ownership: %w", err))
			}
			if exists {
				return connect.NewResponse(&authv1.ChooseStarterResponse{}), nil
			}
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("user already has a starter pokemon"))
		}
	}

	if err := s.RegisterPokemon(ctx, userID, pokemonID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register starter pokemon: %w", err))
	}

	return connect.NewResponse(&authv1.ChooseStarterResponse{}), nil
}

func (s *Service) requireWriteDependencies() error {
	if s.eventStore == nil || s.pool == nil {
		return fmt.Errorf("database not configured")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
