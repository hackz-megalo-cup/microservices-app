package auth

import (
	"fmt"
	"time"
)

// User aggregate root
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	LastLoginAt  *time.Time
	Version      int
}

// NewUser creates a new user aggregate
func NewUser(id, email, passwordHash, role string, createdAt time.Time) *User {
	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    createdAt,
		Version:      0,
	}
}

// ApplyEvent applies a domain event to the user aggregate
func (u *User) ApplyEvent(event DomainEvent) error {
	switch e := event.(type) {
	case UserRegistered:
		u.ID = e.UserID
		u.Email = e.Email
		u.Role = e.Role
		u.CreatedAt = e.Timestamp
		u.Version++
		return nil

	case UserLoggedIn:
		u.LastLoginAt = &e.Timestamp
		u.Version++
		return nil

	default:
		return fmt.Errorf("unknown event type: %T", event)
	}
}

// IsFirstToday checks if this is the user's first login today
func (u *User) IsFirstToday() bool {
	if u.LastLoginAt == nil {
		return true
	}

	lastDate := u.LastLoginAt.Local()
	today := time.Now().Local()

	return lastDate.Year() != today.Year() ||
		lastDate.Month() != today.Month() ||
		lastDate.Day() != today.Day()
}
