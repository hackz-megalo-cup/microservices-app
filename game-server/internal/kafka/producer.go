package kafka

import (
	"encoding/json"
	"log"

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

func BuildBattleFinishedRecord(event BattleFinishedEvent) *kgo.Record {
	val, err := json.Marshal(event)
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
