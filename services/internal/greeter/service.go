package greeter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgxpool"

	callerv1 "github.com/thirdlf03/microservice-app/services/gen/go/caller/v1"
	"github.com/thirdlf03/microservice-app/services/gen/go/caller/v1/callerv1connect"
	greeterv1 "github.com/thirdlf03/microservice-app/services/gen/go/greeter/v1"
)

type Service struct {
	callerClient callerv1connect.CallerServiceClient
	externalURL  string
	timeout      time.Duration
	pool         *pgxpool.Pool
}

func NewService(callerClient callerv1connect.CallerServiceClient, externalURL string, timeout time.Duration, pool *pgxpool.Pool) *Service {
	return &Service{
		callerClient: callerClient,
		externalURL:  externalURL,
		timeout:      timeout,
		pool:         pool,
	}
}

func (s *Service) Greet(ctx context.Context, req *connect.Request[greeterv1.GreetRequest]) (*connect.Response[greeterv1.GreetResponse], error) {
	name := req.Msg.GetName()
	if name == "" {
		name = "World"
	}

	rpcCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	callerResp, err := s.callerClient.CallExternal(rpcCtx, connect.NewRequest(&callerv1.CallExternalRequest{Url: s.externalURL}))
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, err
		}
		if rpcCtx.Err() != nil {
			return nil, connect.NewError(connect.CodeDeadlineExceeded, rpcCtx.Err())
		}
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("caller call failed: %w", err))
	}

	msg := fmt.Sprintf("Hello %s from greeter-service!", name)
	resp := connect.NewResponse(&greeterv1.GreetResponse{
		Message:            msg,
		ExternalStatus:     callerResp.Msg.GetStatusCode(),
		ExternalBodyLength: callerResp.Msg.GetBodyLength(),
	})

	// 同期パターン: レスポンス前にDB書き込み完了を保証
	if s.pool != nil {
		_, dbErr := s.pool.Exec(ctx, "INSERT INTO greetings (name, message, external_status) VALUES ($1, $2, $3)", name, msg, callerResp.Msg.GetStatusCode())
		if dbErr != nil {
			slog.Error("failed to insert greeting", "error", dbErr)
		}
	}

	return resp, nil
}
