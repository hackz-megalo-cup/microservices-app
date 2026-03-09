package order

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ==========================================================.
// Aggregate — your domain entity with event sourcing.
// Add fields that represent the current state.
// ==========================================================.

type OrderAggregate struct {
	platform.AggregateBase
	Input  string
	Output string
	Status string // "created", "failed", "compensated"
}

func NewOrderAggregate(id string) *OrderAggregate {
	return &OrderAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *OrderAggregate) StreamType() string { return "order" }

// ApplyEvent replays a stored event to reconstruct state.
// This is called when loading an aggregate from the event store.
func (a *OrderAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventOrderCreated:
		var d OrderCreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		a.Input = d.Input
		a.Output = d.Output
		a.Status = "created"
	case EventOrderFailed:
		var d OrderFailedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal failed data", "error", err)
		}
		a.Input = d.Input
		a.Status = "failed"
	case EventOrderCompensated:
		a.Status = "compensated"
	}
}

// ==========================================================.
// Command methods — business actions that produce events.
// ==========================================================.

// Create records a successful operation.
func (a *OrderAggregate) Create(input, output string) {
	a.Raise(EventOrderCreated, OrderCreatedData{
		Input:  input,
		Output: output,
	})
	a.Input = input
	a.Output = output
	a.Status = "created"
}

// Fail records a failed operation.
func (a *OrderAggregate) Fail(input string, reason string) {
	a.Raise(EventOrderFailed, OrderFailedData{
		Input: input,
		Error: reason,
	})
	a.Input = input
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated.
func (a *OrderAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventOrderCompensated, OrderCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// OrderTopicMapper maps event types to Kafka topics.
func OrderTopicMapper(eventType string) string {
	switch eventType {
	case EventOrderCreated:
		return platform.TopicOrderCreated
	case EventOrderFailed:
		return platform.TopicOrderFailed
	default:
		return ""
	}
}
