package safety

import (
	"strings"
	"testing"
)

func TestValidateReport(t *testing.T) {
	reason, details, err := ValidateReport(" harassment ", "  insultes répétées ")
	if err != nil || reason != "harassment" || details != "insultes répétées" {
		t.Fatalf("unexpected result %q %q %v", reason, details, err)
	}
	if _, _, err := ValidateReport("i-dont-like-them", ""); err == nil {
		t.Fatal("unknown reason must be rejected")
	}
	if _, _, err := ValidateReport("spam", strings.Repeat("x", 1001)); err == nil {
		t.Fatal("details over 1000 chars must be rejected")
	}
}
