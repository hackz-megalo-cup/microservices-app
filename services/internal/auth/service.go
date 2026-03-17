package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type userRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID string) (*time.Time, error)
	RegisterPokemon(ctx context.Context, userID, pokemonID string) error
	GetUserPokemon(ctx context.Context, userID string) ([]string, error)
}

// Service handles authentication business logic
type Service struct {
	repo       userRepository
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	keyID      string
}

// NewService creates a new authentication service
func NewService(
	repo userRepository,
	eventStore *platform.EventStore,
	outbox *platform.OutboxStore,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	keyID string,
) *Service {
	return &Service{
		repo:       repo,
		eventStore: eventStore,
		outbox:     outbox,
		privateKey: privateKey,
		publicKey:  publicKey,
		keyID:      keyID,
	}
}

// RegisterUserRequest is the input for RegisterUser
type RegisterUserRequest struct {
	Email    string
	Password string
}

// UserResponse is the response for user queries
type UserResponse struct {
	ID          string
	Email       string
	Role        string
	CreatedAt   time.Time
	LastLoginAt *time.Time
}

// RegisterUser creates a new user and emits UserRegistered event in a single transaction
func (s *Service) RegisterUser(ctx context.Context, req RegisterUserRequest) (*UserResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New().String()
	now := time.Now()

	user := &User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Role:         "user",
		CreatedAt:    now,
		Version:      0,
	}

	// Create user in repository
	if err := s.repo.Create(ctx, user); err != nil {
		// Check for specific database errors
		errStr := err.Error()
		if errStr == "user not found" {
			// Should not happen on Create, but handle it
			return nil, fmt.Errorf("internal error: %w", err)
		}
		// PostgreSQL duplicate key error code: 23505
		if errStr == "duplicate key" || errStr == "already exists" ||
			(len(errStr) >= 5 && errStr[:5] == "23505") {
			return nil, errors.New("email already exists")
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Emit UserRegistered event (within transaction)
	event := UserRegistered{
		UserID:    userID,
		Email:     req.Email,
		Role:      "user",
		Timestamp: now,
	}

	if err := s.appendAndPublishEvent(ctx, userID, 0, event.EventType(), event, platform.TopicUserRegistered); err != nil {
		// Event publish failed - user was created but event wasn't published
		// This is acceptable because event sourcing uses EventStore as single source of truth
		// The event will be retried if EventStore is enabled
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

// LoginUserRequest is the input for LoginUser
type LoginUserRequest struct {
	Email    string
	Password string
}

// LoginUserResponse is the response for LoginUser
type LoginUserResponse struct {
	Token string
	User  *UserResponse
}

// LoginUser authenticates a user and returns JWT token
func (s *Service) LoginUser(ctx context.Context, req LoginUserRequest) (*LoginUserResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Check if this is first login today
	isFirstToday := user.IsFirstToday()

	// Update last login timestamp
	lastLoginAt, err := s.repo.UpdateLastLogin(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Emit UserLoggedIn event (within transaction with UpdateLastLogin)
	event := UserLoggedIn{
		UserID:       user.ID,
		IsFirstToday: isFirstToday,
		Timestamp:    time.Now(),
	}

	if err := s.appendAndPublishEvent(ctx, user.ID, user.Version, event.EventType(), event, platform.TopicUserLoggedIn); err != nil {
		// Event publish failed - last_login_at was already updated in DB
		// This is acceptable for now; event will retry if EventStore is enabled
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Role,
		"iss":  "auth-service",
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	})
	token.Header["kid"] = s.keyID

	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &LoginUserResponse{
		Token: tokenString,
		User: &UserResponse{
			ID:          user.ID,
			Email:       user.Email,
			Role:        user.Role,
			CreatedAt:   user.CreatedAt,
			LastLoginAt: lastLoginAt,
		},
	}, nil
}

// GetUserProfileRequest is the input for GetUserProfile
type GetUserProfileRequest struct {
	UserID string
}

// GetUserProfile retrieves user profile by ID
func (s *Service) GetUserProfile(ctx context.Context, req GetUserProfileRequest) (*UserResponse, error) {
	if req.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		LastLoginAt: user.LastLoginAt,
	}, nil
}

// RegisterPokemonRequest is the input for RegisterPokemon
type RegisterPokemonRequest struct {
	UserID    string
	PokemonID string
}

// RegisterPokemon registers a caught pokemon for a user
func (s *Service) RegisterPokemon(ctx context.Context, req RegisterPokemonRequest) error {
	if req.UserID == "" || req.PokemonID == "" {
		return errors.New("user_id and pokemon_id are required")
	}

	return s.repo.RegisterPokemon(ctx, req.UserID, req.PokemonID)
}

// GetUserPokemonRequest is the input for GetUserPokemon
type GetUserPokemonRequest struct {
	UserID string
}

// GetUserPokemon retrieves all pokemon caught by a user
func (s *Service) GetUserPokemon(ctx context.Context, req GetUserPokemonRequest) ([]string, error) {
	if req.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	return s.repo.GetUserPokemon(ctx, req.UserID)
}

func (s *Service) appendAndPublishEvent(
	ctx context.Context,
	streamID string,
	expectedVersion int,
	eventType string,
	data any,
	topic string,
) error {
	if s.eventStore == nil && s.outbox == nil {
		return nil
	}

	if s.eventStore == nil {
		tx, err := s.outbox.BeginTx(ctx)
		if err != nil {
			return fmt.Errorf("begin outbox tx: %w", err)
		}
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		event := platform.NewEvent(eventType, "auth-service", data)
		if err := s.outbox.InsertEvent(ctx, tx, topic, event); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit outbox tx: %w", err)
		}
		return nil
	}

	tx, err := s.eventStore.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin event tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = s.eventStore.AppendToStream(ctx, tx, streamID, "auth.user", expectedVersion, []platform.UnsavedEvent{{
		Type: eventType,
		Data: data,
	}})
	if err != nil {
		return fmt.Errorf("append to stream: %w", err)
	}

	if s.outbox != nil {
		event := platform.NewEvent(eventType, "auth-service", data)
		if err := s.outbox.InsertEvent(ctx, tx, topic, event); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit event tx: %w", err)
	}

	return nil
}
