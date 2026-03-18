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
		a.CreatedAt = d.OccurredAt

	case EventUserLoggedIn:
		var d UserLoggedInData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal UserLoggedInData", "error", err)
		}
		occurredAt := d.OccurredAt
		a.LastLoginAt = &occurredAt
	}
}

// RegisterUser records a user registration
func (a *UserAggregate) RegisterUser(email string, passwordHash string, occurredAt time.Time) {
	a.Raise(EventUserRegistered, UserRegisteredData{
		UserID:     a.AggregateID(),
		Email:      email,
		Role:       "user",
		OccurredAt: occurredAt,
	})
	a.Email = email
	a.PasswordHash = passwordHash
	a.Role = "user"
	a.CreatedAt = occurredAt
}

// LoggedIn records a user login
func (a *UserAggregate) LoggedIn(occurredAt time.Time) {
	isFirstToday := a.LastLoginAt == nil || isFirstLoginToday(a.LastLoginAt, occurredAt)
	a.Raise(EventUserLoggedIn, UserLoggedInData{
		UserID:       a.AggregateID(),
		IsFirstToday: isFirstToday,
		OccurredAt:   occurredAt,
	})
	a.LastLoginAt = &occurredAt
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

// isFirstLoginToday checks if the login on the given timestamp is the first for that day,
// using the provided reference time instead of the current time.
func isFirstLoginToday(lastTime *time.Time, now time.Time) bool {
	if lastTime == nil {
		return true
	}
	lastDate := lastTime.UTC()
	today := now.UTC()
	return lastDate.Year() != today.Year() ||
		lastDate.Month() != today.Month() ||
		lastDate.Day() != today.Day()
}
