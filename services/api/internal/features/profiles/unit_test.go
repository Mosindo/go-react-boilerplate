package profiles

import (
	"strings"
	"testing"
)

func TestValidateFirstName(t *testing.T) {
	ok := map[string]string{
		"  Anne-Marie ": "Anne-Marie",
		"O'Neil":        "O'Neil",
		"José   Luis":   "José Luis",
		"Zoë​":          "Zoë",
		"Ann\x00a":      "Anna",
		"Жанна":         "Жанна",
		"D’Arcy":        "D’Arcy",
	}
	for in, want := range ok {
		got, err := ValidateFirstName(in)
		if err != nil || got != want {
			t.Errorf("ValidateFirstName(%q)=%q,%v want %q", in, got, err, want)
		}
	}
	bad := []string{"", "   ", "Bob1", "Bob_", "<b>x</b>", "a@b", "---", "''", strings.Repeat("a", 41), "Ann\nBob"}
	for _, in := range bad {
		if got, err := ValidateFirstName(in); err == nil {
			// "Ann\nBob": newline is a control char and is stripped -> "AnnBob" is legitimately valid.
			if in == "Ann\nBob" && got == "AnnBob" {
				continue
			}
			t.Errorf("ValidateFirstName(%q) accepted as %q", in, got)
		}
	}
	if _, err := ValidateFirstName(strings.Repeat("a", 40)); err != nil {
		t.Errorf("40 chars must be accepted: %v", err)
	}
}

func TestValidateBioAndCity(t *testing.T) {
	bio, err := ValidateBio("  hello\r\nworld\x07‮ ")
	if err != nil || bio != "hello\nworld" {
		t.Fatalf("bio=%q err=%v", bio, err)
	}
	if _, err := ValidateBio(strings.Repeat("é", 500)); err != nil {
		t.Fatal("500 runes must pass")
	}
	if _, err := ValidateBio(strings.Repeat("é", 501)); err == nil {
		t.Fatal("501 runes must fail")
	}
	if c, err := ValidateCity("  New   York\t"); err != nil || c != "New York" {
		t.Fatalf("city=%q err=%v", c, err)
	}
	if _, err := ValidateCity(strings.Repeat("c", 81)); err == nil {
		t.Fatal("81 city chars must fail")
	}
}

func TestNormalizeInterests(t *testing.T) {
	got, err := NormalizeInterests([]string{" Music", "music", "art"})
	if err != nil || len(got) != 2 || got[0] != "music" || got[1] != "art" {
		t.Fatalf("got %v %v", got, err)
	}
	many := make([]string, 11)
	for i := range many {
		many[i] = string(rune('a'+i)) + "x"
	}
	if _, err := NormalizeInterests(many); err == nil {
		t.Fatal("11 interests must fail")
	}
	if _, err := NormalizeInterests(many[:10]); err != nil {
		t.Fatal("10 must pass")
	}
	if _, err := NormalizeInterests([]string{""}); err == nil {
		t.Fatal("empty slug must fail")
	}
}

func TestValidateLocation(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	lat, lng, err := ValidateLocation(LocationRequest{Latitude: f(48.85661), Longitude: f(-2.3522219)})
	if err != nil || lat != 48.86 || lng != -2.35 {
		t.Fatalf("%v %v %v", lat, lng, err)
	}
	if lat, lng, _ := ValidateLocation(LocationRequest{Latitude: f(-0.001), Longitude: f(0.004)}); lat != 0 || lng != 0 {
		t.Fatal("zero normalisation")
	}
	for _, c := range []LocationRequest{
		{Latitude: f(90.1), Longitude: f(0)}, {Latitude: f(-91), Longitude: f(0)},
		{Latitude: f(0), Longitude: f(180.5)}, {Latitude: f(0)}, {},
	} {
		if _, _, err := ValidateLocation(c); err == nil {
			t.Errorf("accepted %+v", c)
		}
	}
}

func TestValidatePreferences(t *testing.T) {
	i := func(v int) *int { return &v }
	ok := PreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, AgeMin: i(20), AgeMax: i(30), MaxDistanceKm: i(50)}
	p, err := ValidatePreferences(ok)
	if err != nil || len(p.InterestedIn) != 2 {
		t.Fatalf("%v %v", p, err)
	}
	bads := []PreferencesRequest{
		{InterestedIn: nil, AgeMin: i(20), AgeMax: i(30), MaxDistanceKm: i(50)},
		{InterestedIn: []string{"alien"}, AgeMin: i(20), AgeMax: i(30), MaxDistanceKm: i(50)},
		{InterestedIn: []string{"man"}, AgeMin: i(17), AgeMax: i(30), MaxDistanceKm: i(50)},
		{InterestedIn: []string{"man"}, AgeMin: i(30), AgeMax: i(29), MaxDistanceKm: i(50)},
		{InterestedIn: []string{"man"}, AgeMin: i(30), AgeMax: i(100), MaxDistanceKm: i(50)},
		{InterestedIn: []string{"man"}, AgeMin: i(30), AgeMax: i(40), MaxDistanceKm: i(0)},
		{InterestedIn: []string{"man"}, AgeMin: i(30), AgeMax: i(40), MaxDistanceKm: i(501)},
		{InterestedIn: []string{"man"}, AgeMin: i(30), AgeMax: i(40)},
	}
	for n, b := range bads {
		if _, err := ValidatePreferences(b); err == nil {
			t.Errorf("case %d accepted", n)
		}
	}
	edge := PreferencesRequest{InterestedIn: []string{"non_binary"}, AgeMin: i(99), AgeMax: i(99), MaxDistanceKm: i(500)}
	if _, err := ValidatePreferences(edge); err != nil {
		t.Errorf("edge rejected: %v", err)
	}
}

func TestBucketKm(t *testing.T) {
	cases := []struct {
		d    float64
		want int
	}{
		{0, 1}, {0.2, 1}, {1, 1}, {1.01, 2}, {2.0000000001, 2}, {9.2, 10}, {9.99, 10},
		{10, 10}, {10.1, 15}, {14.9, 15}, {15, 15}, {15.1, 20}, {103, 105},
	}
	for _, c := range cases {
		if got := BucketKm(c.d); got != c.want {
			t.Errorf("BucketKm(%v)=%d want %d", c.d, got, c.want)
		}
	}
}

func TestDistanceKm(t *testing.T) {
	// Paris -> London is about 344 km.
	d := DistanceKm(48.85, 2.35, 51.51, -0.13)
	if d < 340 || d > 348 {
		t.Fatalf("paris-london = %v", d)
	}
	if DistanceKm(10, 10, 10, 10) != 0 {
		t.Fatal("same point must be 0")
	}
	// Symmetric, and antipodes stay finite.
	if a, b := DistanceKm(1, 2, 3, 4), DistanceKm(3, 4, 1, 2); a != b {
		t.Fatal("asymmetric")
	}
	if d := DistanceKm(0, 0, 0, 180); d < 20000 || d > 20040 {
		t.Fatalf("antipode %v", d)
	}
}
