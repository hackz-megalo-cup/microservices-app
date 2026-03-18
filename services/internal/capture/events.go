package capture

import "github.com/hackz-megalo-cup/microservices-app/services/internal/platform"

const (
	EventCaptureStarted    = "capture.started"
	EventCaptureItemUsed   = "capture.item_used"
	EventCaptureBallThrown = "capture.ball_thrown"
	EventCaptureCompleted  = "capture.completed"

	EventCaptureFailed      = "capture.failed"      // main.go が参照 — 削除禁止
	EventCaptureCompensated = "capture.compensated" // main.go が参照 — 削除禁止
)

type StartedData struct {
	SessionID       string  `json:"session_id"`
	BattleSessionID string  `json:"battle_session_id"`
	UserID          string  `json:"user_id"`
	PokemonID       string  `json:"pokemon_id"`
	BaseRate        float64 `json:"base_rate"`
}

type ItemUsedData struct {
	SessionID  string  `json:"session_id"`
	ItemID     string  `json:"item_id"`
	RateBefore float64 `json:"rate_before"`
	RateAfter  float64 `json:"rate_after"`
}

type BallThrownData struct {
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
}

type CompletedData struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	PokemonID string `json:"pokemon_id"`
	Result    string `json:"result"`
}

type FailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type CompensatedData struct {
	Reason string `json:"reason"`
}

// CaptureTopicMapper maps event types to Kafka topics.
func CaptureTopicMapper(eventType string) string {
	switch eventType {
	case EventCaptureStarted:
		return platform.TopicCaptureStarted
	case EventCaptureItemUsed:
		return platform.TopicCaptureItemUsed
	case EventCaptureBallThrown:
		return platform.TopicCaptureBallThrown
	case EventCaptureCompleted:
		return platform.TopicCaptureCompleted
	case EventCaptureFailed:
		return platform.TopicCaptureFailed
	case EventCaptureCompensated:
		return platform.TopicCaptureCompensated
	default:
		return ""
	}
}
