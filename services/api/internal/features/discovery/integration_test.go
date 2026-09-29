package discovery

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var zone atomic.Int64

type person struct {
	id, token string
	lat, lon  float64
}

type opts struct {
	gender       string
	age          int
	interestedIn []string
	ageMin       int
	ageMax       int
	maxDist      int
	dLat         float64 // offset in degrees from the zone origin
	discoverable bool
	photo        bool
	location     bool
	showAge      bool
	showDistance bool
	interests    []string
}

func def() opts {
	return opts{gender: "woman", age: 30, interestedIn: []string{"woman", "man", "non_binary"}, ageMin: 18, ageMax: 99,
		maxDist: 50, discoverable: true, photo: true, location: true, showAge: true, showDistance: true}
}

// newZone returns an origin far away from any other test zone so unrelated users never interfere.
func newZone() (lat, lon float64) {
	n := zone.Add(1) + time.Now().UnixNano()%37
	return -80 + float64(n%150)*1.0, 100 + float64(n%70)*0.5
}

func seed(t *testing.T, pool *pgxpool.Pool, zlat, zlon float64, o opts) person {
	t.Helper()
	bd := time.Now().UTC().AddDate(-o.age, 0, -30).Format("2006-01-02")
	id, token := testutil.User(t, pool, "disc", bd)
	ctx := context.Background()
	lat, lon := zlat+o.dLat, zlon
	var latArg, lonArg any
	if o.location {
		latArg, lonArg = lat, lon
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profiles (user_id, first_name, gender, bio, city, latitude, longitude, show_age, show_distance, discoverable)
		VALUES ($1, 'Name', $2, 'bio', 'city', $3, $4, $5, $6, $7)`,
		id, o.gender, latArg, lonArg, o.showAge, o.showDistance, o.discoverable); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO preferences (user_id, interested_in, age_min, age_max, max_distance_km)
		VALUES ($1, $2, $3, $4, $5)`, id, o.interestedIn, o.ageMin, o.ageMax, o.maxDist); err != nil {
		t.Fatal(err)
	}
	if o.photo {
		if _, err := pool.Exec(ctx, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes)
			VALUES ($1, $2, 0, 10, 10, 10)`, id, "k-"+id); err != nil {
			t.Fatal(err)
		}
	}
	for _, slug := range o.interests {
		if _, err := pool.Exec(ctx, `INSERT INTO user_interests (user_id, interest_id)
			SELECT $1, id FROM interests WHERE slug = $2`, id, slug); err != nil {
			t.Fatal(err)
		}
	}
	return person{id: id, token: token, lat: lat, lon: lon}
}

type discoverResp struct {
	Profiles []PublicProfile `json:"profiles"`
}

func setup(t *testing.T) (*pgxpool.Pool, *gin.Engine) {
	pool := testutil.Pool(t)
	d := testutil.Deps(t, nil, nil)
	r := testutil.Router(func(r gin.IRouter) { RegisterRoutes(r, pool, d) })
	return pool, r
}

func discover(t *testing.T, r *gin.Engine, who person, query string) discoverResp {
	t.Helper()
	res := testutil.Do(t, r, "GET", "/discover"+query, nil, who.token)
	if res.Status != 200 {
		t.Fatalf("discover status %d: %s", res.Status, res.Body)
	}
	var out discoverResp
	res.JSON(t, &out)
	return out
}

func ids(r discoverResp) map[string]PublicProfile {
	m := map[string]PublicProfile{}
	for _, p := range r.Profiles {
		m[p.UserID] = p
	}
	return m
}

func TestDiscoverFilters(t *testing.T) {
	pool, r := setup(t)
	zlat, zlon := newZone()
	me := seed(t, pool, zlat, zlon, func() opts {
		o := def()
		o.gender = "man"
		o.interestedIn = []string{"woman"}
		o.ageMin = 25
		o.ageMax = 35
		o.maxDist = 50
		return o
	}())

	good := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.05; o.interests = []string{"travel"}; return o }())
	wrongGender := seed(t, pool, zlat, zlon, func() opts { o := def(); o.gender = "man"; return o }())
	theyDontLikeMe := seed(t, pool, zlat, zlon, func() opts { o := def(); o.interestedIn = []string{"woman"}; return o }())
	tooOld := seed(t, pool, zlat, zlon, func() opts { o := def(); o.age = 40; return o }())
	tooYoung := seed(t, pool, zlat, zlon, func() opts { o := def(); o.age = 22; return o }())
	theirAgeRange := seed(t, pool, zlat, zlon, func() opts { o := def(); o.ageMin = 18; o.ageMax = 29; return o }()) // I am 30
	tooFar := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.9; return o }())                        // ~100 km
	theirMaxDist := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.2; o.maxDist = 10; return o }())  // ~22 km, they accept 10
	hidden := seed(t, pool, zlat, zlon, func() opts { o := def(); o.discoverable = false; return o }())
	noPhoto := seed(t, pool, zlat, zlon, func() opts { o := def(); o.photo = false; return o }())
	noLoc := seed(t, pool, zlat, zlon, func() opts { o := def(); o.location = false; return o }())
	blockedByMe := seed(t, pool, zlat, zlon, def())
	blockedMe := seed(t, pool, zlat, zlon, def())
	swiped := seed(t, pool, zlat, zlon, def())
	matched := seed(t, pool, zlat, zlon, def())

	ctx := context.Background()
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)`, me.id, blockedByMe.id)
	mustExec(`INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1, $2)`, blockedMe.id, me.id)
	mustExec(`INSERT INTO swipes (swiper_id, target_id, action) VALUES ($1, $2, 'pass')`, me.id, swiped.id)
	a, b := me.id, matched.id
	if a > b {
		a, b = b, a
	}
	mustExec(`INSERT INTO matches (user_a, user_b) VALUES ($1, $2)`, a, b)

	got := ids(discover(t, r, me, "?limit=20"))
	if _, ok := got[good.id]; !ok {
		t.Fatalf("expected good candidate, got %v", got)
	}
	for name, p := range map[string]person{
		"wrongGender": wrongGender, "theyDontLikeMe": theyDontLikeMe, "tooOld": tooOld, "tooYoung": tooYoung,
		"theirAgeRange": theirAgeRange, "tooFar": tooFar, "theirMaxDist": theirMaxDist, "hidden": hidden,
		"noPhoto": noPhoto, "noLoc": noLoc, "blockedByMe": blockedByMe, "blockedMe": blockedMe,
		"swiped": swiped, "matched": matched, "self": me,
	} {
		if _, ok := got[p.id]; ok {
			t.Errorf("%s must not be discoverable", name)
		}
	}
	// The mirror check: the good candidate sees me too (compatibility is symmetric here) except the
	// candidate is a woman interested in all genders aged 30 and I am a man aged 30 within range.
	back := ids(discover(t, r, good, "?limit=20"))
	if _, ok := back[me.id]; !ok {
		t.Errorf("good candidate should see me: %v", back)
	}
}

func TestDiscoverCardShape(t *testing.T) {
	pool, r := setup(t)
	zlat, zlon := newZone()
	me := seed(t, pool, zlat, zlon, def())
	near := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.03; o.interests = []string{"music", "art"}; return o }()) // ~3.3 km
	far := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.22; o.showAge = false; o.showDistance = false; return o }())

	got := ids(discover(t, r, me, ""))
	n, f := got[near.id], got[far.id]
	if n.Age == nil || *n.Age != 30 {
		t.Errorf("age = %v", n.Age)
	}
	if n.DistanceKm == nil || *n.DistanceKm != 4 {
		t.Errorf("near distance = %v, want 4", n.DistanceKm)
	}
	if len(n.Interests) != 2 || n.Interests[0] != "art" || len(n.Photos) != 1 || n.Photos[0].URL == "" || n.Photos[0].Position != 0 {
		t.Errorf("interests/photos: %+v", n)
	}
	if f.UserID == "" {
		t.Fatal("far candidate missing")
	}
	if f.Age != nil || f.DistanceKm != nil {
		t.Errorf("hidden age/distance leaked: %+v", f)
	}
	res := testutil.Do(t, r, "GET", "/discover", nil, me.token)
	if s := string(res.Body); contains(s, "latitude") || contains(s, "longitude") || contains(s, "email") || contains(s, "@test.local") {
		t.Errorf("response leaks private fields: %s", s)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestDiscoverRankingAndLimit(t *testing.T) {
	pool, r := setup(t)
	zlat, zlon := newZone()
	me := seed(t, pool, zlat, zlon, func() opts { o := def(); o.interests = []string{"travel", "music"}; return o }())
	farShared := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.3; o.interests = []string{"travel", "music"}; return o }())
	nearNone := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.01; return o }())
	midOne := seed(t, pool, zlat, zlon, func() opts { o := def(); o.dLat = 0.1; o.interests = []string{"travel"}; return o }())

	res := discover(t, r, me, "?limit=20")
	if len(res.Profiles) != 3 {
		t.Fatalf("want 3 profiles, got %d", len(res.Profiles))
	}
	want := []string{farShared.id, midOne.id, nearNone.id}
	for i, p := range res.Profiles {
		if p.UserID != want[i] {
			t.Errorf("rank %d = %s, want %s", i, p.UserID, want[i])
		}
	}
	if got := discover(t, r, me, "?limit=2"); len(got.Profiles) != 2 || got.Profiles[0].UserID != farShared.id {
		t.Errorf("limit=2 wrong: %+v", got)
	}
	for _, q := range []string{"?limit=0", "?limit=21", "?limit=x"} {
		if res := testutil.Do(t, r, "GET", "/discover"+q, nil, me.token); res.Status != 400 {
			t.Errorf("%s status %d", q, res.Status)
		}
	}
}

func TestDiscoverViewerMustBeComplete(t *testing.T) {
	pool, r := setup(t)
	zlat, zlon := newZone()
	noPhoto := seed(t, pool, zlat, zlon, func() opts { o := def(); o.photo = false; return o }())
	noLoc := seed(t, pool, zlat, zlon, func() opts { o := def(); o.location = false; return o }())
	id, token := testutil.User(t, pool, "bare", "1990-01-01")
	_ = id
	for name, tok := range map[string]string{"noPhoto": noPhoto.token, "noLoc": noLoc.token, "noProfile": token} {
		if res := testutil.Do(t, r, "GET", "/discover", nil, tok); res.Status != 422 {
			t.Errorf("%s: status %d, want 422", name, res.Status)
		}
	}
	if res := testutil.Do(t, r, "GET", "/discover", nil, ""); res.Status != 401 {
		t.Errorf("unauthenticated status %d", res.Status)
	}
}
