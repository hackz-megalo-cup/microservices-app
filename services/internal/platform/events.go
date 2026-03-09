package platform

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

func NewEvent(eventType, source string, data any) Event {
	return Event{
		ID:        uuid.NewString(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}
}

type EventPublisher struct {
	publisher message.Publisher
	logger    watermill.LoggerAdapter
}

func NewEventPublisher(brokers []string) (*EventPublisher, error) {
	if len(brokers) == 0 {
		slog.Warn("no Kafka brokers, event publishing disabled")
		return nil, nil
	}
	logger := watermill.NewSlogLogger(slog.Default())
	publisher, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:   brokers,
			Marshaler: kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		slog.Warn("failed to create Kafka publisher", "error", err)
		return nil, nil
	}
	return &EventPublisher{publisher: publisher, logger: logger}, nil
}

func (p *EventPublisher) Publish(ctx context.Context, topic string, event Event) error {
	if p == nil {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := message.NewMessage(event.ID, payload)
	msg.Metadata.Set("event_type", event.Type)
	msg.Metadata.Set("source", event.Source)
	return p.publisher.Publish(topic, msg)
}

// PublishRaw publishes raw bytes to a topic (used for DLQ forwarding).
func (p *EventPublisher) PublishRaw(ctx context.Context, topic string, id string, payload []byte) error {
	if p == nil {
		return nil
	}
	msg := message.NewMessage(id, payload)
	return p.publisher.Publish(topic, msg)
}

func (p *EventPublisher) Close() error {
	if p == nil {
		return nil
	}
	return p.publisher.Close()
}

type EventSubscriber struct {
	subscriber message.Subscriber
}

func NewEventSubscriber(brokers []string, consumerGroup string) (*EventSubscriber, error) {
	if len(brokers) == 0 {
		return nil, nil
	}
	logger := watermill.NewSlogLogger(slog.Default())
	subscriber, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:       brokers,
			ConsumerGroup: consumerGroup,
			Unmarshaler:   kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		return nil, err
	}
	return &EventSubscriber{subscriber: subscriber}, nil
}

func (s *EventSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if s == nil {
		return nil, nil
	}
	return s.subscriber.Subscribe(ctx, topic)
}

func (s *EventSubscriber) Close() error {
	if s == nil {
		return nil
	}
	return s.subscriber.Close()
}

func ParseEvent(msg *message.Message) (Event, error) {
	var event Event
	err := json.Unmarshal(msg.Payload, &event)
	return event, err
}
