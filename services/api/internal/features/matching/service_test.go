package matching

import (
	"testing"
	"time"

	"example.com/api/internal/features/profiles"
)

func TestBuildQueryResolvesMutualCriteria(t *testing.T) {
	today := time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)
	v := Viewer{
		UserID:        "u",
		Birthdate:     time.Date(1996, 11, 24, 0, 0, 0, 0, time.UTC),
		MinAge:        25,
		MaxAge:        35,
		MaxDistanceKm: 111,
	}
	q := BuildQuery(v, today, 10)
	if q.ViewerAge != 29 {
		t.Fatalf("expected viewer age 29, got %d", q.ViewerAge)
	}
	if profiles.AgeOn(q.YoungestBirth, today) != 25 || profiles.AgeOn(q.OldestBirthdate, today) != 35 {
		t.Fatalf("birthdate bounds do not match age range: %s %s", q.OldestBirthdate, q.YoungestBirth)
	}
	if q.LatDelta < 1.0 || q.LatDelta > 1.1 {
		t.Fatalf("unexpected latitude prefilter %f", q.LatDelta)
	}
	if q.Limit != 10 || q.OnlyUserID != "" {
		t.Fatalf("unexpected query %#v", q)
	}
}

func TestSummarizeUsesPrimaryPhoto(t *testing.T) {
	card := profiles.PublicProfile{UserID: "b", FirstName: "Lou", Age: 30, Photos: []profiles.Photo{{ID: "p0", Position: 0}, {ID: "p1", Position: 1}}}
	s := summarize(StoredMatch{ID: "m", ConversationID: "c", OtherUserID: "b"}, card)
	if s.User.Photo == nil || s.User.Photo.ID != "p0" || s.User.FirstName != "Lou" {
		t.Fatalf("unexpected summary %#v", s)
	}
}
