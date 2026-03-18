package lobby

const (
	EventLobbyActivePokemonSet = "lobby.active_pokemon_set"

	EventLobbyFailed      = "lobby.failed"      // main.go が参照 — 削除禁止
	EventLobbyCompensated = "lobby.compensated" // main.go が参照 — 削除禁止
)

// LobbyActivePokemonSetData is the payload for the active pokemon set event.
type LobbyActivePokemonSetData struct {
	UserID    string `json:"user_id"`
	PokemonID string `json:"pokemon_id"`
}

// --- 以下は main.go の補償ハンドラが使用。型名とフィールドは残すこと。 ---

type LobbyFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

type LobbyCompensatedData struct {
	Reason string `json:"reason"`
}
