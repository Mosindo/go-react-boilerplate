// Package validate contains pure input validation and normalisation rules.
package validate

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MinPasswordLen = 8
	MaxPasswordLen = 128
	MinAge         = 18
	MaxEmailLen    = 254
)

// NormalizeEmail trims and lower-cases an address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Email normalises and validates an address, returning the normalised form.
func Email(raw string) (string, error) {
	email := NormalizeEmail(raw)
	if email == "" || len(email) > MaxEmailLen {
		return "", errors.New("a valid email address is required")
	}
	parsed, err := mail.ParseAddress(email)
	// Reject display-name forms such as "Bob <bob@x.y>": the input must be a bare address.
	if err != nil || parsed.Address != email || strings.ContainsAny(email, " <>,;\"") {
		return "", errors.New("a valid email address is required")
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return "", errors.New("a valid email address is required")
	}
	return email, nil
}

// Password enforces the 8..128 character policy (counted in runes).
func Password(pw string) error {
	n := utf8.RuneCountInString(pw)
	if !utf8.ValidString(pw) || n < MinPasswordLen || n > MaxPasswordLen {
		return errors.New("password must be between 8 and 128 characters")
	}
	return nil
}

// Text trims s, rejects invalid UTF-8 and control characters (newlines and tabs
// are only accepted when allowMultiline is set) and enforces a rune length range.
func Text(s string, min, max int, allowMultiline bool) (string, error) {
	s = strings.TrimSpace(s)
	if !utf8.ValidString(s) {
		return "", errors.New("text must be valid UTF-8")
	}
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			if allowMultiline {
				continue
			}
			return "", errors.New("text must not contain line breaks")
		}
		if unicode.IsControl(r) || r == ' ' || r == ' ' || r == 0xFFFD {
			return "", errors.New("text contains invalid characters")
		}
	}
	n := utf8.RuneCountInString(s)
	if n < min || n > max {
		return "", errors.New("text length out of range")
	}
	return s, nil
}

// ParseBirthDate parses YYYY-MM-DD.
func ParseBirthDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, errors.New("birthDate must be formatted YYYY-MM-DD")
	}
	return t, nil
}

// AgeOn returns the full years between birth and the calendar date of now
// (both interpreted in UTC). Feb 29 birthdays turn a year older on Mar 1
// in non-leap years.
func AgeOn(birth, now time.Time) int {
	b := birth.UTC()
	n := now.UTC()
	years := n.Year() - b.Year()
	if n.Month() < b.Month() || (n.Month() == b.Month() && n.Day() < b.Day()) {
		years--
	}
	return years
}

// IsAdult reports whether birth is at least MinAge years before now.
func IsAdult(birth, now time.Time) bool { return AgeOn(birth, now) >= MinAge }

// SameDate compares two dates ignoring time of day.
func SameDate(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}
