package matching

import (
	"testing"
	"time"
)

func TestBuildDiscoverQueryWithLocation(t *testing.T) {
	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	lat, lng := 45.76, 4.84
	v := viewerContext{
		OrganizationID: "org", Gender: "woman", Birthdate: time.Date(1996, 6, 15, 0, 0, 0, 0, time.UTC),
		Latitude: &lat, Longitude: &lng, InterestedIn: []string{"man"}, MinAge: 25, MaxAge: 35, MaxDistanceKm: 50,
	}
	q := buildDiscoverQuery("me", v, []string{}, 10, now)
	if q.Age != 30 || q.Gender != "woman" || q.Limit != 10 {
		t.Fatalf("unexpected query: %+v", q)
	}
	if q.DeltaLat == nil || *q.DeltaLat < 0.45 || q.DeltaLng == nil || *q.DeltaLng < *q.DeltaLat {
		t.Fatalf("bounding box should cover 50 km: %+v", q)
	}
	if q.Latest.After(now.AddDate(-25, 0, 0)) || q.Earliest.Before(now.AddDate(-36, 0, 0)) {
		t.Fatalf("birthdate bounds out of range: %v %v", q.Earliest, q.Latest)
	}
}

func TestBuildDiscoverQueryWithoutLocation(t *testing.T) {
	v := viewerContext{Gender: "man", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), InterestedIn: []string{"woman"}, MinAge: 18, MaxAge: 99, MaxDistanceKm: 50}
	q := buildDiscoverQuery("me", v, nil, 5, time.Now())
	if q.Latitude != nil || q.DeltaLat != nil || q.DeltaLng != nil {
		t.Fatal("no location means no distance filter")
	}
}

func TestOrderedPair(t *testing.T) {
	a, b := orderedPair("b", "a")
	c, d := orderedPair("a", "b")
	if a != c || b != d || a != "a" {
		t.Fatal("pair order must be canonical")
	}
}
