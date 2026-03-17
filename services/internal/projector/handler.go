package projector

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Projector struct {
	subscriber *platform.EventSubscriber
	pool       *pgxpool.Pool
	dlq        *platform.EventPublisher
	projection *ProjectionHandler
}

func New(brokers []string, pool *pgxpool.Pool, dlq *platform.EventPublisher, projection *ProjectionHandler) (*Projector, error) {
	sub, err := platform.NewEventSubscriber(brokers, "projector-group")
	if err != nil {
		return nil, err
	}
	return &Projector{subscriber: sub, pool: pool, dlq: dlq, projection: projection}, nil
}

func (p *Projector) Run(ctx context.Context) error {
	topics := platform.SubscribableTopics()

	for _, topic := range topics {
		ch, err := p.subscriber.Subscribe(ctx, topic)
		if err != nil {
			return err
		}
		go p.processMessages(ctx, topic, ch)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (p *Projector) publishToDLQ(ctx context.Context, topic string, msg *message.Message) {
	dlqTopic := platform.DLQTopic(topic)
	if dlqTopic == "" {
		slog.Warn("no DLQ topic mapped", "source_topic", topic)
		return
	}
	if err := p.dlq.PublishRaw(ctx, dlqTopic, msg.UUID, msg.Payload); err != nil {
		slog.Error("failed to publish to DLQ", "dlq_topic", dlqTopic, "error", err)
	}
}

func (p *Projector) processMessages(ctx context.Context, topic string, ch <-chan *message.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			event, err := platform.ParseEvent(msg)
			if err != nil {
				slog.Error("failed to parse event", "topic", topic, "error", err)
				p.publishToDLQ(ctx, topic, msg)
				msg.Nack()
				continue
			}
			if err := p.handleEvent(ctx, event); err != nil {
				slog.Error("failed to handle event", "type", event.Type, "error", err)
				p.publishToDLQ(ctx, topic, msg)
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}

func (p *Projector) handleEvent(ctx context.Context, event platform.Event) error {
	if p.pool == nil {
		return nil
	}
	data, _ := json.Marshal(event.Data)
	var inserted bool
	err := p.pool.QueryRow(ctx,
		`INSERT INTO event_log (event_id, event_type, source, data, version, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (event_id) DO NOTHING
		 RETURNING TRUE`,
		event.ID, event.Type, event.Source, data, event.Version, event.Timestamp,
	).Scan(&inserted)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// Skip projection if this event was already processed (dedup).
	if !inserted {
		slog.Debug("duplicate event skipped", "event_id", event.ID, "type", event.Type)
		return nil
	}

	// Update read model projections
	if p.projection != nil {
		if projErr := p.projection.HandleEvent(ctx, event); projErr != nil {
			slog.Error("projection error", "type", event.Type, "error", projErr)
			return projErr
		}
	}

	return nil
}

func (p *Projector) Close() error {
	return p.subscriber.Close()
}
