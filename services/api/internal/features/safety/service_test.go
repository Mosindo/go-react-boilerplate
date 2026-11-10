package safety

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateReport(t *testing.T) {
	if d, err := ValidateReport("harassment", "  insults  "); err != nil || d != "insults" {
		t.Fatalf("got %q %v", d, err)
	}
	if _, err := ValidateReport("dislike", ""); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("expected invalid reason, got %v", err)
	}
	if _, err := ValidateReport("other", strings.Repeat("x", 1001)); !errors.Is(err, ErrDetailsLong) {
		t.Fatalf("expected details too long, got %v", err)
	}
}
