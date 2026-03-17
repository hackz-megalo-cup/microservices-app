package kafka_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/kafka"
)

func TestBuildBattleFinishedRecord(t *testing.T) {
	event := kafka.BattleFinishedEvent{
		SessionID:      uuid.New(),
		LobbyID:        uuid.New(),
		BossPokemonID:  uuid.New(),
		Result:         "win",
		ParticipantIDs: []uuid.UUID{uuid.New(), uuid.New()},
	}

	record := kafka.BuildBattleFinishedRecord(event)

	if record.Topic != "battle.finished" {
		t.Errorf("Topic = %s, want battle.finished", record.Topic)
	}
	if len(record.Value) == 0 {
		t.Error("expected non-empty value")
	}
	if len(record.Key) == 0 {
		t.Error("expected non-empty key")
	}
}
