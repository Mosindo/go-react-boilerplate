package profiles

import (
	"strings"
	"testing"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

var fixedNow = time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

func validInput() ProfileInput {
	return ProfileInput{FirstName: "Léa", BirthDate: "1995-06-15", Gender: "woman", Bio: "Bonjour", City: "Lyon", Interests: []string{"cinema", "music"}}
}

func fieldsOf(t *testing.T, err error) map[string]string {
	t.Helper()
	appErr, ok := platformerrors.As(err)
	if !ok {
		t.Fatalf("expected AppError, got %v", err)
	}
	return appErr.Fields
}

func TestAgeOn(t *testing.T) {
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"2000-02-29", 26},
		{"1995-12-31", 30},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeOn(b, fixedNow); got != c.want {
			t.Errorf("AgeOn(%s)=%d want %d", c.birth, got, c.want)
		}
	}
}

func TestNormalizeProfileAcceptsValid(t *testing.T) {
	p, slugs, err := normalizeProfile("u", validInput(), fixedNow)
	if err != nil || p.FirstName != "Léa" || len(slugs) != 2 {
		t.Fatalf("unexpected %+v %v %v", p, slugs, err)
	}
}

func TestNormalizeProfileEnforcesMinimumAge(t *testing.T) {
	in := validInput()
	in.BirthDate = "2008-06-16" // 17 years old
	_, _, err := normalizeProfile("u", in, fixedNow)
	if fieldsOf(t, err)["birthDate"] == "" {
		t.Fatal("under-18 must be rejected")
	}
	in.BirthDate = "1900-01-01"
	if _, _, err := normalizeProfile("u", in, fixedNow); err == nil {
		t.Fatal("impossible age must be rejected")
	}
	in.BirthDate = "not-a-date"
	if _, _, err := normalizeProfile("u", in, fixedNow); err == nil {
		t.Fatal("malformed date must be rejected")
	}
}

func TestNormalizeProfileFieldRules(t *testing.T) {
	cases := map[string]func(*ProfileInput){
		"firstName": func(in *ProfileInput) { in.FirstName = "<script>" },
		"gender":    func(in *ProfileInput) { in.Gender = "robot" },
		"bio":       func(in *ProfileInput) { in.Bio = strings.Repeat("x", 501) },
		"city":      func(in *ProfileInput) { in.City = "" },
		"interests": func(in *ProfileInput) { in.Interests = strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",") },
		"latitude":  func(in *ProfileInput) { v := 10.0; in.Latitude = &v },
	}
	for field, mutate := range cases {
		in := validInput()
		mutate(&in)
		_, _, err := normalizeProfile("u", in, fixedNow)
		if fieldsOf(t, err)[field] == "" {
			t.Errorf("%s: expected a field error", field)
		}
	}
	in := validInput()
	in.FirstName = "  "
	if _, _, err := normalizeProfile("u", in, fixedNow); err == nil {
		t.Error("blank name must be rejected")
	}
}

func TestLocationIsRoundedAndRangeChecked(t *testing.T) {
	in := validInput()
	la, lo := 45.76412, 4.83566
	in.Latitude, in.Longitude = &la, &lo
	p, _, err := normalizeProfile("u", in, fixedNow)
	if err != nil || p.Latitude == nil {
		t.Fatalf("unexpected %v", err)
	}
	if *p.Latitude != 45.76 || *p.Longitude != 4.84 {
		t.Fatalf("coordinates not coarsened: %v %v", *p.Latitude, *p.Longitude)
	}
	bad := 91.0
	in.Latitude = &bad
	if _, _, err := normalizeProfile("u", in, fixedNow); err == nil {
		t.Fatal("out-of-range latitude must be rejected")
	}
	in = validInput()
	in.ClearLocation, in.Latitude, in.Longitude = true, &la, &lo
	p, _, _ = normalizeProfile("u", in, fixedNow)
	if p.Latitude != nil {
		t.Fatal("clearLocation must drop coordinates")
	}
}

func TestInterestsAreDeduplicated(t *testing.T) {
	in := validInput()
	in.Interests = []string{"Cinema", "cinema", " music "}
	_, slugs, err := normalizeProfile("u", in, fixedNow)
	if err != nil || len(slugs) != 2 {
		t.Fatalf("got %v %v", slugs, err)
	}
}

func TestValidatePreferences(t *testing.T) {
	ok := Preferences{InterestedIn: []string{"woman", "woman"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30, Discoverable: true}
	got, err := validatePreferences(ok)
	if err != nil || len(got.InterestedIn) != 1 {
		t.Fatalf("got %+v %v", got, err)
	}
	bads := []Preferences{
		{InterestedIn: nil, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"x"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 40, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 100, MaxDistanceKm: 30},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 0},
		{InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 40, MaxDistanceKm: 501},
	}
	for i, b := range bads {
		if _, err := validatePreferences(b); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
}
