package chat

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeBody(t *testing.T) {
	cases := []struct {
		name, in, want string
		err            error
	}{
		{"trims", "  salut  ", "salut", nil},
		{"keeps line breaks", "a\r\nb", "a\nb", nil},
		{"drops control chars", "he\x00llo\x07", "hello", nil},
		{"empty", "", "", ErrEmptyMessage},
		{"blank", " \n\t ", "", ErrEmptyMessage},
		{"only control", "\x00\x01", "", ErrEmptyMessage},
		{"limit", strings.Repeat("é", MaxMessageRunes), strings.Repeat("é", MaxMessageRunes), nil},
		{"over limit", strings.Repeat("é", MaxMessageRunes+1), "", ErrMessageTooLong},
	}
	for _, c := range cases {
		got, err := normalizeBody(c.in)
		if !errors.Is(err, c.err) && !(c.err == nil && err == nil) {
			t.Errorf("%s: error %v, want %v", c.name, err, c.err)
		}
		if err == nil && got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPreviewTruncatesOnRuneBoundaries(t *testing.T) {
	if got := preview("short   message\nwith  spaces"); got != "short message with spaces" {
		t.Errorf("unexpected preview %q", got)
	}
	long := preview(strings.Repeat("é", 200))
	if r := []rune(long); len(r) != notificationPreviewRunes || r[len(r)-1] != '…' {
		t.Errorf("preview should be %d runes ending with an ellipsis, got %d", notificationPreviewRunes, len(r))
	}
}
