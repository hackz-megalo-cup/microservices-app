package platform

import (
	"context"
	"testing"
)

func TestNewSnapshotStore_NilPool(t *testing.T) {
	s := NewSnapshotStore(nil)
	if s != nil {
		t.Fatal("expected nil when pool is nil")
	}
}

func TestSnapshot_LoadNilStore(t *testing.T) {
	var s *SnapshotStore
	snap, err := s.Load(context.TODO(), "test")
	if snap != nil || err != nil {
		t.Fatal("expected nil, nil from nil store")
	}
}

func TestSnapshot_SaveNilStore(t *testing.T) {
	var s *SnapshotStore
	err := s.Save(context.TODO(), "stream-1", "test", 1, map[string]string{"key": "value"})
	if err != nil {
		t.Fatal("expected nil error from nil store")
	}
}

func TestSaveSnapshot_NilSnapshots(t *testing.T) {
	err := SaveSnapshot(context.TODO(), nil, nil, 10)
	if err != nil {
		t.Fatal("expected nil error")
	}
}
