package greeter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/otel/trace"

	callerv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1"
	greeterv2 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/greeter/v2"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

// ServiceV2 wraps the same Service struct to implement the v2 GreeterServiceHandler.
type ServiceV2 struct {
	*Service
}

func NewServiceV2(svc *Service) *ServiceV2 {
	return &ServiceV2{Service: svc}
}

func (s *ServiceV2) Greet(ctx context.Context, req *connect.Request[greeterv2.GreetRequest]) (*connect.Response[greeterv2.GreetResponse], error) {
	name := req.Msg.GetName()
	if name == "" {
		name = "World"
	}
	locale := req.Msg.GetLocale()

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

	greeting := "Hello"
	if locale != "" {
		greeting = fmt.Sprintf("[%s] Hello", locale)
	}
	msg := fmt.Sprintf("%s %s from greeter-service!", greeting, name)

	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

	resp := connect.NewResponse(&greeterv2.GreetResponse{
		Message:            msg,
		ExternalStatus:     callerResult.GetStatusCode(),
		ExternalBodyLength: int64(callerResult.GetBodyLength()),
		TraceId:            traceID,
	})

	if s.outbox != nil {
		tx, txErr := s.outbox.BeginTx(ctx)
		if txErr != nil {
			slog.Error("failed to begin transaction", "error", txErr)
			return resp, nil
		}
		_, execErr := tx.Exec(ctx,
			"INSERT INTO greetings (name, message, external_status, status) VALUES ($1, $2, $3, $4)",
			name, msg, callerResult.GetStatusCode(), "completed",
		)
		if execErr != nil {
			_ = tx.Rollback(ctx)
			slog.Error("failed to insert greeting", "error", execErr)
			return resp, nil
		}
		event := platform.NewEvent("greeting.created", "greeter-service", map[string]any{
			"name":            name,
			"message":         msg,
			"external_status": callerResult.GetStatusCode(),
		})
		if outboxErr := s.outbox.InsertEvent(ctx, tx, platform.TopicGreetingCreated, event); outboxErr != nil {
			_ = tx.Rollback(ctx)
			slog.Error("failed to insert outbox event", "error", outboxErr)
			return resp, nil
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			slog.Error("failed to commit transaction", "error", commitErr)
		}
	} else if s.pool != nil {
		_, dbErr := s.pool.Exec(ctx, "INSERT INTO greetings (name, message, external_status) VALUES ($1, $2, $3)", name, msg, callerResult.GetStatusCode())
		if dbErr != nil {
			slog.Error("failed to insert greeting", "error", dbErr)
		}
	}

	return resp, nil
}

func (s *ServiceV2) GreetStream(ctx context.Context, req *connect.Request[greeterv2.GreetRequest], stream *connect.ServerStream[greeterv2.GreetResponse]) error {
	name := req.Msg.GetName()
	if name == "" {
		name = "World"
	}
	locale := req.Msg.GetLocale()

	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()

	greeting := "Hello"
	if locale != "" {
		greeting = fmt.Sprintf("[%s] Hello", locale)
	}

	parts := []string{
		greeting,
		name,
		"from greeter-service!",
	}

	for _, part := range parts {
		if err := stream.Send(&greeterv2.GreetResponse{
			Message: part,
			TraceId: traceID,
		}); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}
