package transport

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type WSConn struct {
	conn    *websocket.Conn
	parentCtx context.Context
	mu      sync.Mutex
}

func NewWSConn(ctx context.Context, conn *websocket.Conn) *WSConn {
	return &WSConn{conn: conn, parentCtx: ctx}
}

func (c *WSConn) SendReliable(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, cancel := context.WithTimeout(c.parentCtx, 5*time.Second)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *WSConn) SendUnreliable(data []byte) error {
	return c.SendReliable(data) // WebSocket has no datagram, use reliable
}

func (c *WSConn) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "closed")
}
