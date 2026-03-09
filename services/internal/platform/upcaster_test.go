package platform

import (
	"encoding/json"
	"testing"
)

func TestUpcasterRegistry_Apply(t *testing.T) {
	registry := NewUpcasterRegistry()

	// Register upcaster that adds a "version" field.
	registry.Register("greeting.created", func(data json.RawMessage) json.RawMessage {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		m["schema_version"] = 2
		out, _ := json.Marshal(m)
		return out
	})

	input := json.RawMessage(`{"name":"Alice"}`)
	output := registry.Apply("greeting.created", input)

	var result map[string]any
	_ = json.Unmarshal(output, &result)

	if result["schema_version"] != float64(2) {
		t.Fatalf("expected schema_version=2, got %v", result["schema_version"])
	}
	if result["name"] != "Alice" {
		t.Fatalf("expected name=Alice, got %v", result["name"])
	}
}

func TestUpcasterRegistry_NoUpcaster(t *testing.T) {
	registry := NewUpcasterRegistry()
	input := json.RawMessage(`{"name":"Bob"}`)
	output := registry.Apply("unknown.event", input)

	if string(output) != string(input) {
		t.Fatalf("expected unchanged data, got %s", output)
	}
}

func TestUpcasterRegistry_NilRegistry(t *testing.T) {
	var registry *UpcasterRegistry
	input := json.RawMessage(`{"name":"Charlie"}`)
	output := registry.Apply("test", input)

	if string(output) != string(input) {
		t.Fatalf("expected unchanged data, got %s", output)
	}
}

func TestUpcasterRegistry_ChainedUpcasters(t *testing.T) {
	registry := NewUpcasterRegistry()

	// First upcaster: v1 -> v2 (add field).
	registry.Register("test.event", func(data json.RawMessage) json.RawMessage {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		m["added_v2"] = true
		out, _ := json.Marshal(m)
		return out
	})

	// Second upcaster: v2 -> v3 (rename field).
	registry.Register("test.event", func(data json.RawMessage) json.RawMessage {
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		m["added_v3"] = true
		out, _ := json.Marshal(m)
		return out
	})

	input := json.RawMessage(`{"original":true}`)
	output := registry.Apply("test.event", input)

	var result map[string]any
	_ = json.Unmarshal(output, &result)

	if result["added_v2"] != true {
		t.Fatal("missing added_v2")
	}
	if result["added_v3"] != true {
		t.Fatal("missing added_v3")
	}
	if result["original"] != true {
		t.Fatal("missing original")
	}
}
