package greeter

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// GreetingProjection polls the event store and materializes greeting events
// into the greetings read-model table. This is the "read side" of CQRS.
//
// It uses stream_id as the greetings primary key so that subsequent events
// (e.g. greeting.compensated) can update the same row.
type GreetingProjection struct {
	eventStore *platform.EventStore
	pool       *pgxpool.Pool
	position   int64
}

func NewGreetingProjection(eventStore *platform.EventStore, pool *pgxpool.Pool) *GreetingProjection {
	if eventStore == nil || pool == nil {
		return nil
	}
	return &GreetingProjection{eventStore: eventStore, pool: pool}
}

// Start runs the projection loop in a background goroutine.
func (p *GreetingProjection) Start(ctx context.Context, interval time.Duration) {
	if p == nil {
		return
	}
	go p.run(ctx, interval)
}

func (p *GreetingProjection) run(ctx context.Context, interval time.Duration) {
	slog.Info("greeting projection started", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("greeting projection stopped")
			return
		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				slog.Error("greeting projection poll error", "error", err)
			}
		}
	}
}

func (p *GreetingProjection) poll(ctx context.Context) error {
	events, err := p.eventStore.LoadAllEvents(ctx, p.position, 100)
	if err != nil {
		return err
	}

	for _, e := range events {
		if err := p.apply(ctx, e); err != nil {
			slog.Error("greeting projection apply error",
				"event_type", e.EventType,
				"stream_id", e.StreamID,
				"error", err,
			)
		}
		p.position = e.GlobalPosition
	}
	return nil
}

func (p *GreetingProjection) apply(ctx context.Context, e platform.StoredEvent) error {
	switch e.EventType {
	case EventGreetingCreated:
		var d GreetingCreatedData
		if err := json.Unmarshal(e.Data, &d); err != nil {
			return err
		}
		_, err := p.pool.Exec(ctx,
			`INSERT INTO greetings (id, name, message, external_status, status, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (id) DO UPDATE SET
			   name = EXCLUDED.name,
			   message = EXCLUDED.message,
			   external_status = EXCLUDED.external_status,
			   status = EXCLUDED.status`,
			e.StreamID, d.Name, d.Message, d.ExternalStatus, "created", e.CreatedAt,
		)
		return err

	case EventGreetingCompensated:
		_, err := p.pool.Exec(ctx,
			`UPDATE greetings SET status = 'compensated' WHERE id = $1`,
			e.StreamID,
		)
		return err

	default:
		return nil
	}
}
