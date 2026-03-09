package platform

import (
	"testing"
)

func TestExtractIdempotencyKey(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"has key", map[string]string{"Idempotency-Key": "abc-123"}, "abc-123"},
		{"no key", map[string]string{}, ""},
		{"empty key", map[string]string{"Idempotency-Key": ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractIdempotencyKey(tt.headers)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
