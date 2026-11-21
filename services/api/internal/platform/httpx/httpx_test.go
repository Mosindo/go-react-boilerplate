package httpx

import "testing"

func TestParseLimit(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", 20}, {"abc", 20}, {"-3", 20}, {"0", 20}, {"5", 5}, {"500", 100},
	}
	for _, tc := range cases {
		if got := ParseLimit(tc.raw, 20, 100); got != tc.want {
			t.Fatalf("ParseLimit(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestParseOffset(t *testing.T) {
	if ParseOffset("-1") != 0 || ParseOffset("x") != 0 || ParseOffset("12") != 12 || ParseOffset("999999") != 10_000 {
		t.Fatal("unexpected offset parsing")
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("11111111-1111-1111-1111-111111111111") {
		t.Fatal("expected valid uuid")
	}
	for _, bad := range []string{"", "1111", "11111111-1111-1111-1111-11111111111g", "11111111x1111-1111-1111-111111111111"} {
		if IsUUID(bad) {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}
