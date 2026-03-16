package transport_test

import (
	"testing"

	"github.com/hackz-megalo-cup/microservices-app/game-server/internal/transport"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantT   string
		wantErr bool
	}{
		{"tap", []byte(`{"t":"tap"}`), "tap", false},
		{"join", []byte(`{"t":"join","userId":"550e8400-e29b-41d4-a716-446655440000"}`), "join", false},
		{"special", []byte(`{"t":"special","userId":"550e8400-e29b-41d4-a716-446655440000"}`), "special", false},
		{"invalid json", []byte(`not json`), "", true},
		{"missing type", []byte(`{"userId":"foo"}`), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := transport.ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && msg.T != tt.wantT {
				t.Errorf("ParseMessage().T = %s, want %s", msg.T, tt.wantT)
			}
		})
	}
}
