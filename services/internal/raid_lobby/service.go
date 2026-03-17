package raidlobby

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type lobbyUpdate struct {
	eventType string
	payload   string
}

type broadcaster struct {
	mu   sync.Mutex
	subs map[string][]chan lobbyUpdate
}

func newBroadcaster() *broadcaster {
	return &broadcaster{subs: make(map[string][]chan lobbyUpdate)}
}

func (b *broadcaster) subscribe(lobbyID string) chan lobbyUpdate {
	ch := make(chan lobbyUpdate, 4)
	b.mu.Lock()
	b.subs[lobbyID] = append(b.subs[lobbyID], ch)
	b.mu.Unlock()
	return ch
}

func (b *broadcaster) unsubscribe(lobbyID string, ch chan lobbyUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subs[lobbyID]
	for i, s := range subs {
		if s == ch {
			b.subs[lobbyID] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

func (b *broadcaster) publish(lobbyID string, update lobbyUpdate) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subs[lobbyID] {
		select {
		case ch <- update:
		default:
		}
	}
}

type Service struct {
	eventStore  *platform.EventStore
	outbox      *platform.OutboxStore
	db          *pgxpool.Pool
	broadcaster *broadcaster
}

func NewService(eventStore *platform.EventStore, outbox *platform.OutboxStore, db *pgxpool.Pool) *Service {
	return &Service{
		eventStore:  eventStore,
		outbox:      outbox,
		db:          db,
		broadcaster: newBroadcaster(),
	}
}

func (s *Service) CreateRaid(ctx context.Context, req *connect.Request[pb.CreateRaidRequest]) (*connect.Response[pb.CreateRaidResponse], error) {
	bossPokemonID := req.Msg.GetBossPokemonId()
	if bossPokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("boss_pokemon_id is required"))
	}

	agg := NewRaidLobbyAggregate(uuid.NewString())
	agg.Create(bossPokemonID)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, RaidLobbyTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	if s.db != nil {
		if _, err := s.db.Exec(ctx,
			`INSERT INTO raid_lobby (id, boss_pokemon_id, status, created_at) VALUES ($1, $2, 'waiting', $3)`,
			agg.AggregateID(), bossPokemonID, time.Now().UTC(),
		); err != nil {
			slog.Error("failed to insert raid_lobby read model", "error", err)
		}
	}

	return connect.NewResponse(&pb.CreateRaidResponse{
		LobbyId: agg.AggregateID(),
	}), nil
}

func (s *Service) JoinRaid(_ context.Context, req *connect.Request[pb.JoinRaidRequest]) (*connect.Response[pb.JoinRaidResponse], error) {
	if req.Msg.GetLobbyId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}
	return connect.NewResponse(&pb.JoinRaidResponse{
		ParticipantId: uuid.NewString(),
	}), nil
}

func (s *Service) StartBattle(_ context.Context, req *connect.Request[pb.StartBattleRequest]) (*connect.Response[pb.StartBattleResponse], error) {
	if req.Msg.GetLobbyId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}
	return connect.NewResponse(&pb.StartBattleResponse{
		BattleSessionId: uuid.NewString(),
	}), nil
}

func (s *Service) StreamLobby(ctx context.Context, req *connect.Request[pb.StreamLobbyRequest], stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	if req.Msg.GetLobbyId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}

	lobbyID := req.Msg.GetLobbyId()
	ch := s.broadcaster.subscribe(lobbyID)
	defer s.broadcaster.unsubscribe(lobbyID, ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&pb.StreamLobbyResponse{
				EventType: update.eventType,
				LobbyId:   lobbyID,
				Payload:   update.payload,
			}); err != nil {
				return err
			}
		}
	}
}

// HandleBattleFinished is called by the battle.finished Kafka consumer.
// Transitions the lobby to finished in event store + read model, then notifies StreamLobby subscribers.
func (s *Service) HandleBattleFinished(ctx context.Context, lobbyID, sessionID, result string) error {
	agg := NewRaidLobbyAggregate(lobbyID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		return fmt.Errorf("load aggregate: %w", err)
	}
	if agg.Status == "finished" {
		return nil // idempotent
	}

	agg.Finish(sessionID, result)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, RaidLobbyTopicMapper); err != nil {
		return fmt.Errorf("save aggregate: %w", err)
	}

	if s.db != nil {
		if _, err := s.db.Exec(ctx,
			`UPDATE raid_lobby SET status = 'finished' WHERE id = $1`,
			lobbyID,
		); err != nil {
			slog.Error("failed to update raid_lobby status", "lobby_id", lobbyID, "error", err)
		}
	}

	payload, _ := json.Marshal(RaidLobbyFinishedData{
		LobbyID:   lobbyID,
		SessionID: sessionID,
		Result:    result,
	})
	s.broadcaster.publish(lobbyID, lobbyUpdate{
		eventType: EventRaidLobbyFinished,
		payload:   string(payload),
	})
	return nil
}
