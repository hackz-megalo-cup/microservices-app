package projector

import (
	"testing"
)

func TestNewRebuilder_NilPool(t *testing.T) {
	r := NewRebuilder(nil, nil, nil)
	if r != nil {
		t.Fatal("expected nil when pool is nil")
	}
}
