package chat

import "testing"

func TestCleanBody(t *testing.T) {
	cases := map[string]string{
		"  hello  ":            "hello",
		"a\r\nb":               "a\nb",
		"bell\x07 and nul\x00": "bell and nul",
		"   \n\t ":             "",
		"emoji 😀 ok":           "emoji 😀 ok",
	}
	for in, want := range cases {
		if got := cleanBody(in); got != want {
			t.Errorf("cleanBody(%q) = %q, want %q", in, got, want)
		}
	}
}
