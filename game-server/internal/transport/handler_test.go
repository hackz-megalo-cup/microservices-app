package transport

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/battle"
)

func TestHandleMessage_Join(t *testing.T) {
	hub := NewHub()
	matchups := battle.TypeMatchup{}
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, matchups, 300*time.Second)

	conn := &mockConn{}
	userID := uuid.New()
	hub.Register(userID, conn)

	handler := NewHandler(hub, session)
	handler.HandleMessage(userID, []byte(`{"t":"join","userId":"`+userID.String()+`"}`))

	if !session.HasParticipant(userID) {
		t.Error("user should be registered as participant after join")
	}

	if len(conn.reliableMsgs) == 0 {
		t.Error("expected joined message via reliable send")
	}
}

func TestHandleMessage_Tap(t *testing.T) {
	hub := NewHub()
	matchups := battle.TypeMatchup{}
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, matchups, 300*time.Second)

	conn := &mockConn{}
	userID := uuid.New()
	hub.Register(userID, conn)

	session.AddParticipant(userID, &battle.Participant{
		UserID:        userID,
		PokemonAttack: 100,
		PokemonType:   "normal",
	})

	handler := NewHandler(hub, session)
	handler.HandleMessage(userID, []byte(`{"t":"tap"}`))

	if len(conn.unreliableMsgs) == 0 {
		t.Error("expected hp message via unreliable broadcast")
	}

	if session.Info().BossHP >= 50000 {
		t.Error("boss HP should have decreased after tap")
	}
}

func TestHandleMessage_Special_NotEnoughTaps(t *testing.T) {
	hub := NewHub()
	matchups := battle.TypeMatchup{}
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, matchups, 300*time.Second)

	conn := &mockConn{}
	userID := uuid.New()
	hub.Register(userID, conn)

	session.AddParticipant(userID, &battle.Participant{
		UserID:             userID,
		PokemonAttack:      100,
		PokemonType:        "normal",
		SpecialMoveDamage:  500,
		RequiredForSpecial: 10,
		TapCount:           0,
	})

	handler := NewHandler(hub, session)
	handler.HandleMessage(userID, []byte(`{"t":"special","userId":"`+userID.String()+`"}`))

	if len(conn.reliableMsgs) != 0 {
		t.Error("should not send special_used when taps insufficient")
	}
}

func TestHandleMessage_BossDefeated(t *testing.T) {
	hub := NewHub()
	matchups := battle.TypeMatchup{}
	session := battle.NewSession(uuid.New(), uuid.New(), 50, matchups, 300*time.Second)

	conn := &mockConn{}
	userID := uuid.New()
	hub.Register(userID, conn)

	session.AddParticipant(userID, &battle.Participant{
		UserID:        userID,
		PokemonAttack: 100,
		PokemonType:   "normal",
	})

	handler := NewHandler(hub, session)
	handler.HandleMessage(userID, []byte(`{"t":"tap"}`))

	if !session.IsFinished() {
		t.Error("session should be finished after boss defeated")
	}

	hasFinished := false
	for _, msg := range conn.reliableMsgs {
		if string(msg) != "" {
			hasFinished = true
		}
	}
	if !hasFinished && len(conn.reliableMsgs) == 0 {
		t.Error("expected finished message")
	}
}
