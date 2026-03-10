package greeter

import (
	"context"
	"errors"
	"fmt"
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

	s.persistGreeting(ctx, name, msg, callerResult.GetStatusCode())

	return resp, nil
}

func (s *ServiceV2) GreetStream(ctx context.Context, req *connect.Request[greeterv2.GreetStreamRequest], stream *connect.ServerStream[greeterv2.GreetStreamResponse]) error {
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
		if err := stream.Send(&greeterv2.GreetStreamResponse{
			Message: part,
			TraceId: traceID,
		}); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil
}
