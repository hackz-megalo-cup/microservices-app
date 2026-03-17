package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// UserRepository handles user data access
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, role, created_at, last_login_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.LastLoginAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	user := &User{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, role, created_at, last_login_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt, &user.LastLoginAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt, time.Now(),
	)
	return err
}

// UpdateLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) (*time.Time, error) {
	var lastLoginAt time.Time
	err := r.db.QueryRowContext(
		ctx,
		`UPDATE users SET last_login_at = NOW() WHERE id = $1 RETURNING last_login_at`,
		userID,
	).Scan(&lastLoginAt)

	if err != nil {
		return nil, err
	}

	return &lastLoginAt, nil
}

// RegisterPokemon registers a caught pokemon for a user
func (r *UserRepository) RegisterPokemon(ctx context.Context, userID, pokemonID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO user_pokemon (user_id, pokemon_id, caught_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id, pokemon_id) DO NOTHING`,
		userID, pokemonID,
	)
	return err
}

// GetUserPokemon retrieves all pokemon caught by a user
func (r *UserRepository) GetUserPokemon(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT pokemon_id FROM user_pokemon WHERE user_id = $1 ORDER BY caught_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pokemonList []string
	for rows.Next() {
		var pokemonID string
		if err := rows.Scan(&pokemonID); err != nil {
			return nil, err
		}
		pokemonList = append(pokemonList, pokemonID)
	}

	return pokemonList, rows.Err()
}
