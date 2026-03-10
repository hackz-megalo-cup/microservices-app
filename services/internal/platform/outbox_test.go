package platform

import (
	"context"
	"testing"
	"time"
)

func TestNewOutboxStore_NilPool(t *testing.T) {
	store := NewOutboxStore(nil, nil)
	if store != nil {
		t.Fatal("expected nil OutboxStore when pool is nil")
	}
}

func TestOutboxStore_PublishPending_NilStore(t *testing.T) {
	var store *OutboxStore
	n, err := store.PublishPending(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 published, got %d", n)
	}
}

func TestOutboxStore_Cleanup_NilStore(t *testing.T) {
	var store *OutboxStore
	err := store.Cleanup(context.Background(), 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOutboxStore_StartPoller_NilStore(t *testing.T) {
	var store *OutboxStore
	// Should not panic
	store.StartPoller(context.Background(), time.Second)
}
