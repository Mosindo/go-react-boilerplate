package httpx

import "testing"

func TestIsUUID(t *testing.T) {
	good := []string{"123e4567-e89b-12d3-a456-426614174000", "00000000-0000-0000-0000-000000000000", "ABCDEF12-e89b-12d3-a456-426614174000"}
	bad := []string{"", "nope", "123e4567e89b12d3a456426614174000", "123e4567-e89b-12d3-a456-42661417400g", "123e4567-e89b-12d3-a456-4266141740000", "'; DROP TABLE users;--"}
	for _, s := range good {
		if !IsUUID(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range bad {
		if IsUUID(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}
