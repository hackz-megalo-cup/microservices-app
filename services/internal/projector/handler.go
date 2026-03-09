package projector

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Projector struct {
	subscriber *platform.EventSubscriber
	pool       *pgxpool.Pool
}

func New(brokers []string, pool *pgxpool.Pool) (*Projector, error) {
	sub, err := platform.NewEventSubscriber(brokers, "projector-group")
	if err != nil {
		return nil, err
	}
	return &Projector{subscriber: sub, pool: pool}, nil
}

func (p *Projector) Run(ctx context.Context) error {
	topics := []string{
		platform.TopicGreetingCreated,
		platform.TopicCallCompleted,
		platform.TopicInvocationCreated,
		platform.TopicUserRegistered,
	}

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
				msg.Nack()
				continue
			}
			if err := p.handleEvent(ctx, event); err != nil {
				slog.Error("failed to handle event", "type", event.Type, "error", err)
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
	_, err := p.pool.Exec(ctx,
		`INSERT INTO event_log (event_id, event_type, source, data, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (event_id) DO NOTHING`,
		event.ID, event.Type, event.Source, data, event.Timestamp,
	)
	return err
}

func (p *Projector) Close() error {
	return p.subscriber.Close()
}
