package raidlobby

const (
	EventCreated       = "raid_lobby.created"
	EventFinished      = "raid_lobby.finished"
	EventUserJoined    = "raid.user_joined"
	EventBattleStarted = "raid.battle_started"

	EventFailed      = "raid_lobby.failed"      // main.go が参照 — 削除禁止
	EventCompensated = "raid_lobby.compensated" // main.go が参照 — 削除禁止
)

type CreatedData struct {
	BossPokemonID string `json:"boss_pokemon_id"`
}

type FinishedData struct {
	LobbyID   string `json:"lobby_id"`
	SessionID string `json:"session_id"`
	Result    string `json:"result"`
}

type UserJoinedData struct {
	LobbyID       string `json:"lobby_id"`
	UserID        string `json:"user_id"`
	ParticipantID string `json:"participant_id"`
}

type BattleStartedData struct {
	LobbyID            string   `json:"lobby_id"`
	BossPokemonID      string   `json:"boss_pokemon_id"`
	ParticipantUserIDs []string `json:"participant_user_ids"`
	SessionID          string   `json:"session_id"`
}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type FailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type CompensatedData struct {
	Reason string `json:"reason"`
}
