package profiles

import (
	"testing"
	"time"
)

func TestValidateFirstName(t *testing.T) {
	ok := []string{"Léa", "  Jean   Paul ", "Anne-Sophie", "O'Neil", "O’Neil", "Zoë"}
	for _, name := range ok {
		if _, err := validateFirstName(name); err != nil {
			t.Fatalf("expected %q to be valid: %v", name, err)
		}
	}
	if got, _ := validateFirstName("  Jean   Paul "); got != "Jean Paul" {
		t.Fatalf("expected normalised spaces, got %q", got)
	}
	bad := []string{"", "   ", "1337", "<script>", "-Lea", "Abcdefghijklmnopqrstuvwxyzabcdef"}
	for _, name := range bad {
		if _, err := validateFirstName(name); err == nil {
			t.Fatalf("expected %q to be rejected", name)
		}
	}
}

func TestValidateBirthdateEnforcesAdulthood(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	if _, err := validateBirthdate("2008-06-01", now); err != nil {
		t.Fatalf("18th birthday today must be accepted: %v", err)
	}
	if _, err := validateBirthdate("2008-06-02", now); err != ErrUnderage {
		t.Fatalf("expected underage, got %v", err)
	}
	if _, err := validateBirthdate("1890-01-01", now); err != ErrInvalidAge {
		t.Fatalf("expected implausible age, got %v", err)
	}
	if _, err := validateBirthdate("2001-02-30", now); err != ErrInvalidBirthdate {
		t.Fatalf("expected invalid date, got %v", err)
	}
}

func TestCleanText(t *testing.T) {
	if got, err := cleanText("  hello\n\n\n\nworld  ", 100, true); err != nil || got != "hello\n\nworld" {
		t.Fatalf("unexpected clean result %q %v", got, err)
	}
	if _, err := cleanText("bad\x00byte", 100, true); err != ErrInvalidText {
		t.Fatalf("expected control chars to be rejected, got %v", err)
	}
	if _, err := cleanText("abcdef", 5, false); err != ErrTextTooLong {
		t.Fatalf("expected too long, got %v", err)
	}
}

func TestValidatePreferences(t *testing.T) {
	prefs, err := validatePreferences(PreferencesRequest{InterestedIn: []string{"woman", "woman", "man"}, MinAge: 25, MaxAge: 40, MaxDistanceKm: 30})
	if err != nil || len(prefs.InterestedIn) != 2 {
		t.Fatalf("unexpected result %+v %v", prefs, err)
	}
	bad := []PreferencesRequest{
		{InterestedIn: nil, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"alien"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 40, MaxAge: 30, MaxDistanceKm: 10},
		{InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 30, MaxDistanceKm: 900},
	}
	for i, req := range bad {
		if _, err := validatePreferences(req); err == nil {
			t.Fatalf("case %d should be rejected", i)
		}
	}
}
