package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"8f14e45f-ceea-467f-a8f4-6d1f2b0c9e11": true,
		"8F14E45F-CEEA-467F-A8F4-6D1F2B0C9E11": true,
		"8f14e45f-ceea-467f-a8f4-6d1f2b0c9e1":  false,
		"8f14e45fxceea-467f-a8f4-6d1f2b0c9e11": false,
		"../../etc/passwd":                     false,
		"":                                     false,
	}
	for input, want := range cases {
		if got := IsUUID(input); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", input, got, want)
		}
	}
}
