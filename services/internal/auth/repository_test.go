package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		user := &User{
			ID:           "user-123",
			Email:        "test@example.com",
			PasswordHash: "hashed",
			Role:         "user",
			CreatedAt:    time.Now(),
		}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "role", "created_at", "last_login_at"}).
			AddRow("user-123", "test@example.com", "hashed", "user", time.Now(), nil)

		mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
			WithArgs("test@example.com").
			WillReturnRows(rows)

		user, err := repo.GetByEmail(context.Background(), "test@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Email != "test@example.com" {
			t.Errorf("got email %q, want %q", user.Email, "test@example.com")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
			WithArgs("nonexistent@example.com").
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetByEmail(context.Background(), "nonexistent@example.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "role", "created_at", "last_login_at"}).
			AddRow("user-123", "test@example.com", "hashed", "user", time.Now(), nil)

		mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
			WithArgs("user-123").
			WillReturnRows(rows)

		user, err := repo.GetByID(context.Background(), "user-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != "user-123" {
			t.Errorf("got ID %q, want %q", user.ID, "user-123")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM users WHERE id").
			WithArgs("nonexistent").
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetByID(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"last_login_at"}).
			AddRow(now)

		mock.ExpectQuery("UPDATE users SET last_login_at").
			WithArgs("user-123").
			WillReturnRows(rows)

		lastLoginAt, err := repo.UpdateLastLogin(context.Background(), "user-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lastLoginAt == nil {
			t.Fatal("expected timestamp, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}

func TestUserRepository_RegisterPokemon(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO user_pokemon").
			WithArgs("user-123", "pikachu").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.RegisterPokemon(context.Background(), "user-123", "pikachu")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})

	t.Run("conflict (idempotent)", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO user_pokemon").
			WithArgs("user-123", "pikachu").
			WillReturnResult(sqlmock.NewResult(0, 0)) // ON CONFLICT DO NOTHING

		err := repo.RegisterPokemon(context.Background(), "user-123", "pikachu")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}

func TestUserRepository_GetUserPokemon(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)

	t.Run("success with results", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"pokemon_id"}).
			AddRow("pikachu").
			AddRow("charmander")

		mock.ExpectQuery("SELECT pokemon_id FROM user_pokemon WHERE user_id").
			WithArgs("user-123").
			WillReturnRows(rows)

		list, err := repo.GetUserPokemon(context.Background(), "user-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("got %d pokemon, want 2", len(list))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})

	t.Run("success with empty results", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"pokemon_id"})

		mock.ExpectQuery("SELECT pokemon_id FROM user_pokemon WHERE user_id").
			WithArgs("user-456").
			WillReturnRows(rows)

		list, err := repo.GetUserPokemon(context.Background(), "user-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("got %d pokemon, want 0", len(list))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled expectations: %v", err)
		}
	})
}
