package httpx

import "testing"

func TestLimitAndOffset(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{{"", 20}, {"abc", 20}, {"-3", 20}, {"0", 20}, {"5", 5}, {"500", 50}}
	for _, tc := range cases {
		if got := Limit(tc.raw, 20, 50); got != tc.want {
			t.Fatalf("Limit(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
	if Offset("-1", 100) != 0 || Offset("7", 100) != 7 || Offset("1000", 100) != 100 {
		t.Fatal("unexpected offset parsing")
	}
}

func TestIsUUID(t *testing.T) {
	if !IsUUID("0f8fad5b-d9cb-469f-a165-70867728950e") {
		t.Fatal("expected valid uuid")
	}
	for _, v := range []string{"", "not-a-uuid", "0f8fad5bd9cb469fa16570867728950e", "0f8fad5b-d9cb-469f-a165-70867728950z"} {
		if IsUUID(v) {
			t.Fatalf("expected %q to be rejected", v)
		}
	}
}
