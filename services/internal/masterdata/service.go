package masterdata

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/masterdata/v1"
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

// ==========================================================.
// 各 RPC を実装する際は、以下のパターンを参考にしてください。
//
// 新規作成パターン:
//   agg := NewMasterdataAggregate(uuid.NewString())
//   agg.Create(...)
//   platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, MasterdataTopicMapper)
//   return agg.AggregateID()  // 生成された ID
//
// 既存更新パターン:
//   agg := NewMasterdataAggregate(id)
//   platform.LoadAggregate(ctx, s.eventStore, agg)  // イベント再生で状態復元
//   agg.SomeCommand()
//   platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, MasterdataTopicMapper)
// ==========================================================.

func (s *Service) CreatePokemon(_ context.Context, _ *connect.Request[pb.CreatePokemonRequest]) (*connect.Response[pb.CreatePokemonResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) GetPokemon(_ context.Context, _ *connect.Request[pb.GetPokemonRequest]) (*connect.Response[pb.GetPokemonResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) ListPokemon(_ context.Context, _ *connect.Request[pb.ListPokemonRequest]) (*connect.Response[pb.ListPokemonResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) CreateTypeMatchup(_ context.Context, _ *connect.Request[pb.CreateTypeMatchupRequest]) (*connect.Response[pb.CreateTypeMatchupResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) ListTypeMatchups(_ context.Context, _ *connect.Request[pb.ListTypeMatchupsRequest]) (*connect.Response[pb.ListTypeMatchupsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) CreateItem(_ context.Context, _ *connect.Request[pb.CreateItemRequest]) (*connect.Response[pb.CreateItemResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) GetItem(_ context.Context, _ *connect.Request[pb.GetItemRequest]) (*connect.Response[pb.GetItemResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

func (s *Service) ListItems(_ context.Context, _ *connect.Request[pb.ListItemsRequest]) (*connect.Response[pb.ListItemsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}
