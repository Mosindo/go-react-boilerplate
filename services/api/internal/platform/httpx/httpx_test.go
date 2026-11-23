package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"7b1d7f0e-4e43-4a3f-9a5a-1f1f1f1f1f1f": true,
		"7B1D7F0E-4E43-4A3F-9A5A-1F1F1F1F1F1F": true,
		"7b1d7f0e4e434a3f9a5a1f1f1f1f1f1f":     false,
		"7b1d7f0e-4e43-4a3f-9a5a-1f1f1f1f1f1g": false,
		"":                                     false,
		"../../etc/passwd":                     false,
	}
	for input, want := range cases {
		if got := IsUUID(input); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseLimit(t *testing.T) {
	if got := ParseLimit("", 20, 50); got != 20 {
		t.Fatalf("expected fallback, got %d", got)
	}
	if got := ParseLimit("-3", 20, 50); got != 20 {
		t.Fatalf("expected fallback for negative, got %d", got)
	}
	if got := ParseLimit("500", 20, 50); got != 50 {
		t.Fatalf("expected clamp, got %d", got)
	}
	if got := ParseLimit(" 7 ", 20, 50); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}
