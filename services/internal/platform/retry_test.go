package platform

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryWithBackoff_Success(t *testing.T) {
	attempts := 0
	err := RetryWithBackoff(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	}, WithMaxRetries(5))
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithBackoff_PermanentError(t *testing.T) {
	attempts := 0
	err := RetryWithBackoff(context.Background(), func() error {
		attempts++
		return NewPermanentError(errors.New("fatal"))
	}, WithMaxRetries(5))
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt for permanent error, got %d", attempts)
	}
}

func TestRetryWithBackoff_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := RetryWithBackoff(ctx, func() error {
		return errors.New("always fail")
	}, WithMaxRetries(100))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
