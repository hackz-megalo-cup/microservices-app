package caller

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgxpool"

	callerv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/caller/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	httpClient *http.Client
	timeout    time.Duration
	pool       *pgxpool.Pool
	publisher  *platform.EventPublisher
}

func NewService(httpClient *http.Client, timeout time.Duration, pool *pgxpool.Pool, publisher *platform.EventPublisher) *Service {
	return &Service{httpClient: httpClient, timeout: timeout, pool: pool, publisher: publisher}
}

func (s *Service) CallExternal(ctx context.Context, req *connect.Request[callerv1.CallExternalRequest]) (*connect.Response[callerv1.CallExternalResponse], error) {
	targetURL := req.Msg.GetUrl()
	if _, err := url.ParseRequestURI(targetURL); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid URL: %w", err))
	}

	rpcCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(rpcCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build request: %w", err))
	}

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		if rpcCtx.Err() != nil {
			return nil, connect.NewError(connect.CodeDeadlineExceeded, rpcCtx.Err())
		}
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("call failed: %w", err))
	}
	defer httpResp.Body.Close()

	n, err := io.Copy(io.Discard, httpResp.Body)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read response: %w", err))
	}

	statusCode := int32(httpResp.StatusCode)
	bodyLength := int32(n)

	// 非同期パターン: レイテンシ影響なし、ログ欠損の可能性あり
	if s.pool != nil {
		capturedURL := targetURL
		go func() {
			_, err := s.pool.Exec(context.Background(), "INSERT INTO call_logs (url, status_code, body_length) VALUES ($1, $2, $3)", capturedURL, statusCode, bodyLength)
			if err != nil {
				slog.Error("failed to insert call log", "error", err)
			}
		}()
	}

	// Fire-and-forget: エラーはログに記録するがメイン処理は失敗させない
	if err := s.publisher.Publish(ctx, platform.TopicCallCompleted, platform.NewEvent(
		"call.completed",
		"caller-service",
		map[string]any{
			"url":         targetURL,
			"status_code": statusCode,
			"body_length": bodyLength,
		},
	)); err != nil {
		slog.Error("failed to publish call.completed event", "error", err)
	}

	resp := connect.NewResponse(&callerv1.CallExternalResponse{
		StatusCode: statusCode,
		BodyLength: bodyLength,
	})
	return resp, nil
}
