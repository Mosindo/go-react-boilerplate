package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoveryFilters(t *testing.T) {
	env := newTestEnv(t)
	viewer := env.register("viewer")
	env.onboard(viewer, profileSpec{Name: "Viewer", BirthDate: "1990-06-01", Gender: "man", InterestedIn: []string{"woman"}, MinAge: 25, MaxAge: 40, MaxKm: 30, Interests: []string{"music"}})

	young := time.Now().AddDate(-22, 0, 0).Format("2006-01-02")
	mk := func(name string, s profileSpec) *user {
		u := env.register(name)
		s.Name = name
		if s.BirthDate == "" {
			s.BirthDate = "1992-02-02"
		}
		if s.Gender == "" {
			s.Gender = "woman"
		}
		if s.InterestedIn == nil {
			s.InterestedIn = []string{"man"}
		}
		env.onboard(u, s)
		return u
	}

	ok := mk("ok", profileSpec{})
	far := mk("far", profileSpec{Lat: 45.76, Lon: 4.83})                           // Lyon, ~390 km
	tooYoung := mk("tooyoung", profileSpec{BirthDate: young})                      // outside viewer's range
	wantsWomen := mk("wantswomen", profileSpec{InterestedIn: []string{"woman"}})   // not into the viewer
	viewerTooOld := mk("picky", profileSpec{MaxAge: 30})                           // viewer (35) outside her range
	theirRadius := mk("shortradius", profileSpec{Lat: 48.60, Lon: 2.35, MaxKm: 5}) // ~28 km away but only accepts 5 km
	noPhoto := mk("nophoto", profileSpec{Photos: -1})
	hidden := mk("hidden", profileSpec{})
	blocker := mk("blocker", profileSpec{})
	blocked := mk("blocked", profileSpec{})
	sameInterest := mk("shared", profileSpec{Interests: []string{"music"}})

	if res := env.do(http.MethodPatch, "/me/privacy", hidden.Token, map[string]any{"discoverable": false}, hidden.addr); res.Status != http.StatusOK {
		t.Fatalf("privacy: %d %s", res.Status, res.Body)
	}
	if res := env.do(http.MethodPost, "/blocks", blocker.Token, map[string]string{"userId": viewer.ID}, blocker.addr); res.Status != http.StatusNoContent {
		t.Fatalf("block: %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/blocks", viewer.Token, map[string]string{"userId": blocked.ID}, viewer.addr); res.Status != http.StatusNoContent {
		t.Fatalf("block: %d", res.Status)
	}

	ids := env.discoverIDs(viewer)
	for _, excluded := range []*user{far, tooYoung, wantsWomen, viewerTooOld, theirRadius, noPhoto, hidden, blocker, blocked, viewer} {
		if contains(ids, excluded.ID) {
			t.Errorf("%s must not be discoverable", excluded.Email)
		}
	}
	for _, included := range []*user{ok, sameInterest} {
		if !contains(ids, included.ID) {
			t.Errorf("%s should be discoverable", included.Email)
		}
	}
	if len(ids) != 2 || ids[0] != sameInterest.ID {
		t.Errorf("shared interests should rank first, got %v (shared=%s)", ids, sameInterest.ID)
	}

	// Hidden or blocked profiles cannot be opened directly either.
	for _, u := range []*user{hidden, blocker, blocked} {
		if res := env.do(http.MethodGet, "/profiles/"+u.ID, viewer.Token, nil, viewer.addr); res.Status != http.StatusNotFound {
			t.Errorf("profile of %s must be 404, got %d", u.Email, res.Status)
		}
	}
	// ...and cannot be swiped on.
	if res := env.do(http.MethodPost, "/swipes", viewer.Token, map[string]string{"targetId": hidden.ID, "action": "like"}, viewer.addr); res.Status != http.StatusNotFound {
		t.Errorf("swiping a hidden profile: got %d", res.Status)
	}
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("val")
	base := func() map[string]any {
		return map[string]any{"firstName": "Val", "birthDate": "1990-01-01", "gender": "woman", "city": "Paris", "latitude": 48.8566, "longitude": 2.3522}
	}
	put := func(mod func(map[string]any)) int {
		p := base()
		mod(p)
		return env.do(http.MethodPut, "/me/profile", u.Token, p, u.addr).Status
	}

	adult18 := time.Now().AddDate(-18, 0, 1).Format("2006-01-02") // turns 18 tomorrow
	cases := map[string]func(map[string]any){
		"minor":          func(p map[string]any) { p["birthDate"] = adult18 },
		"future":         func(p map[string]any) { p["birthDate"] = "2999-01-01" },
		"bad date":       func(p map[string]any) { p["birthDate"] = "01/01/1990" },
		"bad gender":     func(p map[string]any) { p["gender"] = "robot" },
		"empty name":     func(p map[string]any) { p["firstName"] = "   " },
		"long name":      func(p map[string]any) { p["firstName"] = strings.Repeat("x", 41) },
		"long bio":       func(p map[string]any) { p["bio"] = strings.Repeat("x", 501) },
		"bad latitude":   func(p map[string]any) { p["latitude"] = 123.0 },
		"bad longitude":  func(p map[string]any) { p["longitude"] = -300.0 },
		"bad interest":   func(p map[string]any) { p["interests"] = []string{"nope"} },
		"bad lookingFor": func(p map[string]any) { p["lookingFor"] = "marriage-now" },
		"too many interests": func(p map[string]any) {
			p["interests"] = []string{"music", "travel", "cinema", "series", "reading", "hiking", "running", "fitness", "yoga", "cycling", "football"}
		},
	}
	for name, mod := range cases {
		if got := put(mod); got != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, got)
		}
	}

	if got := put(func(p map[string]any) {
		p["firstName"] = "  Val\x00entine\n "
		p["latitude"] = 48.856789
		p["longitude"] = 2.352211
	}); got != http.StatusOK {
		t.Fatalf("valid profile: %d", got)
	}
	prof := env.do(http.MethodGet, "/me/profile", u.Token, nil, u.addr).Map(t)["profile"].(map[string]any)
	if prof["firstName"] != "Valentine" {
		t.Errorf("name should be sanitised, got %q", prof["firstName"])
	}
	if prof["latitude"] != 48.86 || prof["longitude"] != 2.35 {
		t.Errorf("coordinates must be rounded to ~1km, got %v,%v", prof["latitude"], prof["longitude"])
	}

	// The date of birth is the age gate: it cannot be edited afterwards.
	if got := put(func(p map[string]any) { p["birthDate"] = "1980-01-01" }); got != http.StatusBadRequest {
		t.Errorf("changing birth date: got %d", got)
	}

	prefs := func(m map[string]any) int {
		return env.do(http.MethodPut, "/me/preferences", u.Token, m, u.addr).Status
	}
	for name, m := range map[string]map[string]any{
		"no gender":     {"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 30},
		"under 18":      {"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 40, "maxDistanceKm": 30},
		"inverted":      {"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 30},
		"over 99":       {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 120, "maxDistanceKm": 30},
		"zero distance": {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 0},
		"huge distance": {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 9000},
		"bad gender":    {"interestedIn": []string{"x"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 30},
	} {
		if got := prefs(m); got != http.StatusBadRequest {
			t.Errorf("preferences %s: got %d, want 400", name, got)
		}
	}

	// show_distance=false removes the distance from what others see.
	v := env.register("viewer")
	env.onboard(v, profileSpec{Name: "V", BirthDate: "1990-01-01", Gender: "man", InterestedIn: []string{"woman"}})
	env.do(http.MethodPut, "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 60, "maxDistanceKm": 50}, u.addr)
	env.upload(u.Token, u.addr, gradientJPEG(t, 300, 300), "image/jpeg")
	if card := env.do(http.MethodGet, "/profiles/"+u.ID, v.Token, nil, v.addr).Map(t); card["distanceKm"] == nil {
		t.Fatalf("distance should be visible by default: %v", card)
	}
	env.do(http.MethodPatch, "/me/privacy", u.Token, map[string]any{"showDistance": false}, u.addr)
	if card := env.do(http.MethodGet, "/profiles/"+u.ID, v.Token, nil, v.addr).Map(t); card["distanceKm"] != nil {
		t.Fatalf("distance must be hidden: %v", card)
	}
}

func TestPhotoManagement(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("photo")
	other := env.register("other")

	// Dangerous or invalid uploads are refused.
	for name, tc := range map[string]struct {
		data []byte
		want int
	}{
		"html":      {[]byte("<html><script>alert(1)</script></html>"), http.StatusUnprocessableEntity},
		"svg":       {[]byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`), http.StatusUnprocessableEntity},
		"too small": {gradientJPEG(t, 40, 40), http.StatusUnprocessableEntity},
		"too big":   {append(gradientJPEG(t, 300, 300), make([]byte, 9<<20)...), http.StatusRequestEntityTooLarge},
	} {
		if res := env.upload(u.Token, u.addr, tc.data, "image/jpeg"); res.Status != tc.want {
			t.Errorf("%s: got %d, want %d (%s)", name, res.Status, tc.want, res.Body)
		}
	}
	// A wrong declared MIME type does not matter: content decides.
	if res := env.upload(u.Token, u.addr, gradientJPEG(t, 300, 300), "text/html"); res.Status != http.StatusCreated {
		t.Fatalf("valid image with odd MIME: %d %s", res.Status, res.Body)
	}

	var ids []string
	ids = append(ids, env.do(http.MethodGet, "/me/profile", u.Token, nil, u.addr).Map(t)["photos"].([]any)[0].(map[string]any)["id"].(string))
	for i := 0; i < 5; i++ {
		res := env.upload(u.Token, u.addr, gradientJPEG(t, 300+i, 300), "image/jpeg")
		if res.Status != http.StatusCreated {
			t.Fatalf("upload %d: %d %s", i, res.Status, res.Body)
		}
		ids = append(ids, res.Map(t)["id"].(string))
	}
	if res := env.upload(u.Token, u.addr, gradientJPEG(t, 300, 300), "image/jpeg"); res.Status != http.StatusConflict {
		t.Fatalf("7th photo: got %d, want 409", res.Status)
	}

	order := func() []string {
		photos := env.do(http.MethodGet, "/me/profile", u.Token, nil, u.addr).Map(t)["photos"].([]any)
		out := make([]string, len(photos))
		for i, p := range photos {
			out[i] = p.(map[string]any)["id"].(string)
		}
		return out
	}

	// Main photo and ordering.
	if res := env.do(http.MethodPut, "/me/photos/"+ids[3]+"/main", u.Token, nil, u.addr); res.Status != http.StatusNoContent {
		t.Fatalf("make main: %d %s", res.Status, res.Body)
	}
	if got := order(); got[0] != ids[3] || got[1] != ids[0] {
		t.Fatalf("main photo should lead: %v", got)
	}
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	if res := env.do(http.MethodPut, "/me/photos/order", u.Token, map[string]any{"photoIds": rev}, u.addr); res.Status != http.StatusNoContent {
		t.Fatalf("reorder: %d %s", res.Status, res.Body)
	}
	if got := order(); strings.Join(got, ",") != strings.Join(rev, ",") {
		t.Fatalf("order not applied: %v", got)
	}
	if res := env.do(http.MethodPut, "/me/photos/order", u.Token, map[string]any{"photoIds": rev[:3]}, u.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("partial order must be rejected: %d", res.Status)
	}
	if res := env.do(http.MethodPut, "/me/photos/order", u.Token, map[string]any{"photoIds": []string{rev[0], rev[0], rev[1], rev[2], rev[3], rev[4]}}, u.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("duplicate order must be rejected: %d", res.Status)
	}

	// Ownership: someone else cannot touch my photos.
	if res := env.do(http.MethodDelete, "/me/photos/"+ids[0], other.Token, nil, other.addr); res.Status != http.StatusNotFound {
		t.Fatalf("deleting someone else's photo: %d", res.Status)
	}
	if res := env.do(http.MethodPut, "/me/photos/order", other.Token, map[string]any{"photoIds": []string{ids[0]}}, other.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("reordering someone else's photo: %d", res.Status)
	}

	// Signed URLs: valid works, tampered/expired-looking ones do not.
	photos := env.do(http.MethodGet, "/me/profile", u.Token, nil, u.addr).Map(t)["photos"].([]any)
	photoURL := photos[0].(map[string]any)["url"].(string)
	res := env.do(http.MethodGet, photoURL, "", nil, "")
	if res.Status != http.StatusOK || res.Header.Get("Content-Type") != "image/jpeg" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("signed photo: %d %v", res.Status, res.Header)
	}
	if res := env.do(http.MethodGet, strings.Replace(photoURL, "sig=", "sig=0", 1), "", nil, ""); res.Status != http.StatusNotFound {
		t.Fatalf("tampered signature: %d", res.Status)
	}
	if res := env.do(http.MethodGet, strings.SplitN(photoURL, "?", 2)[0], "", nil, ""); res.Status != http.StatusNotFound {
		t.Fatalf("unsigned URL: %d", res.Status)
	}

	// Replace keeps the slot; delete compacts positions and removes the file.
	if res := env.do(http.MethodPut, "/me/photos/"+rev[0], u.Token, nil, u.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("replace without file: %d", res.Status)
	}
	var key string
	if err := env.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE id = $1`, rev[1]).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if res := env.do(http.MethodDelete, "/me/photos/"+rev[1], u.Token, nil, u.addr); res.Status != http.StatusNoContent {
		t.Fatalf("delete: %d", res.Status)
	}
	if got := order(); len(got) != 5 || got[0] != rev[0] || got[1] != rev[2] {
		t.Fatalf("positions should stay contiguous: %v", got)
	}
	var positions []int
	rows, _ := env.pool.Query(t.Context(), `SELECT position FROM photos WHERE user_id = $1 ORDER BY position`, u.ID)
	for rows.Next() {
		var p int
		rows.Scan(&p)
		positions = append(positions, p)
	}
	rows.Close()
	for i, p := range positions {
		if p != i {
			t.Fatalf("positions not compacted: %v", positions)
		}
	}
	if _, err := os.Stat(filepath.Join(env.storeDir, filepath.FromSlash(key))); !os.IsNotExist(err) {
		t.Fatalf("file must be removed from storage (stat err=%v)", err)
	}
}

func TestBlockAndReport(t *testing.T) {
	env := newTestEnv(t)
	a, b := env.register("a"), env.register("b")
	env.onboard(a, profileSpec{Name: "A", BirthDate: "1994-01-01", Gender: "woman", InterestedIn: []string{"man"}})
	env.onboard(b, profileSpec{Name: "B", BirthDate: "1993-01-01", Gender: "man", InterestedIn: []string{"woman"}})
	env.swipe(a, b, "like")
	matchID := env.swipe(b, a, "like")["matchId"].(string)
	env.do(http.MethodPost, "/conversations/"+matchID+"/messages", a.Token, map[string]string{"body": "salut"}, a.addr)

	// Self-block and bad ids are rejected.
	if res := env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": a.ID}, a.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("self block: %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": "00000000-0000-4000-8000-000000000000"}, a.addr); res.Status != http.StatusNotFound {
		t.Fatalf("unknown block target: %d", res.Status)
	}

	if res := env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": b.ID}, a.addr); res.Status != http.StatusNoContent {
		t.Fatalf("block: %d", res.Status)
	}
	// Idempotent.
	if res := env.do(http.MethodPost, "/blocks", a.Token, map[string]string{"userId": b.ID}, a.addr); res.Status != http.StatusNoContent {
		t.Fatalf("block twice: %d", res.Status)
	}

	// The conversation is gone for both, nobody can see or contact the other.
	for _, u := range []*user{a, b} {
		if res := env.do(http.MethodGet, "/conversations/"+matchID+"/messages", u.Token, nil, u.addr); res.Status != http.StatusNotFound {
			t.Errorf("%s: conversation after block: %d", u.Email, res.Status)
		}
		if convs := env.do(http.MethodGet, "/conversations", u.Token, nil, u.addr).Map(t)["conversations"].([]any); len(convs) != 0 {
			t.Errorf("%s: conversations after block: %v", u.Email, convs)
		}
		if ms := env.do(http.MethodGet, "/matches", u.Token, nil, u.addr).Map(t)["matches"].([]any); len(ms) != 0 {
			t.Errorf("%s: matches after block: %v", u.Email, ms)
		}
	}
	if res := env.do(http.MethodPost, "/conversations/"+matchID+"/messages", b.Token, map[string]string{"body": "hello?"}, b.addr); res.Status != http.StatusNotFound {
		t.Fatalf("blocked user sending: %d", res.Status)
	}
	if res := env.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil, b.addr); res.Status != http.StatusNotFound {
		t.Fatalf("blocked user viewing blocker: %d", res.Status)
	}
	if ids := env.discoverIDs(b); contains(ids, a.ID) {
		t.Fatal("blocker must vanish from the blocked user's discovery")
	}

	list := env.do(http.MethodGet, "/blocks", a.Token, nil, a.addr).Map(t)["blocks"].([]any)
	if len(list) != 1 {
		t.Fatalf("block list: %v", list)
	}
	if res := env.do(http.MethodDelete, "/blocks/"+b.ID, a.Token, nil, a.addr); res.Status != http.StatusNoContent {
		t.Fatalf("unblock: %d", res.Status)
	}
	if list := env.do(http.MethodGet, "/blocks", a.Token, nil, a.addr).Map(t)["blocks"].([]any); len(list) != 0 {
		t.Fatalf("block list after unblock: %v", list)
	}

	// Reports: validated, stored once while open, reporter-only.
	report := func(reason string) int {
		return env.do(http.MethodPost, "/reports", a.Token, map[string]any{"userId": b.ID, "reason": reason, "details": "  spam links  "}, a.addr).Status
	}
	if got := report("not-a-reason"); got != http.StatusBadRequest {
		t.Fatalf("bad reason: %d", got)
	}
	if got := report("spam"); got != http.StatusAccepted {
		t.Fatalf("report: %d", got)
	}
	if got := report("spam"); got != http.StatusAccepted {
		t.Fatalf("repeat report: %d", got)
	}
	var n int
	var details string
	if err := env.pool.QueryRow(t.Context(), `SELECT COUNT(*), MIN(details) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n, &details); err != nil {
		t.Fatal(err)
	}
	if n != 1 || details != "spam links" {
		t.Fatalf("expected one trimmed report, got %d %q", n, details)
	}
	if res := env.do(http.MethodPost, "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"}, a.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("self report: %d", res.Status)
	}
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	env := newTestEnv(t)
	a, b := env.register("a"), env.register("b")
	env.onboard(a, profileSpec{Name: "A", BirthDate: "1994-01-01", Gender: "woman", InterestedIn: []string{"man"}, Interests: []string{"music"}, Photos: 2})
	env.onboard(b, profileSpec{Name: "B", BirthDate: "1993-01-01", Gender: "man", InterestedIn: []string{"woman"}})
	env.swipe(a, b, "like")
	matchID := env.swipe(b, a, "like")["matchId"].(string)
	env.do(http.MethodPost, "/conversations/"+matchID+"/messages", a.Token, map[string]string{"body": "avant de partir"}, a.addr)
	env.do(http.MethodPost, "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "other"}, b.addr)

	var keys []string
	rows, _ := env.pool.Query(t.Context(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID)
	for rows.Next() {
		var k string
		rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) != 2 {
		t.Fatalf("expected 2 stored photos, got %d", len(keys))
	}
	for _, k := range keys {
		if _, err := os.Stat(filepath.Join(env.storeDir, filepath.FromSlash(k))); err != nil {
			t.Fatalf("photo file should exist before deletion: %v", err)
		}
	}

	if res := env.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": "wrong-password"}, a.addr); res.Status != http.StatusForbidden {
		t.Fatalf("wrong password: %d", res.Status)
	}
	if res := env.do(http.MethodDelete, "/me", a.Token, map[string]string{}, a.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("missing password: %d", res.Status)
	}
	if res := env.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": testPassword}, a.addr); res.Status != http.StatusNoContent {
		t.Fatalf("delete account: %d", res.Status)
	}

	for table, col := range map[string]string{
		"users": "id", "profiles": "user_id", "preferences": "user_id", "user_interests": "user_id", "photos": "user_id",
		"sessions": "user_id", "notifications": "user_id",
	} {
		var n int
		if err := env.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s still holds %d rows for the deleted user (%v)", table, n, err)
		}
	}
	var n int
	for q, args := range map[string][]any{
		`SELECT COUNT(*) FROM swipes WHERE actor_id = $1 OR target_id = $1`:       {a.ID},
		`SELECT COUNT(*) FROM matches WHERE user_a = $1 OR user_b = $1`:           {a.ID},
		`SELECT COUNT(*) FROM messages WHERE sender_id = $1`:                      {a.ID},
		`SELECT COUNT(*) FROM reports WHERE reporter_id = $1 OR reported_id = $1`: {a.ID},
	} {
		if err := env.pool.QueryRow(t.Context(), q, args...).Scan(&n); err != nil || n != 0 {
			t.Errorf("leftover rows for %q: %d (%v)", q, n, err)
		}
	}
	for _, k := range keys {
		if _, err := os.Stat(filepath.Join(env.storeDir, filepath.FromSlash(k))); !os.IsNotExist(err) {
			t.Errorf("photo file %s should be deleted (err=%v)", k, err)
		}
	}

	if res := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": a.Email, "password": testPassword}, a.addr); res.Status != http.StatusUnauthorized {
		t.Fatalf("login after deletion: %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": a.Refresh}, a.addr); res.Status != http.StatusUnauthorized {
		t.Fatalf("refresh after deletion: %d", res.Status)
	}
	if convs := env.do(http.MethodGet, "/conversations", b.Token, nil, b.addr).Map(t)["conversations"].([]any); len(convs) != 0 {
		t.Fatalf("the other person must not keep a ghost conversation: %v", convs)
	}
	if ids := env.discoverIDs(b); contains(ids, a.ID) {
		t.Fatal("deleted user must vanish from discovery")
	}
}

func TestAuthFlowOverHTTP(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("auth")

	me := env.do(http.MethodGet, "/me", u.Token, nil, u.addr).Map(t)
	if me["email"] != u.Email || me["id"] != u.ID {
		t.Fatalf("/me: %v", me)
	}
	if res := env.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": testPassword}, u.addr); res.Status != http.StatusConflict {
		t.Fatalf("duplicate registration with different case: %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "not-an-email", "password": testPassword}, u.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("invalid email: %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "nope-nope"}, u.addr); res.Status != http.StatusUnauthorized {
		t.Fatalf("bad login: %d", res.Status)
	}

	// Recovery over HTTP: always 202, code arrives by mail, session list is revoked.
	if res := env.do(http.MethodPost, "/auth/forgot-password", "", map[string]string{"email": "ghost@example.test"}, u.addr); res.Status != http.StatusAccepted {
		t.Fatalf("forgot (unknown): %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/auth/forgot-password", "", map[string]string{"email": u.Email}, u.addr); res.Status != http.StatusAccepted {
		t.Fatalf("forgot: %d", res.Status)
	}
	var body string
	for i := 0; i < 100 && body == ""; i++ { // the email is sent in the background
		env.mail.mu.Lock()
		if len(env.mail.bodies) > 0 {
			body = env.mail.bodies[len(env.mail.bodies)-1]
		}
		env.mail.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	code := body[strings.Index(body, ": ")+2:][:10]
	if res := env.do(http.MethodPost, "/auth/reset-password", "", map[string]string{"email": u.Email, "code": code, "newPassword": "BrandNew123"}, u.addr); res.Status != http.StatusNoContent {
		t.Fatalf("reset: %d %s", res.Status, res.Body)
	}
	if res := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "BrandNew123"}, u.addr); res.Status != http.StatusOK {
		t.Fatalf("login with new password: %d", res.Status)
	}
	if res := env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}, u.addr); res.Status != http.StatusUnauthorized {
		t.Fatalf("old refresh token after reset: %d", res.Status)
	}
}

func TestRateLimitingOnCredentialEndpoints(t *testing.T) {
	env := newTestEnv(t)
	addr := "203.0.113.50:5000"
	var last int
	for i := 0; i < 25; i++ {
		last = env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "nobody@example.test", "password": "wrong-password"}, addr).Status
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after repeated logins, got %d", last)
	}
	// Another client is unaffected.
	if res := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "nobody@example.test", "password": "wrong-password"}, "203.0.113.51:5000"); res.Status != http.StatusUnauthorized {
		t.Fatalf("other client: %d", res.Status)
	}
}

func TestInterestsCatalogue(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("i")
	list := env.do(http.MethodGet, "/interests", u.Token, nil, u.addr).Map(t)["interests"].([]any)
	if len(list) < 20 {
		t.Fatalf("catalogue too small: %d", len(list))
	}
}

func TestSuspendedAccountCannotSignIn(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("sus")
	if _, err := env.pool.Exec(t.Context(), `UPDATE users SET banned_at = NOW() WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if res := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": testPassword}, u.addr); res.Status != http.StatusForbidden {
		t.Fatalf("suspended login: got %d, want 403", res.Status)
	}
	if res := env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}, u.addr); res.Status != http.StatusUnauthorized {
		t.Fatalf("suspended refresh: got %d, want 401", res.Status)
	}
}

func TestHardening(t *testing.T) {
	env := newTestEnv(t)
	a, b := env.register("a"), env.register("b")
	env.onboard(a, profileSpec{Name: "A", BirthDate: "1994-01-01", Gender: "woman", InterestedIn: []string{"man"}})
	env.onboard(b, profileSpec{Name: "B", BirthDate: "1993-01-01", Gender: "man", InterestedIn: []string{"woman"}})

	// Upper-case ids must behave like lower-case ones (and cannot dodge the self check).
	if res := env.do(http.MethodPost, "/swipes", a.Token, map[string]string{"targetId": strings.ToUpper(a.ID), "action": "like"}, a.addr); res.Status != http.StatusBadRequest {
		t.Fatalf("self swipe in upper case: %d", res.Status)
	}
	env.swipe(a, &user{ID: strings.ToUpper(b.ID)}, "like")
	if res := env.swipe(b, a, "like"); res["matched"] != true {
		t.Fatalf("mutual like with mixed-case id must match: %v", res)
	}

	// Logout and suspension take effect on access tokens immediately (cache aside).
	login := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": b.Email, "password": testPassword}, b.addr).Map(t)
	token := login["accessToken"].(string)
	if res := env.do(http.MethodGet, "/me", token, nil, b.addr); res.Status != http.StatusOK {
		t.Fatalf("fresh token: %d", res.Status)
	}
	if _, err := env.pool.Exec(t.Context(), `UPDATE users SET banned_at = NOW() WHERE id = $1`, b.ID); err != nil {
		t.Fatal(err)
	}
	// A new check is needed after the positive cache expires; sessions of fresh tokens are checked on first use.
	fresh := env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": login["refreshToken"].(string)}, b.addr)
	if fresh.Status != http.StatusUnauthorized {
		t.Fatalf("banned refresh: %d", fresh.Status)
	}

	// Oversized JSON bodies are refused before being buffered.
	big := []byte(`{"email":"` + strings.Repeat("a", 200_000) + `"}`)
	if res := env.do(http.MethodPost, "/auth/login", "", big, "198.51.100.9:1"); res.Status == http.StatusOK || res.Status == http.StatusUnauthorized {
		t.Fatalf("oversized body must be rejected as a bad request, got %d", res.Status)
	}

	// The stored location can only be moved once a day.
	c := env.register("c")
	env.onboard(c, profileSpec{Name: "C", BirthDate: "1990-01-01", Gender: "woman", InterestedIn: []string{"man"}})
	move := func(lat, lon float64) int {
		return env.do(http.MethodPut, "/me/profile", c.Token, map[string]any{"firstName": "C", "birthDate": "1990-01-01", "gender": "woman", "city": "X", "latitude": lat, "longitude": lon}, c.addr).Status
	}
	if got := move(45.76, 4.83); got != http.StatusOK {
		t.Fatalf("first move: %d", got)
	}
	if got := move(43.29, 5.37); got != http.StatusBadRequest {
		t.Fatalf("second move within a day must be refused: %d", got)
	}
	if got := move(45.76, 4.83); got != http.StatusOK {
		t.Fatalf("saving the same location stays allowed: %d", got)
	}
}
