package greeter

import (
	"encoding/json"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// Event types for the greeting aggregate.
const (
	EventGreetingCreated     = "greeting.created"
	EventGreetingFailed      = "greeting.failed"
	EventGreetingCompensated = "greeting.compensated"
)

// GreetingCreatedData is the payload for greeting.created events.
type GreetingCreatedData struct {
	Name           string `json:"name"`
	Message        string `json:"message"`
	ExternalStatus int32  `json:"external_status"`
}

// GreetingFailedData is the payload for greeting.failed events.
type GreetingFailedData struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// GreetingCompensatedData is the payload for greeting.compensated events.
type GreetingCompensatedData struct {
	Reason string `json:"reason"`
}

// GreetingAggregate is the event-sourced aggregate for greetings.
type GreetingAggregate struct {
	platform.AggregateBase
	Name           string
	Message        string
	ExternalStatus int32
	Status         string // "created", "failed", "compensated"
}

func NewGreetingAggregate(id string) *GreetingAggregate {
	return &GreetingAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *GreetingAggregate) StreamType() string { return "greeting" }

// ApplyEvent applies a stored event to reconstruct state.
func (a *GreetingAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventGreetingCreated:
		var d GreetingCreatedData
		_ = json.Unmarshal(data, &d)
		a.Name = d.Name
		a.Message = d.Message
		a.ExternalStatus = d.ExternalStatus
		a.Status = "created"
	case EventGreetingFailed:
		var d GreetingFailedData
		_ = json.Unmarshal(data, &d)
		a.Name = d.Name
		a.Status = "failed"
	case EventGreetingCompensated:
		a.Status = "compensated"
	}
}

// Command methods below.

// Create records a successful greeting.
func (a *GreetingAggregate) Create(name, message string, externalStatus int32) {
	a.Raise(EventGreetingCreated, GreetingCreatedData{
		Name:           name,
		Message:        message,
		ExternalStatus: externalStatus,
	})
	// Apply immediately to keep in-memory state consistent
	a.Name = name
	a.Message = message
	a.ExternalStatus = externalStatus
	a.Status = "created"
}

// Fail records a failed greeting attempt.
func (a *GreetingAggregate) Fail(name string, reason string) {
	a.Raise(EventGreetingFailed, GreetingFailedData{
		Name:  name,
		Error: reason,
	})
	a.Name = name
	a.Status = "failed"
}

// Compensate marks this greeting as compensated.
func (a *GreetingAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return // idempotent
	}
	a.Raise(EventGreetingCompensated, GreetingCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// GreetingTopicMapper maps event types to Kafka topics.
func GreetingTopicMapper(eventType string) string {
	switch eventType {
	case EventGreetingCreated:
		return platform.TopicGreetingCreated
	case EventGreetingFailed:
		return platform.TopicGreetingFailed
	default:
		return ""
	}
}
