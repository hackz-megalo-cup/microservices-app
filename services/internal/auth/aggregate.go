package auth

import (
	"encoding/json"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// UserAggregate is the event-sourced aggregate for users
type UserAggregate struct {
	platform.AggregateBase
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

// NewUserAggregate creates a new user aggregate with the given ID
func NewUserAggregate(id string) *UserAggregate {
	return &UserAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

// StreamType returns the stream type for this aggregate
func (a *UserAggregate) StreamType() string { return "user" }

// ApplyEvent applies a stored event to reconstruct state
func (a *UserAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventUserRegistered:
		var d UserRegisteredData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal UserRegisteredData", "error", err)
		}
		a.Email = d.Email
		a.Role = d.Role
		a.CreatedAt = time.Now()

	case EventUserLoggedIn:
		var d UserLoggedInData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal UserLoggedInData", "error", err)
		}
		now := time.Now()
		a.LastLoginAt = &now
	}
}

// RegisterUser records a user registration
func (a *UserAggregate) RegisterUser(email string, passwordHash string) {
	a.Raise(EventUserRegistered, UserRegisteredData{
		UserID: a.AggregateID(),
		Email:  email,
		Role:   "user",
	})
	a.Email = email
	a.PasswordHash = passwordHash
	a.Role = "user"
	a.CreatedAt = time.Now()
}

// LoggedIn records a user login
func (a *UserAggregate) LoggedIn() {
	isFirstToday := a.LastLoginAt == nil || isFirstToday(a.LastLoginAt)
	a.Raise(EventUserLoggedIn, UserLoggedInData{
		UserID:       a.AggregateID(),
		IsFirstToday: isFirstToday,
	})
	now := time.Now()
	a.LastLoginAt = &now
}

// Fail records a failed operation (for saga compensation)
func (a *UserAggregate) Fail(reason string) {
	a.Raise(EventUserFailed, UserFailedData{
		UserID: a.AggregateID(),
		Error:  reason,
	})
}

// Compensate compensates a failed operation
func (a *UserAggregate) Compensate(reason string) {
	a.Raise(EventUserCompensated, UserCompensatedData{
		Reason: reason,
	})
}

// VerifyPassword verifies the user's password
func (a *UserAggregate) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password))
	return err == nil
}

// UserTopicMapper maps event types to Kafka topics
func UserTopicMapper(eventType string) string {
	switch eventType {
	case EventUserRegistered:
		return platform.TopicUserRegistered
	case EventUserLoggedIn:
		return platform.TopicUserLoggedIn
	case EventUserFailed:
		return platform.TopicUserFailed
	case EventUserCompensated:
		return platform.TopicUserCompensated
	default:
		return ""
	}
}

// Helper functions

// isFirstToday checks if the given time is from before today
func isFirstToday(lastTime *time.Time) bool {
	if lastTime == nil {
		return true
	}
	lastDate := lastTime.Local()
	today := time.Now().Local()
	return lastDate.Year() != today.Year() ||
		lastDate.Month() != today.Month() ||
		lastDate.Day() != today.Day()
}
