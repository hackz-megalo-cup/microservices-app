package platform

import (
	"context"
	"errors"
	"testing"

	"github.com/sony/gobreaker/v2"
)

func TestDefaultCBConfig(t *testing.T) {
	cfg := DefaultCBConfig("test-service")
	if cfg.Name != "test-service" {
		t.Errorf("got name %q, want %q", cfg.Name, "test-service")
	}
	if cfg.MaxRequests != 3 {
		t.Errorf("got MaxRequests %d, want 3", cfg.MaxRequests)
	}
	if cfg.TripThreshold != 5 {
		t.Errorf("got TripThreshold %d, want 5", cfg.TripThreshold)
	}
}

func TestNewCircuitBreaker_Success(t *testing.T) {
	cb := NewCircuitBreaker[string](DefaultCBConfig("test"))

	result, err := CBExecute(cb, func() (string, error) {
		return "hello", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("got %q, want %q", result, "hello")
	}
}

func TestNewCircuitBreaker_PropagatesError(t *testing.T) {
	cb := NewCircuitBreaker[string](DefaultCBConfig("test"))

	_, err := CBExecute(cb, func() (string, error) {
		return "", errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "boom" {
		t.Errorf("got %q, want %q", err.Error(), "boom")
	}
}

func TestNewCircuitBreaker_TripsAfterThreshold(t *testing.T) {
	cfg := DefaultCBConfig("test")
	cfg.TripThreshold = 3
	cb := NewCircuitBreaker[string](cfg)

	simulatedErr := errors.New("fail")
	for i := 0; i < 3; i++ {
		_, _ = CBExecute(cb, func() (string, error) {
			return "", simulatedErr
		})
	}

	// Next call should hit the open circuit
	_, err := CBExecute(cb, func() (string, error) {
		return "should not reach", nil
	})
	if err == nil {
		t.Fatal("expected circuit open error, got nil")
	}
	if !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("expected ErrOpenState, got %v", err)
	}
}

func TestNewCircuitBreaker_CanceledIsSuccessful(t *testing.T) {
	cb := NewCircuitBreaker[string](DefaultCBConfig("test"))

	// context.Canceled should be treated as successful (not trip the breaker)
	for i := 0; i < 10; i++ {
		_, _ = CBExecute(cb, func() (string, error) {
			return "", context.Canceled
		})
	}

	// Circuit should still be closed
	result, err := CBExecute(cb, func() (string, error) {
		return "still open", nil
	})
	if err != nil {
		t.Fatalf("circuit should still be closed, got error: %v", err)
	}
	if result != "still open" {
		t.Errorf("got %q, want %q", result, "still open")
	}
}
