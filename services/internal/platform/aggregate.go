package platform

import (
	"context"
	"encoding/json"
	"log/slog"
)

// Aggregate is the interface that all event-sourced aggregates must implement.
type Aggregate interface {
	AggregateID() string
	StreamType() string
	Version() int
	SetVersion(int)
	Changes() []UnsavedEvent
	ClearChanges()
	ApplyEvent(eventType string, data json.RawMessage)
}

// AggregateBase provides common fields and methods for aggregates.
type AggregateBase struct {
	id      string
	version int
	changes []UnsavedEvent
}

func NewAggregateBase(id string) AggregateBase {
	return AggregateBase{id: id}
}

func (a *AggregateBase) AggregateID() string     { return a.id }
func (a *AggregateBase) Version() int            { return a.version }
func (a *AggregateBase) SetVersion(v int)        { a.version = v }
func (a *AggregateBase) Changes() []UnsavedEvent { return a.changes }
func (a *AggregateBase) ClearChanges()           { a.changes = nil }

// Raise adds an uncommitted event. The concrete aggregate should call this
// from its command methods, then immediately apply the state change.
func (a *AggregateBase) Raise(eventType string, data any) {
	a.changes = append(a.changes, UnsavedEvent{Type: eventType, Data: data})
}

// LoadAggregate replays stored events onto the aggregate.
func LoadAggregate(ctx context.Context, store *EventStore, agg Aggregate) error {
	if store == nil {
		return nil
	}
	events, err := store.LoadStream(ctx, agg.AggregateID())
	if err != nil {
		return err
	}
	for _, e := range events {
		agg.ApplyEvent(e.EventType, e.Data)
	}
	if len(events) > 0 {
		agg.SetVersion(events[len(events)-1].Version)
	}
	return nil
}

// TopicMapper maps an event type to a Kafka topic.
type TopicMapper func(eventType string) string

// SaveAggregate persists uncommitted events to the event store and outbox
// within a single transaction.
func SaveAggregate(ctx context.Context, store *EventStore, outbox *OutboxStore, agg Aggregate, topicMapper TopicMapper) error {
	changes := agg.Changes()
	if len(changes) == 0 {
		return nil
	}
	if store == nil {
		return nil
	}

	tx, err := store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	newVersion, err := store.AppendToStream(ctx, tx, agg.AggregateID(), agg.StreamType(), agg.Version(), changes)
	if err != nil {
		return err
	}

	// Also insert into outbox for Kafka publishing
	if outbox != nil && topicMapper != nil {
		for _, change := range changes {
			topic := topicMapper(change.Type)
			if topic == "" {
				continue
			}
			event := NewEvent(change.Type, agg.StreamType()+"-service", change.Data)
			// Add stream_id to event data for downstream consumers
			enrichedData := map[string]any{
				"stream_id": agg.AggregateID(),
			}
			if m, ok := change.Data.(map[string]any); ok {
				for k, v := range m {
					enrichedData[k] = v
				}
			}
			event.Data = enrichedData
			if outboxErr := outbox.InsertEvent(ctx, tx, topic, event); outboxErr != nil {
				slog.Error("failed to insert outbox event", "type", change.Type, "error", outboxErr)
				err = outboxErr
				return err
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	agg.SetVersion(newVersion)
	agg.ClearChanges()
	return nil
}
