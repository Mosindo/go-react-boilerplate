// Package textx normalises user-provided text before validation and storage.
package textx

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Clean trims the value, removes control characters (except newlines when
// multiline is true), collapses runs of spaces and limits blank lines.
func Clean(value string, multiline bool) string {
	var b strings.Builder
	b.Grow(len(value))
	newlines := 0
	lastSpace := false
	for _, r := range value {
		switch {
		case r == '\n' || r == '\r':
			if !multiline {
				if !lastSpace {
					b.WriteRune(' ')
					lastSpace = true
				}
				continue
			}
			if r == '\r' {
				continue
			}
			newlines++
			if newlines <= 2 {
				b.WriteRune('\n')
			}
			lastSpace = false
		case r == '\t' || r == ' ' || unicode.IsSpace(r):
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
		case unicode.IsControl(r) || r == utf8.RuneError || isInvisibleFormat(r):
			continue
		default:
			b.WriteRune(r)
			newlines = 0
			lastSpace = false
		}
	}
	lines := strings.Split(b.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// Len counts runes.
func Len(value string) int { return utf8.RuneCountInString(value) }

// isInvisibleFormat drops bidi overrides and zero-width characters often used
// to spoof or hide content.
func isInvisibleFormat(r rune) bool {
	switch {
	case r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069, r == 0x200B, r == 0xFEFF:
		return true
	}
	return false
}
