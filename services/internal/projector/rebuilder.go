package projector

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

const (
	// ProjectionGreetingsView is the name of the greetings view projection.
	ProjectionGreetingsView = "greetings_view"
	rebuildBatchSize        = 500
)

// Rebuilder replays events from the event store to rebuild projections.
type Rebuilder struct {
	pool       *pgxpool.Pool
	eventStore *platform.EventStore
	projection *ProjectionHandler
}

// NewRebuilder creates a Rebuilder. Returns nil if pool is nil.
func NewRebuilder(pool *pgxpool.Pool, eventStore *platform.EventStore, projection *ProjectionHandler) *Rebuilder {
	if pool == nil {
		return nil
	}
	return &Rebuilder{pool: pool, eventStore: eventStore, projection: projection}
}

// Rebuild replays all events from the event store and rebuilds the named projection from scratch.
func (r *Rebuilder) Rebuild(ctx context.Context, projectionName string) error {
	if r == nil || r.eventStore == nil || r.projection == nil {
		return nil
	}

	slog.Info("rebuild: starting", "projection", projectionName)

	// Truncate the target view.
	if projectionName == ProjectionGreetingsView {
		if _, err := r.pool.Exec(ctx, "TRUNCATE greetings_view"); err != nil {
			return err
		}
	}

	// Reset checkpoint.
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO projection_checkpoints (projection_name, last_position, updated_at)
		 VALUES ($1, 0, NOW())
		 ON CONFLICT (projection_name) DO UPDATE SET last_position = 0, updated_at = NOW()`,
		projectionName,
	); err != nil {
		return err
	}

	var lastPosition int64
	totalProcessed := 0

	for {
		events, err := r.eventStore.LoadAllEvents(ctx, lastPosition, rebuildBatchSize)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}

		for _, stored := range events {
			evt := platform.Event{
				Type: stored.EventType,
			}
			// Enrich with stream_id for projection handlers.
			var dataMap map[string]any
			if err := json.Unmarshal(stored.Data, &dataMap); err == nil {
				dataMap["stream_id"] = stored.StreamID
				evt.Data = dataMap
			}

			if err := r.projection.HandleEvent(ctx, evt); err != nil {
				slog.Error("rebuild: projection error", "event_type", stored.EventType, "error", err)
			}
			lastPosition = stored.GlobalPosition
			totalProcessed++
		}

		// Update checkpoint after each batch.
		if _, err := r.pool.Exec(ctx,
			`UPDATE projection_checkpoints SET last_position = $1, updated_at = NOW() WHERE projection_name = $2`,
			lastPosition, projectionName,
		); err != nil {
			return err
		}

		slog.Info("rebuild: batch processed", "projection", projectionName, "processed", totalProcessed, "last_position", lastPosition)
	}

	slog.Info("rebuild: complete", "projection", projectionName, "total_events", totalProcessed)
	return nil
}

// GetCheckpoint returns the last processed global position for a projection.
func (r *Rebuilder) GetCheckpoint(ctx context.Context, projectionName string) (int64, error) {
	if r == nil {
		return 0, nil
	}
	var pos int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(last_position, 0) FROM projection_checkpoints WHERE projection_name = $1`,
		projectionName,
	).Scan(&pos)
	if err != nil {
		return 0, nil //nolint:nilerr // No checkpoint yet is not an error.
	}
	return pos, nil
}
