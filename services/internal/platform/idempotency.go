package platform

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func ExtractIdempotencyKey(headers map[string]string) string {
	return headers["Idempotency-Key"]
}

type IdempotencyStore struct {
	pool *pgxpool.Pool
}

func NewIdempotencyStore(pool *pgxpool.Pool) *IdempotencyStore {
	if pool == nil {
		return nil
	}
	return &IdempotencyStore{pool: pool}
}

// Check returns cached response if key exists and not expired. Returns nil if not found.
func (s *IdempotencyStore) Check(ctx context.Context, key string) ([]byte, int, bool, error) {
	if s == nil || key == "" {
		return nil, 0, false, nil
	}
	var response []byte
	var statusCode int
	err := s.pool.QueryRow(ctx,
		"SELECT response, status_code FROM idempotency_keys WHERE key = $1 AND expires_at > NOW()",
		key,
	).Scan(&response, &statusCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	return response, statusCode, true, nil
}

// Store saves response for idempotency key with 24h TTL.
func (s *IdempotencyStore) Store(ctx context.Context, key string, response []byte, statusCode int) error {
	if s == nil || key == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO idempotency_keys (key, response, status_code, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (key) DO UPDATE SET response = $2, status_code = $3`,
		key, response, statusCode, time.Now().Add(24*time.Hour),
	)
	return err
}

// Cleanup removes expired keys. Call periodically.
func (s *IdempotencyStore) Cleanup(ctx context.Context) error {
	if s == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx, "DELETE FROM idempotency_keys WHERE expires_at < NOW()")
	return err
}

// responseTypeRegistry caches the reflect.Type of connect.Response[T] for each
// RPC procedure. This is needed because connect.NewResponse is generic and we
// cannot instantiate it with a runtime type directly. The registry is populated
// on the first successful (non-cached) call for each procedure and reused for
// subsequent cache hits.
var responseTypeRegistry sync.Map // map[string]reflect.Type (procedure -> Response[T] reflect.Type)

// newCachedResponse reconstructs a connect.AnyResponse from a deserialized
// proto.Message using the cached reflect.Type for the given procedure.
// Returns nil if the response type has not been seen yet.
func newCachedResponse(procedure string, msg proto.Message) connect.AnyResponse {
	respTypeVal, ok := responseTypeRegistry.Load(procedure)
	if !ok {
		return nil
	}
	respType := respTypeVal.(reflect.Type)
	resp := reflect.New(respType)
	resp.Elem().Field(0).Set(reflect.ValueOf(msg))
	return resp.Interface().(connect.AnyResponse)
}

// recordResponseType stores the reflect.Type of the connect.Response for the
// given procedure so it can be used to reconstruct cached responses later.
func recordResponseType(procedure string, resp connect.AnyResponse) {
	if resp == nil {
		return
	}
	respType := reflect.TypeOf(resp).Elem()
	responseTypeRegistry.Store(procedure, respType)
}

// NewIdempotencyInterceptor returns a connect-go interceptor that checks for
// duplicate requests using the Idempotency-Key header.
// If store is nil, the interceptor is a no-op pass-through.
// StartIdempotencyCleanup runs a background goroutine that periodically cleans up expired keys.
func StartIdempotencyCleanup(ctx context.Context, store *IdempotencyStore) {
	if store == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := store.Cleanup(ctx); err != nil {
					slog.Error("idempotency cleanup failed", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func NewIdempotencyInterceptor(store *IdempotencyStore) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if store == nil {
				return next(ctx, req)
			}
			key := req.Header().Get("Idempotency-Key")
			if key == "" {
				return next(ctx, req)
			}

			procedure := req.Spec().Procedure

			if cachedResp := tryResolveCached(ctx, store, key, procedure); cachedResp != nil {
				return cachedResp, nil
			}

			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			}

			recordResponseType(procedure, resp)
			storeIdempotencyResponse(ctx, store, key, resp)

			return resp, nil
		}
	})
}

func tryResolveCached(ctx context.Context, store *IdempotencyStore, key, procedure string) connect.AnyResponse {
	cached, _, found, err := store.Check(ctx, key)
	if err != nil {
		slog.Warn("idempotency check failed, proceeding without cache", "key", key, "error", err)
		return nil
	}
	if !found {
		return nil
	}
	var wrapper anypb.Any
	if unmarshalErr := proto.Unmarshal(cached, &wrapper); unmarshalErr != nil {
		slog.Warn("failed to unmarshal cached idempotency response", "key", key, "error", unmarshalErr)
		return nil
	}
	msg, unmarshalNewErr := wrapper.UnmarshalNew()
	if unmarshalNewErr != nil {
		slog.Warn("failed to unmarshal cached proto message", "key", key, "error", unmarshalNewErr)
		return nil
	}
	cachedResp := newCachedResponse(procedure, msg)
	if cachedResp != nil {
		slog.Debug("returning cached idempotency response", "key", key)
	}
	return cachedResp
}

func storeIdempotencyResponse(ctx context.Context, store *IdempotencyStore, key string, resp connect.AnyResponse) {
	msg, ok := resp.Any().(proto.Message)
	if !ok {
		slog.Warn("idempotency: response is not a proto.Message, skipping cache store", "key", key)
		return
	}
	wrapper, wrapErr := anypb.New(msg)
	if wrapErr != nil {
		slog.Warn("idempotency: failed to wrap response in Any", "key", key, "error", wrapErr)
		return
	}
	data, marshalErr := proto.Marshal(wrapper)
	if marshalErr != nil {
		slog.Warn("idempotency: failed to marshal response", "key", key, "error", marshalErr)
		return
	}
	if storeErr := store.Store(ctx, key, data, 0); storeErr != nil {
		slog.Warn("idempotency: failed to store response", "key", key, "error", storeErr)
	}
}
