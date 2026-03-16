package transport

import (
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

	joined := JoinedMessage{
		T:            "joined",
		SessionID:    h.session.SessionID.String(),
		BossHP:       h.session.BossHP,
		BossMaxHP:    h.session.BossMaxHP,
		Participants: participants,
		TimeoutSec:   int(h.session.TimeoutDuration / time.Second),
	}

	data, err := MarshalJSON(joined)
	if err != nil {
		log.Printf("marshal joined error: %v", err)
		return
	}
	h.hub.BroadcastReliable(data)

	log.Printf("player %s joined session %s", userID, h.session.SessionID)
}

func (h *Handler) handleTap(userID uuid.UUID) {
	if h.session == nil {
		return
	}

	dmg := h.session.ApplyTap(userID)
	if dmg == 0 {
		return
	}

	hp := HPMessage{
		T:       "hp",
		HP:      h.session.BossHP,
		MaxHP:   h.session.BossMaxHP,
		LastDmg: dmg,
		By:      userID.String(),
	}

	data, err := MarshalJSON(hp)
	if err != nil {
		return
	}
	h.hub.Broadcast(data)

	if h.session.IsFinished() {
		h.broadcastFinished()
	}
}

func (h *Handler) handleSpecial(userID uuid.UUID) {
	if h.session == nil {
		return
	}

	dmg, ok := h.session.ApplySpecial(userID)
	if !ok {
		return
	}

	special := SpecialUsedMessage{
		T:        "special_used",
		UserID:   userID.String(),
		MoveName: "Debug Blast",
		Dmg:      dmg,
		BossHP:   h.session.BossHP,
	}

	data, err := MarshalJSON(special)
	if err != nil {
		return
	}
	h.hub.BroadcastReliable(data)

	// Also send HP update
	hp := HPMessage{
		T:       "hp",
		HP:      h.session.BossHP,
		MaxHP:   h.session.BossMaxHP,
		LastDmg: dmg,
		By:      userID.String(),
	}
	hpData, _ := MarshalJSON(hp)
	h.hub.Broadcast(hpData)

	if h.session.IsFinished() {
		h.broadcastFinished()
	}
}

func (h *Handler) broadcastFinished() {
	elapsed := int(time.Since(h.session.StartedAt).Seconds())
	finished := FinishedMessage{
		T:       "finished",
		Result:  h.session.Result(),
		BossHP:  h.session.BossHP,
		Elapsed: elapsed,
	}

	data, err := MarshalJSON(finished)
	if err != nil {
		return
	}
	h.hub.BroadcastReliable(data)
}
