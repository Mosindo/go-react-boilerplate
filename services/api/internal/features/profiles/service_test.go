package profiles

import (
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // birthday tomorrow
		{"2008-06-14", 18},
		{"2000-02-29", 26},
		{"1990-12-31", 35},
	}
	for _, c := range cases {
		birth, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeAt(birth, now); got != c.want {
			t.Errorf("AgeAt(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestApproxDistance(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		in   *float64
		want int
	}{{f(0), 1}, {f(0.9), 1}, {f(1.1), 5}, {f(5), 5}, {f(5.01), 10}, {f(43.2), 45}}
	for _, c := range cases {
		got := ApproxDistance(c.in)
		if got == nil || *got != c.want {
			t.Errorf("ApproxDistance(%v) = %v, want %d", *c.in, got, c.want)
		}
	}
	if ApproxDistance(nil) != nil {
		t.Error("nil distance must stay nil")
	}
}

func TestCoarsen(t *testing.T) {
	if got := coarsen(48.85661234); got != 48.86 {
		t.Errorf("coarsen = %v", got)
	}
	if got := coarsen(-2.35219876); got != -2.35 {
		t.Errorf("coarsen = %v", got)
	}
}

func TestCleanText(t *testing.T) {
	if got, err := cleanText("  hello \n", 10, false); err != nil || got != "hello" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := cleanText("a\nb", 10, true); err == nil {
		t.Error("single-line text must reject newlines")
	}
	if _, err := cleanText("a\x00b", 10, false); err == nil {
		t.Error("NUL must be rejected")
	}
	if _, err := cleanText("ééééé", 4, false); err == nil {
		t.Error("limit counts runes, not bytes")
	}
	if _, err := cleanText("\xff\xfe", 10, false); err == nil {
		t.Error("invalid UTF-8 must be rejected")
	}
}

func TestUniqueInterestIDs(t *testing.T) {
	got, err := uniqueInterestIDs([]int{3, 1, 3})
	if err != nil || len(got) != 2 {
		t.Errorf("dedupe failed: %v %v", got, err)
	}
	if _, err := uniqueInterestIDs([]int{0}); err == nil {
		t.Error("id 0 invalid")
	}
	if _, err := uniqueInterestIDs([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}); err == nil {
		t.Error("more than 10 interests must fail")
	}
}
