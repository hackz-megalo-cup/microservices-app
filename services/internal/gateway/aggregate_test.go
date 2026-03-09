package gateway

import (
	"encoding/json"
	"testing"
)

func TestInvocationAggregate_Create(t *testing.T) {
	agg := NewInvocationAggregate("test-id")
	agg.Create("Alice", "hello from custom-lang")

	if agg.Name != "Alice" {
		t.Fatalf("expected Name=Alice, got %s", agg.Name)
	}
	if agg.Message != "hello from custom-lang" {
		t.Fatalf("expected Message='hello from custom-lang', got %s", agg.Message)
	}
	if agg.Status != "completed" {
		t.Fatalf("expected Status=completed, got %s", agg.Status)
	}
	if len(agg.Changes()) != 1 {
		t.Fatalf("expected 1 change, got %d", len(agg.Changes()))
	}
}

func TestInvocationAggregate_Fail(t *testing.T) {
	agg := NewInvocationAggregate("test-id")
	agg.Fail("Bob", "connection refused")

	if agg.Status != "failed" {
		t.Fatalf("expected Status=failed, got %s", agg.Status)
	}
	if len(agg.Changes()) != 1 {
		t.Fatalf("expected 1 change, got %d", len(agg.Changes()))
	}
}

func TestInvocationAggregate_Compensate(t *testing.T) {
	agg := NewInvocationAggregate("test-id")
	agg.Fail("Bob", "error")
	agg.Compensate("downstream failed")

	if agg.Status != "compensated" {
		t.Fatalf("expected Status=compensated, got %s", agg.Status)
	}
	if len(agg.Changes()) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(agg.Changes()))
	}

	// Idempotent: second compensate should not add another event.
	agg.Compensate("duplicate")
	if len(agg.Changes()) != 2 {
		t.Fatalf("expected still 2 changes, got %d", len(agg.Changes()))
	}
}

func TestInvocationAggregate_ApplyEvent(t *testing.T) {
	agg := NewInvocationAggregate("test-id")

	data, _ := json.Marshal(InvocationCreatedData{Name: "Alice", Message: "hello"})
	agg.ApplyEvent(EventInvocationCreated, data)

	if agg.Name != "Alice" || agg.Message != "hello" || agg.Status != "completed" {
		t.Fatalf("unexpected state after apply: Name=%s Message=%s Status=%s", agg.Name, agg.Message, agg.Status)
	}
}

func TestInvocationAggregate_StreamType(t *testing.T) {
	agg := NewInvocationAggregate("test-id")
	if agg.StreamType() != "invocation" {
		t.Fatalf("expected stream type 'invocation', got %s", agg.StreamType())
	}
}
