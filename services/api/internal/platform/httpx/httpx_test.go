package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"3f2b8a52-6f0e-4c5b-9a40-0c7f6d1e2a11": true,
		"3F2B8A52-6F0E-4C5B-9A40-0C7F6D1E2A11": true,
		"not-a-uuid":                           false,
		"":                                     false,
		"3f2b8a52-6f0e-4c5b-9a40-0c7f6d1e2a1":  false,
		"3f2b8a52-6f0e-4c5b-9a40-0c7f6d1e2a11'; DROP TABLE users;--": false,
	}
	for in, want := range cases {
		if got := IsUUID(in); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestBoundedParsing(t *testing.T) {
	if got := Limit("", 20, 50); got != 20 {
		t.Errorf("default limit = %d", got)
	}
	if got := Limit("999", 20, 50); got != 50 {
		t.Errorf("clamped limit = %d", got)
	}
	if got := Limit("-4", 20, 50); got != 1 {
		t.Errorf("negative limit = %d", got)
	}
	if got := Limit("abc", 20, 50); got != 20 {
		t.Errorf("invalid limit = %d", got)
	}
	if got := Offset("-5", 1000); got != 0 {
		t.Errorf("negative offset = %d", got)
	}
	if got := Offset("5000", 1000); got != 1000 {
		t.Errorf("clamped offset = %d", got)
	}
}
