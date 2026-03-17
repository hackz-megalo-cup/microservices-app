package projector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"hash/fnv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/item"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ProjectionHandler updates read models based on events.
type ProjectionHandler struct {
	pool       *pgxpool.Pool
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
}

// NewProjectionHandler creates a new ProjectionHandler.
func NewProjectionHandler(pool *pgxpool.Pool, eventStore *platform.EventStore, outbox *platform.OutboxStore) *ProjectionHandler {
	if pool == nil {
		return nil
	}
	return &ProjectionHandler{pool: pool, eventStore: eventStore, outbox: outbox}
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
	case "user.logged_in":
		return h.onUserLoggedIn(ctx, event)
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

func (h *ProjectionHandler) onUserLoggedIn(ctx context.Context, event platform.Event) error {
	data, err := toMap(event.Data)
	if err != nil {
		slog.Warn("projection: invalid user.logged_in data", "event_id", event.ID)
		return nil //nolint:nilerr // Intentionally skip malformed events.
	}

	// is_first_today = true の場合のみ付与
	isFirstToday, _ := data["is_first_today"].(bool)
	if !isFirstToday {
		return nil // 今日初回ログインじゃない → スキップ
	}

	userID, _ := data["user_id"].(string)
	if userID == "" {
		return nil
	}

	if h.eventStore == nil || h.outbox == nil {
		slog.Warn("login bonus skipped: item event store not configured", "user_id", userID)
		return nil
	}

	// アイテムを選定（イベントIDから決定的に選ぶ → リプレイ安全）
	itemID := selectItem(event.ID)
	quantity := int32(1)

	// 集約をロードして付与
	aggID := fmt.Sprintf("%s:%s", userID, itemID)
	agg := item.NewItemAggregate(aggID)
	if err := platform.LoadAggregate(ctx, h.eventStore, agg); err != nil {
		slog.Warn("load aggregate (may be new)", "error", err)
	}
	agg.Grant(userID, itemID, quantity, "login_bonus")

	if err := platform.SaveAggregate(ctx, h.eventStore, h.outbox, agg, item.ItemTopicMapper); err != nil {
		slog.Error("failed to grant login bonus", "error", err)
		return err
	}

	slog.Info("login bonus granted", "user_id", userID, "item_id", itemID)
	return nil
}

// selectItem はイベントIDから決定的にアイテムを選ぶ。
// 同じイベントIDなら常に同じアイテムを返すため、リプレイ時も安全。
func selectItem(eventID string) string {
	items := []string{"potion", "super_ball", "revive", "lure"}
	h := fnv.New32a()
	_, _ = h.Write([]byte(eventID)) //nolint:errcheck // hash.Write never fails.
	return items[h.Sum32()%uint32(len(items))]
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
