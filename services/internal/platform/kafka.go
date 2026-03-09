package platform

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ParseKafkaBrokers parses comma-separated broker addresses.
func ParseKafkaBrokers(env string) []string {
	if env == "" {
		return nil
	}
	return strings.Split(env, ",")
}

// NewKafkaProducer creates a franz-go client configured for producing.
// Returns nil if brokers is empty (graceful fallback for environments without Redpanda).
func NewKafkaProducer(ctx context.Context, brokers []string, opts ...kgo.Opt) (*kgo.Client, error) {
	if len(brokers) == 0 {
		slog.Warn("KAFKA_BROKERS not set, Kafka producer disabled")
		return nil, nil
	}
	baseOpts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ProduceRequestTimeout(5 * time.Second),
		kgo.RecordRetries(3),
	}
	baseOpts = append(baseOpts, opts...)
	client, err := kgo.NewClient(baseOpts...)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}
	// Ping to verify connectivity
	if err := client.Ping(ctx); err != nil {
		slog.Warn("kafka broker unreachable, continuing without kafka", "error", err)
		client.Close()
		return nil, nil
	}
	slog.Info("kafka producer connected", "brokers", brokers)
	return client, nil
}

// NewKafkaConsumer creates a franz-go client configured for consuming from a consumer group.
// Returns nil if brokers is empty.
func NewKafkaConsumer(ctx context.Context, brokers []string, group string, topics []string, opts ...kgo.Opt) (*kgo.Client, error) {
	if len(brokers) == 0 {
		slog.Warn("KAFKA_BROKERS not set, Kafka consumer disabled")
		return nil, nil
	}
	baseOpts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),
		kgo.FetchMaxWait(5 * time.Second),
	}
	baseOpts = append(baseOpts, opts...)
	client, err := kgo.NewClient(baseOpts...)
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}
	slog.Info("kafka consumer connected", "brokers", brokers, "group", group, "topics", topics)
	return client, nil
}

// EnsureTopics creates topics if they don't exist using kadm.
func EnsureTopics(ctx context.Context, client *kgo.Client, topics map[string]int32) error {
	if client == nil {
		return nil
	}
	adm := kadm.NewClient(client)
	defer adm.Close()
	for topic, partitions := range topics {
		_, err := adm.CreateTopic(ctx, partitions, 1, nil, topic)
		if err != nil {
			slog.Warn("topic creation (may already exist)", "topic", topic, "error", err)
		}
	}
	return nil
}
