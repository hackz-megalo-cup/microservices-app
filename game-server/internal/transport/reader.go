// game-server/internal/transport/reader.go
package transport

import (
	"context"
	"io"
	"log"

	"github.com/coder/websocket"
	webtransport "github.com/quic-go/webtransport-go"
)

// ReadWT reads messages from a WebTransport session (bidirectional streams + datagrams)
// and sends them all to the returned channel.
func ReadWT(ctx context.Context, session *webtransport.Session) <-chan []byte {
	ch := make(chan []byte, 64)

	// Read from bidirectional streams (join, special)
	go func() {
		for {
			stream, err := session.AcceptStream(ctx)
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				data, err := io.ReadAll(stream)
				if err != nil {
					log.Printf("wt stream read error: %v", err)
					return
				}
				select {
				case ch <- data:
				case <-ctx.Done():
				}
			}()
		}
	}()

	// Read datagrams (tap)
	go func() {
		for {
			data, err := session.ReceiveDatagram(ctx)
			if err != nil {
				return
			}
			select {
			case ch <- data:
			case <-ctx.Done():
			}
		}
	}()

	// Close channel when context is done
	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch
}

// ReadWS reads messages from a WebSocket connection and sends them to the returned channel.
func ReadWS(ctx context.Context, conn *websocket.Conn) <-chan []byte {
	ch := make(chan []byte, 64)

	go func() {
		defer close(ch)
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
					log.Printf("ws read error: %v", err)
				}
				return
			}
			select {
			case ch <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}
