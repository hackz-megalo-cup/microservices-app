package transport

import (
	webtransport "github.com/quic-go/webtransport-go"
)

type WTConn struct {
	session *webtransport.Session
}

func NewWTConn(session *webtransport.Session) *WTConn {
	return &WTConn{session: session}
}

func (c *WTConn) SendReliable(data []byte) error {
	stream, err := c.session.OpenUniStream()
	if err != nil {
		return err
	}
	defer stream.Close()
	_, err = stream.Write(data)
	return err
}

func (c *WTConn) SendUnreliable(data []byte) error {
	return c.session.SendDatagram(data)
}

func (c *WTConn) Close() error {
	return c.session.CloseWithError(0, "closed")
}
