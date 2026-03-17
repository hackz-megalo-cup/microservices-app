package raid_lobby

const (
	EventRaidLobbyCreated = "raid_lobby.created"
	// ↓ ドメインイベントを追加する
	// 例: EventRaidLobbyCompleted = "raid_lobby.completed"
	// ⚠ 新しいイベントを追加したら platform/topics.go にもトピック定数と DefaultTopics() を追加すること

	EventRaidLobbyFailed      = "raid_lobby.failed"      // main.go が参照 — 削除禁止
	EventRaidLobbyCompensated = "raid_lobby.compensated" // main.go が参照 — 削除禁止
)

// RaidLobbyCreatedData — 作成イベントのペイロード。
// ドメインに合わせてフィールドを書き換える。
type RaidLobbyCreatedData struct {
	// 例: Title string `json:"title"`
}

// ↓ 追加イベントのペイロードをここに定義する
// 例: type RaidLobbyCompletedData struct{}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type RaidLobbyFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type RaidLobbyCompensatedData struct {
	Reason string `json:"reason"`
}
