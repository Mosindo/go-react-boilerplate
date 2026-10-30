package safety

import (
	"context"
	"errors"
	"strings"
	"testing"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type fakeRepo struct {
	reports int
	blocks  int
}

func (f *fakeRepo) Block(context.Context, string, string) ([]string, error) {
	f.blocks++
	return nil, nil
}
func (f *fakeRepo) Unblock(context.Context, string, string) error { return nil }
func (f *fakeRepo) ListBlocked(context.Context, string) ([]BlockedUser, error) {
	return nil, nil
}
func (f *fakeRepo) Report(context.Context, string, string, string, string) error {
	f.reports++
	return nil
}

type noEvents struct{}

func (noEvents) Publish(string, realtime.Event) {}

func TestReportValidation(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, noEvents{})
	ctx := context.Background()
	me := "11111111-1111-1111-1111-111111111111"
	other := "22222222-2222-2222-2222-222222222222"

	for name, req := range map[string]ReportRequest{
		"self":       {UserID: me, Reason: "spam"},
		"bad reason": {UserID: other, Reason: "dislike"},
		"long":       {UserID: other, Reason: "spam", Details: strings.Repeat("x", 1001)},
	} {
		err := svc.Report(ctx, me, req)
		if _, ok := platformerrors.AsValidation(err); !ok {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
	if repo.reports != 0 {
		t.Fatal("invalid reports must not reach the repository")
	}

	if err := svc.Report(ctx, me, ReportRequest{UserID: other, Reason: "harassment", Block: true}); err != nil {
		t.Fatalf("valid report: %v", err)
	}
	if repo.reports != 1 || repo.blocks != 1 {
		t.Fatalf("expected report and block, got reports=%d blocks=%d", repo.reports, repo.blocks)
	}
}

func TestCannotBlockSelf(t *testing.T) {
	svc := NewService(&fakeRepo{}, noEvents{})
	id := "11111111-1111-1111-1111-111111111111"
	if err := svc.Block(context.Background(), id, id); err == nil || errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
