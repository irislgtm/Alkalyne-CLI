package models

import "testing"

func TestRelayStatusValues(t *testing.T) {
	values := []RelayStatus{RelayOnline, RelayOffline}
	if len(values) != 2 {
		t.Errorf("expected 2 status values, got %d", len(values))
	}
}
