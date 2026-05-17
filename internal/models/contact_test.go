package models

import "testing"

func TestContactStatusValues(t *testing.T) {
	values := []ContactStatus{ContactOnline, ContactOffline, ContactPending}
	if len(values) != 3 {
		t.Errorf("expected 3 status values, got %d", len(values))
	}
}
