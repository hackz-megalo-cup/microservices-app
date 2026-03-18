package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

type WSProxyHandler struct {
	locator *raidLocator
	store   *AllocationStore
}

func NewWSProxyHandler(namespace string, store *AllocationStore) *WSProxyHandler {
	locator, err := newRaidLocator(namespace)
	if err != nil {
		slog.Warn("agones lookup not available for websocket proxy", "error", err)
	}
	return &WSProxyHandler{locator: locator, store: store}
}

func (h *WSProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lobbyID := r.URL.Query().Get("lobbyId")

	target, ok, err := h.resolveTarget(r.Context(), lobbyID)
	if err != nil {
		http.Error(w, "failed to resolve raid target", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "no active raid found", http.StatusNotFound)
		return
	}

	clientConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Warn("failed to accept websocket client", "error", err)
		return
	}
	defer clientConn.CloseNow()

	dialCtx, cancel := context.WithCancel(r.Context())
	defer cancel()

	backendConn, resp, err := websocket.Dial(dialCtx, fmt.Sprintf("wss://%s:%d/ws", target.Host, target.Port), &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	})
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		_ = clientConn.Close(websocket.StatusInternalError, "failed to connect to raid server")
		slog.Error("failed to dial backend websocket", "lobbyId", lobbyID, "host", target.Host, "port", target.Port, "error", err)
		return
	}
	defer backendConn.CloseNow()

	errCh := make(chan error, 2)
	go h.proxyLoop(dialCtx, clientConn, backendConn, "client->backend", errCh)
	go h.proxyLoop(dialCtx, backendConn, clientConn, "backend->client", errCh)

	err = <-errCh
	cancel()

	if status := websocket.CloseStatus(err); status != -1 {
		_ = clientConn.Close(status, "closed")
		_ = backendConn.Close(status, "closed")
		return
	}

	if err != nil {
		slog.Warn("websocket proxy loop ended with error", "lobbyId", lobbyID, "error", err)
		_ = clientConn.Close(websocket.StatusInternalError, "proxy error")
		_ = backendConn.Close(websocket.StatusInternalError, "proxy error")
		return
	}

	_ = clientConn.Close(websocket.StatusNormalClosure, "closed")
	_ = backendConn.Close(websocket.StatusNormalClosure, "closed")
}

func (h *WSProxyHandler) resolveTarget(ctx context.Context, lobbyID string) (AllocateResponse, bool, error) {
	if lobbyID != "" {
		if h.locator != nil {
			resp, ok, err := h.locator.findByLobbyID(ctx, lobbyID)
			if err != nil {
				return AllocateResponse{}, false, err
			}
			if ok {
				return resp, true, nil
			}
		}

		resp, ok := h.store.Get(lobbyID)
		return resp, ok, nil
	}

	if h.locator != nil {
		entry, ok, err := h.locator.findFirstActive(ctx)
		if err != nil {
			return AllocateResponse{}, false, err
		}
		if ok {
			return AllocateResponse{
				Host:     entry.Host,
				Port:     entry.Port,
				CertHash: entry.CertHash,
			}, true, nil
		}
	}

	entry, ok := h.store.First()
	if !ok {
		return AllocateResponse{}, false, nil
	}

	return AllocateResponse{
		Host:     entry.Host,
		Port:     entry.Port,
		CertHash: entry.CertHash,
	}, true, nil
}

func (h *WSProxyHandler) proxyLoop(ctx context.Context, src, dst *websocket.Conn, direction string, errCh chan<- error) {
	for {
		typ, payload, err := src.Read(ctx)
		if err != nil {
			errCh <- err
			return
		}
		if err := dst.Write(ctx, typ, payload); err != nil {
			slog.Warn("failed to proxy websocket message", "direction", direction, "error", err)
			errCh <- err
			return
		}
	}
}
