// Package validate contains domain-agnostic input rules enforced server-side.
package validate

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MinAge = 18
	MaxAge = 120
)

var (
	ErrInvalidBirthDate = errors.New("invalid birth date")
	ErrUnderage         = errors.New("must be at least 18 years old")
)

// AgeAt returns the age in full years of someone born on birth, at time now.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// ParseBirthDate parses YYYY-MM-DD and enforces the access-age rule.
func ParseBirthDate(raw string, now time.Time) (time.Time, error) {
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthDate
	}
	age := AgeAt(birth, now)
	if birth.After(now) || age > MaxAge {
		return time.Time{}, ErrInvalidBirthDate
	}
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	return birth, nil
}

// CleanText trims, collapses control characters and enforces a rune limit.
// It returns false when the text exceeds max runes or contains invalid UTF-8.
func CleanText(s string, max int) (string, bool) {
	s = strings.TrimSpace(s)
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > max {
		return "", false
	}
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return cleaned, true
}

// NormalizeEmail lowercases and trims an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
