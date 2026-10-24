package profiles

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today: already 18
		{"2008-06-16", 17}, // birthday tomorrow
		{"1990-01-01", 36},
		{"2000-02-29", 26},
	}
	for _, c := range cases {
		b, _ := time.Parse(dateLayout, c.birth)
		if got := AgeAt(b, now); got != c.want {
			t.Errorf("AgeAt(%s)=%d want %d", c.birth, got, c.want)
		}
	}
}

func newTestService() *Service {
	s := NewService(nil)
	s.now = func() time.Time { return time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) }
	return s
}

func TestValidateProfile(t *testing.T) {
	s := newTestService()
	valid := UpsertProfileRequest{FirstName: "  Léa-Marie ", BirthDate: "1995-03-02", Gender: GenderWoman, Bio: "Salut\x00\n\n\n\nbio", City: " Lyon "}
	in, err := s.validateProfile(valid)
	if err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	if in.FirstName != "Léa-Marie" || in.City != "Lyon" || strings.ContainsRune(in.Bio, 0) || strings.Contains(in.Bio, "\n\n\n") {
		t.Fatalf("input not normalized: %+v", in)
	}
	if !in.Discoverable || !in.ShowDistance {
		t.Fatal("privacy flags default to visible")
	}

	bad := map[string]UpsertProfileRequest{
		"minor":       {FirstName: "A", BirthDate: "2008-06-16", Gender: GenderMan},
		"ancient":     {FirstName: "A", BirthDate: "1890-01-01", Gender: GenderMan},
		"bad gender":  {FirstName: "A", BirthDate: "1995-03-02", Gender: "x"},
		"name chars":  {FirstName: "A<script>", BirthDate: "1995-03-02", Gender: GenderMan},
		"too many":    {FirstName: "A", BirthDate: "1995-03-02", Gender: GenderMan, Interests: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}},
		"name length": {FirstName: strings.Repeat("a", 41), BirthDate: "1995-03-02", Gender: GenderMan},
	}
	for name, req := range bad {
		_, err := s.validateProfile(req)
		var ve ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: expected a validation error, got %v", name, err)
		}
	}
}

func TestCleanText(t *testing.T) {
	if got := cleanText("  a\tb\r\nc\x07 ", false); got != "a b  c" {
		t.Fatalf("got %q", got)
	}
}
