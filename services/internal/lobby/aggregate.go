package lobby

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// Aggregate tracks the active pokemon for a user.
type Aggregate struct {
	platform.AggregateBase
	Status    string
	UserID    string
	PokemonID string
}

func NewAggregate(id string) *Aggregate {
	return &Aggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *Aggregate) StreamType() string { return "lobby" }

// ApplyEvent replays events to restore aggregate state.
func (a *Aggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventLobbyActivePokemonSet:
		var d LobbyActivePokemonSetData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal active_pokemon_set data", "error", err)
		}
		a.UserID = d.UserID
		a.PokemonID = d.PokemonID
		a.Status = "active"
	case EventLobbyFailed:
		a.Status = "failed"
	case EventLobbyCompensated:
		a.Status = "compensated"
	}
}

// SetActivePokemon records a new active pokemon for the user.
func (a *Aggregate) SetActivePokemon(userID, pokemonID string) {
	a.Raise(EventLobbyActivePokemonSet, LobbyActivePokemonSetData{
		UserID:    userID,
		PokemonID: pokemonID,
	})
	a.UserID = userID
	a.PokemonID = pokemonID
	a.Status = "active"
}

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *Aggregate) Fail(input string, reason string) {
	a.Raise(EventLobbyFailed, LobbyFailedData{
		Input: input,
		Error: reason,
	})
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *Aggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventLobbyCompensated, LobbyCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// LobbyTopicMapper maps event types to Kafka topics.
func LobbyTopicMapper(eventType string) string {
	switch eventType {
	case EventLobbyActivePokemonSet:
		return "" // stored in event store only; no Kafka publishing needed
	case EventLobbyFailed:
		return platform.TopicLobbyFailed
	default:
		return ""
	}
}
