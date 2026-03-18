package transport

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/battle"
)

type Handler struct {
	hub     *Hub
	session *battle.Session
}

func NewHandler(hub *Hub, session *battle.Session) *Handler {
	return &Handler{hub: hub, session: session}
}

func (h *Handler) HandleMessage(userID uuid.UUID, raw []byte) {
	msg, err := ParseMessage(raw)
	if err != nil {
		log.Printf("parse error from %s: %v", userID, err)
		return
	}

	switch msg.T {
	case "join":
		h.handleJoin(userID)
	case "tap":
		h.handleTap(userID)
	case "special":
		h.handleSpecial(userID)
	default:
		log.Printf("unknown message type %q from %s", msg.T, userID)
	}
}

func (h *Handler) handleJoin(userID uuid.UUID) {
	if h.session == nil {
		log.Printf("join from %s but no session", userID)
		return
	}

	h.session.AddParticipant(userID, &battle.Participant{
		UserID:             userID,
		PokemonAttack:      100,
		PokemonSpeed:       50,
		PokemonType:        "normal",
		SpecialMoveName:    "Debug Blast",
		SpecialMoveDamage:  500,
		RequiredForSpecial: 10,
	})

	participantIDs := h.session.ParticipantIDs()
	participants := make([]string, 0, len(participantIDs))
	for _, uid := range participantIDs {
		participants = append(participants, uid.String())
	}

	info := h.session.Info()
	joined := JoinedMessage{
		T:            "joined",
		SessionID:    info.SessionID.String(),
		BossHP:       info.BossHP,
		BossMaxHP:    info.BossMaxHP,
		Participants: participants,
		TimeoutSec:   int(info.TimeoutDuration / time.Second),
	}

	data, err := MarshalJSON(joined)
	if err != nil {
		log.Printf("marshal joined error: %v", err)
		return
	}
	h.hub.BroadcastReliable(data)

	log.Printf("player %s joined session %s", userID, info.SessionID)
}

func (h *Handler) handleTap(userID uuid.UUID) {
	if h.session == nil {
		return
	}

	dmg, currentHP, maxHP, justFinished := h.session.ApplyTap(userID)
	if dmg == 0 {
		return
	}

	hp := HPMessage{
		T:       "hp",
		HP:      currentHP,
		MaxHP:   maxHP,
		LastDmg: dmg,
		By:      userID.String(),
	}

	data, err := MarshalJSON(hp)
	if err != nil {
		return
	}
	h.hub.Broadcast(data)

	if justFinished {
		h.broadcastFinished()
	}
}

func (h *Handler) handleSpecial(userID uuid.UUID) {
	if h.session == nil {
		return
	}

	dmg, currentHP, maxHP, ok, justFinished := h.session.ApplySpecial(userID)
	if !ok {
		return
	}

	special := SpecialUsedMessage{
		T:        "special_used",
		UserID:   userID.String(),
		MoveName: "Debug Blast",
		Dmg:      dmg,
		BossHP:   currentHP,
	}

	data, err := MarshalJSON(special)
	if err != nil {
		return
	}
	h.hub.BroadcastReliable(data)

	// Also send HP update
	hp := HPMessage{
		T:       "hp",
		HP:      currentHP,
		MaxHP:   maxHP,
		LastDmg: dmg,
		By:      userID.String(),
	}
	hpData, err := MarshalJSON(hp)
	if err != nil {
		log.Printf("marshal hp error: %v", err)
		return
	}
	h.hub.Broadcast(hpData)

	if justFinished {
		h.broadcastFinished()
	}
}

func (h *Handler) StartTimeSync(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.session.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(h.session.StartedAt)
			remaining := h.session.TimeoutDuration - elapsed
			if remaining < 0 {
				remaining = 0
			}

			msg := TimeSyncMessage{
				T:            "time_sync",
				RemainingSec: int(remaining.Seconds()),
			}
			data, err := MarshalJSON(msg)
			if err != nil {
				continue
			}
			h.hub.Broadcast(data)
		}
	}
}

func (h *Handler) broadcastFinished() {
	info := h.session.Info()
	finished := FinishedMessage{
		T:       "finished",
		Result:  h.session.Result(),
		BossHP:  info.BossHP,
		Elapsed: int(time.Since(h.session.StartedAt).Seconds()),
	}

	data, err := MarshalJSON(finished)
	if err != nil {
		return
	}
	h.hub.BroadcastReliable(data)
}
