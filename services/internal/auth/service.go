package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

	if email == "" || password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("email and password are required"))
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	// Create aggregate
	agg := NewUserAggregate(uuid.NewString())
	agg.RegisterUser(email, string(passwordHash))

	// Save to EventStore + Outbox
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, UserTopicMapper); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	// Write projection synchronously for immediate availability
	_, err = s.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		agg.AggregateID(), email, passwordHash, agg.Role,
	)
	if err != nil {
		slog.Error("failed to write user projection", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to register"))
	}

	return connect.NewResponse(&authv1.RegisterUserResponse{
		User: &authv1.User{
			Id:        agg.AggregateID(),
			Email:     agg.Email,
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

	// Lookup user by email from projection table
	var userID, passwordHash, role string
	err := s.pool.QueryRow(ctx,
		"SELECT id, password_hash, role FROM users WHERE email = $1",
		email,
	).Scan(&userID, &passwordHash, &role)
	if err != nil {
		slog.Error("failed to lookup user by email", "email", email, "error", err)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid email or password"))
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

	// Record login
	agg.LoggedIn()
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, UserTopicMapper); err != nil {
		slog.Error("failed to save login event", "user_id", userID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to login"))
	}

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
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found"))
	}

	return connect.NewResponse(&authv1.GetUserProfileResponse{
		User: &authv1.User{
			Id:          agg.AggregateID(),
			Email:       agg.Email,
			Role:        agg.Role,
			CreatedAt:   timestampFromTime(agg.CreatedAt),
			LastLoginAt: timestampFromTime(*agg.LastLoginAt),
		},
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

// RegisterPokemon implements the pokemonRegistrar interface for Kafka consumer
func (s *Service) RegisterPokemon(ctx context.Context, userID, pokemonID string) error {
	if userID == "" || pokemonID == "" {
		return fmt.Errorf("user_id and pokemon_id are required")
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO user_pokemon (user_id, pokemon_id) VALUES ($1, $2)
		 ON CONFLICT (user_id, pokemon_id) DO NOTHING`,
		userID, pokemonID,
	)
	return err
}
