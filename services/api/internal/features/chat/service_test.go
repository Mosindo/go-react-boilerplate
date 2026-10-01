package chat

import "testing"

func TestCleanBody(t *testing.T) {
	for in, want := range map[string]string{
		"  hello  ":           "hello",
		"a\r\nb":              "a\nb",
		"bell\x07 and\x00nul": "bell andnul",
		"tab\there":           "tab\there",
		"\n\n  \t ":           "",
		"émoji 😀":             "émoji 😀",
	} {
		if got := cleanBody(in); got != want {
			t.Errorf("cleanBody(%q) = %q, want %q", in, got, want)
		}
	}
}
