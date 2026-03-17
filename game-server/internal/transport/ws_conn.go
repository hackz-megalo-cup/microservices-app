package transport

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type WSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewWSConn(conn *websocket.Conn) *WSConn {
	return &WSConn{conn: conn}
}

func (c *WSConn) SendReliable(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *WSConn) SendUnreliable(data []byte) error {
	return c.SendReliable(data) // WebSocket has no datagram, use reliable
}

func (c *WSConn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "closed")
}
