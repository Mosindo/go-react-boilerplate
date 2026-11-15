package textx

import "testing"

func TestClean(t *testing.T) {
	cases := []struct {
		in        string
		multiline bool
		want      string
	}{
		{"  hello   world ", false, "hello world"},
		{"a\nb", false, "a b"},
		{"a\n\n\n\nb", true, "a\n\nb"},
		{"evil‮text​", false, "eviltext"},
		{"tab\tsep\x00", false, "tab sep"},
		{"  line one  \n  line two ", true, "line one\nline two"},
	}
	for _, c := range cases {
		if got := Clean(c.in, c.multiline); got != c.want {
			t.Errorf("Clean(%q, %v) = %q, want %q", c.in, c.multiline, got, c.want)
		}
	}
}
