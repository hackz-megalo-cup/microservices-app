package platform

import (
	"testing"
)

func TestParseKafkaBrokers(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected []string
	}{
		{"single broker", "localhost:9092", []string{"localhost:9092"}},
		{"multiple brokers", "broker1:9092,broker2:9092", []string{"broker1:9092", "broker2:9092"}},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseKafkaBrokers(tt.env)
			if len(got) != len(tt.expected) {
				t.Errorf("ParseKafkaBrokers(%q) = %v, want %v", tt.env, got, tt.expected)
			}
		})
	}
}
