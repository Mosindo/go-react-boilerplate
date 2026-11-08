// Package validate contains small input rules shared by several features.
package validate

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MinAge = 18
	MaxAge = 99
)

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func ValidEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && strings.Contains(email[strings.LastIndex(email, "@"):], ".")
}

// ParseBirthDate parses YYYY-MM-DD.
func ParseBirthDate(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(raw))
}

// Age returns the full years elapsed between birth and now (both interpreted in UTC dates).
func Age(birth, now time.Time) int {
	birth, now = birth.UTC(), now.UTC()
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// CleanText trims, collapses nothing else, and enforces a rune limit. It rejects control characters
// except newlines when allowNewlines is set.
func CleanText(s string, maxRunes int, allowNewlines bool) (string, bool) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxRunes || !utf8.ValidString(s) {
		return "", false
	}
	for _, r := range s {
		if r == '\n' && allowNewlines {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}
	return s, true
}
