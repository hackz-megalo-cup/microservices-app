package capture

const (
	EventCaptureCreated = "capture.created"
	// ↓ ドメインイベントを追加する
	// 例: EventCaptureCompleted = "capture.completed"
	// ⚠ 新しいイベントを追加したら platform/topics.go にもトピック定数と DefaultTopics() を追加すること

	EventCaptureFailed      = "capture.failed"      // main.go が参照 — 削除禁止
	EventCaptureCompensated = "capture.compensated" // main.go が参照 — 削除禁止
)

// CaptureCreatedData — 作成イベントのペイロード。
// ドメインに合わせてフィールドを書き換える。
type CaptureCreatedData struct {
	// 例: Title string `json:"title"`
}

// ↓ 追加イベントのペイロードをここに定義する
// 例: type CaptureCompletedData struct{}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type CaptureFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type CaptureCompensatedData struct {
	Reason string `json:"reason"`
}
