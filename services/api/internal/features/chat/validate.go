package chat

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidMessage = errors.New("message must contain between 1 and 2000 characters")

// NormalizeMessage trims the text, drops control and invisible formatting
// characters (keeping line breaks) and enforces the length limits.
func NormalizeMessage(raw string) (string, error) {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\r' || r == '\t':
			b.WriteRune(' ')
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r) && r != '‍' && r != '️':
			// dropped (zero-width joiner and variation selector are kept for emoji)
		default:
			b.WriteRune(r)
		}
	}
	lines := strings.Split(b.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	body := strings.TrimSpace(strings.Join(lines, "\n"))
	if n := utf8.RuneCountInString(body); n < 1 || n > MaxMessageRunes {
		return "", ErrInvalidMessage
	}
	return body, nil
}

// Preview shortens a body for notifications and conversation lists.
func Preview(body string, max int) string {
	runes := []rune(strings.Join(strings.Fields(body), " "))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max-1]) + "…"
}
