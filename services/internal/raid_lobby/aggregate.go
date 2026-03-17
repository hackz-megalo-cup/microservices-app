package raidlobby

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Aggregate struct {
	platform.AggregateBase
	Status        string
	BossPokemonID string
	Participants  []string
}

func NewAggregate(id string) *Aggregate {
	return &Aggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *Aggregate) StreamType() string { return "raid_lobby" }

func (a *Aggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventCreated:
		var d CreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		a.BossPokemonID = d.BossPokemonID
		a.Status = "waiting"
	case EventUserJoined:
		var d UserJoinedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal user joined data", "error", err)
		}
		a.Participants = append(a.Participants, d.UserID)
	case EventBattleStarted:
		a.Status = "in_battle"
	case EventFinished:
		a.Status = "finished"
	case EventFailed:
		a.Status = "failed"
	case EventCompensated:
		a.Status = "compensated"
	}
}

// Create initialises a new raid lobby.
func (a *Aggregate) Create(bossPokemonID string) {
	a.Raise(EventCreated, CreatedData{
		BossPokemonID: bossPokemonID,
	})
	a.BossPokemonID = bossPokemonID
	a.Status = "waiting"
}

// Join adds a participant to the lobby.
func (a *Aggregate) Join(userID, participantID string) {
	a.Raise(EventUserJoined, UserJoinedData{
		LobbyID:       a.AggregateID(),
		UserID:        userID,
		ParticipantID: participantID,
	})
	a.Participants = append(a.Participants, userID)
}

// StartBattle transitions the lobby to in_battle status.
func (a *Aggregate) StartBattle(sessionID string) {
	a.Raise(EventBattleStarted, BattleStartedData{
		LobbyID:            a.AggregateID(),
		BossPokemonID:      a.BossPokemonID,
		ParticipantUserIDs: a.Participants,
		SessionID:          sessionID,
	})
	a.Status = "in_battle"
}

// Finish marks the lobby as finished after battle completion.
func (a *Aggregate) Finish(sessionID, result string) {
	a.Raise(EventFinished, FinishedData{
		LobbyID:   a.AggregateID(),
		SessionID: sessionID,
		Result:    result,
	})
	a.Status = "finished"
}

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *Aggregate) Fail(input string, reason string) {
	a.Raise(EventFailed, FailedData{
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
	a.Raise(EventCompensated, CompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// TopicMapper maps event types to Kafka topics.
func TopicMapper(eventType string) string {
	switch eventType {
	case EventCreated:
		return platform.TopicRaidLobbyCreated
	case EventUserJoined:
		return platform.TopicRaidUserJoined
	case EventBattleStarted:
		return platform.TopicRaidBattleStarted
	case EventFinished:
		return platform.TopicRaidLobbyFinished
	case EventFailed:
		return platform.TopicRaidLobbyFailed
	default:
		return ""
	}
}
