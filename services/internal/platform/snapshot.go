package platform

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Snapshot holds a point-in-time state of an aggregate.
type Snapshot struct {
	StreamID   string
	StreamType string
	Version    int
	State      json.RawMessage
}

// SnapshotStore manages aggregate snapshots.
type SnapshotStore struct {
	pool *pgxpool.Pool
}

// NewSnapshotStore creates a SnapshotStore. Returns nil if pool is nil.
func NewSnapshotStore(pool *pgxpool.Pool) *SnapshotStore {
	if pool == nil {
		return nil
	}
	return &SnapshotStore{pool: pool}
}

// Save persists a snapshot of the aggregate's current state.
func (s *SnapshotStore) Save(ctx context.Context, streamID, streamType string, version int, state any) error {
	if s == nil {
		return nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO snapshots (stream_id, stream_type, version, state, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (stream_id) DO UPDATE SET
		   version = EXCLUDED.version,
		   state = EXCLUDED.state,
		   created_at = NOW()`,
		streamID, streamType, version, data,
	)
	return err
}

// Load retrieves the latest snapshot for a stream. Returns nil if none exists.
func (s *SnapshotStore) Load(ctx context.Context, streamID string) (*Snapshot, error) {
	if s == nil {
		return nil, nil
	}
	var snap Snapshot
	err := s.pool.QueryRow(ctx,
		`SELECT stream_id, stream_type, version, state FROM snapshots WHERE stream_id = $1`,
		streamID,
	).Scan(&snap.StreamID, &snap.StreamType, &snap.Version, &snap.State)
	if err != nil {
		return nil, nil //nolint:nilerr // No snapshot is not an error.
	}
	return &snap, nil
}

// Snapshotable is an optional interface aggregates can implement for snapshot support.
type Snapshotable interface {
	MarshalSnapshot() ([]byte, error)
	UnmarshalSnapshot(data []byte) error
}

// LoadAggregateWithSnapshot loads an aggregate using snapshot + remaining events.
// If the aggregate implements Snapshotable, it will try to load a snapshot first
// and then replay only events after the snapshot version. Upcasters are applied
// to each event during replay if provided.
func LoadAggregateWithSnapshot(ctx context.Context, store *EventStore, snapshots *SnapshotStore, upcasters *UpcasterRegistry, agg Aggregate) error {
	if store == nil {
		return nil
	}

	fromVersion := 0

	// Try loading snapshot if aggregate supports it.
	if snappable, ok := agg.(Snapshotable); ok && snapshots != nil {
		snap, err := snapshots.Load(ctx, agg.AggregateID())
		if err != nil {
			slog.Warn("snapshot load failed, falling back to full replay", "error", err)
		}
		if snap != nil {
			if err := snappable.UnmarshalSnapshot(snap.State); err != nil {
				slog.Warn("snapshot unmarshal failed, falling back to full replay", "error", err)
			} else {
				agg.SetVersion(snap.Version)
				fromVersion = snap.Version
			}
		}
	}

	// Replay events after snapshot (or all events if no snapshot).
	events, err := store.LoadStreamFrom(ctx, agg.AggregateID(), fromVersion)
	if err != nil {
		return err
	}
	for _, e := range events {
		data := e.Data
		if upcasters != nil {
			data = upcasters.Apply(e.EventType, data)
		}
		agg.ApplyEvent(e.EventType, data)
	}
	if len(events) > 0 {
		agg.SetVersion(events[len(events)-1].Version)
	}
	return nil
}

// SaveSnapshot saves a snapshot if the aggregate supports it and has enough events.
func SaveSnapshot(ctx context.Context, snapshots *SnapshotStore, agg Aggregate, everyN int) error {
	if snapshots == nil {
		return nil
	}
	snappable, ok := agg.(Snapshotable)
	if !ok {
		return nil
	}
	if agg.Version() > 0 && agg.Version()%everyN == 0 {
		data, err := snappable.MarshalSnapshot()
		if err != nil {
			return err
		}
		return snapshots.Save(ctx, agg.AggregateID(), agg.StreamType(), agg.Version(), json.RawMessage(data))
	}
	return nil
}
