package battle

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	SessionID       uuid.UUID
	LobbyID         uuid.UUID
	BossPokemonID   uuid.UUID
	BossHP          int32
	BossMaxHP       int32
	BossType        string
	TypeMatchups    TypeMatchup
	Participants    map[uuid.UUID]*Participant
	StartedAt       time.Time
	TimeoutDuration time.Duration
	finished        bool
	result          string // "win", "timeout"
	doneCh          chan struct{}
	mu              sync.RWMutex
}

func NewSession(lobbyID, bossPokemonID uuid.UUID, bossHP int32, bossType string, matchups TypeMatchup, timeout time.Duration) *Session {
	return &Session{
		SessionID:       uuid.New(),
		LobbyID:         lobbyID,
		BossPokemonID:   bossPokemonID,
		BossHP:          bossHP,
		BossMaxHP:       bossHP,
		BossType:        bossType,
		TypeMatchups:    matchups,
		Participants:    make(map[uuid.UUID]*Participant),
		StartedAt:       time.Now(),
		TimeoutDuration: timeout,
		doneCh:          make(chan struct{}),
	}
}

type SessionInfo struct {
	SessionID       uuid.UUID
	BossHP          int32
	BossMaxHP       int32
	TimeoutDuration time.Duration
}

func (s *Session) Info() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SessionInfo{
		SessionID:       s.SessionID,
		BossHP:          s.BossHP,
		BossMaxHP:       s.BossMaxHP,
		TimeoutDuration: s.TimeoutDuration,
	}
}

func (s *Session) AddParticipant(userID uuid.UUID, p *Participant) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Participants[userID] = p
}

func (s *Session) HasParticipant(userID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Participants[userID]
	return ok
}

func (s *Session) ApplyTap(userID uuid.UUID) (dmg, currentHP, maxHP int32, justFinished bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return 0, s.BossHP, s.BossMaxHP, false
	}

	p, ok := s.Participants[userID]
	if !ok {
		return 0, s.BossHP, s.BossMaxHP, false
	}

	p.TapCount++
	dmg = CalcTapDamage(p.PokemonAttack, p.PokemonType, s.BossType, s.TypeMatchups)
	s.BossHP -= dmg
	if s.BossHP <= 0 {
		s.BossHP = 0
		s.finished = true
		s.result = "win"
		close(s.doneCh)
		return dmg, s.BossHP, s.BossMaxHP, true
	}
	return dmg, s.BossHP, s.BossMaxHP, false
}

func (s *Session) ApplySpecial(userID uuid.UUID) (dmg, currentHP, maxHP int32, ok, justFinished bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return 0, s.BossHP, s.BossMaxHP, false, false
	}

	p, found := s.Participants[userID]
	if !found {
		return 0, s.BossHP, s.BossMaxHP, false, false
	}

	if p.TapCount < p.RequiredForSpecial {
		return 0, s.BossHP, s.BossMaxHP, false, false
	}

	p.TapCount = 0
	dmg = CalcSpecialDamage(p.SpecialMoveDamage, p.PokemonType, s.BossType, s.TypeMatchups)
	s.BossHP -= dmg
	if s.BossHP <= 0 {
		s.BossHP = 0
		s.finished = true
		s.result = "win"
		close(s.doneCh)
		return dmg, s.BossHP, s.BossMaxHP, true, true
	}
	return dmg, s.BossHP, s.BossMaxHP, true, false
}

func (s *Session) Timeout() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.finished {
		s.finished = true
		s.result = "timeout"
		close(s.doneCh)
	}
}

func (s *Session) IsFinished() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.finished
}

func (s *Session) Result() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.result
}

func (s *Session) ParticipantIDs() []uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]uuid.UUID, 0, len(s.Participants))
	for uid := range s.Participants {
		ids = append(ids, uid)
	}
	return ids
}

func (s *Session) Done() <-chan struct{} {
	return s.doneCh
}

func (s *Session) GetParticipantMoveName(userID uuid.UUID) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.Participants[userID]; ok && p.SpecialMoveName != "" {
		return p.SpecialMoveName
	}
	return "Special Attack"
}

func (s *Session) RemainingTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	remaining := s.TimeoutDuration - time.Since(s.StartedAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}
