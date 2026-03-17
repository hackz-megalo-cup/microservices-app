package raidlobby

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/raid_lobby/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
}

func NewService(eventStore *platform.EventStore, outbox *platform.OutboxStore) *Service {
	return &Service{
		eventStore: eventStore,
		outbox:     outbox,
	}
}

func (s *Service) CreateRaid(ctx context.Context, req *connect.Request[pb.CreateRaidRequest]) (*connect.Response[pb.CreateRaidResponse], error) {
	if req.Msg.GetBossPokemonId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("boss_pokemon_id is required"))
	}

	agg := NewRaidLobbyAggregate(uuid.NewString())
	agg.Create()
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, RaidLobbyTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
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

func (s *Service) StreamLobby(_ context.Context, req *connect.Request[pb.StreamLobbyRequest], stream *connect.ServerStream[pb.StreamLobbyResponse]) error {
	if req.Msg.GetLobbyId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("lobby_id is required"))
	}
	return nil
}
