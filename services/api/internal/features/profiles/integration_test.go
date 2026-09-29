package profiles_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	pool *pgxpool.Pool
	d    deps.Common
	r    *gin.Engine
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := testutil.Pool(t)
	d := testutil.Deps(t, nil, nil)
	r := testutil.Router(func(r gin.IRouter) { profiles.RegisterRoutes(r, pool, d) })
	return fixture{pool: pool, d: d, r: r}
}

func body(name, gender string, extra map[string]any) map[string]any {
	m := map[string]any{"firstName": name, "gender": gender, "bio": "hi", "city": "Lyon", "interests": []string{"music", "art"}}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func (f fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestProfileCRUD(t *testing.T) {
	f := setup(t)
	id, tok := testutil.User(t, f.pool, "prof", "1995-03-10")

	if s := testutil.Do(t, f.r, "GET", "/me/profile", nil, tok).Status; s != 404 {
		t.Fatalf("get before create want 404 got %d", s)
	}
	if s := testutil.Do(t, f.r, "GET", "/me/preferences", nil, tok).Status; s != 404 {
		t.Fatalf("prefs before create want 404 got %d", s)
	}
	if s := testutil.Do(t, f.r, "PUT", "/me/location", map[string]float64{"latitude": 1, "longitude": 1}, tok).Status; s != 404 {
		t.Fatalf("location before profile want 404 got %d", s)
	}
	if s := testutil.Do(t, f.r, "GET", "/me/profile", nil, "").Status; s != 401 {
		t.Fatalf("unauth want 401 got %d", s)
	}

	var il struct {
		Interests []struct{ Slug, Label string } `json:"interests"`
	}
	testutil.Do(t, f.r, "GET", "/interests", nil, tok).JSON(t, &il)
	if len(il.Interests) < 20 {
		t.Fatalf("interests: %v", il)
	}

	res := testutil.Do(t, f.r, "PUT", "/me/profile", body("  anne-marie ", "woman", map[string]any{"interests": []string{"Music", "music", "art"}}), tok)
	if res.Status != 200 {
		t.Fatalf("put: %d %s", res.Status, res.Body)
	}
	var mine struct {
		UserID       string   `json:"userId"`
		FirstName    string   `json:"firstName"`
		Age          *int     `json:"age"`
		Interests    []string `json:"interests"`
		Photos       []any    `json:"photos"`
		DistanceKm   *int     `json:"distanceKm"`
		HasLocation  bool     `json:"hasLocation"`
		ShowAge      bool     `json:"showAge"`
		ShowDistance bool     `json:"showDistance"`
		Discoverable bool     `json:"discoverable"`
		IsComplete   bool     `json:"isComplete"`
	}
	res.JSON(t, &mine)
	if mine.UserID != id || mine.FirstName != "anne-marie" || mine.Age == nil || *mine.Age < 30 ||
		len(mine.Interests) != 2 || mine.DistanceKm != nil || mine.HasLocation || mine.IsComplete ||
		!mine.ShowAge || !mine.ShowDistance || !mine.Discoverable || mine.Photos == nil {
		t.Fatalf("unexpected: %+v (%s)", mine, res.Body)
	}

	var prefs profiles.Preferences
	res = testutil.Do(t, f.r, "GET", "/me/preferences", nil, tok)
	res.JSON(t, &prefs)
	if res.Status != 200 || len(prefs.InterestedIn) != 3 || prefs.AgeMin != 18 || prefs.AgeMax != 99 || prefs.MaxDistanceKm != 50 {
		t.Fatalf("default prefs: %d %s", res.Status, res.Body)
	}

	// Update: change fields, replace interests, flip flags; preferences must not be reset.
	f.exec(t, `UPDATE preferences SET age_min=25 WHERE user_id=$1`, id)
	res = testutil.Do(t, f.r, "PUT", "/me/profile", body("Anne", "non_binary", map[string]any{
		"interests": []string{"yoga"}, "showAge": false, "discoverable": false,
	}), tok)
	res.JSON(t, &mine)
	if res.Status != 200 || len(mine.Interests) != 1 || mine.Interests[0] != "yoga" || mine.ShowAge || mine.Discoverable || !mine.ShowDistance {
		t.Fatalf("update: %d %s", res.Status, res.Body)
	}
	testutil.Do(t, f.r, "GET", "/me/preferences", nil, tok).JSON(t, &prefs)
	if prefs.AgeMin != 25 {
		t.Fatal("preferences reset by profile update")
	}
	// omitted flags keep their values
	testutil.Do(t, f.r, "PUT", "/me/profile", body("Anne", "woman", nil), tok).JSON(t, &mine)
	if mine.ShowAge || mine.Discoverable {
		t.Fatal("omitted flags must keep stored values")
	}
	// Own age is always visible even with showAge=false.
	if mine.Age == nil {
		t.Fatal("own age must be set")
	}
}

func TestProfileValidationErrors(t *testing.T) {
	f := setup(t)
	_, tok := testutil.User(t, f.pool, "val", "1995-03-10")
	eleven := []string{"music", "art", "yoga", "travel", "books", "wine", "pets", "food", "nature", "coffee", "sports"}
	cases := map[string]map[string]any{
		"empty name":      body("", "woman", nil),
		"digits":          body("Bob2", "man", nil),
		"long name":       body(strings.Repeat("a", 41), "man", nil),
		"bad gender":      body("Bob", "robot", nil),
		"long bio":        body("Bob", "man", map[string]any{"bio": strings.Repeat("x", 501)}),
		"long city":       body("Bob", "man", map[string]any{"city": strings.Repeat("x", 81)}),
		"unknown interst": body("Bob", "man", map[string]any{"interests": []string{"nope"}}),
		"11 interests":    body("Bob", "man", map[string]any{"interests": eleven}),
	}
	for name, b := range cases {
		if s := testutil.Do(t, f.r, "PUT", "/me/profile", b, tok).Status; s != 400 {
			t.Errorf("%s: want 400 got %d", name, s)
		}
	}
	if s := testutil.Do(t, f.r, "GET", "/me/profile", nil, tok).Status; s != 404 {
		t.Fatalf("failed writes must not create a profile (got %d)", s)
	}
	// Exactly 10 works.
	if s := testutil.Do(t, f.r, "PUT", "/me/profile", body("Bob", "man", map[string]any{"interests": eleven[:10]}), tok).Status; s != 200 {
		t.Fatalf("10 interests want 200 got %d", s)
	}
	// Failed interest update leaves previous interests untouched (transaction).
	testutil.Do(t, f.r, "PUT", "/me/profile", body("Bob", "man", map[string]any{"interests": []string{"nope"}}), tok)
	var mine struct {
		Interests []string `json:"interests"`
	}
	testutil.Do(t, f.r, "GET", "/me/profile", nil, tok).JSON(t, &mine)
	if len(mine.Interests) != 10 {
		t.Fatalf("interests changed by failed update: %v", mine.Interests)
	}
}

func TestPreferencesEndpoints(t *testing.T) {
	f := setup(t)
	_, tok := testutil.User(t, f.pool, "pref", "1995-03-10")
	testutil.Do(t, f.r, "PUT", "/me/profile", body("Pat", "woman", nil), tok)

	ok := map[string]any{"interestedIn": []string{"man"}, "ageMin": 25, "ageMax": 40, "maxDistanceKm": 120}
	var p profiles.Preferences
	res := testutil.Do(t, f.r, "PUT", "/me/preferences", ok, tok)
	res.JSON(t, &p)
	if res.Status != 200 || p.AgeMin != 25 || p.AgeMax != 40 || p.MaxDistanceKm != 120 || len(p.InterestedIn) != 1 || p.InterestedIn[0] != "man" {
		t.Fatalf("%d %s", res.Status, res.Body)
	}
	testutil.Do(t, f.r, "GET", "/me/preferences", nil, tok).JSON(t, &p)
	if p.AgeMin != 25 {
		t.Fatal("not persisted")
	}
	bads := []map[string]any{
		{"interestedIn": []string{}, "ageMin": 25, "ageMax": 40, "maxDistanceKm": 10},
		{"interestedIn": []string{"x"}, "ageMin": 25, "ageMax": 40, "maxDistanceKm": 10},
		{"interestedIn": []string{"man"}, "ageMin": 17, "ageMax": 40, "maxDistanceKm": 10},
		{"interestedIn": []string{"man"}, "ageMin": 41, "ageMax": 40, "maxDistanceKm": 10},
		{"interestedIn": []string{"man"}, "ageMin": 20, "ageMax": 100, "maxDistanceKm": 10},
		{"interestedIn": []string{"man"}, "ageMin": 20, "ageMax": 40, "maxDistanceKm": 501},
		{"interestedIn": []string{"man"}, "ageMin": 20, "ageMax": 40, "maxDistanceKm": 0},
	}
	for i, b := range bads {
		if s := testutil.Do(t, f.r, "PUT", "/me/preferences", b, tok).Status; s != 400 {
			t.Errorf("bad %d want 400 got %d", i, s)
		}
	}
}

func TestLocationRounding(t *testing.T) {
	f := setup(t)
	id, tok := testutil.User(t, f.pool, "loc", "1995-03-10")
	testutil.Do(t, f.r, "PUT", "/me/profile", body("Lou", "woman", nil), tok)

	if s := testutil.Do(t, f.r, "PUT", "/me/location", map[string]float64{"latitude": 45.767299, "longitude": 4.834999}, tok).Status; s != 204 {
		t.Fatalf("want 204 got %d", s)
	}
	var lat, lng float64
	var updated bool
	if err := f.pool.QueryRow(context.Background(), `SELECT latitude, longitude, location_updated_at IS NOT NULL FROM profiles WHERE user_id=$1`, id).Scan(&lat, &lng, &updated); err != nil {
		t.Fatal(err)
	}
	if lat != 45.77 || lng != 4.83 || !updated {
		t.Fatalf("stored %v %v %v", lat, lng, updated)
	}
	for _, b := range []map[string]float64{{"latitude": 91, "longitude": 0}, {"latitude": 0, "longitude": -181}, {"latitude": 10}} {
		if s := testutil.Do(t, f.r, "PUT", "/me/location", b, tok).Status; s != 400 {
			t.Errorf("%v want 400 got %d", b, s)
		}
	}
	var mine struct {
		HasLocation bool `json:"hasLocation"`
		IsComplete  bool `json:"isComplete"`
	}
	f.exec(t, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,0,1,1,1)`, id, "photos/"+id+"/a.jpg")
	testutil.Do(t, f.r, "GET", "/me/profile", nil, tok).JSON(t, &mine)
	if !mine.HasLocation || !mine.IsComplete {
		t.Fatalf("%+v", mine)
	}
}

type pub struct {
	UserID     string   `json:"userId"`
	FirstName  string   `json:"firstName"`
	Age        *int     `json:"age"`
	DistanceKm *int     `json:"distanceKm"`
	Interests  []string `json:"interests"`
	Photos     []struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Position int    `json:"position"`
	} `json:"photos"`
}

func (f fixture) mkProfile(t *testing.T, id, name string, lat, lng float64, extra ...string) {
	t.Helper()
	f.exec(t, `INSERT INTO profiles (user_id, first_name, gender, latitude, longitude, location_updated_at) VALUES ($1,$2,'woman',$3,$4,NOW())`, id, name, lat, lng)
	f.exec(t, `INSERT INTO user_interests (user_id, interest_id) SELECT $1, id FROM interests WHERE slug IN ('music','art')`, id)
	f.exec(t, `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,1,1,1,1),($1,$3,0,1,1,1)`, id, "photos/"+id+"/b.jpg", "photos/"+id+"/a.jpg")
}

func TestPublicProfileVisibility(t *testing.T) {
	f := setup(t)
	viewer, vTok := testutil.User(t, f.pool, "viewer", "1990-01-01")
	target, _ := testutil.User(t, f.pool, "target", "1996-01-01")
	f.mkProfile(t, viewer, "Vic", 48.85, 2.35)
	f.mkProfile(t, target, "Tara", 48.90, 2.35) // ~5.5 km north

	get := func(id string) (int, pub, []byte) {
		res := testutil.Do(t, f.r, "GET", "/profiles/"+id, nil, vTok)
		var p pub
		if res.Status == 200 {
			res.JSON(t, &p)
		}
		return res.Status, p, res.Body
	}

	s, p, raw := get(target)
	if s != 200 || p.FirstName != "Tara" || p.Age == nil || *p.Age < 29 {
		t.Fatalf("discoverable: %d %s", s, raw)
	}
	if p.DistanceKm == nil || *p.DistanceKm != 6 {
		t.Fatalf("distance want 6 got %v (%s)", p.DistanceKm, raw)
	}
	if len(p.Interests) != 2 || len(p.Photos) != 2 || p.Photos[0].Position != 0 || !strings.HasPrefix(p.Photos[0].URL, "/photos/"+p.Photos[0].ID+"/file?exp=") {
		t.Fatalf("interests/photos: %s", raw)
	}
	if !f.d.Signer.Verify(p.Photos[0].ID, mustExp(t, p.Photos[0].URL), mustSig(p.Photos[0].URL)) {
		t.Fatal("photo URL signature invalid")
	}

	// Hidden age and distance.
	f.exec(t, `UPDATE profiles SET show_age=FALSE, show_distance=FALSE WHERE user_id=$1`, target)
	s, p, raw = get(target)
	if s != 200 || p.Age != nil || p.DistanceKm != nil || !strings.Contains(string(raw), `"age":null`) || !strings.Contains(string(raw), `"distanceKm":null`) {
		t.Fatalf("hidden: %d %s", s, raw)
	}
	f.exec(t, `UPDATE profiles SET show_age=TRUE, show_distance=TRUE WHERE user_id=$1`, target)

	// Viewer without location -> no distance.
	f.exec(t, `UPDATE profiles SET latitude=NULL, longitude=NULL WHERE user_id=$1`, viewer)
	if _, p, raw = get(target); p.DistanceKm != nil {
		t.Fatalf("distance without viewer location: %s", raw)
	}
	f.exec(t, `UPDATE profiles SET latitude=48.85, longitude=2.35 WHERE user_id=$1`, viewer)

	// Not discoverable -> 404, unless matched.
	f.exec(t, `UPDATE profiles SET discoverable=FALSE WHERE user_id=$1`, target)
	if s, _, _ = get(target); s != 404 {
		t.Fatalf("hidden profile want 404 got %d", s)
	}
	a, b := viewer, target
	if b < a {
		a, b = b, a
	}
	f.exec(t, `INSERT INTO matches (user_a, user_b) VALUES ($1,$2)`, a, b)
	if s, _, _ = get(target); s != 200 {
		t.Fatalf("matched user must be visible, got %d", s)
	}
	f.exec(t, `DELETE FROM matches WHERE user_a=$1 AND user_b=$2`, a, b)

	// Self is always visible, age shown, no distance.
	f.exec(t, `UPDATE profiles SET discoverable=FALSE, show_age=FALSE WHERE user_id=$1`, viewer)
	if s, p, raw = get(viewer); s != 200 || p.Age == nil || p.DistanceKm != nil {
		t.Fatalf("self: %d %s", s, raw)
	}

	// Blocks in either direction -> 404.
	f.exec(t, `UPDATE profiles SET discoverable=TRUE WHERE user_id=$1`, target)
	f.exec(t, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1,$2)`, target, viewer)
	if s, _, _ = get(target); s != 404 {
		t.Fatalf("blocked by target want 404 got %d", s)
	}
	f.exec(t, `DELETE FROM blocks`+` WHERE blocker_id=$1`, target)
	f.exec(t, `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1,$2)`, viewer, target)
	if s, _, _ = get(target); s != 404 {
		t.Fatalf("viewer blocked target want 404 got %d", s)
	}

	// Unknown id, malformed id, user without profile.
	noProf, _ := testutil.User(t, f.pool, "noprof", "1990-01-01")
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "not-a-uuid", noProf} {
		if s, _, _ = get(id); s != http.StatusNotFound {
			t.Errorf("%s want 404 got %d", id, s)
		}
	}
}
