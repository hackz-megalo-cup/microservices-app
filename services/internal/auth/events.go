package auth

import "time"

// Event type constants
const (
	EventUserRegistered  = "user.registered"
	EventUserLoggedIn    = "user.logged_in"
	EventUserFailed      = "user.failed"
	EventUserCompensated = "user.compensated"
)

// UserRegisteredData is the payload for user.registered events
type UserRegisteredData struct {
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	OccurredAt time.Time `json:"occurred_at"`
}

// UserLoggedInData is the payload for user.logged_in events
type UserLoggedInData struct {
	UserID       string    `json:"user_id"`
	IsFirstToday bool      `json:"is_first_today"`
	OccurredAt   time.Time `json:"occurred_at"`
}

// UserFailedData is the payload for user.failed events
type UserFailedData struct {
	UserID string `json:"user_id"`
	Error  string `json:"error"`
}

// UserCompensatedData is the payload for user.compensated events
type UserCompensatedData struct {
	Reason string `json:"reason"`
}
