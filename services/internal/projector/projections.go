package projector

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ProjectionHandler updates read models based on events.
type ProjectionHandler struct {
	pool *pgxpool.Pool
}

// NewProjectionHandler creates a new ProjectionHandler.
func NewProjectionHandler(pool *pgxpool.Pool) *ProjectionHandler {
	if pool == nil {
		return nil
	}
	return &ProjectionHandler{pool: pool}
}

// HandleEvent dispatches an event to the appropriate projection.
func (h *ProjectionHandler) HandleEvent(ctx context.Context, event platform.Event) error {
	if h == nil {
		return nil
	}
	switch event.Type {
	case "greeting.created":
		return h.onGreetingCreated(ctx, event)
	case "greeting.failed":
		return h.onGreetingFailed(ctx, event)
	case "greeting.compensated":
		return h.onGreetingCompensated(ctx, event)
	case "invocation.created":
		return h.onInvocationCreated(ctx, event)
	case "invocation.failed":
		return h.onInvocationFailed(ctx, event)
	case "invocation.compensated":
		return h.onInvocationCompensated(ctx, event)
	default:
		return nil
	}
}

func (h *ProjectionHandler) onGreetingCreated(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid greeting.created data", "event_id", event.ID)
		return nil //nolint:nilerr // Intentionally skip malformed events.
	}
	streamID, _ := data["stream_id"].(string)
	if streamID == "" {
		streamID = event.ID // fallback for legacy events
	}
	name, _ := data["name"].(string)
	message, _ := data["message"].(string)
	var extStatus int32
	if v, ok := data["external_status"].(float64); ok {
		extStatus = int32(v)
	}

	_, err = h.pool.Exec(ctx,
		`INSERT INTO greetings_view (id, name, message, external_status, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'created', NOW(), NOW())
		 ON CONFLICT (id) DO UPDATE SET
		   name = EXCLUDED.name,
		   message = EXCLUDED.message,
		   external_status = EXCLUDED.external_status,
		   status = 'created',
		   updated_at = NOW()`,
		streamID, name, message, extStatus,
	)
	if err != nil {
		slog.Error("projection: failed to upsert greetings_view", "error", err)
	}
	return err
}

func (h *ProjectionHandler) onGreetingFailed(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid greeting.failed data", "event_id", event.ID)
		return nil //nolint:nilerr // Intentionally skip malformed events.
	}
	streamID, _ := data["stream_id"].(string)
	if streamID == "" {
		streamID = event.ID
	}
	name, _ := data["name"].(string)

	_, err = h.pool.Exec(ctx,
		`INSERT INTO greetings_view (id, name, status, created_at, updated_at)
		 VALUES ($1, $2, 'failed', NOW(), NOW())
		 ON CONFLICT (id) DO UPDATE SET
		   status = 'failed',
		   updated_at = NOW()`,
		streamID, name,
	)
	if err != nil {
		slog.Error("projection: failed to upsert greetings_view (failed)", "error", err)
	}
	return err
}

func (h *ProjectionHandler) onGreetingCompensated(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid greeting.compensated data", "event_id", event.ID)
		return nil //nolint:nilerr // Intentionally skip malformed events.
	}
	streamID, _ := data["stream_id"].(string)
	if streamID == "" {
		streamID = event.ID
	}

	_, err = h.pool.Exec(ctx,
		`UPDATE greetings_view SET status = 'compensated', updated_at = NOW() WHERE id = $1`,
		streamID,
	)
	if err != nil {
		slog.Error("projection: failed to update greetings_view (compensated)", "error", err)
	}
	return err
}

func (h *ProjectionHandler) onInvocationCreated(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid invocation.created data", "event_id", event.ID)
		return nil //nolint:nilerr // Invalid data is not retryable.
	}
	streamID, _ := data["stream_id"].(string)
	if streamID == "" {
		streamID = event.ID
	}
	name, _ := data["name"].(string)
	message, _ := data["message"].(string)

	_, err = h.pool.Exec(ctx,
		`INSERT INTO invocations_view (id, name, message, status, created_at, updated_at)
		 VALUES ($1, $2, $3, 'completed', NOW(), NOW())
		 ON CONFLICT (id) DO UPDATE SET
		   name = EXCLUDED.name,
		   message = EXCLUDED.message,
		   status = 'completed',
		   updated_at = NOW()`,
		streamID, name, message,
	)
	if err != nil {
		slog.Error("projection: failed to upsert invocations_view", "error", err)
	}
	return err
}

func (h *ProjectionHandler) onInvocationFailed(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid invocation.failed data", "event_id", event.ID)
		return nil //nolint:nilerr // Invalid data is not retryable.
	}
	streamID, _ := data["stream_id"].(string)
	if streamID == "" {
		streamID = event.ID
	}
	name, _ := data["name"].(string)

	_, err = h.pool.Exec(ctx,
		`INSERT INTO invocations_view (id, name, status, created_at, updated_at)
		 VALUES ($1, $2, 'failed', NOW(), NOW())
		 ON CONFLICT (id) DO UPDATE SET
		   status = 'failed',
		   updated_at = NOW()`,
		streamID, name,
	)
	if err != nil {
		slog.Error("projection: failed to upsert invocations_view (failed)", "error", err)
	}
	return err
}

func (h *ProjectionHandler) onInvocationCompensated(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid invocation.compensated data", "event_id", event.ID)
		return nil //nolint:nilerr // Invalid data is not retryable.
	}
	streamID, _ := data["stream_id"].(string)
	if streamID == "" {
		streamID = event.ID
	}

	_, err = h.pool.Exec(ctx,
		`UPDATE invocations_view SET status = 'compensated', updated_at = NOW() WHERE id = $1`,
		streamID,
	)
	if err != nil {
		slog.Error("projection: failed to update invocations_view (compensated)", "error", err)
	}
	return err
}

func toMap(data any) (map[string]any, error) {
	switch v := data.(type) {
	case map[string]any:
		return v, nil
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		var m map[string]any
		return m, json.Unmarshal(b, &m)
	}
}
