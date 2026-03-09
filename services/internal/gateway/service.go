package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sony/gobreaker/v2"

	gatewayv1 "github.com/hackz-megalo-cup/microservices-app/services/gen/go/gateway/v1"
	"github.com/hackz-megalo-cup/microservices-app/services/internal/platform"
)

type Service struct {
	httpClient         *http.Client
	baseURL            string
	timeout            time.Duration
	breaker            *gobreaker.CircuitBreaker[invokeResult]
	retryBudget        *RetryBudget
	customLangBulkhead *platform.Bulkhead
	pool               *pgxpool.Pool
	publisher          *platform.EventPublisher
}

type invokeResult struct {
	message string
}

type statusError struct {
	status int
	body   string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("downstream status %d: %s", e.status, e.body)
}

type RetryBudget struct {
	tokens chan struct{}
}

func NewRetryBudget(capacity, refillPerSecond int) *RetryBudget {
	b := &RetryBudget{tokens: make(chan struct{}, capacity)}
	for i := 0; i < capacity; i++ {
		b.tokens <- struct{}{}
	}
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for i := 0; i < refillPerSecond; i++ {
				select {
				case b.tokens <- struct{}{}:
				default:
				}
			}
		}
	}()
	return b
}

func (b *RetryBudget) Allow() bool {
	select {
	case <-b.tokens:
		return true
	default:
		return false
	}
}

func NewService(httpClient *http.Client, baseURL string, timeout time.Duration, pool *pgxpool.Pool, publisher *platform.EventPublisher) *Service {
	return &Service{
		httpClient:         httpClient,
		baseURL:            strings.TrimRight(baseURL, "/"),
		timeout:            timeout,
		breaker:            platform.NewCircuitBreaker[invokeResult](platform.DefaultCBConfig("custom-lang-service")),
		retryBudget:        NewRetryBudget(20, 10),
		customLangBulkhead: platform.NewBulkhead(10),
		pool:               pool,
		publisher:          publisher,
	}
}

func (s *Service) InvokeCustom(ctx context.Context, req *connect.Request[gatewayv1.InvokeCustomRequest]) (*connect.Response[gatewayv1.InvokeCustomResponse], error) {
	name := req.Msg.GetName()
	if name == "" {
		name = "World"
	}

	var result invokeResult
	err := s.customLangBulkhead.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = platform.CBExecute(s.breaker, func() (invokeResult, error) {
			return s.callCustom(ctx, name)
		})
		if cbErr != nil && shouldRetry(cbErr) && s.retryBudget.Allow() {
			result, cbErr = platform.CBExecute(s.breaker, func() (invokeResult, error) {
				return s.callCustom(ctx, name)
			})
		}
		return cbErr
	})
	if err != nil {
		// 同期パターン: 失敗もDB記録
		if s.pool != nil {
			_, dbErr := s.pool.Exec(ctx, "INSERT INTO invocations (name, result_message, success) VALUES ($1, $2, $3)", name, err.Error(), false)
			if dbErr != nil {
				slog.Error("failed to insert invocation", "error", dbErr)
			}
		}
		return nil, mapError(err)
	}

	// 同期パターン: レスポンス前にDB書き込み完了を保証
	if s.pool != nil {
		_, dbErr := s.pool.Exec(ctx, "INSERT INTO invocations (name, result_message, success) VALUES ($1, $2, $3)", name, result.message, true)
		if dbErr != nil {
			slog.Error("failed to insert invocation", "error", dbErr)
		}
	}

	// Fire-and-forget: エラーはログに記録するがメイン処理は失敗させない
	if err := s.publisher.Publish(ctx, platform.TopicInvocationCreated, platform.NewEvent(
		"invocation.created",
		"gateway-service",
		map[string]any{
			"name":    name,
			"message": result.message,
		},
	)); err != nil {
		slog.Error("failed to publish invocation.created event", "error", err)
	}

	return connect.NewResponse(&gatewayv1.InvokeCustomResponse{Message: result.message}), nil
}

func (s *Service) callCustom(ctx context.Context, name string) (invokeResult, error) {
	payload, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return invokeResult{}, err
	}

	rpcCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(rpcCtx, http.MethodPost, s.baseURL+"/invoke", bytes.NewReader(payload))
	if err != nil {
		return invokeResult{}, err
	}
	httpReq.Header.Set("content-type", "application/json")

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return invokeResult{}, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return invokeResult{}, err
	}

	if httpResp.StatusCode >= 400 {
		return invokeResult{}, &statusError{status: httpResp.StatusCode, body: string(body)}
	}

	var resp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return invokeResult{}, err
	}
	if resp.Message == "" {
		resp.Message = "custom-lang-service returned an empty message"
	}
	return invokeResult{message: resp.Message}, nil
}

func shouldRetry(err error) bool {
	var se *statusError
	if errors.As(err, &se) {
		switch se.status {
		case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func mapError(err error) error {
	if errors.Is(err, gobreaker.ErrOpenState) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	}
	var se *statusError
	if errors.As(err, &se) {
		return connect.NewError(MapHTTPStatusToConnectCode(se.status), err)
	}
	return connect.NewError(connect.CodeInternal, err)
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
