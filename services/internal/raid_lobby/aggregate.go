package raidlobby

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type RaidLobbyAggregate struct {
	platform.AggregateBase
	Status        string
	BossPokemonID string
	Participants  []string
}

func NewRaidLobbyAggregate(id string) *RaidLobbyAggregate {
	return &RaidLobbyAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *RaidLobbyAggregate) StreamType() string { return "raid_lobby" }

func (a *RaidLobbyAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventRaidLobbyCreated:
		var d RaidLobbyCreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		a.BossPokemonID = d.BossPokemonID
		a.Status = "waiting"
	case EventRaidUserJoined:
		var d RaidUserJoinedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal user joined data", "error", err)
		}
		a.Participants = append(a.Participants, d.UserID)
	case EventRaidLobbyFailed:
		a.Status = "failed"
	case EventRaidLobbyCompensated:
		a.Status = "compensated"
	}
}

// Create initialises a new raid lobby.
func (a *RaidLobbyAggregate) Create(bossPokemonID string) {
	a.Raise(EventRaidLobbyCreated, RaidLobbyCreatedData{
		BossPokemonID: bossPokemonID,
	})
	a.BossPokemonID = bossPokemonID
	a.Status = "waiting"
}

// Join adds a participant to the lobby.
func (a *RaidLobbyAggregate) Join(userID, participantID string) {
	a.Raise(EventRaidUserJoined, RaidUserJoinedData{
		LobbyID:       a.AggregateID(),
		UserID:        userID,
		ParticipantID: participantID,
	})
	a.Participants = append(a.Participants, userID)
}

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *RaidLobbyAggregate) Fail(input string, reason string) {
	a.Raise(EventRaidLobbyFailed, RaidLobbyFailedData{
		Input: input,
		Error: reason,
	})
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *RaidLobbyAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventRaidLobbyCompensated, RaidLobbyCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// RaidLobbyTopicMapper maps event types to Kafka topics.
func RaidLobbyTopicMapper(eventType string) string {
	switch eventType {
	case EventRaidLobbyCreated:
		return platform.TopicRaidLobbyCreated
	case EventRaidUserJoined:
		return platform.TopicRaidUserJoined
	case EventRaidLobbyFailed:
		return platform.TopicRaidLobbyFailed
	default:
		return ""
	}
}
