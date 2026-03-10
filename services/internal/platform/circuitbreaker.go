package platform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"
)

type CircuitBreakerConfig struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	TripThreshold uint32
}

func DefaultCBConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:          name,
		MaxRequests:   3,
		Interval:      10 * time.Second,
		Timeout:       30 * time.Second,
		TripThreshold: 5,
	}
}

func NewCircuitBreaker[T any](cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker[T] {
	return gobreaker.NewCircuitBreaker[T](gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cfg.TripThreshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state change",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
		IsSuccessful: func(err error) bool {
			return err == nil || errors.Is(err, context.Canceled)
		},
	})
}

func CBExecute[T any](cb *gobreaker.CircuitBreaker[T], fn func() (T, error)) (T, error) {
	result, err := cb.Execute(fn)
	if err != nil {
		var zero T
		if errors.Is(err, gobreaker.ErrOpenState) {
			return zero, fmt.Errorf("service unavailable (circuit open): %w", err)
		}
		if errors.Is(err, gobreaker.ErrTooManyRequests) {
			return zero, fmt.Errorf("too many requests (circuit half-open): %w", err)
		}
		return zero, err
	}
	return result, nil
}
