package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"3f1c5a3e-9a4b-4c9e-8a8f-1b2c3d4e5f60": true,
		"3F1C5A3E-9A4B-4C9E-8A8F-1B2C3D4E5F60": true,
		"3f1c5a3e9a4b4c9e8a8f1b2c3d4e5f60":     false,
		"3f1c5a3e-9a4b-4c9e-8a8f-1b2c3d4e5f6z": false,
		"":                                     false,
		"' OR 1=1 --":                          false,
	}
	for input, want := range cases {
		if got := IsUUID(input); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestLimitAndOffset(t *testing.T) {
	if got := Limit("", 20, 50); got != 20 {
		t.Fatalf("empty limit: got %d", got)
	}
	if got := Limit("-3", 20, 50); got != 20 {
		t.Fatalf("negative limit: got %d", got)
	}
	if got := Limit("500", 20, 50); got != 50 {
		t.Fatalf("clamped limit: got %d", got)
	}
	if got := Limit("7", 20, 50); got != 7 {
		t.Fatalf("explicit limit: got %d", got)
	}
	if got := Offset("abc", 1000); got != 0 {
		t.Fatalf("invalid offset: got %d", got)
	}
	if got := Offset("5000", 1000); got != 1000 {
		t.Fatalf("clamped offset: got %d", got)
	}
}
