package safety

import (
	"context"
	"errors"
	"strings"
	"testing"

	"example.com/api/internal/platform/realtime"
)

func TestServiceRejectsInvalidReports(t *testing.T) {
	s := NewService(nil, realtime.NopPublisher{})
	ctx := context.Background()
	const me = "11111111-1111-1111-1111-111111111111"
	const other = "22222222-2222-2222-2222-222222222222"
	if err := s.Report(ctx, me, me, ReportRequest{Reason: "spam"}); !errors.Is(err, ErrSelf) {
		t.Fatalf("self report: %v", err)
	}
	if err := s.Report(ctx, me, other, ReportRequest{Reason: "bogus"}); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("bad reason: %v", err)
	}
	if err := s.Report(ctx, me, other, ReportRequest{Reason: "spam", Details: strings.Repeat("x", 1001)}); !errors.Is(err, ErrDetailsLong) {
		t.Fatalf("long details: %v", err)
	}
	if err := s.Block(ctx, me, "not-a-uuid"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("bad id: %v", err)
	}
}
