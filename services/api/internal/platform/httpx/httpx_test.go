package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"3f2504e0-4f89-11d3-9a0c-0305e82c3301": true,
		"3F2504E0-4F89-11D3-9A0C-0305E82C3301": true,
		"3f2504e0-4f89-11d3-9a0c-0305e82c330":  false,
		"3f2504e0x4f89-11d3-9a0c-0305e82c3301": false,
		"zf2504e0-4f89-11d3-9a0c-0305e82c3301": false,
		"":                                     false,
	}
	for input, want := range cases {
		if got := IsUUID(input); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestLimitAndOffset(t *testing.T) {
	if got := Limit("", 20, 50); got != 20 {
		t.Fatalf("default limit: got %d", got)
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
