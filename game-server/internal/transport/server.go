package transport

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/quic-go/quic-go/http3"
	webtransport "github.com/quic-go/webtransport-go"
)

type DualServer struct {
	port   int32
	wtCert tls.Certificate
	wsCert tls.Certificate
	onWT   func(*webtransport.Session)
	onWS   func(http.ResponseWriter, *http.Request)
}

func NewDualServer(port int32, wtCert, wsCert tls.Certificate, onWT func(*webtransport.Session), onWS func(http.ResponseWriter, *http.Request)) *DualServer {
	return &DualServer{port: port, wtCert: wtCert, wsCert: wsCert, onWT: onWT, onWS: onWS}
}

func (ds *DualServer) Start() error {
	addr := fmt.Sprintf(":%d", ds.port)

	// UDP: WebTransport
	h3s := &http3.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			NextProtos:   []string{http3.NextProtoH3},
			Certificates: []tls.Certificate{ds.wtCert},
		},
	}
	webtransport.ConfigureHTTP3Server(h3s)

	wtServer := &webtransport.Server{
		H3:          h3s,
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/wt", func(w http.ResponseWriter, r *http.Request) {
		session, err := wtServer.Upgrade(w, r)
		if err != nil {
			log.Printf("wt upgrade error: %v", err)
			return
		}
		ds.onWT(session)
	})
	mux.HandleFunc("/ws", ds.onWS)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h3s.Handler = mux

	// TCP: WebSocket (HTTPS)
	tcpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			Certificates: []tls.Certificate{ds.wsCert},
		},
	}

	errCh := make(chan error, 2)
	go func() { errCh <- wtServer.ListenAndServe() }()
	go func() {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			errCh <- err
			return
		}
		errCh <- tcpServer.ServeTLS(ln, "", "")
	}()

	return <-errCh
}
