package raidlobby

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	masterdatav1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1/masterdatav1connect"
	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore       *platform.EventStore
	outbox           *platform.OutboxStore
	db               *pgxpool.Pool
	masterdataClient masterdatav1connect.MasterdataServiceClient
	brokers          []string
}

func NewService(
	eventStore *platform.EventStore,
	outbox *platform.OutboxStore,
	db *pgxpool.Pool,
	masterdataClient masterdatav1connect.MasterdataServiceClient,
	brokers []string,
) *Service {
	return &Service{
		eventStore:       eventStore,
		outbox:           outbox,
		db:               db,
		masterdataClient: masterdataClient,
		brokers:          brokers,
	}
}

func (s *Service) CreateRaid(ctx context.Context, req *connect.Request[pb.CreateRaidRequest]) (*connect.Response[pb.CreateRaidResponse], error) {
	bossPokemonID := req.Msg.GetBossPokemonId()
	if bossPokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("boss_pokemon_id is required"))
	}

	if s.masterdataClient != nil {
		_, err := s.masterdataClient.GetPokemon(ctx, connect.NewRequest(&masterdatav1.GetPokemonRequest{
			Id: bossPokemonID,
		}))
		if err != nil {
			slog.Error("failed to get pokemon from masterdata", "pokemon_id", bossPokemonID, "error", err)
			if connect.CodeOf(err) == connect.CodeNotFound {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pokemon not found: %s", bossPokemonID))
			}
			return nil, err
		}
	}

	lobbyID := uuid.NewString()

	agg := NewAggregate(lobbyID)
	agg.Create(bossPokemonID)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	if s.db != nil {
		if _, err := s.db.Exec(ctx,
			`INSERT INTO raid_lobby (id, boss_pokemon_id, status, created_at) VALUES ($1, $2, 'waiting', $3)`,
			lobbyID, bossPokemonID, time.Now().UTC(),
		); err != nil {
			slog.Error("failed to insert raid_lobby", "error", err)
		}
	}

	return connect.NewResponse(&pb.CreateRaidResponse{
		LobbyId: lobbyID,
	}), nil
}

func (s *Service) JoinRaid(ctx context.Context, req *connect.Request[pb.JoinRaidRequest]) (*connect.Response[pb.JoinRaidResponse], error) {
	lobbyID := req.Msg.GetLobbyId()
	userID := req.Msg.GetUserId()
	if lobbyID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	if s.db != nil {
		var status string
		err := s.db.QueryRow(ctx, `SELECT status FROM raid_lobby WHERE id = $1`, lobbyID).Scan(&status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("lobby not found: %s", lobbyID))
			}
			slog.Error("failed to query lobby", "lobby_id", lobbyID, "error", err)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query lobby"))
		}
		if status != "waiting" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("lobby is not accepting participants (status: %s)", status))
		}
	}

	participantID := uuid.NewString()

	agg := NewAggregate(lobbyID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Error("failed to load aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load lobby"))
	}
	agg.Join(userID, participantID)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to join lobby"))
	}

	if s.db != nil {
		if _, err := s.db.Exec(ctx,
			`INSERT INTO raid_participant (id, lobby_id, user_id, joined_at) VALUES ($1, $2, $3, $4)`,
			participantID, lobbyID, userID, time.Now().UTC(),
		); err != nil {
			slog.Error("failed to insert raid_participant", "error", err)
		}
	}

	return connect.NewResponse(&pb.JoinRaidResponse{
		ParticipantId: participantID,
	}), nil
}

func (s *Service) validateLobbyForBattle(ctx context.Context, lobbyID string) error {
	if s.db == nil {
		return nil
	}
	var status string
	err := s.db.QueryRow(ctx, `SELECT status FROM raid_lobby WHERE id = $1`, lobbyID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("lobby not found: %s", lobbyID))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query lobby: %w", err))
	}
	if status != "waiting" {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("lobby is not in waiting status (status: %s)", status))
	}
	return nil
}

func (s *Service) StartBattle(ctx context.Context, req *connect.Request[pb.StartBattleRequest]) (*connect.Response[pb.StartBattleResponse], error) {
	lobbyID := req.Msg.GetLobbyId()
	if lobbyID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}

	if err := s.validateLobbyForBattle(ctx, lobbyID); err != nil {
		return nil, err
	}

	agg := NewAggregate(lobbyID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Error("failed to load aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load lobby: %w", err))
	}

	battleSessionID := uuid.NewString()
	agg.StartBattle(battleSessionID)

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save battle start: %w", err))
	}

	if s.db != nil {
		if _, err := s.db.Exec(ctx, `UPDATE raid_lobby SET status = 'in_battle' WHERE id = $1`, lobbyID); err != nil {
			slog.Error("failed to update raid_lobby status", "error", err)
		}
	}

	return connect.NewResponse(&pb.StartBattleResponse{
		BattleSessionId: battleSessionID,
	}), nil
}

// HandleBattleFinished is called by the battle.finished Kafka consumer.
// Transitions the lobby to finished in event store + read model.
func (s *Service) HandleBattleFinished(ctx context.Context, lobbyID, sessionID, result string) error {
	agg := NewAggregate(lobbyID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		return fmt.Errorf("load aggregate: %w", err)
	}
	if agg.Status == "finished" {
		return nil // idempotent
	}

	agg.Finish(sessionID, result)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, TopicMapper); err != nil {
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
	return nil
}

// StreamLobby streams lobby state changes to the client.
// On connect: sends existing participants as snapshots.
// Then: streams raid.user_joined, raid.battle_started, raid_lobby.finished events in real time.
func (s *Service) StreamLobby(ctx context.Context, req *connect.Request[pb.StreamLobbyRequest], stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	lobbyID := req.Msg.GetLobbyId()
	if lobbyID == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}

	if err := s.sendParticipantSnapshots(ctx, lobbyID, stream); err != nil {
		return err
	}

	if len(s.brokers) == 0 {
		return nil
	}

	return s.streamKafkaEvents(ctx, lobbyID, stream)
}

func (s *Service) sendParticipantSnapshots(ctx context.Context, lobbyID string, stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, joined_at FROM raid_participant WHERE lobby_id = $1 ORDER BY joined_at ASC`,
		lobbyID,
	)
	if err != nil {
		slog.Error("StreamLobby: failed to query participants", "error", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var participantID, userID string
		var joinedAt time.Time
		if err := rows.Scan(&participantID, &userID, &joinedAt); err != nil {
			continue
		}
		payload, _ := json.Marshal(UserJoinedData{
			LobbyID:       lobbyID,
			UserID:        userID,
			ParticipantID: participantID,
		})
		if sendErr := stream.Send(&pb.StreamLobbyResponse{
			EventType: "raid.participant_snapshot",
			LobbyId:   lobbyID,
			Payload:   string(payload),
		}); sendErr != nil {
			return sendErr
		}
	}
	return nil
}

// newTopicSub creates a subscriber and subscribes to a topic.
// Returns nil channel on failure (caller should treat as no-op).
func (s *Service) newTopicSub(ctx context.Context, group, topic string) (*platform.EventSubscriber, <-chan *message.Message) {
	sub, err := platform.NewEventSubscriber(s.brokers, group)
	if err != nil || sub == nil {
		slog.Warn("StreamLobby: failed to create subscriber", "topic", topic, "error", err)
		return nil, nil
	}
	msgs, err := sub.Subscribe(ctx, topic)
	if err != nil || msgs == nil {
		slog.Warn("StreamLobby: failed to subscribe", "topic", topic, "error", err)
		sub.Close()
		return nil, nil
	}
	return sub, msgs
}

func (s *Service) streamKafkaEvents(ctx context.Context, lobbyID string, stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	groupBase := "stream-lobby-" + uuid.NewString()

	sub1, msgs1 := s.newTopicSub(ctx, groupBase+"-joined", platform.TopicRaidUserJoined)
	if sub1 != nil {
		defer sub1.Close()
	}
	sub2, msgs2 := s.newTopicSub(ctx, groupBase+"-battle", platform.TopicRaidBattleStarted)
	if sub2 != nil {
		defer sub2.Close()
	}
	sub3, msgs3 := s.newTopicSub(ctx, groupBase+"-finished", platform.TopicRaidLobbyFinished)
	if sub3 != nil {
		defer sub3.Close()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs1:
			if !ok {
				return nil
			}
			msg.Ack()
			if err := s.handleJoinedEvent(msg, lobbyID, stream); err != nil {
				return err
			}
		case msg, ok := <-msgs2:
			if !ok {
				return nil
			}
			msg.Ack()
			matched, err := s.handleBattleStartedEvent(msg, lobbyID, stream)
			if err != nil {
				return err
			}
			if matched {
				return nil
			}
		case msg, ok := <-msgs3:
			if !ok {
				return nil
			}
			msg.Ack()
			matched, err := s.handleFinishedEvent(msg, lobbyID, stream)
			if err != nil {
				return err
			}
			if matched {
				return nil
			}
		}
	}
}

func (s *Service) handleJoinedEvent(msg *message.Message, lobbyID string, stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	event, err := platform.ParseEvent(msg)
	if err != nil {
		slog.Warn("StreamLobby: failed to parse event", "error", err)
		return nil
	}
	raw, _ := json.Marshal(event.Data)
	var data UserJoinedData
	_ = json.Unmarshal(raw, &data)
	if data.LobbyID != lobbyID {
		return nil
	}
	return stream.Send(&pb.StreamLobbyResponse{
		EventType: event.Type,
		LobbyId:   data.LobbyID,
		Payload:   string(raw),
	})
}

func (s *Service) handleBattleStartedEvent(msg *message.Message, lobbyID string, stream *connect.ServerStream[pb.StreamLobbyResponse]) (bool, error) {
	event, err := platform.ParseEvent(msg)
	if err != nil {
		slog.Warn("StreamLobby: failed to parse battle_started event", "error", err)
		return false, nil
	}
	raw, _ := json.Marshal(event.Data)
	var data BattleStartedData
	_ = json.Unmarshal(raw, &data)
	if data.LobbyID != lobbyID {
		return false, nil
	}
	return true, stream.Send(&pb.StreamLobbyResponse{
		EventType: event.Type,
		LobbyId:   data.LobbyID,
		Payload:   string(raw),
	})
}

func (s *Service) handleFinishedEvent(msg *message.Message, lobbyID string, stream *connect.ServerStream[pb.StreamLobbyResponse]) (bool, error) {
	event, err := platform.ParseEvent(msg)
	if err != nil {
		slog.Warn("StreamLobby: failed to parse finished event", "error", err)
		return false, nil
	}
	raw, _ := json.Marshal(event.Data)
	var data FinishedData
	_ = json.Unmarshal(raw, &data)
	if data.LobbyID != lobbyID {
		return false, nil
	}
	return true, stream.Send(&pb.StreamLobbyResponse{
		EventType: event.Type,
		LobbyId:   data.LobbyID,
		Payload:   string(raw),
	})
}
