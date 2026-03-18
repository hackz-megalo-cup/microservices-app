package lobby

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/lobby/v1"
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

func (s *Service) SetActivePokemon(ctx context.Context, req *connect.Request[pb.SetActivePokemonRequest]) (*connect.Response[pb.SetActivePokemonResponse], error) {
	userID := req.Msg.GetUserId()
	pokemonID := req.Msg.GetPokemonId()
	if userID == "" || pokemonID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and pokemon_id are required"))
	}

	agg := NewLobbyAggregate(uuid.NewString())
	agg.Create()
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, LobbyTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	return connect.NewResponse(&pb.SetActivePokemonResponse{Success: true}), nil
}

func (s *Service) GetActivePokemon(ctx context.Context, req *connect.Request[pb.GetActivePokemonRequest]) (*connect.Response[pb.GetActivePokemonResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	return connect.NewResponse(&pb.GetActivePokemonResponse{}), nil
}

func (s *Service) GetLobbyOverview(ctx context.Context, req *connect.Request[pb.GetLobbyOverviewRequest]) (*connect.Response[pb.GetLobbyOverviewResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	return connect.NewResponse(&pb.GetLobbyOverviewResponse{}), nil
}
