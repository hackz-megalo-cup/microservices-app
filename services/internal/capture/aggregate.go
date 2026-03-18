package capture

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type CaptureAggregate struct {
	platform.AggregateBase
	BattleSessionID string
	UserID          string
	PokemonID       string
	BaseRate        float64
	CurrentRate     float64
	Result          string // pending, success, fail, escaped
}

func NewCaptureAggregate(id string) *CaptureAggregate {
	return &CaptureAggregate{
		AggregateBase: platform.NewAggregateBase(id),
		Result:        "pending",
	}
}

func (a *CaptureAggregate) StreamType() string { return "capture" }

func (a *CaptureAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventCaptureStarted:
		var d StartedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal started data", "error", err)
		}
		a.BattleSessionID = d.BattleSessionID
		a.UserID = d.UserID
		a.PokemonID = d.PokemonID
		a.BaseRate = d.BaseRate
		a.CurrentRate = d.BaseRate
		a.Result = "pending"
	case EventCaptureItemUsed:
		var d ItemUsedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal item_used data", "error", err)
		}
		a.CurrentRate = d.RateAfter
	case EventCaptureBallThrown:
		var d BallThrownData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal ball_thrown data", "error", err)
		}
		a.Result = d.Result
	case EventCaptureCompleted:
		var d CompletedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal completed data", "error", err)
		}
		a.Result = d.Result
	case EventCaptureFailed:
		a.Result = "failed"
	case EventCaptureCompensated:
		a.Result = "compensated"
	}
}

// Start creates a new capture session.
func (a *CaptureAggregate) Start(battleSessionID, userID, pokemonID string, baseRate float64) {
	a.Raise(EventCaptureStarted, StartedData{
		SessionID:       a.AggregateID(),
		BattleSessionID: battleSessionID,
		UserID:          userID,
		PokemonID:       pokemonID,
		BaseRate:        baseRate,
	})
	a.BattleSessionID = battleSessionID
	a.UserID = userID
	a.PokemonID = pokemonID
	a.BaseRate = baseRate
	a.CurrentRate = baseRate
	a.Result = "pending"
}

// UseItem records item usage and rate change.
func (a *CaptureAggregate) UseItem(itemID string, rateBefore, rateAfter float64) {
	a.Raise(EventCaptureItemUsed, ItemUsedData{
		SessionID:  a.AggregateID(),
		ItemID:     itemID,
		RateBefore: rateBefore,
		RateAfter:  rateAfter,
	})
	a.CurrentRate = rateAfter
}

// ThrowBall records a ball throw result.
func (a *CaptureAggregate) ThrowBall(result string) {
	a.Raise(EventCaptureBallThrown, BallThrownData{
		SessionID: a.AggregateID(),
		Result:    result,
	})
	a.Result = result
}

// Complete records the final capture result.
func (a *CaptureAggregate) Complete(result string) {
	a.Raise(EventCaptureCompleted, CompletedData{
		SessionID: a.AggregateID(),
		UserID:    a.UserID,
		PokemonID: a.PokemonID,
		Result:    result,
	})
	a.Result = result
}

// Escape sets capture rate to 0 and marks as escaped.
func (a *CaptureAggregate) Escape() {
	a.CurrentRate = 0
	a.Result = "escaped"
	a.Raise(EventCaptureCompleted, CompletedData{
		SessionID: a.AggregateID(),
		UserID:    a.UserID,
		PokemonID: a.PokemonID,
		Result:    "escaped",
	})
}

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *CaptureAggregate) Fail(input string, reason string) {
	a.Raise(EventCaptureFailed, FailedData{
		Input: input,
		Error: reason,
	})
	a.Result = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *CaptureAggregate) Compensate(reason string) {
	if a.Result == "compensated" {
		return
	}
	a.Raise(EventCaptureCompensated, CompensatedData{
		Reason: reason,
	})
	a.Result = "compensated"
}
