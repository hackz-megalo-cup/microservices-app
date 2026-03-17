package masterdata

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ==========================================================.
// Aggregate — ドメインエンティティ。
// ↓ ドメインの状態フィールドを追加する（例: Title string）
// ==========================================================.

type MasterdataAggregate struct {
	platform.AggregateBase
	Status string
}

func NewMasterdataAggregate(id string) *MasterdataAggregate {
	return &MasterdataAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *MasterdataAggregate) StreamType() string { return "masterdata" }

// ApplyEvent はイベントを再生して状態を復元する。
// Created の case を書き換え、追加イベントの case を足す。
func (a *MasterdataAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventMasterdataCreated:
		var d MasterdataCreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		// ↓ フィールドの復元を書く（例: a.Title = d.Title）
		a.Status = "created"
	// ↓ 追加イベントの case をここに足す
	// 例:
	// case EventMasterdataCompleted:
	//     a.Status = "completed"
	case EventMasterdataFailed:
		a.Status = "failed"
	case EventMasterdataCompensated:
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

// Create — 引数をドメインに合わせて変更する（例: Create(title string)）.
func (a *MasterdataAggregate) Create() {
	a.Raise(EventMasterdataCreated, MasterdataCreatedData{
		// ↓ フィールドを渡す（例: Title: title）
	})
	// ↓ 状態を更新する（例: a.Title = title）
	a.Status = "created"
}

// ↓ 追加コマンドをここに定義する。
// 例:
// func (a *MasterdataAggregate) Complete() {
//     a.Raise(EventMasterdataCompleted, MasterdataCompletedData{})
//     a.Status = "completed"
// }.

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *MasterdataAggregate) Fail(input string, reason string) {
	a.Raise(EventMasterdataFailed, MasterdataFailedData{
		Input: input,
		Error: reason,
	})
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *MasterdataAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventMasterdataCompensated, MasterdataCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// MasterdataTopicMapper maps event types to Kafka topics.
func MasterdataTopicMapper(eventType string) string {
	switch eventType {
	case EventMasterdataCreated:
		return platform.TopicMasterdataCreated
	case EventMasterdataFailed:
		return platform.TopicMasterdataFailed
	default:
		return ""
	}
}
