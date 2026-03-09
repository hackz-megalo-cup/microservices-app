package platform

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	e := NewEvent("greeting.created", "greeter-service", map[string]string{"name": "test"})
	if e.ID == "" {
		t.Fatal("event ID should not be empty")
	}
	if e.Type != "greeting.created" {
		t.Fatalf("expected type greeting.created, got %s", e.Type)
	}
	if e.Source != "greeter-service" {
		t.Fatalf("expected source greeter-service, got %s", e.Source)
	}
	if time.Since(e.Timestamp) > time.Second {
		t.Fatal("timestamp should be recent")
	}
}

func TestEventSerialization(t *testing.T) {
	e := NewEvent("test", "test-service", map[string]string{"key": "value"})
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed.ID != e.ID {
		t.Fatalf("ID mismatch: %s vs %s", parsed.ID, e.ID)
	}
}

func TestNewEventPublisher_NoBrokers(t *testing.T) {
	pub, err := NewEventPublisher(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub != nil {
		t.Fatal("publisher should be nil when no brokers")
	}
}
