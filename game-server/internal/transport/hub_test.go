package transport

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

// mockConn implements Conn for testing
type mockConn struct {
	mu             sync.Mutex
	reliableMsgs   [][]byte
	unreliableMsgs [][]byte
	closed         bool
}

func (m *mockConn) SendReliable(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reliableMsgs = append(m.reliableMsgs, data)
	return nil
}

func (m *mockConn) SendUnreliable(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unreliableMsgs = append(m.unreliableMsgs, data)
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func TestHub_RegisterAndBroadcast(t *testing.T) {
	hub := NewHub()

	c1 := &mockConn{}
	c2 := &mockConn{}
	id1 := uuid.New()
	id2 := uuid.New()

	hub.Register(id1, c1)
	hub.Register(id2, c2)

	hub.Broadcast([]byte(`{"t":"hp"}`))

	if len(c1.unreliableMsgs) != 1 {
		t.Errorf("c1 got %d unreliable msgs, want 1", len(c1.unreliableMsgs))
	}
	if len(c2.unreliableMsgs) != 1 {
		t.Errorf("c2 got %d unreliable msgs, want 1", len(c2.unreliableMsgs))
	}
}

func TestHub_BroadcastReliable(t *testing.T) {
	hub := NewHub()

	c1 := &mockConn{}
	id1 := uuid.New()
	hub.Register(id1, c1)

	hub.BroadcastReliable([]byte(`{"t":"finished"}`))

	if len(c1.reliableMsgs) != 1 {
		t.Errorf("c1 got %d reliable msgs, want 1", len(c1.reliableMsgs))
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := NewHub()

	c1 := &mockConn{}
	id1 := uuid.New()
	hub.Register(id1, c1)
	hub.Unregister(id1)

	hub.Broadcast([]byte(`{"t":"hp"}`))

	if len(c1.unreliableMsgs) != 0 {
		t.Errorf("unregistered conn got %d msgs, want 0", len(c1.unreliableMsgs))
	}
}

func TestHub_ClientCount(t *testing.T) {
	hub := NewHub()

	id1 := uuid.New()
	id2 := uuid.New()
	hub.Register(id1, &mockConn{})
	hub.Register(id2, &mockConn{})

	if hub.ClientCount() != 2 {
		t.Errorf("ClientCount() = %d, want 2", hub.ClientCount())
	}

	hub.Unregister(id1)

	if hub.ClientCount() != 1 {
		t.Errorf("ClientCount() = %d, want 1", hub.ClientCount())
	}
}
