package auth

import (
	"errors"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestAgeOn(t *testing.T) {
	cases := []struct {
		birth, now string
		want       int
	}{
		{"2000-06-15", "2026-06-14", 25},
		{"2000-06-15", "2026-06-15", 26},
		{"2000-06-15", "2026-06-16", 26},
		{"2008-02-29", "2026-02-28", 17},
		{"2008-02-29", "2026-03-01", 18},
		{"2026-01-01", "2026-01-01", 0},
		{"2027-01-01", "2026-01-01", -1},
	}
	for _, c := range cases {
		if got := AgeOn(d(c.birth), d(c.now)); got != c.want {
			t.Errorf("AgeOn(%s,%s)=%d want %d", c.birth, c.now, got, c.want)
		}
	}
}

func TestValidateBirthDate(t *testing.T) {
	now := d("2026-09-29")
	cases := []struct {
		birth string
		want  error
	}{
		{"2008-09-29", nil},
		{"2008-09-30", ErrUnderage},
		{"2026-09-30", ErrBirthDateFuture},
		{"2030-01-01", ErrBirthDateFuture},
		{"1900-01-01", ErrBirthDateAbsurd},
		{"1905-09-29", nil},
		{"1905-09-28", ErrBirthDateAbsurd},
		{"1990-01-01", nil},
	}
	for _, c := range cases {
		if err := ValidateBirthDate(d(c.birth), now); !errors.Is(err, c.want) {
			t.Errorf("ValidateBirthDate(%s)=%v want %v", c.birth, err, c.want)
		}
	}
}
