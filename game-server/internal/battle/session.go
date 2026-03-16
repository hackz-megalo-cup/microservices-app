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

func NewSession(lobbyID, bossPokemonID uuid.UUID, bossHP int32, matchups TypeMatchup, timeout time.Duration) *Session {
	return &Session{
		SessionID:       uuid.New(),
		LobbyID:         lobbyID,
		BossPokemonID:   bossPokemonID,
		BossHP:          bossHP,
		BossMaxHP:       bossHP,
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

func (s *Session) ApplyTap(userID uuid.UUID) (int32, int32, int32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return 0, s.BossHP, s.BossMaxHP
	}

	p, ok := s.Participants[userID]
	if !ok {
		return 0, s.BossHP, s.BossMaxHP
	}

	p.TapCount++
	dmg := CalcTapDamage(p.PokemonAttack, p.PokemonType, s.BossType, s.TypeMatchups)
	s.BossHP -= dmg
	if s.BossHP <= 0 {
		s.BossHP = 0
		s.finished = true
		s.result = "win"
		close(s.doneCh)
	}
	return dmg, s.BossHP, s.BossMaxHP
}

func (s *Session) ApplySpecial(userID uuid.UUID) (int32, int32, int32, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.finished {
		return 0, s.BossHP, s.BossMaxHP, false
	}

	p, ok := s.Participants[userID]
	if !ok {
		return 0, s.BossHP, s.BossMaxHP, false
	}

	if p.TapCount < p.RequiredForSpecial {
		return 0, s.BossHP, s.BossMaxHP, false
	}

	p.TapCount = 0
	dmg := CalcSpecialDamage(p.SpecialMoveDamage, p.PokemonType, s.BossType, s.TypeMatchups)
	s.BossHP -= dmg
	if s.BossHP <= 0 {
		s.BossHP = 0
		s.finished = true
		s.result = "win"
		close(s.doneCh)
	}
	return dmg, s.BossHP, s.BossMaxHP, true
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
