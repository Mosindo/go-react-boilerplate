package moderation

import "testing"

func TestNormalizeTarget(t *testing.T) {
	self := "123e4567-e89b-12d3-a456-426614174000"
	other := "123E4567-E89B-12D3-A456-426614174001"
	got, err := normalizeTarget(self, other)
	if err != nil || got != "123e4567-e89b-12d3-a456-426614174001" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := normalizeTarget(self, self); err == nil {
		t.Error("self target must be rejected")
	}
	if _, err := normalizeTarget(self, "not-a-uuid"); err == nil {
		t.Error("malformed id must be rejected")
	}
}
