package raidlobby

import (
	"encoding/json"
	"log/slog"

	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ==========================================================.
// Aggregate — ドメインエンティティ。
// ↓ ドメインの状態フィールドを追加する（例: Title string）
// ==========================================================.

type RaidLobbyAggregate struct {
	platform.AggregateBase
	Status string
}

func NewRaidLobbyAggregate(id string) *RaidLobbyAggregate {
	return &RaidLobbyAggregate{
		AggregateBase: platform.NewAggregateBase(id),
	}
}

func (a *RaidLobbyAggregate) StreamType() string { return "raid_lobby" }

// ApplyEvent はイベントを再生して状態を復元する。
// Created の case を書き換え、追加イベントの case を足す。
func (a *RaidLobbyAggregate) ApplyEvent(eventType string, data json.RawMessage) {
	switch eventType {
	case EventRaidLobbyCreated:
		var d RaidLobbyCreatedData
		if err := json.Unmarshal(data, &d); err != nil {
			slog.Warn("failed to unmarshal created data", "error", err)
		}
		// ↓ フィールドの復元を書く（例: a.Title = d.Title）
		a.Status = "created"
	// ↓ 追加イベントの case をここに足す
	// 例:
	// case EventRaidLobbyCompleted:
	//     a.Status = "completed"
	case EventRaidLobbyFailed:
		a.Status = "failed"
	case EventRaidLobbyCompensated:
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
func (a *RaidLobbyAggregate) Create() {
	a.Raise(EventRaidLobbyCreated, RaidLobbyCreatedData{
		// ↓ フィールドを渡す（例: Title: title）
	})
	// ↓ 状態を更新する（例: a.Title = title）
	a.Status = "created"
}

// ↓ 追加コマンドをここに定義する
// 例:
// func (a *RaidLobbyAggregate) Complete() {
//     a.Raise(EventRaidLobbyCompleted, RaidLobbyCompletedData{})
//     a.Status = "completed"
// }

// Fail records a failed operation — main.go が参照、削除禁止。
func (a *RaidLobbyAggregate) Fail(input string, reason string) {
	a.Raise(EventRaidLobbyFailed, RaidLobbyFailedData{
		Input: input,
		Error: reason,
	})
	a.Status = "failed"
}

// Compensate marks this aggregate as compensated — main.go が参照、削除禁止。
func (a *RaidLobbyAggregate) Compensate(reason string) {
	if a.Status == "compensated" {
		return
	}
	a.Raise(EventRaidLobbyCompensated, RaidLobbyCompensatedData{
		Reason: reason,
	})
	a.Status = "compensated"
}

// RaidLobbyTopicMapper maps event types to Kafka topics.
func RaidLobbyTopicMapper(eventType string) string {
	switch eventType {
	case EventRaidLobbyCreated:
		return platform.TopicRaidLobbyCreated
	case EventRaidLobbyFailed:
		return platform.TopicRaidLobbyFailed
	default:
		return ""
	}
}
