package capture

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ==========================================================.
// Aggregate — ドメインエンティティ。
// ↓ ドメインの状態フィールドを追加する（例: Title string）
// ==========================================================.

type Aggregate struct {
	platform.AggregateBase
	Status string
}

func NewAggregate(id string) *Aggregate {
	return &Aggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *Aggregate) StreamType() string { return "capture" }

// ApplyEvent はイベントを再生して状態を復元する。
// Created の case を書き換え、追加イベントの case を足す。
func (a *Aggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventCaptureCreated:
		var d CreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		// ↓ フィールドの復元を書く（例: a.Title = d.Title）
		a.Status = "created"
	// ↓ 追加イベントの case をここに足す
	// 例:
	// case EventCaptureCompleted:
	//     a.Status = "completed"
	case EventCaptureFailed:
		a.Status = "failed"
	case EventCaptureCompensated:
		a.Status = "compensated"
	}
}

// ==========================================================.
// コマンドメソッド — Raise() でイベントを発行し、状態を更新する。
//
// Fail / Compensate は main.go の補償ハンドラが参照 — 削除禁止。
// AggregateID() で集約の ID を取得できる。
// 既存集約のロード: platform.LoadAggregate(ctx, eventStore, agg)
// ==========================================================.

// Create — 引数をドメインに合わせて変更する（例: Create(title string)）
func (a *Aggregate) Create() {
	a.Raise(EventCaptureCreated, CreatedData{
		// ↓ フィールドを渡す（例: Title: title）
	})
	// ↓ 状態を更新する（例: a.Title = title）
	a.Status = "created"
}

// ↓ 追加コマンドをここに定義する
// 例:
// func (a *Aggregate) Complete() {
//     a.Raise(EventCaptureCompleted, CaptureCompletedData{})
//     a.Status = "completed"
// }

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *Aggregate) Fail(input string, reason string) {
	a.Raise(EventCaptureFailed, FailedData{
		Input: input,
		Error: reason,
	})
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *Aggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventCaptureCompensated, CompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// TopicMapper maps event types to Kafka topics.
func TopicMapper(eventType string) string {
	switch eventType {
	case EventCaptureCreated:
		return platform.TopicCaptureCreated
	case EventCaptureFailed:
		return platform.TopicCaptureFailed
	case EventCaptureCompensated:
		return platform.TopicCaptureCompensated
	default:
		return ""
	}
}
