package httpx

import "testing"

func TestLimit(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{{"", 20}, {"abc", 20}, {"-3", 20}, {"0", 20}, {"5", 5}, {"500", 50}}
	for _, tc := range cases {
		if got := Limit(tc.raw, 20, 50); got != tc.want {
			t.Errorf("Limit(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestOffset(t *testing.T) {
	if Offset("-1", 100) != 0 || Offset("x", 100) != 0 || Offset("30", 100) != 30 || Offset("1000", 100) != 100 {
		t.Fatal("unexpected offset parsing")
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("3f2504e0-4f89-11d3-9a0c-0305e82c3301") {
		t.Fatal("expected valid uuid")
	}
	for _, bad := range []string{"", "not-a-uuid", "3f2504e0-4f89-11d3-9a0c-0305e82c330g", "3f2504e04f8911d39a0c0305e82c3301xxxx"} {
		if IsUUID(bad) {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}
