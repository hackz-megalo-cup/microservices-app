package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	webtransport "github.com/quic-go/webtransport-go"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/agones"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/battle"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/cert"
	gamekafka "github.com/hackz-megalo-cup/microservices-app/game-server/internal/kafka"
	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/transport"
)

func main() {
	log.Println("game-server starting...")

	localDev := os.Getenv("LOCAL_DEV") == "true"

	if localDev {
		runLocalDev()
	} else {
		runProduction()
	}
}

func runLocalDev() {
	log.Println("=== LOCAL DEV MODE ===")

	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "7777"
	}
	port := int32(0)
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		log.Fatalf("invalid PORT %q: %v", portStr, err)
	}

	// Generate ephemeral cert for WebTransport
	wtCert, certHash, err := cert.GenerateEphemeral()
	if err != nil {
		log.Fatalf("cert gen: %v", err)
	}

	// Load mkcert certificate for WebSocket (browser needs trusted CA)
	wsCertPath := os.Getenv("WS_CERT_PATH")
	if wsCertPath == "" {
		wsCertPath = "/tmp"
	}
	wsCert, err := tls.LoadX509KeyPair(wsCertPath+"/tls.crt", wsCertPath+"/tls.key")
	if err != nil {
		log.Printf("mkcert load failed (WS will use ephemeral cert — browser may reject): %v", err)
		wsCert = wtCert
	}

	log.Printf("port: %d", port)
	log.Printf("cert hash: %s", certHash)
	log.Println("======================")

	// Create session immediately
	session := battle.NewSession(uuid.New(), uuid.New(), 50000, "normal", battle.TypeMatchup{}, 300*time.Second)
	hub := transport.NewHub()
	handler := transport.NewHandler(hub, session)

	log.Println("battle session created (local dev)")

	// Start timeout timer
	time.AfterFunc(session.TimeoutDuration, func() {
		session.Timeout()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	onWT := func(wtSession *webtransport.Session) {
		userID := uuid.New()
		conn := transport.NewWTConn(wtSession)
		hub.Register(userID, conn)
		log.Printf("wt client connected: %s", userID)

		messages := transport.ReadWT(ctx, wtSession)
		for msg := range messages {
			handler.HandleMessage(userID, msg)
		}
		hub.Unregister(userID)
		log.Printf("wt client disconnected: %s", userID)
	}

	onWS := func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws accept error: %v", err)
			return
		}
		userID := uuid.New()
		conn := transport.NewWSConn(ctx, wsConn)
		hub.Register(userID, conn)
		log.Printf("ws client connected: %s", userID)

		messages := transport.ReadWS(ctx, wsConn)
		for msg := range messages {
			handler.HandleMessage(userID, msg)
		}
		hub.Unregister(userID)
		log.Printf("ws client disconnected: %s", userID)
	}

	srv := transport.NewDualServer(port, wtCert, wsCert, onWT, onWS)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	log.Printf("game-server listening on :%d (WebTransport + WebSocket)", port)

	<-session.Done()
	log.Printf("battle finished: result=%s", session.Result())
	cancel()
	log.Println("game-server shut down")
}

func runProduction() {
	// 1. Agones SDK init
	lc, err := agones.NewLifecycle()
	if err != nil {
		log.Fatalf("agones init: %v", err)
	}

	// 2. Get allocated port
	port, err := lc.Port()
	if err != nil {
		log.Fatalf("get port: %v", err)
	}
	log.Printf("allocated port: %d", port)

	// 3. Generate ECDSA ephemeral cert for WebTransport
	wtCert, certHash, err := cert.GenerateEphemeral()
	if err != nil {
		log.Fatalf("cert gen: %v", err)
	}
	log.Printf("cert hash: %s", certHash)

	// 4. Load mkcert certificate for WebSocket
	wsCertPath := os.Getenv("WS_CERT_PATH")
	if wsCertPath == "" {
		wsCertPath = "/etc/tls/mkcert"
	}
	wsCert, err := tls.LoadX509KeyPair(wsCertPath+"/tls.crt", wsCertPath+"/tls.key")
	if err != nil {
		log.Printf("mkcert load failed (using ephemeral cert for WS too): %v", err)
		wsCert = wtCert
	}

	// Hub and handler (created when session starts), protected by mu
	var (
		mu      sync.RWMutex
		hub     *transport.Hub
		handler *transport.Handler
		session *battle.Session
	)

	sessionReady := make(chan struct{})

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "redpanda.redpanda.svc.cluster.local:9093"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. WebTransport handler
	onWT := func(wtSession *webtransport.Session) {
		mu.RLock()
		h, hdl := hub, handler
		mu.RUnlock()
		if h == nil || hdl == nil {
			log.Println("wt connection rejected: no active session")
			wtSession.CloseWithError(0, "no session")
			return
		}
		userID := uuid.New()
		conn := transport.NewWTConn(wtSession)
		h.Register(userID, conn)
		log.Printf("wt client connected: %s", userID)

		messages := transport.ReadWT(ctx, wtSession)
		for msg := range messages {
			hdl.HandleMessage(userID, msg)
		}
		h.Unregister(userID)
		log.Printf("wt client disconnected: %s", userID)
	}

	// 6. WebSocket handler
	onWS := func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		h, hdl := hub, handler
		mu.RUnlock()
		if h == nil || hdl == nil {
			http.Error(w, "no active session", http.StatusServiceUnavailable)
			return
		}
		wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws accept error: %v", err)
			return
		}
		userID := uuid.New()
		conn := transport.NewWSConn(ctx, wsConn)
		h.Register(userID, conn)
		log.Printf("ws client connected: %s", userID)

		messages := transport.ReadWS(ctx, wsConn)
		for msg := range messages {
			hdl.HandleMessage(userID, msg)
		}
		h.Unregister(userID)
		log.Printf("ws client disconnected: %s", userID)
	}

	// 7. Start dual-stack server
	srv := transport.NewDualServer(port, wtCert, wsCert, onWT, onWS)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// 8. Watch for Allocation
	var allocateOnce sync.Once
	if err := lc.WatchAllocated(func(annotations map[string]string) {
		allocateOnce.Do(func() {
			lobbyIDStr := annotations["raid.lobby-id"]
			bossPokemonIDStr := annotations["raid.boss-pokemon-id"]

			lobbyID, err := uuid.Parse(lobbyIDStr)
			if err != nil {
				log.Fatalf("invalid lobby-id annotation %q: %v", lobbyIDStr, err)
			}
			bossPokemonID, err := uuid.Parse(bossPokemonIDStr)
			if err != nil {
				log.Fatalf("invalid boss-pokemon-id annotation %q: %v", bossPokemonIDStr, err)
			}

			matchups := battle.TypeMatchup{}

			mu.Lock()
			session = battle.NewSession(lobbyID, bossPokemonID, 50000, "normal", matchups, 300*time.Second)
			hub = transport.NewHub()
			handler = transport.NewHandler(hub, session)
			mu.Unlock()

			// Start timeout timer
			time.AfterFunc(session.TimeoutDuration, func() {
				session.Timeout()
			})

			close(sessionReady)

			log.Printf("battle session created: lobby=%s boss=%s", lobbyIDStr, bossPokemonIDStr)
		})
	}); err != nil {
		log.Fatalf("watch allocated: %v", err)
	}

	// 9. Publish cert hash
	if err := lc.SetCertHash(certHash); err != nil {
		log.Fatalf("set cert hash: %v", err)
	}

	// 10. Mark as Ready
	if err := lc.Ready(); err != nil {
		log.Fatalf("ready: %v", err)
	}
	log.Println("game-server ready, waiting for allocation...")

	// Wait for session then battle completion
	<-sessionReady
	mu.RLock()
	s := session
	mu.RUnlock()
	<-s.Done()
	log.Printf("battle finished: result=%s", s.Result())

	// Kafka publish
	kClient, err := kgo.NewClient(kgo.SeedBrokers(strings.Split(kafkaBrokers, ",")...))
	if err != nil {
		log.Printf("kafka client error: %v", err)
	} else {
		participantIDs := s.ParticipantIDs()
		event := gamekafka.BattleFinishedEvent{
			SessionID:      s.SessionID,
			LobbyID:        s.LobbyID,
			BossPokemonID:  s.BossPokemonID,
			Result:         s.Result(),
			ParticipantIDs: participantIDs,
		}
		record := gamekafka.BuildBattleFinishedRecord(event)
		if record != nil {
			pctx, pcancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer pcancel()
			if err := kClient.ProduceSync(pctx, record).FirstErr(); err != nil {
				log.Printf("kafka publish error: %v", err)
			} else {
				log.Println("battle.finished published to Kafka")
			}
		}
		kClient.Close()
	}

	// Shutdown
	cancel()
	if err := lc.Shutdown(); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("game-server shut down")
}
