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
}

// ConsumerConfig holds Kafka consumer configuration
type ConsumerConfig struct {
	Client *kgo.Client
	Repo   pokemonRegistrar
}

// RunConsumer starts consuming capture.caught events and registers pokemon
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
					slog.Error("failed to process capture.caught event", "error", err)
					// Don't commit offset on error - let Kafka retry
					return
				}
			}
		})
	}
}

// handleCaughtPokemon processes a PokemonCaught event and registers the pokemon
func handleCaughtPokemon(ctx context.Context, repo pokemonRegistrar, data []byte) error {
	var event PokemonCaught
	if err := json.Unmarshal(data, &event); err != nil {
		slog.Error("failed to unmarshal capture.caught event", "error", err)
		return err
	}

	if event.UserID == "" || event.PokemonID == "" {
		slog.Warn("invalid capture.caught event: missing user_id or pokemon_id")
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
