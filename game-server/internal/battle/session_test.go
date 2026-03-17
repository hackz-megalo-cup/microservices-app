package battle

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewSession(t *testing.T) {
	lobbyID := uuid.New()
	bossID := uuid.New()
	matchups := TypeMatchup{"static_typing": {"dynamic_typing": 2.0}}

	s := NewSession(lobbyID, bossID, 50000, "normal", matchups, 300*time.Second)

	if s.LobbyID != lobbyID {
		t.Errorf("LobbyID = %v, want %v", s.LobbyID, lobbyID)
	}
	if s.BossHP != 50000 || s.BossMaxHP != 50000 {
		t.Error("boss HP not initialized correctly")
	}
	if s.IsFinished() {
		t.Error("new session should not be finished")
	}
}

func TestSession_ApplyTap(t *testing.T) {
	s := newTestSession(50000)
	userID := uuid.New()
	s.AddParticipant(userID, &Participant{
		UserID:        userID,
		PokemonAttack: 100,
		PokemonType:   "static_typing",
	})

	dmg, currentHP, _, _ := s.ApplyTap(userID)
	if dmg <= 0 {
		t.Errorf("expected positive damage, got %d", dmg)
	}
	if currentHP != 50000-dmg {
		t.Errorf("BossHP = %d, want %d", currentHP, 50000-dmg)
	}
}

func TestSession_BossDefeated(t *testing.T) {
	s := newTestSession(100) // low HP
	userID := uuid.New()
	s.AddParticipant(userID, &Participant{
		UserID:        userID,
		PokemonAttack: 200,
		PokemonType:   "static_typing",
	})

	s.ApplyTap(userID) // returns are unused here

	if !s.IsFinished() {
		t.Error("session should be finished when boss HP <= 0")
	}
	if s.Result() != "win" {
		t.Errorf("Result() = %s, want win", s.Result())
	}
}

func TestSession_UnknownParticipant(t *testing.T) {
	s := newTestSession(50000)
	unknownID := uuid.New()

	dmg, _, _, _ := s.ApplyTap(unknownID)
	if dmg != 0 {
		t.Errorf("expected 0 damage for unknown participant, got %d", dmg)
	}
}

func newTestSession(bossHP int32) *Session {
	matchups := TypeMatchup{"static_typing": {"dynamic_typing": 2.0}}
	return NewSession(uuid.New(), uuid.New(), bossHP, "normal", matchups, 300*time.Second)
}
