package chat

import "testing"

func TestCleanMessage(t *testing.T) {
	cases := map[string]string{
		"  hello  ":     "hello",
		"a\x00b\x07c":   "abc",
		"line1\nline2":  "line1\nline2",
		"\t\n  ":        "",
		"emoji 😀 stays": "emoji 😀 stays",
		"‮ reversed":    "‮ reversed",
	}
	for in, want := range cases {
		if got := cleanMessage(in); got != want {
			t.Fatalf("cleanMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClamp(t *testing.T) {
	if clamp(0, 20, 50) != 20 || clamp(-1, 20, 50) != 20 || clamp(99, 20, 50) != 50 || clamp(7, 20, 50) != 7 {
		t.Fatal("clamp broken")
	}
}
