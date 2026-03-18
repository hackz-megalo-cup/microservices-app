package capture

const (
	EventCaptureCreated = "capture.created"
	// ↓ ドメインイベントを追加する
	// 例: EventCaptureCompleted = "capture.completed"
	// ⚠ 新しいイベントを追加したら platform/topics.go にもトピック定数と DefaultTopics() を追加すること

	EventCaptureFailed      = "capture.failed"      // main.go が参照 — 削除禁止
	EventCaptureCompensated = "capture.compensated" // main.go が参照 — 削除禁止
)

// CreatedData — 作成イベントのペイロード。
// ドメインに合わせてフィールドを書き換える。
type CreatedData struct {
	// 例: Title string `json:"title"`
}

// ↓ 追加イベントのペイロードをここに定義する
// 例: type CompletedData struct{}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type FailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type CompensatedData struct {
	Reason string `json:"reason"`
}
