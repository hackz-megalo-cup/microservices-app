package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgxpool"

	gatewayv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/gateway/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	pool       *pgxpool.Pool
	outbox     *platform.OutboxStore
	eventStore *platform.EventStore
}

func NewService(pool *pgxpool.Pool, outbox *platform.OutboxStore, eventStore *platform.EventStore) *Service {
	return &Service{
		pool:       pool,
		outbox:     outbox,
		eventStore: eventStore,
	}
}

func (s *Service) InvokeCustom(ctx context.Context, req *connect.Request[gatewayv1.InvokeCustomRequest]) (*connect.Response[gatewayv1.InvokeCustomResponse], error) {
	name := req.Msg.GetName()
	if name == "" {
		name = "World"
	}

	message := fmt.Sprintf("Hello, %s!", name)
	s.recordInvocation(ctx, name, message, true, "completed", platform.TopicInvocationCreated, "invocation.created")
	return connect.NewResponse(&gatewayv1.InvokeCustomResponse{Message: message}), nil
}

func (s *Service) recordInvocation(ctx context.Context, name, message string, success bool, status, topic, eventType string) {
	// Event Sourcing path (preferred).
	if s.eventStore != nil {
		s.recordViaEventStore(ctx, name, message, success)
		return
	}
	// Legacy outbox path.
	if s.outbox != nil {
		s.recordViaOutbox(ctx, name, message, success, status, topic, eventType)
		return
	}
	// Direct DB fallback.
	if s.pool != nil {
		_, dbErr := s.pool.Exec(ctx, "INSERT INTO invocations (name, result_message, success) VALUES ($1, $2, $3)", name, message, success)
		if dbErr != nil {
			slog.Error("failed to insert invocation", "error", dbErr)
		}
	}
}

func (s *Service) recordViaEventStore(ctx context.Context, name, message string, success bool) {
	agg := NewInvocationAggregate(uuid.NewString())
	if success {
		agg.Create(name, message)
	} else {
		agg.Fail(name, message)
	}
	if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, InvocationTopicMapper); err != nil {
		slog.Error("failed to save invocation aggregate", "error", err)
	}
}

func (s *Service) recordViaOutbox(ctx context.Context, name, message string, success bool, status, topic, eventType string) {
	tx, txErr := s.outbox.BeginTx(ctx)
	if txErr != nil {
		slog.Error("failed to begin transaction", "error", txErr)
		return
	}
	_, execErr := tx.Exec(ctx,
		"INSERT INTO invocations (name, result_message, success, status) VALUES ($1, $2, $3, $4)",
		name, message, success, status,
	)
	if execErr != nil {
		_ = tx.Rollback(ctx)
		slog.Error("failed to insert invocation", "error", execErr)
		return
	}
	data := map[string]any{"name": name}
	if success {
		data["message"] = message
	} else {
		data["error"] = message
	}
	event := platform.NewEvent(eventType, "gateway-service", data)
	if outboxErr := s.outbox.InsertEvent(ctx, tx, topic, event); outboxErr != nil {
		_ = tx.Rollback(ctx)
		slog.Error("failed to insert outbox event", "error", outboxErr)
		return
	}
	if commitErr := tx.Commit(ctx); commitErr != nil {
		slog.Error("failed to commit transaction", "error", commitErr)
	}
}

func MapHTTPStatusToConnectCode(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAlreadyExists
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	case http.StatusGatewayTimeout:
		return connect.CodeDeadlineExceeded
	default:
		if status >= 500 {
			return connect.CodeInternal
		}
		return connect.CodeUnknown
	}
}
