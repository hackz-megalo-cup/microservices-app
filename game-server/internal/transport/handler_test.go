package transport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/battle"
)

func TestHandleMessage_Join(t *testing.T) {
	hub := NewHub()
	matchups := battle.TypeMatchup{}
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, "normal", matchups, 300*time.Second)

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
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, "normal", matchups, 300*time.Second)

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
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, "normal", matchups, 300*time.Second)

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
	session := battle.NewSession(uuid.New(), uuid.New(), 50, "normal", matchups, 300*time.Second)

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

func TestStartTimeSync(t *testing.T) {
	hub := NewHub()
	matchups := battle.TypeMatchup{}
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, "normal", matchups, 300*time.Second)

	conn := &mockConn{}
	userID := uuid.New()
	hub.Register(userID, conn)

	handler := NewHandler(hub, session)

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	go handler.StartTimeSync(ctx)

	// Wait for at least 2 ticks
	time.Sleep(2200 * time.Millisecond)

	conn.mu.Lock()
	msgs := make([][]byte, len(conn.unreliableMsgs))
	copy(msgs, conn.unreliableMsgs)
	conn.mu.Unlock()

	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 time_sync messages, got %d", len(msgs))
	}

	var tsMsg TimeSyncMessage
	if err := json.Unmarshal(msgs[0], &tsMsg); err != nil {
		t.Fatalf("failed to unmarshal time_sync: %v", err)
	}
	if tsMsg.T != "time_sync" {
		t.Errorf("expected type time_sync, got %q", tsMsg.T)
	}
	if tsMsg.RemainingSec <= 0 || tsMsg.RemainingSec > 300 {
		t.Errorf("unexpected remaining seconds: %d", tsMsg.RemainingSec)
	}
}
