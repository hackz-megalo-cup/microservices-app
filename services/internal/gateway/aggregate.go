package gateway

import (
	"encoding/json"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// Event types for the invocation aggregate.
const (
	EventInvocationCreated     = "invocation.created"
	EventInvocationFailed      = "invocation.failed"
	EventInvocationCompensated = "invocation.compensated"
)

// InvocationCreatedData is the payload for invocation.created events.
type InvocationCreatedData struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// InvocationFailedData is the payload for invocation.failed events.
type InvocationFailedData struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

// InvocationCompensatedData is the payload for invocation.compensated events.
type InvocationCompensatedData struct {
	Reason string `json:"reason"`
}

// InvocationAggregate is the event-sourced aggregate for invocations.
type InvocationAggregate struct {
	platform.AggregateBase
	Name    string
	Message string
	Status  string // "completed", "failed", "compensated".
}

func NewInvocationAggregate(id string) *InvocationAggregate {
	return &InvocationAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *InvocationAggregate) StreamType() string { return "invocation" }

// ApplyEvent applies a stored event to reconstruct state.
func (a *InvocationAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventInvocationCreated:
		var d InvocationCreatedData
		_ = json.Unmarshal(data, &d)
		a.Name = d.Name
		a.Message = d.Message
		a.Status = "completed"
	case EventInvocationFailed:
		var d InvocationFailedData
		_ = json.Unmarshal(data, &d)
		a.Name = d.Name
		a.Status = "failed"
	case EventInvocationCompensated:
		a.Status = "compensated"
	}
}

// Create records a successful invocation.
func (a *InvocationAggregate) Create(name, message string) {
	a.Raise(EventInvocationCreated, InvocationCreatedData{
		Name:    name,
		Message: message,
	})
	a.Name = name
	a.Message = message
	a.Status = "completed"
}

// Fail records a failed invocation.
func (a *InvocationAggregate) Fail(name, reason string) {
	a.Raise(EventInvocationFailed, InvocationFailedData{
		Name:  name,
		Error: reason,
	})
	a.Name = name
	a.Status = "failed"
}

// Compensate marks this invocation as compensated.
func (a *InvocationAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return // Idempotent.
	}
	a.Raise(EventInvocationCompensated, InvocationCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// InvocationTopicMapper maps event types to Kafka topics.
func InvocationTopicMapper(eventType string) string {
	switch eventType {
	case EventInvocationCreated:
		return platform.TopicInvocationCreated
	case EventInvocationFailed:
		return platform.TopicInvocationFailed
	default:
		return ""
	}
}
