package raidlobby

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
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

	// マスターデータからボスポケモン情報を取得して存在確認
	if s.masterdataClient != nil {
		_, err := s.masterdataClient.GetPokemon(ctx, connect.NewRequest(&masterdatav1.GetPokemonRequest{
			Id: bossPokemonID,
		}))
		if err != nil {
			slog.Error("failed to get pokemon from masterdata", "pokemon_id", bossPokemonID, "error", err)
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("pokemon not found: %s", bossPokemonID))
		}
	}

	lobbyID := uuid.NewString()

	// イベントソーシング: ロビー作成イベントを保存 + Kafka 発行
	agg := NewRaidLobbyAggregate(lobbyID)
	agg.Create(bossPokemonID)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, RaidLobbyTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	// 読み取りモデル: raid_lobby テーブルに直接挿入
	if s.db != nil {
		_, err := s.db.Exec(ctx,
			`INSERT INTO raid_lobby (id, boss_pokemon_id, status, created_at) VALUES ($1, $2, 'waiting', $3)`,
			lobbyID, bossPokemonID, time.Now().UTC(),
		)
		if err != nil {
			slog.Error("failed to insert raid_lobby", "error", err)
			// イベントは既に保存済みのため、ここではエラーをログのみ
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

	// ロビー存在確認
	if s.db != nil {
		var status string
		err := s.db.QueryRow(ctx, `SELECT status FROM raid_lobby WHERE id = $1`, lobbyID).Scan(&status)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("lobby not found: %s", lobbyID))
		}
		if status != "waiting" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("lobby is not accepting participants (status: %s)", status))
		}
	}

	participantID := uuid.NewString()

	// 既存の集約を読み込んで参加イベントを発行
	agg := NewRaidLobbyAggregate(lobbyID)
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Error("failed to load aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load lobby"))
	}
	agg.Join(userID, participantID)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, RaidLobbyTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to join lobby"))
	}

	// 読み取りモデル: raid_participant テーブルに挿入
	if s.db != nil {
		_, err := s.db.Exec(ctx,
			`INSERT INTO raid_participant (id, lobby_id, user_id, joined_at) VALUES ($1, $2, $3, $4)`,
			participantID, lobbyID, userID, time.Now().UTC(),
		)
		if err != nil {
			slog.Error("failed to insert raid_participant", "error", err)
		}
	}

	return connect.NewResponse(&pb.JoinRaidResponse{
		ParticipantId: participantID,
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

// StreamLobby streams lobby participant changes.
// On connect: sends existing participants as "raid.participant_snapshot" events.
// Then: streams new raid.user_joined events in real time.
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
		payload, _ := json.Marshal(RaidUserJoinedData{
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

func (s *Service) streamKafkaEvents(ctx context.Context, lobbyID string, stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	consumerGroup := "stream-lobby-" + uuid.NewString()
	subscriber, err := platform.NewEventSubscriber(s.brokers, consumerGroup)
	if err != nil || subscriber == nil {
		slog.Warn("failed to create event subscriber for StreamLobby", "error", err)
		return nil
	}
	defer subscriber.Close()

	msgs, err := subscriber.Subscribe(ctx, platform.TopicRaidUserJoined)
	if err != nil || msgs == nil {
		slog.Warn("failed to subscribe to raid.user_joined", "error", err)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			msg.Ack()
			if err := s.handleJoinedEvent(msg, lobbyID, stream); err != nil {
				return err
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
	var data RaidUserJoinedData
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
