package safety

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateReport(t *testing.T) {
	reason, details, err := ValidateReport(" Harassment ", "  insults  ")
	if err != nil || reason != "harassment" || details != "insults" {
		t.Fatalf("got %q %q %v", reason, details, err)
	}
	if _, _, err := ValidateReport("dislike", ""); !errors.Is(err, ErrInvalidReason) {
		t.Fatal("unknown reason must be rejected")
	}
	if _, _, err := ValidateReport("spam", strings.Repeat("x", 1001)); !errors.Is(err, ErrDetailsTooLong) {
		t.Fatal("long details must be rejected")
	}
}
