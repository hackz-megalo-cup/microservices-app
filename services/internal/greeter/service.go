package greeter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker/v2"

	callerv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1/callerv1connect"
	greeterv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	callerClient   callerv1connect.CallerServiceClient
	callerCB       *gobreaker.CircuitBreaker[*callerv1.CallExternalResponse]
	callerBulkhead *platform.Bulkhead
	externalURL    string
	timeout        time.Duration
	pool           *pgxpool.Pool
	outbox         *platform.OutboxStore
	eventStore     *platform.EventStore
}

func NewService(callerClient callerv1connect.CallerServiceClient, externalURL string, timeout time.Duration, pool *pgxpool.Pool, outbox *platform.OutboxStore, eventStore *platform.EventStore) *Service {
	return &Service{
		callerClient:   callerClient,
		callerCB:       platform.NewCircuitBreaker[*callerv1.CallExternalResponse](platform.DefaultCBConfig("greeter-to-caller")),
		callerBulkhead: platform.NewBulkhead(10),
		externalURL:    externalURL,
		timeout:        timeout,
		pool:           pool,
		outbox:         outbox,
		eventStore:     eventStore,
	}
}

func (s *Service) Greet(ctx context.Context, req *connect.Request[greeterv1.GreetRequest]) (*connect.Response[greeterv1.GreetResponse], error) {
	name := req.Msg.GetName()
	if name == "" {
		name = "World"
	}

	rpcCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var callerResult *callerv1.CallExternalResponse
	err := s.callerBulkhead.Execute(rpcCtx, func() error {
		return platform.RetryWithBackoff(rpcCtx, func() error {
			result, cbErr := platform.CBExecute(s.callerCB, func() (*callerv1.CallExternalResponse, error) {
				resp, err := s.callerClient.CallExternal(rpcCtx, connect.NewRequest(&callerv1.CallExternalRequest{Url: s.externalURL}))
				if err != nil {
					return nil, err
				}
				return resp.Msg, nil
			})
			if cbErr != nil {
				return cbErr
			}
			callerResult = result
			return nil
		}, platform.WithMaxRetries(3))
	})
	if err != nil {
		// Saga: publish greeting.failed via outbox
		s.publishFailedEvent(ctx, name, err)

		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, err
		}
		if errors.Is(err, gobreaker.ErrOpenState) {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("caller service circuit open: %w", err))
		}
		if rpcCtx.Err() != nil {
			return nil, connect.NewError(connect.CodeDeadlineExceeded, rpcCtx.Err())
		}
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("caller call failed: %w", err))
	}

	msg := fmt.Sprintf("Hello %s from greeter-service!", name)
	resp := connect.NewResponse(&greeterv1.GreetResponse{
		Message:            msg,
		ExternalStatus:     callerResult.GetStatusCode(),
		ExternalBodyLength: callerResult.GetBodyLength(),
	})

	s.persistGreeting(ctx, name, msg, callerResult.GetStatusCode())

	return resp, nil
}

func (s *Service) persistGreeting(ctx context.Context, name, msg string, statusCode int32) {
	switch {
	case s.eventStore != nil:
		agg := NewGreetingAggregate(uuid.NewString())
		agg.Create(name, msg, statusCode)
		if saveErr := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, GreetingTopicMapper); saveErr != nil {
			slog.Error("failed to save greeting aggregate", "error", saveErr)
		}
	case s.outbox != nil:
		tx, txErr := s.outbox.BeginTx(ctx)
		if txErr != nil {
			slog.Error("failed to begin transaction", "error", txErr)
			return
		}
		_, execErr := tx.Exec(ctx,
			"INSERT INTO greetings (name, message, external_status, status) VALUES ($1, $2, $3, $4)",
			name, msg, statusCode, "completed",
		)
		if execErr != nil {
			_ = tx.Rollback(ctx)
			slog.Error("failed to insert greeting", "error", execErr)
			return
		}
		event := platform.NewEvent("greeting.created", "greeter-service", map[string]any{
			"name":            name,
			"message":         msg,
			"external_status": statusCode,
		})
		if outboxErr := s.outbox.InsertEvent(ctx, tx, platform.TopicGreetingCreated, event); outboxErr != nil {
			_ = tx.Rollback(ctx)
			slog.Error("failed to insert outbox event", "error", outboxErr)
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			slog.Error("failed to commit transaction", "error", commitErr)
		}
	case s.pool != nil:
		_, dbErr := s.pool.Exec(ctx, "INSERT INTO greetings (name, message, external_status) VALUES ($1, $2, $3)", name, msg, statusCode)
		if dbErr != nil {
			slog.Error("failed to insert greeting", "error", dbErr)
		}
	}
}

func (s *Service) publishFailedEvent(ctx context.Context, name string, originalErr error) {
	// Event Sourcing path (preferred)
	if s.eventStore != nil {
		agg := NewGreetingAggregate(uuid.NewString())
		agg.Fail(name, originalErr.Error())
		if err := platform.SaveAggregate(ctx, s.eventStore, s.outbox, agg, GreetingTopicMapper); err != nil {
			slog.Error("failed to save failed greeting aggregate", "error", err)
		}
		return
	}
	// Legacy outbox path
	if s.outbox == nil {
		return
	}
	tx, txErr := s.outbox.BeginTx(ctx)
	if txErr != nil {
		slog.Error("failed to begin tx for greeting.failed", "error", txErr)
		return
	}
	event := platform.NewEvent("greeting.failed", "greeter-service", map[string]any{
		"name":  name,
		"error": originalErr.Error(),
	})
	if err := s.outbox.InsertEvent(ctx, tx, platform.TopicGreetingFailed, event); err != nil {
		_ = tx.Rollback(ctx)
		slog.Error("failed to insert greeting.failed outbox event", "error", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Error("failed to commit greeting.failed event", "error", err)
	}
}
