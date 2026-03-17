package auth

import "time"

// UserRegistered domain event
type UserRegistered struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Timestamp time.Time `json:"timestamp"`
}

// EventType returns the event type name
func (e UserRegistered) EventType() string {
	return "user.registered"
}

// UserLoggedIn domain event
type UserLoggedIn struct {
	UserID       string    `json:"user_id"`
	IsFirstToday bool      `json:"is_first_today"`
	Timestamp    time.Time `json:"timestamp"`
}

// EventType returns the event type name
func (e UserLoggedIn) EventType() string {
	return "user.logged_in"
}

// PokemonCaught domain event (from capture service)
type PokemonCaught struct {
	UserID    string    `json:"user_id"`
	PokemonID string    `json:"pokemon_id"`
	CaughtAt  time.Time `json:"caught_at"`
}

// EventType returns the event type name
func (e PokemonCaught) EventType() string {
	return "capture.completed"
}

// DomainEvent interface
type DomainEvent interface {
	EventType() string
}
