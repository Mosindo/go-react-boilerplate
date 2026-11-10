package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	cases := map[string]bool{
		"8f14e45f-ceea-467f-a0f6-4d3a2c7b9e10": true,
		"8F14E45F-CEEA-467F-A0F6-4D3A2C7B9E10": true,
		"8f14e45f-ceea-467f-a0f6-4d3a2c7b9e1":  false,
		"8f14e45fxceea-467f-a0f6-4d3a2c7b9e10": false,
		"zf14e45f-ceea-467f-a0f6-4d3a2c7b9e10": false,
		"' OR 1=1 --":                          false,
		"":                                     false,
	}
	for input, want := range cases {
		if got := IsUUID(input); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", input, got, want)
		}
	}
}
