package kafka

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

const TopicBattleFinished = "battle.finished"

type BattleFinishedEvent struct {
	SessionID      uuid.UUID   `json:"sessionId"`
	LobbyID        uuid.UUID   `json:"lobbyId"`
	BossPokemonID  uuid.UUID   `json:"bossPokemonId"`
	Result         string      `json:"result"`
	ParticipantIDs []uuid.UUID `json:"participantUserIds"`
}

// battleFinishedData uses snake_case to match what the services consumers expect.
type battleFinishedData struct {
	SessionID      string   `json:"session_id"`
	LobbyID        string   `json:"lobby_id"`
	BossPokemonID  string   `json:"boss_pokemon_id"`
	Result         string   `json:"result"`
	ParticipantIDs []string `json:"participant_user_ids"`
}

// eventEnvelope matches the platform.Event format used by service consumers.
type eventEnvelope struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Version   int       `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

func BuildBattleFinishedRecord(event BattleFinishedEvent) *kgo.Record {
	participantIDs := make([]string, 0, len(event.ParticipantIDs))
	for _, id := range event.ParticipantIDs {
		participantIDs = append(participantIDs, id.String())
	}

	envelope := eventEnvelope{
		ID:        uuid.NewString(),
		Type:      TopicBattleFinished,
		Source:    "game-server",
		Version:   1,
		Timestamp: time.Now().UTC(),
		Data: battleFinishedData{
			SessionID:      event.SessionID.String(),
			LobbyID:        event.LobbyID.String(),
			BossPokemonID:  event.BossPokemonID.String(),
			Result:         event.Result,
			ParticipantIDs: participantIDs,
		},
	}

	val, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("marshal battle.finished error: %v", err)
		return nil
	}
	return &kgo.Record{
		Topic: TopicBattleFinished,
		Key:   []byte(event.SessionID.String()),
		Value: val,
	}
}
