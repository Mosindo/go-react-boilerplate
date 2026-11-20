package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	if !IsUUID("3f0f4c1e-8a7b-4f7e-9b0e-2d7f1c3a9b10") {
		t.Fatal("expected valid uuid")
	}
	for _, value := range []string{"", "abc", "3f0f4c1e8a7b4f7e9b0e2d7f1c3a9b10", "3f0f4c1e-8a7b-4f7e-9b0e-2d7f1c3a9b1z"} {
		if IsUUID(value) {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestParseInt(t *testing.T) {
	if v, err := parseInt("", 20, 1, 50); err != nil || v != 20 {
		t.Fatalf("expected fallback, got %d %v", v, err)
	}
	if v, err := parseInt(" 7 ", 20, 1, 50); err != nil || v != 7 {
		t.Fatalf("expected 7, got %d %v", v, err)
	}
	for _, raw := range []string{"0", "51", "-1", "x", "1.5"} {
		if _, err := parseInt(raw, 20, 1, 50); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}
