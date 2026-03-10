package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoredEvent is an event loaded from the event store.
type StoredEvent struct {
	StreamID       string
	Version        int
	EventType      string
	Data           json.RawMessage
	Metadata       json.RawMessage
	CreatedAt      time.Time
	GlobalPosition int64
}

// UnsavedEvent represents an event to be appended to a stream.
type UnsavedEvent struct {
	Type     string
	Data     any
	Metadata map[string]string
}

// ErrConcurrencyConflict is returned when optimistic concurrency check fails.
var ErrConcurrencyConflict = fmt.Errorf("event store: concurrency conflict")

// EventStore provides append and load operations on event streams.
type EventStore struct {
	pool *pgxpool.Pool
}

// NewEventStore creates an EventStore. Returns nil if pool is nil.
func NewEventStore(pool *pgxpool.Pool) *EventStore {
	if pool == nil {
		return nil
	}
	return &EventStore{pool: pool}
}

// AppendToStream appends events to a stream within the given transaction.
// ExpectedVersion is the last known version (0 for new streams).
// Returns the new version after appending.
func (s *EventStore) AppendToStream(ctx context.Context, tx pgx.Tx, streamID, streamType string, expectedVersion int, events []UnsavedEvent) (int, error) {
	if s == nil || len(events) == 0 {
		return expectedVersion, nil
	}

	// Acquire a transaction-scoped advisory lock for this stream to prevent
	// concurrent appends. We use hashtext() so the stream_id string maps to
	// a stable int suitable for pg_advisory_xact_lock.
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`, streamID,
	); err != nil {
		return 0, fmt.Errorf("event store: advisory lock: %w", err)
	}

	var currentVersion int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM event_store WHERE stream_id = $1`,
		streamID,
	).Scan(&currentVersion)
	if err != nil {
		return 0, fmt.Errorf("event store: check version: %w", err)
	}
	if currentVersion != expectedVersion {
		return 0, ErrConcurrencyConflict
	}

	version := expectedVersion
	for _, e := range events {
		version++
		data, err := json.Marshal(e.Data)
		if err != nil {
			return 0, fmt.Errorf("event store: marshal data: %w", err)
		}
		var metadata []byte
		if e.Metadata != nil {
			metadata, _ = json.Marshal(e.Metadata)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO event_store (stream_id, stream_type, version, event_id, event_type, data, metadata, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			streamID, streamType, version, uuid.NewString(), e.Type, data, metadata, time.Now().UTC(),
		)
		if err != nil {
			return 0, fmt.Errorf("event store: insert event: %w", err)
		}
	}
	return version, nil
}

// LoadStream loads all events for a stream, ordered by version.
func (s *EventStore) LoadStream(ctx context.Context, streamID string) ([]StoredEvent, error) {
	return s.LoadStreamFrom(ctx, streamID, 0)
}

// LoadStreamFrom loads events for a stream starting from a given version (exclusive).
func (s *EventStore) LoadStreamFrom(ctx context.Context, streamID string, fromVersion int) ([]StoredEvent, error) {
	if s == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT stream_id, version, event_type, data, metadata, created_at
		 FROM event_store
		 WHERE stream_id = $1 AND version > $2
		 ORDER BY version ASC`,
		streamID, fromVersion,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []StoredEvent
	for rows.Next() {
		var e StoredEvent
		if err := rows.Scan(&e.StreamID, &e.Version, &e.EventType, &e.Data, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// LoadAllEvents loads all events across all streams from a given global position.
// Used for projection rebuilds.
func (s *EventStore) LoadAllEvents(ctx context.Context, afterPosition int64, limit int) ([]StoredEvent, error) {
	if s == nil {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT stream_id, version, event_type, data, metadata, created_at, global_position
		 FROM event_store
		 WHERE global_position > $1
		 ORDER BY global_position ASC
		 LIMIT $2`,
		afterPosition, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []StoredEvent
	for rows.Next() {
		var e StoredEvent
		if err := rows.Scan(&e.StreamID, &e.Version, &e.EventType, &e.Data, &e.Metadata, &e.CreatedAt, &e.GlobalPosition); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// BeginTx starts a new transaction.
func (s *EventStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}
