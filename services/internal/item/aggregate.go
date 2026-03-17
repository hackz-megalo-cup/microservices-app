package item

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ==========================================================.
// Aggregate — ドメインエンティティ。
// ↓ ドメインの状態フィールドを追加する（例: Title string）
// ==========================================================.

type ItemAggregate struct {
	platform.AggregateBase
	Status string
}

func NewItemAggregate(id string) *ItemAggregate {
	return &ItemAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *ItemAggregate) StreamType() string { return "item" }

// ApplyEvent はイベントを再生して状態を復元する。
// Created の case を書き換え、追加イベントの case を足す。
func (a *ItemAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventItemCreated:
		var d CreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		// ↓ フィールドの復元を書く（例: a.Title = d.Title）。
		a.Status = "created"
	// ↓ 追加イベントの case をここに足す。
	// 例:。
	// case EventItemCompleted:。
	//     a.Status = "completed"。
	case EventItemFailed:
		a.Status = "failed"
	case EventItemCompensated:
		a.Status = "compensated"
	}
}

// ==========================================================.
// コマンドメソッド — Raise() でイベントを発行し、状態を更新する。
//
// Fail / Compensate は main.go の補償ハンドラが参照 — 削除禁止。
// AggregateID() で集約の ID を取得できる。
// 既存集約のロード: platform.LoadAggregate(ctx, eventStore, agg)。
// ==========================================================.

// Create — 引数をドメインに合わせて変更する（例: Create(title string)）。
func (a *ItemAggregate) Create(userID, itemID string, quantity int32, reason string) {
	a.Raise(EventItemCreated, CreatedData{
		UserID:   userID,
		ItemID:   itemID,
		Quantity: quantity,
		Reason:   reason,
	})
	a.Status = "created"
}

// ↓ 追加コマンドをここに定義する。
// 例:
// func (a *ItemAggregate) Complete() {
//     a.Raise(EventItemCompleted, ItemCompletedData{})
//     a.Status = "completed"
// }.

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *ItemAggregate) Fail(input string, reason string) {
	a.Raise(EventItemFailed, FailedData{
		Input: input,
		Error: reason,
	})
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *ItemAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventItemCompensated, CompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// ItemTopicMapper maps event types to Kafka topics.
func ItemTopicMapper(eventType string) string {
	switch eventType {
	case EventItemCreated:
		return platform.TopicItemCreated
	case EventItemFailed:
		return platform.TopicItemFailed
	default:
		return ""
	}
}
