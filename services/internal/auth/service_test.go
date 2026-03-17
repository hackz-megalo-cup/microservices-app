package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"
)

// mockUserRepository implements UserRepository interface for testing
type mockUserRepository struct {
	createErr           error
	getByEmailUser      *User
	getByEmailErr       error
	getByIDUser         *User
	getByIDErr          error
	updateLastLoginTime *time.Time
	updateLastLoginErr  error
	registerPokemonErr  error
	getUserPokemonList  []string
	getUserPokemonErr   error
}

func (m *mockUserRepository) Create(ctx context.Context, user *User) error {
	return m.createErr
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	return m.getByEmailUser, m.getByEmailErr
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	return m.getByIDUser, m.getByIDErr
}

func (m *mockUserRepository) UpdateLastLogin(ctx context.Context, userID string) (*time.Time, error) {
	return m.updateLastLoginTime, m.updateLastLoginErr
}

func (m *mockUserRepository) RegisterPokemon(ctx context.Context, userID, pokemonID string) error {
	return m.registerPokemonErr
}

func (m *mockUserRepository) GetUserPokemon(ctx context.Context, userID string) ([]string, error) {
	return m.getUserPokemonList, m.getUserPokemonErr
}

func TestService_RegisterUser(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	t.Run("success", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		resp, err := svc.RegisterUser(context.Background(), RegisterUserRequest{
			Email:    "test@example.com",
			Password: "password123",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Email != "test@example.com" {
			t.Errorf("got email %q, want %q", resp.Email, "test@example.com")
		}
		if resp.Role != "user" {
			t.Errorf("got role %q, want %q", resp.Role, "user")
		}
	})

	t.Run("empty email", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.RegisterUser(context.Background(), RegisterUserRequest{
			Email:    "",
			Password: "password123",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.RegisterUser(context.Background(), RegisterUserRequest{
			Email:    "test@example.com",
			Password: "",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockUserRepository{
			createErr: errors.New("db error"),
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.RegisterUser(context.Background(), RegisterUserRequest{
			Email:    "test@example.com",
			Password: "password123",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestService_LoginUser(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	// Pre-hashed password for "password123"
	hashedPassword := "$2a$10$YourHashedPasswordHere"

	t.Run("invalid email or password - user not found", func(t *testing.T) {
		repo := &mockUserRepository{
			getByEmailErr: errors.New("user not found"),
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.LoginUser(context.Background(), LoginUserRequest{
			Email:    "test@example.com",
			Password: "password123",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "invalid email or password" {
			t.Errorf("got error %q, want %q", err.Error(), "invalid email or password")
		}
	})

	t.Run("empty email", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.LoginUser(context.Background(), LoginUserRequest{
			Email:    "",
			Password: "password123",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty password", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.LoginUser(context.Background(), LoginUserRequest{
			Email:    "test@example.com",
			Password: "",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("update last login error", func(t *testing.T) {
		now := time.Now()
		repo := &mockUserRepository{
			getByEmailUser: &User{
				ID:           "user-123",
				Email:        "test@example.com",
				PasswordHash: hashedPassword,
				Role:         "user",
				CreatedAt:    now,
			},
			updateLastLoginErr: errors.New("db error"),
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.LoginUser(context.Background(), LoginUserRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestService_GetUserProfile(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		repo := &mockUserRepository{
			getByIDUser: &User{
				ID:        "user-123",
				Email:     "test@example.com",
				Role:      "user",
				CreatedAt: now,
			},
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		resp, err := svc.GetUserProfile(context.Background(), GetUserProfileRequest{
			UserID: "user-123",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != "user-123" {
			t.Errorf("got ID %q, want %q", resp.ID, "user-123")
		}
		if resp.Email != "test@example.com" {
			t.Errorf("got email %q, want %q", resp.Email, "test@example.com")
		}
	})

	t.Run("empty user_id", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.GetUserProfile(context.Background(), GetUserProfileRequest{
			UserID: "",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("user not found", func(t *testing.T) {
		repo := &mockUserRepository{
			getByIDErr: errors.New("user not found"),
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.GetUserProfile(context.Background(), GetUserProfileRequest{
			UserID: "nonexistent",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestUser_IsFirstToday(t *testing.T) {
	tests := []struct {
		name         string
		lastLoginAt  *time.Time
		expectedBool bool
	}{
		{
			name:         "no last login",
			lastLoginAt:  nil,
			expectedBool: true,
		},
		{
			name: "last login today",
			lastLoginAt: func() *time.Time {
				t := time.Now()
				return &t
			}(),
			expectedBool: false,
		},
		{
			name: "last login yesterday",
			lastLoginAt: func() *time.Time {
				t := time.Now().AddDate(0, 0, -1)
				return &t
			}(),
			expectedBool: true,
		},
		{
			name: "last login last month",
			lastLoginAt: func() *time.Time {
				t := time.Now().AddDate(0, -1, 0)
				return &t
			}(),
			expectedBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				LastLoginAt: tt.lastLoginAt,
			}
			got := user.IsFirstToday()
			if got != tt.expectedBool {
				t.Errorf("IsFirstToday() = %v, want %v", got, tt.expectedBool)
			}
		})
	}
}

func TestService_RegisterPokemon(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	t.Run("success", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		err := svc.RegisterPokemon(context.Background(), RegisterPokemonRequest{
			UserID:    "user-123",
			PokemonID: "pikachu",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty user_id", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		err := svc.RegisterPokemon(context.Background(), RegisterPokemonRequest{
			UserID:    "",
			PokemonID: "pikachu",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty pokemon_id", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		err := svc.RegisterPokemon(context.Background(), RegisterPokemonRequest{
			UserID:    "user-123",
			PokemonID: "",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockUserRepository{
			registerPokemonErr: errors.New("db error"),
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		err := svc.RegisterPokemon(context.Background(), RegisterPokemonRequest{
			UserID:    "user-123",
			PokemonID: "pikachu",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestService_GetUserPokemon(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	t.Run("success", func(t *testing.T) {
		repo := &mockUserRepository{
			getUserPokemonList: []string{"pikachu", "charmander"},
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		list, err := svc.GetUserPokemon(context.Background(), GetUserPokemonRequest{
			UserID: "user-123",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("got %d pokemon, want 2", len(list))
		}
	})

	t.Run("empty user_id", func(t *testing.T) {
		repo := &mockUserRepository{}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.GetUserPokemon(context.Background(), GetUserPokemonRequest{
			UserID: "",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockUserRepository{
			getUserPokemonErr: errors.New("db error"),
		}
		svc := NewService(repo, nil, nil, privateKey, publicKey, "test-kid")

		_, err := svc.GetUserPokemon(context.Background(), GetUserPokemonRequest{
			UserID: "user-123",
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
