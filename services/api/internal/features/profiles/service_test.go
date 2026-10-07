package profiles

import (
	"testing"
	"time"
)

func TestAgeOn(t *testing.T) {
	ref := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-03-10", 18}, // birthday today
		{"2008-03-11", 17}, // birthday tomorrow
		{"2000-12-31", 25},
		{"1990-03-09", 36},
	}
	for _, tc := range cases {
		birth, _ := time.Parse("2006-01-02", tc.birth)
		if got := AgeOn(birth, ref); got != tc.want {
			t.Errorf("AgeOn(%s) = %d, want %d", tc.birth, got, tc.want)
		}
	}
}

func TestValidateFirstName(t *testing.T) {
	valid := []string{"Léa", "Anne-Marie", "O'Neil", "Zoë", "李雷"}
	for _, n := range valid {
		if err := validateFirstName(n); err != nil {
			t.Errorf("%q should be valid: %v", n, err)
		}
	}
	invalid := []string{"", "   ", "<script>", "John3", "http://x.io", "....", "A very very very very very long name exceeding the limit"}
	for _, n := range invalid {
		if err := validateFirstName(n); err == nil {
			t.Errorf("%q should be invalid", n)
		}
	}
}
