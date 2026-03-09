package platform

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxStore manages the transactional outbox pattern.
type OutboxStore struct {
	pool      *pgxpool.Pool
	publisher *EventPublisher
}

// NewOutboxStore creates an OutboxStore. Returns nil if pool is nil.
func NewOutboxStore(pool *pgxpool.Pool, publisher *EventPublisher) *OutboxStore {
	if pool == nil {
		return nil
	}
	return &OutboxStore{pool: pool, publisher: publisher}
}

// InsertEvent writes an event to the outbox table within the given transaction.
func (s *OutboxStore) InsertEvent(ctx context.Context, tx pgx.Tx, topic string, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO outbox_events (id, event_type, topic, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		event.ID, event.Type, topic, payload, event.Timestamp,
	)
	return err
}

// PublishPending polls unpublished events and publishes them to Kafka.
func (s *OutboxStore) PublishPending(ctx context.Context) (int, error) {
	if s == nil || s.publisher == nil {
		return 0, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, event_type, topic, payload
		 FROM outbox_events
		 WHERE published = FALSE
		 ORDER BY created_at ASC
		 LIMIT 50
		 FOR UPDATE SKIP LOCKED`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type outboxRow struct {
		id        string
		eventType string
		topic     string
		payload   []byte
	}

	var pending []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.id, &r.eventType, &r.topic, &r.payload); err != nil {
			return 0, err
		}
		pending = append(pending, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	published := 0
	for _, r := range pending {
		if err := s.publisher.PublishRaw(ctx, r.topic, r.id, r.payload); err != nil {
			slog.Error("outbox: failed to publish event", "id", r.id, "topic", r.topic, "error", err)
			continue
		}
		if _, err := s.pool.Exec(ctx,
			`UPDATE outbox_events SET published = TRUE, published_at = $1 WHERE id = $2`,
			time.Now().UTC(), r.id,
		); err != nil {
			slog.Error("outbox: failed to mark event as published", "id", r.id, "error", err)
		}
		published++
	}
	return published, nil
}

// Cleanup deletes published events older than the given duration.
func (s *OutboxStore) Cleanup(ctx context.Context, maxAge time.Duration) error {
	if s == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM outbox_events WHERE published = TRUE AND published_at < $1`,
		time.Now().UTC().Add(-maxAge),
	)
	return err
}

// StartPoller runs a background goroutine that polls for unpublished events.
// It uses exponential backoff: the interval doubles (up to 5s) when idle,
// and resets to the base interval when events are found.
func (s *OutboxStore) StartPoller(ctx context.Context, interval time.Duration) {
	if s == nil {
		return
	}
	const maxInterval = 5 * time.Second
	go func() {
		current := interval
		timer := time.NewTimer(current)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				n, err := s.PublishPending(ctx)
				switch {
				case err != nil:
					slog.Error("outbox poller error", "error", err)
					current = min(current*2, maxInterval)
				case n > 0:
					slog.Info("outbox poller published events", "count", n)
					current = interval
				default:
					current = min(current*2, maxInterval)
				}
				timer.Reset(current)
			}
		}
	}()
}

// BeginTx starts a new transaction from the pool.
func (s *OutboxStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}
