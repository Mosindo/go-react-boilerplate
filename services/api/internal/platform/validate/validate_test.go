package validate

import "testing"

func TestIsUUID(t *testing.T) {
	good := "123e4567-e89b-12d3-a456-426614174000"
	if !IsUUID(good) {
		t.Fatalf("%s should be valid", good)
	}
	for _, bad := range []string{"", "abc", good + "x", "123e4567e89b12d3a456426614174000", "' OR 1=1 --"} {
		if IsUUID(bad) {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}
