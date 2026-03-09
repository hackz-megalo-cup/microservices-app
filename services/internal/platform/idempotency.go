package platform

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
