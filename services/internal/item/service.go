package item

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/item/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	eventStore *platform.EventStore
	outbox     *platform.OutboxStore
	pool       *pgxpool.Pool
}

func NewService(eventStore *platform.EventStore, outbox *platform.OutboxStore, pool *pgxpool.Pool) *Service {
	return &Service{
		eventStore: eventStore,
		outbox:     outbox,
		pool:       pool,
	}
}

// GrantItem — アイテムを付与する
func (s *Service) GrantItem(ctx context.Context, req *connect.Request[pb.GrantItemRequest]) (*connect.Response[pb.GrantItemResponse], error) {
	userID := req.Msg.GetUserId()
	itemID := req.Msg.GetItemId()
	quantity := req.Msg.GetQuantity()
	reason := req.Msg.GetReason()

	if userID == "" || itemID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and item_id are required"))
	}
	if quantity <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("quantity must be positive"))
	}

	// 集約ID = user_id:item_id
	aggID := fmt.Sprintf("%s:%s", userID, itemID)
	agg := NewItemAggregate(aggID)

	// 既存の集約があればロード（初回は空）
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Warn("load aggregate (may be new)", "error", err)
	}

	agg.Grant(userID, itemID, quantity, reason)

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, ItemTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	return connect.NewResponse(&pb.GrantItemResponse{
		Id: agg.AggregateID(),
	}), nil
}

// UseItem — アイテムを使用する
func (s *Service) UseItem(ctx context.Context, req *connect.Request[pb.UseItemRequest]) (*connect.Response[pb.UseItemResponse], error) {
	userID := req.Msg.GetUserId()
	itemID := req.Msg.GetItemId()
	quantity := req.Msg.GetQuantity()

	if userID == "" || itemID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id and item_id are required"))
	}
	// もし無を表示するならここ消して
	if quantity <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("quantity must be positive"))
	}

	aggID := fmt.Sprintf("%s:%s", userID, itemID)
	agg := NewItemAggregate(aggID)

	// 既存の集約をロード（存在しなければアイテム未所持）
	if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
		slog.Error("failed to load aggregate", "error", err)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("item not found for user"))
	}

	// ドメインバリデーション（数量不足チェック）
	if err := agg.Use(quantity); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, ItemTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	return connect.NewResponse(&pb.UseItemResponse{}), nil
}

// GetInventory — ユーザーの所持アイテム一覧を取得する
func (s *Service) GetInventory(ctx context.Context, req *connect.Request[pb.GetInventoryRequest]) (*connect.Response[pb.GetInventoryResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	// Read Model からユーザーの所持アイテムを取得
	rows, err := s.pool.Query(ctx,
		`SELECT item_id, quantity FROM user_item WHERE user_id = $1 AND quantity > 0`,
		userID,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query inventory"))
	}
	defer rows.Close()

	var items []*pb.InventoryItem
	for rows.Next() {
		var itemID string
		var qty int32
		if err := rows.Scan(&itemID, &qty); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan inventory row"))
		}
		items = append(items, &pb.InventoryItem{
			ItemId:   itemID,
			Quantity: qty,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to iterate inventory rows"))
	}

	return connect.NewResponse(&pb.GetInventoryResponse{Items: items}), nil
}
