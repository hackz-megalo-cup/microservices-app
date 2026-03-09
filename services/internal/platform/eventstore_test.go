package platform

import (
	"testing"
)

func TestStoredEvent_HasGlobalPosition(t *testing.T) {
	e := StoredEvent{}
	if e.GlobalPosition != 0 {
		t.Fatal("expected zero value for GlobalPosition")
	}
}
