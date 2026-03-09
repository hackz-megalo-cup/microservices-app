package platform

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"
)

// CompensationRouter routes compensation events to registered handlers.
type CompensationRouter struct {
	handlers map[string]func(context.Context, Event) error
}

// NewCompensationRouter creates a new CompensationRouter.
func NewCompensationRouter() *CompensationRouter {
	return &CompensationRouter{
		handlers: make(map[string]func(context.Context, Event) error),
	}
}

// Handle registers a handler for a given event type.
func (r *CompensationRouter) Handle(eventType string, fn func(context.Context, Event) error) {
	r.handlers[eventType] = fn
}

// Run subscribes to the given topics and dispatches events to registered handlers.
func (r *CompensationRouter) Run(ctx context.Context, brokers []string, consumerGroup string) error {
	if len(r.handlers) == 0 {
		return nil
	}

	sub, err := NewEventSubscriber(brokers, consumerGroup)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}
	defer sub.Close()

	var topics []string
	for t := range r.handlers {
		topics = append(topics, t)
	}

	for _, topic := range topics {
		ch, err := sub.Subscribe(ctx, topic)
		if err != nil {
			return err
		}
		go r.processMessages(ctx, topic, ch)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (r *CompensationRouter) processMessages(ctx context.Context, topic string, ch <-chan *message.Message) {
	handler := r.handlers[topic]
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			event, err := ParseEvent(msg)
			if err != nil {
				slog.Error("compensation: failed to parse event", "topic", topic, "error", err)
				msg.Nack()
				continue
			}
			if err := handler(ctx, event); err != nil {
				slog.Error("compensation: handler failed", "topic", topic, "type", event.Type, "error", err)
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}
}
