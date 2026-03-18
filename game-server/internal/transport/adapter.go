package transport

import (
	"encoding/json"
	"fmt"
)

// ClientMessage is the incoming message from a client.
type ClientMessage struct {
	T      string `json:"t"`
	UserID string `json:"userId,omitempty"`
}

// ServerMessage types sent to clients.
type JoinedMessage struct {
	T            string   `json:"t"` // "joined"
	SessionID    string   `json:"sessionId"`
	BossHP       int32    `json:"bossHp"`
	BossMaxHP    int32    `json:"bossMaxHp"`
	Participants []string `json:"participants"`
	TimeoutSec   int      `json:"timeoutSec"`
}

type HPMessage struct {
	T       string `json:"t"` // "hp"
	HP      int32  `json:"hp"`
	MaxHP   int32  `json:"maxHp"`
	LastDmg int32  `json:"lastDmg"`
	By      string `json:"by"`
}

type SpecialUsedMessage struct {
	T        string `json:"t"` // "special_used"
	UserID   string `json:"userId"`
	MoveName string `json:"moveName"`
	Dmg      int32  `json:"dmg"`
	BossHP   int32  `json:"bossHp"`
}

type FinishedMessage struct {
	T       string `json:"t"` // "finished"
	Result  string `json:"result"`
	BossHP  int32  `json:"bossHp"`
	Elapsed int    `json:"elapsed"`
}

type TimeSyncMessage struct {
	T            string `json:"t"` // "time_sync"
	RemainingSec int    `json:"remainingSec"`
}

// Conn abstracts a client connection (WebTransport session or WebSocket conn).
type Conn interface {
	// SendReliable sends a message that must be delivered (stream/websocket frame).
	SendReliable(data []byte) error
	// SendUnreliable sends a best-effort message (datagram/websocket frame).
	SendUnreliable(data []byte) error
	// Close closes the connection.
	Close() error
}

func ParseMessage(data []byte) (ClientMessage, error) {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, fmt.Errorf("invalid message: %w", err)
	}
	if msg.T == "" {
		return msg, fmt.Errorf("missing message type")
	}
	return msg, nil
}

func MarshalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
