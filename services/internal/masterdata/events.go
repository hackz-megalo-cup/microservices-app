package masterdata

const (
	EventMasterdataCreated = "masterdata.created"
	// ↓ ドメインイベントを追加する
	// 例: EventMasterdataCompleted = "masterdata.completed"
	// ⚠ 新しいイベントを追加したら platform/topics.go にもトピック定数と DefaultTopics() を追加すること。

	EventMasterdataFailed      = "masterdata.failed"      // main.go が参照 — 削除禁止
	EventMasterdataCompensated = "masterdata.compensated" // main.go が参照 — 削除禁止
)

// MasterdataCreatedData — 作成イベントのペイロード。
// ドメインに合わせてフィールドを書き換える。
type MasterdataCreatedData struct {
	// 例: Title string `json:"title"`
}

// ↓ 追加イベントのペイロードをここに定義する。
// 例: type MasterdataCompletedData struct{}.

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。

type MasterdataFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type MasterdataCompensatedData struct {
	Reason string `json:"reason"`
}
