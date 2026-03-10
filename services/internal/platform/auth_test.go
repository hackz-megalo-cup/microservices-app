package platform

import (
	"testing"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
		err      bool
	}{
		{"valid", "Bearer abc123", "abc123", false},
		{"no prefix", "abc123", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBearerToken(tt.header)
			if (err != nil) != tt.err {
				t.Fatalf("error = %v, wantErr %v", err, tt.err)
			}
			if got != tt.expected {
				t.Fatalf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
