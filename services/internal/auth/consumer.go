package auth

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

type pokemonRegistrar interface {
	RegisterPokemon(ctx context.Context, userID, pokemonID string) error
}

// PokemonCaught represents a pokemon caught event from the capture service
type PokemonCaught struct {
	UserID    string `json:"user_id"`
	PokemonID string `json:"pokemon_id"`
	Result    string `json:"result"`
}

// ConsumerConfig holds Kafka consumer configuration
type ConsumerConfig struct {
	Client *kgo.Client
	Repo   pokemonRegistrar
}

// RunConsumer starts consuming capture.completed events and registers pokemon
func RunConsumer(ctx context.Context, cfg ConsumerConfig) error {
	if cfg.Client == nil || cfg.Repo == nil {
		slog.Info("consumer skipped (no kafka client or repo)")
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fetches := cfg.Client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			if err := ctx.Err(); err != nil {
				return err
			}
			return nil
		}

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				if err := handleCaughtPokemon(ctx, cfg.Repo, record.Value); err != nil {
					slog.Error("failed to process capture.completed event", "error", err)
					// Do not commit offset on error so Kafka can retry this record.
					return
				}

				if err := cfg.Client.CommitRecords(ctx, record); err != nil {
					slog.Error("failed to commit kafka offset", "topic", record.Topic, "partition", record.Partition, "offset", record.Offset, "error", err)
					return
				}
			}
		})
	}
}

// handleCaughtPokemon processes a PokemonCaught event and registers the pokemon
func handleCaughtPokemon(ctx context.Context, repo pokemonRegistrar, data []byte) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		slog.Error("failed to unmarshal capture.completed event envelope", "error", err)
		return err
	}

	var event PokemonCaught
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		slog.Error("failed to unmarshal capture.completed event data", "error", err)
		return err
	}

	if event.Result != "" && event.Result != "success" {
		slog.Info("skip capture.completed event: result is not success", "result", event.Result)
		return nil
	}

	if event.UserID == "" || event.PokemonID == "" {
		slog.Warn("invalid capture.completed event: missing user_id or pokemon_id")
		return nil
	}

	if err := repo.RegisterPokemon(ctx, event.UserID, event.PokemonID); err != nil {
		slog.Error("failed to register pokemon",
			"user_id", event.UserID,
			"pokemon_id", event.PokemonID,
			"error", err,
		)
		return err
	}

	slog.Info("pokemon registered",
		"user_id", event.UserID,
		"pokemon_id", event.PokemonID,
	)
	return nil
}
