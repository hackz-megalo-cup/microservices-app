package lobby

const (
	EventLobbyCreated = "lobby.created"
	// ↓ ドメインイベントを追加する
	// 例: EventLobbyCompleted = "lobby.completed"
	// ⚠ 新しいイベントを追加したら platform/topics.go にもトピック定数と DefaultTopics() を追加すること

	EventLobbyFailed      = "lobby.failed"      // main.go が参照 — 削除禁止
	EventLobbyCompensated = "lobby.compensated" // main.go が参照 — 削除禁止
)

// LobbyCreatedData — 作成イベントのペイロード。
// ドメインに合わせてフィールドを書き換える。
type LobbyCreatedData struct {
	// 例: Title string `json:"title"`
}

// ↓ 追加イベントのペイロードをここに定義する
// 例: type LobbyCompletedData struct{}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type LobbyFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type LobbyCompensatedData struct {
	Reason string `json:"reason"`
}
