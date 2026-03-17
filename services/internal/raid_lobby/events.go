package raidlobby

const (
	EventRaidLobbyCreated  = "raid_lobby.created"
	EventRaidLobbyFinished = "raid_lobby.finished"

	EventRaidLobbyFailed      = "raid_lobby.failed"      // main.go が参照 — 削除禁止
	EventRaidLobbyCompensated = "raid_lobby.compensated" // main.go が参照 — 削除禁止
)

type RaidLobbyCreatedData struct {
	BossPokemonID string `json:"boss_pokemon_id"`
}

type RaidLobbyFinishedData struct {
	LobbyID   string `json:"lobby_id"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type RaidLobbyFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type RaidLobbyCompensatedData struct {
	Reason string `json:"reason"`
}
