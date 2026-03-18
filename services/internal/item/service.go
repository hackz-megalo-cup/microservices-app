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
	dbPool     *pgxpool.Pool
}

func NewService(eventStore *platform.EventStore, outbox *platform.OutboxStore, dbPool *pgxpool.Pool) *Service {
	return &Service{
		eventStore: eventStore,
		outbox:     outbox,
		dbPool:     dbPool,
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

// GetUserItems — ユーザーが所持しているアイテムの一覧を取得する
func (s *Service) GetUserItems(ctx context.Context, req *connect.Request[pb.GetUserItemsRequest]) (*connect.Response[pb.GetUserItemsResponse], error) {
	userID := req.Msg.GetUserId()
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	// EventStore から user_id にマッチする全ストリームを取得
	// stream_id パターン: "user_id:item_id"
	rows, err := s.dbPool.Query(ctx, `
		SELECT DISTINCT stream_id
		FROM event_store
		WHERE stream_type = 'item'
		  AND stream_id LIKE $1 || ':%'
		ORDER BY stream_id
	`, userID)
	if err != nil {
		slog.Error("failed to query user items", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query items"))
	}
	defer rows.Close()

	var items []*pb.UserItem
	for rows.Next() {
		var streamID string
		if err := rows.Scan(&streamID); err != nil {
			continue
		}

		// 各集約をロードして現在の状態を取得
		agg := NewItemAggregate(streamID)
		if err := platform.LoadAggregate(ctx, s.eventStore, agg); err != nil {
			slog.Warn("failed to load aggregate", "stream_id", streamID, "error", err)
			continue
		}

		// quantity が 0 より大きいもののみ返す
		if agg.Quantity > 0 {
			items = append(items, &pb.UserItem{
				ItemId:   agg.ItemID,
				Quantity: agg.Quantity,
				Status:   agg.Status,
			})
		}
	}

	return connect.NewResponse(&pb.GetUserItemsResponse{
		Items: items,
	}), nil
}
