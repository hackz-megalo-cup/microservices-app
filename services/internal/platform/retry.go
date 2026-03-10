package platform

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v5"
)

type RetryConfig struct {
	MaxRetries   uint
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

type RetryOption func(*RetryConfig)

func WithMaxRetries(n uint) RetryOption {
	return func(c *RetryConfig) { c.MaxRetries = n }
}

func WithInitialDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.InitialDelay = d }
}

func WithMaxDelay(d time.Duration) RetryOption {
	return func(c *RetryConfig) { c.MaxDelay = d }
}

func NewPermanentError(err error) error {
	return backoff.Permanent(err)
}

// RetryWithBackoff executes op with exponential backoff + jitter using cenkalti/backoff v5.
func RetryWithBackoff(ctx context.Context, op func() error, opts ...RetryOption) error {
	cfg := &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	}
	for _, o := range opts {
		o(cfg)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = cfg.InitialDelay
	b.MaxInterval = cfg.MaxDelay
	b.RandomizationFactor = 0.5 // Full jitter

	// v5 API: context is first arg, WithMaxTries replaces WithMaxRetries,
	// Operation is generic: func() (T, error)
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		return struct{}{}, op()
	},
		backoff.WithBackOff(b),
		backoff.WithMaxTries(cfg.MaxRetries),
	)
	return err
}

// RetryWithBackoffV returns a value from the retried operation (uses v5 generic API directly).
func RetryWithBackoffV[T any](ctx context.Context, op func() (T, error), opts ...RetryOption) (T, error) {
	cfg := &RetryConfig{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
	}
	for _, o := range opts {
		o(cfg)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = cfg.InitialDelay
	b.MaxInterval = cfg.MaxDelay
	b.RandomizationFactor = 0.5

	return backoff.Retry(ctx, op,
		backoff.WithBackOff(b),
		backoff.WithMaxTries(cfg.MaxRetries),
	)
}
