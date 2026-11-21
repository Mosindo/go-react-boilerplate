package safety

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeReport(t *testing.T) {
	reason, details, err := NormalizeReport(" Harassment ", " insults\x00 ")
	if err != nil || reason != "harassment" || details != "insults" {
		t.Fatalf("unexpected %q %q %v", reason, details, err)
	}
	if _, _, err := NormalizeReport("i-dont-like-them", ""); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("expected invalid reason, got %v", err)
	}
	if _, _, err := NormalizeReport("spam", strings.Repeat("x", 1001)); !errors.Is(err, ErrDetailsLength) {
		t.Fatalf("expected details length error, got %v", err)
	}
}
