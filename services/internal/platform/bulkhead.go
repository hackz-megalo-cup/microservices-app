package platform

import (
	"context"
	"fmt"

	"golang.org/x/sync/semaphore"
)

type Bulkhead struct {
	sem     *semaphore.Weighted
	maxSize int64
}

func NewBulkhead(maxConcurrent int64) *Bulkhead {
	return &Bulkhead{
		sem:     semaphore.NewWeighted(maxConcurrent),
		maxSize: maxConcurrent,
	}
}

func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	if err := b.sem.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("bulkhead acquire: %w", err)
	}
	defer b.sem.Release(1)
	return fn()
}
