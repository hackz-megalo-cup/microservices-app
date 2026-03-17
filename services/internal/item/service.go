package item

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1"
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
// Proto を編集して RPC を定義したら、このメソッドを書き換える。
//
// 新規作成パターン:
//   agg := NewItemAggregate(uuid.NewString())
//   agg.Create(...)
//   platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, ItemTopicMapper)
//   return agg.AggregateID()  // 生成された ID
//
// 既存更新パターン:
//   agg := NewItemAggregate(id)
//   platform.LoadAggregate(ctx, s.eventStore, agg)  // イベント再生で状態復元
//   agg.SomeCommand()
//   platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, ItemTopicMapper)
// ==========================================================.

func (s *Service) GrantItem(ctx context.Context, req *connect.Request[pb.GrantItemRequest]) (*connect.Response[pb.GrantItemResponse], error) {
	userID := req.Msg.GetUserId()
	itemID := req.Msg.GetItemId()
	quantity := req.Msg.GetQuantity()

	if userID == "" || itemID == "" || quantity <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("userId, itemId and quantity are required"))
	}

	agg := NewItemAggregate(uuid.NewString())
	agg.Create(userID, itemID, quantity, req.Msg.GetReason())
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, ItemTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	return connect.NewResponse(&pb.GrantItemResponse{
		Id: agg.AggregateID(),
	}), nil
}

func (s *Service) UseItem(ctx context.Context, req *connect.Request[pb.UseItemRequest]) (*connect.Response[pb.UseItemResponse], error) {
	// TODO: Implement UseItem logic
	return connect.NewResponse(&pb.UseItemResponse{}), nil
}
