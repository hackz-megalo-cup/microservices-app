package order

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pb "github.com/hackz-megalo-cup/microservices-app/services/gen/go/order/v1"
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
// Invoke — implement your business logic here.
// The infrastructure (ES, Outbox, Kafka) is handled for you.
// ==========================================================.

func (s *Service) Invoke(ctx context.Context, req *connect.Request[pb.OrderRequest]) (*connect.Response[pb.OrderResponse], error) {
	input := req.Msg.GetInput()
	if input == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("input is required"))
	}

	// TODO: Implement your business logic here.
	output := fmt.Sprintf("Hello %s from order!", input)

	// Save via Event Sourcing (EventStore + Outbox + Kafka — all automatic)
	agg := NewOrderAggregate(uuid.NewString())
	agg.Create(input, output)
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, OrderTopicMapper); err != nil {
		slog.Error("failed to save aggregate", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to save"))
	}

	return connect.NewResponse(&pb.OrderResponse{
		Output: output,
	}), nil
}
