package item

const (
	EventItemCreated = "item.created"
	// ↓ ドメインイベントを追加する
	// 例: EventItemCompleted = "item.completed"
	// ⚠ 新しいイベントを追加したら platform/topics.go にもトピック定数と DefaultTopics() を追加すること。

	EventItemFailed      = "item.failed"      // main.go が参照 — 削除禁止
	EventItemCompensated = "item.compensated" // main.go が参照 — 削除禁止
)

// ItemCreatedData — 作成イベントのペイロード。
// ドメインに合わせてフィールドを書き換える。
type CreatedData struct {
	UserID   string `json:"user_id"`
	ItemID   string `json:"item_id"`
	Quantity int32  `json:"quantity"`
	Reason   string `json:"reason"`
}

// ↓ 追加イベントのペイロードをここに定義する
// 例: type ItemCompletedData struct{}。

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---。

type FailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type CompensatedData struct {
	Reason string `json:"reason"`
}
