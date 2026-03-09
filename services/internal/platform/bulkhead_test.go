package platform

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkhead_LimitsConcurrency(t *testing.T) {
	bh := NewBulkhead(2)
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bh.Execute(context.Background(), func() error {
				cur := concurrent.Add(1)
				for {
					old := maxConcurrent.Load()
					if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				concurrent.Add(-1)
				return nil
			})
		}()
	}
	wg.Wait()

	if maxConcurrent.Load() > 2 {
		t.Fatalf("max concurrent was %d, expected <= 2", maxConcurrent.Load())
	}
}

func TestBulkhead_ContextCancelled(t *testing.T) {
	bh := NewBulkhead(1)
	ctx, cancel := context.WithCancel(context.Background())

	// Fill the bulkhead
	done := make(chan struct{})
	go func() {
		_ = bh.Execute(context.Background(), func() error {
			<-done
			return nil
		})
	}()
	time.Sleep(5 * time.Millisecond)

	cancel()
	err := bh.Execute(ctx, func() error { return nil })
	close(done)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
