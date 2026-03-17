package transport

import (
	"log"
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	clients map[uuid.UUID]Conn
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]Conn),
	}
}

func (h *Hub) Register(userID uuid.UUID, conn Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[userID] = conn
}

func (h *Hub) Unregister(userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, userID)
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for uid, conn := range h.clients {
		if err := conn.SendUnreliable(data); err != nil {
			log.Printf("broadcast to %s failed: %v", uid, err)
		}
	}
}

func (h *Hub) BroadcastReliable(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for uid, conn := range h.clients {
		if err := conn.SendReliable(data); err != nil {
			log.Printf("broadcast reliable to %s failed: %v", uid, err)
		}
	}
}
