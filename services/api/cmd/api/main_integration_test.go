package main

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/testutil"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.must(e.call("GET", "/health", "", nil), 200)
	if r.json()["status"] != "ok" {
		t.Fatalf("unexpected health body %s", r.Body)
	}
}

func TestErrorShapeAndAuthRequired(t *testing.T) {
	e := newEnv(t)
	for _, tc := range []struct{ method, path string }{
		{"GET", "/me"}, {"GET", "/discover"}, {"GET", "/matches"}, {"GET", "/notifications"},
		{"POST", "/swipes"}, {"GET", "/blocks"}, {"GET", "/interests"}, {"PUT", "/me/profile"},
		{"GET", "/photos/00000000-0000-0000-0000-000000000000/content"},
	} {
		r := e.call(tc.method, tc.path, "", nil)
		if r.Status != 401 || r.code() != "unauthorized" || r.json()["error"] == "" {
			t.Errorf("%s %s: want 401 unauthorized, got %d %s", tc.method, tc.path, r.Status, r.Body)
		}
	}
	r := e.call("GET", "/nope", "", nil)
	if r.Status != 404 || r.code() != "not_found" {
		t.Errorf("unknown route: %d %s", r.Status, r.Body)
	}
	if r := e.call("GET", "/me", "not.a.jwt", nil); r.Status != 401 {
		t.Errorf("garbage token: %d", r.Status)
	}
}

func TestAuthRegisterLoginRefreshMe(t *testing.T) {
	e := newEnv(t)

	// validation
	for _, body := range []map[string]string{
		{"email": "nope", "password": testPassword},
		{"email": "a@b.invalid", "password": "short"},
		{"email": "a@b.invalid", "password": strings.Repeat("x", 129)},
		{"email": "", "password": testPassword},
		{"email": "Bob <b@b.invalid>", "password": testPassword},
	} {
		r := e.call("POST", "/auth/register", "", body)
		if r.Status != 400 || r.code() != "invalid_request" {
			t.Errorf("register %v: want 400 invalid_request, got %d %s", body, r.Status, r.Body)
		}
	}
	// unknown JSON garbage
	if r := e.raw("POST", "/auth/register", "", "application/json", strings.NewReader("{not json")); r.Status != 400 {
		t.Errorf("bad json: %d", r.Status)
	}

	// case/space normalisation and duplicate detection
	r := e.must(e.call("POST", "/auth/register", "", map[string]string{"email": "  Alice@Test.Invalid ", "password": testPassword}), 201)
	j := r.json()
	if j["user"].(map[string]any)["email"] != "alice@test.invalid" {
		t.Fatalf("email not normalised: %s", r.Body)
	}
	if _, has := j["token"]; has {
		t.Errorf("legacy token field must be gone")
	}
	if r.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("auth responses must be no-store")
	}
	if r := e.call("POST", "/auth/register", "", map[string]string{"email": "ALICE@test.invalid", "password": testPassword}); r.Status != 409 || r.code() != "conflict" {
		t.Errorf("duplicate register: %d %s", r.Status, r.Body)
	}

	// 72+ byte passwords work (pre-hash) and are compared fully
	long := strings.Repeat("a", 100)
	e.must(e.call("POST", "/auth/register", "", map[string]string{"email": "long@test.invalid", "password": long}), 201)
	e.must(e.call("POST", "/auth/login", "", map[string]string{"email": "long@test.invalid", "password": long}), 200)
	if r := e.call("POST", "/auth/login", "", map[string]string{"email": "long@test.invalid", "password": long[:80]}); r.Status != 401 {
		t.Errorf("truncated long password must not log in: %d", r.Status)
	}

	// login
	if r := e.call("POST", "/auth/login", "", map[string]string{"email": "alice@test.invalid", "password": "wrong-password"}); r.Status != 401 || r.code() != "unauthorized" {
		t.Errorf("wrong password: %d %s", r.Status, r.Body)
	}
	if r := e.call("POST", "/auth/login", "", map[string]string{"email": "ghost@test.invalid", "password": testPassword}); r.Status != 401 {
		t.Errorf("unknown user: %d", r.Status)
	}
	login := e.must(e.call("POST", "/auth/login", "", map[string]string{"email": "ALICE@test.invalid", "password": testPassword}), 200).json()
	access, refresh := login["accessToken"].(string), login["refreshToken"].(string)

	// /me for a fresh account
	me := e.must(e.call("GET", "/me", access, nil), 200).json()
	if me["email"] != "alice@test.invalid" || me["profile"] != nil || me["profileComplete"] != false {
		t.Fatalf("unexpected /me: %v", me)
	}
	prefs := me["preferences"].(map[string]any)
	if prefs["minAge"] != float64(18) || prefs["maxAge"] != float64(99) || prefs["maxDistanceKm"] != float64(50) || len(prefs["interestedIn"].([]any)) != 3 {
		t.Fatalf("unexpected default preferences: %v", prefs)
	}
	if _, has := me["organizationId"]; has {
		t.Errorf("no organization in /me")
	}

	// refresh rotates: the old refresh token dies, the new one works
	ref := e.must(e.call("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), 200).json()
	newRefresh := ref["refreshToken"].(string)
	if newRefresh == refresh {
		t.Fatal("refresh token was not rotated")
	}
	if r := e.call("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}); r.Status != 401 {
		t.Errorf("reusing a rotated refresh token: %d", r.Status)
	}
	e.must(e.call("GET", "/me", ref["accessToken"].(string), nil), 200)

	// logout revokes the session: refresh and access token both stop working
	e.must(e.call("POST", "/auth/logout", "", map[string]string{"refreshToken": newRefresh}), 204)
	if r := e.call("POST", "/auth/refresh", "", map[string]string{"refreshToken": newRefresh}); r.Status != 401 {
		t.Errorf("refresh after logout: %d", r.Status)
	}
	if r := e.call("GET", "/me", ref["accessToken"].(string), nil); r.Status != 401 {
		t.Errorf("access token after logout must be rejected (session revoked): %d", r.Status)
	}
	// logout of garbage is still 204 (no oracle)
	e.must(e.call("POST", "/auth/logout", "", map[string]string{"refreshToken": "garbage"}), 204)
}

func TestConcurrentRefreshOnlyOneWins(t *testing.T) {
	e := newEnv(t)
	u := e.register()
	var wg sync.WaitGroup
	results := make([]int, 6)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = e.call("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}).Status
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, s := range results {
		if s == 200 {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("expected exactly one refresh to succeed, got %d (%v)", ok, results)
	}
}

func TestPasswordResetFlow(t *testing.T) {
	e := newEnv(t)
	e.d.Mailer = e.mail
	u := e.register()

	// unknown email: 204 and no mail
	e.must(e.call("POST", "/auth/forgot-password", "", map[string]string{"email": "ghost@test.invalid"}), 204)
	if _, ok := e.mail.last("ghost@test.invalid"); ok {
		t.Fatal("no mail expected for unknown account")
	}

	code := func() string {
		t.Helper()
		e.must(e.call("POST", "/auth/forgot-password", "", map[string]string{"email": u.Email}), 204)
		var msg string
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if m, ok := e.mail.last(u.Email); ok && strings.Contains(m.Body, "reset code is ") {
				msg = m.Body
				// use the freshest one: wait until the count stabilises
				time.Sleep(50 * time.Millisecond)
				m2, _ := e.mail.last(u.Email)
				msg = m2.Body
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if msg == "" {
			t.Fatal("reset mail not sent")
		}
		i := strings.Index(msg, "reset code is ") + len("reset code is ")
		return msg[i : i+8]
	}

	c1 := code()
	if len(c1) != 8 {
		t.Fatalf("bad code %q", c1)
	}
	var stored string
	if err := e.pool.QueryRow(context.Background(), `SELECT code_hash FROM password_resets WHERE user_id=$1`, u.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, c1) || len(stored) != 64 {
		t.Fatalf("reset code must be stored hashed, got %q", stored)
	}

	body := func(code, pw string) map[string]string {
		return map[string]string{"email": u.Email, "code": code, "newPassword": pw}
	}
	if r := e.call("POST", "/auth/reset-password", "", body("AAAAAAAA", "brand-new-password")); r.Status != 400 || r.code() != "invalid_request" {
		t.Fatalf("wrong code: %d %s", r.Status, r.Body)
	}
	if r := e.call("POST", "/auth/reset-password", "", body(c1, "short")); r.Status != 400 {
		t.Fatalf("weak new password: %d", r.Status)
	}
	// attempts: 5 total guesses allowed; burn the rest then the right code fails
	for i := 0; i < 4; i++ {
		e.call("POST", "/auth/reset-password", "", body("BBBBBBBB", "brand-new-password"))
	}
	if r := e.call("POST", "/auth/reset-password", "", body(c1, "brand-new-password")); r.Status != 400 {
		t.Fatalf("code must be locked after 5 attempts: %d", r.Status)
	}

	// a new code works (lower case + spaces tolerated), single use, revokes sessions
	c2 := code()
	pw := "brand-new-password"
	e.must(e.call("POST", "/auth/reset-password", "", body(" "+strings.ToLower(c2)+" ", pw)), 204)
	if r := e.call("POST", "/auth/reset-password", "", body(c2, "another-new-password")); r.Status != 400 {
		t.Fatalf("code must be single use: %d", r.Status)
	}
	if r := e.call("GET", "/me", u.Access, nil); r.Status != 401 {
		t.Fatalf("sessions must be revoked after reset: %d", r.Status)
	}
	if r := e.call("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}); r.Status != 401 {
		t.Fatalf("refresh must be revoked after reset: %d", r.Status)
	}
	if r := e.call("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": testPassword}); r.Status != 401 {
		t.Fatalf("old password must stop working: %d", r.Status)
	}
	e.must(e.call("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": pw}), 200)

	// expiry
	c3 := code()
	if _, err := e.pool.Exec(context.Background(), `UPDATE password_resets SET expires_at = NOW() - INTERVAL '1 minute' WHERE user_id=$1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if r := e.call("POST", "/auth/reset-password", "", body(c3, "yet-another-password")); r.Status != 400 {
		t.Fatalf("expired code: %d", r.Status)
	}
}

func TestChangePassword(t *testing.T) {
	e := newEnv(t)
	u := e.register()
	other := e.must(e.call("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": testPassword}), 200).json()

	if r := u.call("POST", "/me/password", map[string]string{"currentPassword": "nope-nope-nope", "newPassword": "a-new-password-1"}); r.Status != 403 {
		t.Fatalf("wrong current password: %d %s", r.Status, r.Body)
	}
	if r := u.call("POST", "/me/password", map[string]string{"currentPassword": testPassword, "newPassword": "short"}); r.Status != 400 {
		t.Fatalf("weak new password: %d", r.Status)
	}
	e.must(u.call("POST", "/me/password", map[string]string{"currentPassword": testPassword, "newPassword": "a-new-password-1"}), 204)
	// current session survives, other sessions are revoked
	e.must(u.call("GET", "/me", nil), 200)
	if r := e.call("GET", "/me", other["accessToken"].(string), nil); r.Status != 401 {
		t.Fatalf("other session must be revoked: %d", r.Status)
	}
	e.must(e.call("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "a-new-password-1"}), 200)
}

func TestProfileLocationPreferencesPhotos(t *testing.T) {
	e := newEnv(t)
	u := e.register()

	// interests catalogue
	items := e.must(u.call("GET", "/interests", nil), 200).json()["items"].([]any)
	if len(items) < 25 {
		t.Fatalf("expected ~30 interests, got %d", len(items))
	}
	first := items[0].(map[string]any)
	if first["id"] == nil || first["slug"] == nil || first["label"] == nil {
		t.Fatalf("bad interest shape: %v", first)
	}
	interestIDs := []int{}
	for _, it := range items[:3] {
		interestIDs = append(interestIDs, int(it.(map[string]any)["id"].(float64)))
	}

	valid := func() map[string]any {
		return map[string]any{"firstName": "  Sam ", "birthDate": birthFor(30), "gender": "non_binary", "bio": "Hi", "interestIds": interestIDs}
	}
	mod := func(k string, v any) map[string]any { m := valid(); m[k] = v; return m }

	// location before profile is refused
	if r := u.call("PUT", "/me/location", map[string]any{"latitude": 1.0, "longitude": 2.0}); r.Status != 409 || r.code() != "profile_incomplete" {
		t.Fatalf("location without profile: %d %s", r.Status, r.Body)
	}

	// validation
	if r := u.call("PUT", "/me/profile", mod("birthDate", birthFor(17))); r.Status != 422 || r.code() != "underage" {
		t.Fatalf("underage: %d %s", r.Status, r.Body)
	}
	// exactly 18 today is allowed, one day short is not
	today := time.Now().UTC()
	exactly18 := time.Date(today.Year()-18, today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	tomorrow18 := time.Date(today.Year()-18, today.Month(), today.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1).Format("2006-01-02")
	if r := u.call("PUT", "/me/profile", mod("birthDate", tomorrow18)); r.Status != 422 || r.code() != "underage" {
		t.Fatalf("17y364d must be underage: %d %s", r.Status, r.Body)
	}
	for name, body := range map[string]map[string]any{
		"empty name":    mod("firstName", "   "),
		"long name":     mod("firstName", strings.Repeat("a", 51)),
		"newline name":  mod("firstName", "a\nb"),
		"bad gender":    mod("gender", "robot"),
		"long bio":      mod("bio", strings.Repeat("a", 501)),
		"nul in bio":    mod("bio", "a\u0000b"),
		"bad date":      mod("birthDate", "30/01/1990"),
		"future date":   mod("birthDate", time.Now().AddDate(1, 0, 0).Format("2006-01-02")),
		"11 interests":  mod("interestIds", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}),
		"dup interests": mod("interestIds", []int{1, 1}),
		"bad interest":  mod("interestIds", []int{9999}),
		"zero interest": mod("interestIds", []int{0}),
	} {
		if r := u.call("PUT", "/me/profile", body); r.Status != 400 || r.code() != "invalid_request" {
			t.Errorf("%s: want 400 invalid_request, got %d %s", name, r.Status, r.Body)
		}
	}

	// create with exactly 18
	r := e.must(u.call("PUT", "/me/profile", mod("birthDate", exactly18)), 200).json()
	if r["firstName"] != "Sam" || r["age"] != float64(18) || r["hasLocation"] != false || r["birthDate"] != exactly18 {
		t.Fatalf("unexpected profile: %v", r)
	}
	if r["showDistance"] != true || r["isDiscoverable"] != true {
		t.Fatalf("defaults should be true: %v", r)
	}
	if len(r["interests"].([]any)) != 3 || len(r["photos"].([]any)) != 0 {
		t.Fatalf("interests/photos wrong: %v", r)
	}
	// birth date is immutable
	if r := u.call("PUT", "/me/profile", mod("birthDate", birthFor(40))); r.Status != 422 || r.code() != "invalid_request" {
		t.Fatalf("birthDate change: %d %s", r.Status, r.Body)
	}
	// update keeps unspecified flags, applies specified ones
	off := false
	upd := valid()
	upd["birthDate"] = exactly18
	upd["showDistance"] = off
	upd["bio"] = "Updated"
	r = e.must(u.call("PUT", "/me/profile", upd), 200).json()
	if r["showDistance"] != false || r["isDiscoverable"] != true || r["bio"] != "Updated" {
		t.Fatalf("flags after update: %v", r)
	}

	// location: validation, rounding, no coordinates in any response
	for name, body := range map[string]map[string]any{
		"lat range":  {"latitude": 91, "longitude": 0},
		"lon range":  {"latitude": 0, "longitude": 181},
		"missing":    {"latitude": 10},
		"long label": {"latitude": 10, "longitude": 10, "label": strings.Repeat("x", 81)},
	} {
		if r := u.call("PUT", "/me/location", body); r.Status != 400 {
			t.Errorf("location %s: %d %s", name, r.Status, r.Body)
		}
	}
	r = e.must(u.call("PUT", "/me/location", map[string]any{"latitude": 48.856613, "longitude": 2.352222, "label": " Paris "}), 200).json()
	if r["hasLocation"] != true || r["locationLabel"] != "Paris" {
		t.Fatalf("location response: %v", r)
	}
	for k := range r {
		if k == "latitude" || k == "longitude" || k == "lat" || k == "lon" {
			t.Fatalf("coordinates must never be returned: %v", r)
		}
	}
	var lat, lon float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id=$1`, u.ID).Scan(&lat, &lon); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lon != 2.35 {
		t.Fatalf("stored coordinates must be rounded to 2 decimals, got %v,%v", lat, lon)
	}
	// label optional on later update: keeps previous
	r = e.must(u.call("PUT", "/me/location", map[string]any{"latitude": 48.85, "longitude": 2.35}), 200).json()
	if r["locationLabel"] != "Paris" {
		t.Fatalf("label should be kept: %v", r)
	}

	// preferences
	if r := u.call("PUT", "/me/preferences", map[string]any{"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}); r.Status != 400 {
		t.Errorf("empty interestedIn: %d", r.Status)
	}
	for name, body := range map[string]map[string]any{
		"unknown gender": {"interestedIn": []string{"x"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10},
		"dup gender":     {"interestedIn": []string{"man", "man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10},
		"min 17":         {"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 30, "maxDistanceKm": 10},
		"max < min":      {"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 25, "maxDistanceKm": 10},
		"max 100":        {"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 100, "maxDistanceKm": 10},
		"dist 0":         {"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 40, "maxDistanceKm": 0},
		"dist 501":       {"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 40, "maxDistanceKm": 501},
	} {
		if r := u.call("PUT", "/me/preferences", body); r.Status != 400 || r.code() != "invalid_request" {
			t.Errorf("prefs %s: %d %s", name, r.Status, r.Body)
		}
	}
	p := e.must(u.call("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"woman", "non_binary"}, "minAge": 21, "maxAge": 35, "maxDistanceKm": 25}), 200).json()
	if p["minAge"] != float64(21) || p["maxDistanceKm"] != float64(25) {
		t.Fatalf("prefs: %v", p)
	}
	if g := e.must(u.call("GET", "/me/preferences", nil), 200).json(); g["maxAge"] != float64(35) {
		t.Fatalf("get prefs: %v", g)
	}

	// not complete until a photo exists
	if me := e.must(u.call("GET", "/me", nil), 200).json(); me["profileComplete"] != false {
		t.Fatalf("no photo yet: %v", me)
	}

	// --- photos ---
	// non-images and disguised files are rejected by content
	if r := e.upload("POST", "/me/photos", u.Access, "evil.jpg", []byte("<html>not an image</html>")); r.Status != 415 || r.code() != "unsupported_media" {
		t.Errorf("text as jpg: %d %s", r.Status, r.Body)
	}
	if r := e.upload("POST", "/me/photos", u.Access, "cat.jpg", testutil.GIF()); r.Status != 415 {
		t.Errorf("gif renamed .jpg: %d %s", r.Status, r.Body)
	}
	if r := e.upload("POST", "/me/photos", u.Access, "trunc.jpg", testutil.JPEG(100, 100)[:200]); r.Status != 415 {
		t.Errorf("truncated jpeg: %d %s", r.Status, r.Body)
	}
	if r := e.raw("POST", "/me/photos", u.Access, "application/json", strings.NewReader(`{}`)); r.Status != 400 {
		t.Errorf("non multipart: %d", r.Status)
	}
	// oversize (>8 MB) is refused with 413
	big := make([]byte, 9<<20)
	copy(big, testutil.JPEG(10, 10))
	if r := e.upload("POST", "/me/photos", u.Access, "big.jpg", big); r.Status != 413 || r.code() != "payload_too_large" {
		t.Errorf("oversize: %d %s", r.Status, r.Body)
	}

	// jpeg with EXIF/GPS: metadata stripped, long side downscaled to <= 1280
	withExif := injectExif(testutil.JPEG(2000, 1000))
	if !strings.Contains(string(withExif), "Exif") {
		t.Fatal("test setup: exif not injected")
	}
	ph := e.must(e.upload("POST", "/me/photos", u.Access, "a.jpg", withExif), 201).json()
	photoID := ph["id"].(string)
	if ph["position"] != float64(0) || ph["url"] != "/photos/"+photoID+"/content" {
		t.Fatalf("photo shape: %v", ph)
	}
	content := e.must(e.raw("GET", "/photos/"+photoID+"/content", u.Access, "", nil), 200)
	if content.Header.Get("Content-Type") != "image/jpeg" || content.Header.Get("Cache-Control") != "private, max-age=86400" || content.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("photo headers: %v", content.Header)
	}
	if strings.Contains(string(content.Body), "Exif") || strings.Contains(string(content.Body), "GPS") {
		t.Error("EXIF must be stripped")
	}
	cfg := decodeConfig(t, content.Body)
	if cfg.Width != 1280 || cfg.Height != 640 {
		t.Errorf("expected 1280x640 after downscale, got %dx%d", cfg.Width, cfg.Height)
	}
	// PNG accepted and converted to JPEG
	png := e.must(e.upload("POST", "/me/photos", u.Access, "p.png", testutil.PNG(64, 64)), 201).json()
	pc := e.must(e.raw("GET", "/photos/"+png["id"].(string)+"/content", u.Access, "", nil), 200)
	if !strings.HasPrefix(string(pc.Body), "\xff\xd8") {
		t.Error("png must be re-encoded to jpeg")
	}

	me := e.must(u.call("GET", "/me", nil), 200).json()
	if me["profileComplete"] != true || len(me["profile"].(map[string]any)["photos"].([]any)) != 2 {
		t.Fatalf("profile should be complete: %v", me)
	}

	// max 6 photos
	for i := 0; i < 4; i++ {
		e.must(e.upload("POST", "/me/photos", u.Access, "x.jpg", testutil.JPEG(50, 50)), 201)
	}
	if r := e.upload("POST", "/me/photos", u.Access, "x.jpg", testutil.JPEG(50, 50)); r.Status != 409 || r.code() != "conflict" {
		t.Errorf("7th photo: %d %s", r.Status, r.Body)
	}

	// reorder must be a permutation
	photos := e.must(u.call("GET", "/me", nil), 200).json()["profile"].(map[string]any)["photos"].([]any)
	ids := make([]string, len(photos))
	for i, p := range photos {
		ids[i] = p.(map[string]any)["id"].(string)
	}
	if r := u.call("PUT", "/me/photos/order", map[string]any{"photoIds": ids[:5]}); r.Status != 400 {
		t.Errorf("partial order: %d", r.Status)
	}
	if r := u.call("PUT", "/me/photos/order", map[string]any{"photoIds": append(append([]string{}, ids[:5]...), ids[0])}); r.Status != 400 {
		t.Errorf("dup order: %d", r.Status)
	}
	if r := u.call("PUT", "/me/photos/order", map[string]any{"photoIds": []string{"nope"}}); r.Status != 400 {
		t.Errorf("bad id order: %d", r.Status)
	}
	rev := make([]string, len(ids))
	for i := range ids {
		rev[i] = ids[len(ids)-1-i]
	}
	ordered := e.must(u.call("PUT", "/me/photos/order", map[string]any{"photoIds": rev}), 200).json()["items"].([]any)
	for i, p := range ordered {
		m := p.(map[string]any)
		if m["id"] != rev[i] || m["position"] != float64(i) {
			t.Fatalf("order not applied: %v", ordered)
		}
	}
	// replace keeps position
	rep := e.must(e.upload("PUT", "/me/photos/"+rev[2], u.Access, "r.jpg", testutil.JPEG(30, 30)), 200).json()
	if rep["id"] != rev[2] || rep["position"] != float64(2) {
		t.Fatalf("replace: %v", rep)
	}
	if r := e.upload("PUT", "/me/photos/"+rev[2], u.Access, "r.jpg", testutil.GIF()); r.Status != 415 {
		t.Errorf("replace with gif: %d", r.Status)
	}
	// delete compacts positions
	e.must(u.call("DELETE", "/me/photos/"+rev[0], nil), 204)
	if r := u.call("DELETE", "/me/photos/"+rev[0], nil); r.Status != 404 {
		t.Errorf("delete twice: %d", r.Status)
	}
	after := e.must(u.call("GET", "/me", nil), 200).json()["profile"].(map[string]any)["photos"].([]any)
	if len(after) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(after))
	}
	for i, p := range after {
		if p.(map[string]any)["position"] != float64(i) {
			t.Fatalf("positions not compacted: %v", after)
		}
	}
	if r := u.call("DELETE", "/me/photos/not-a-uuid", nil); r.Status != 404 {
		t.Errorf("delete bad id: %d", r.Status)
	}
}

func TestPhotoAuthorization(t *testing.T) {
	e := newEnv(t)
	owner := e.person(spec{Lat: 10, Lon: 10})
	stranger := e.person(spec{Lat: 10, Lon: 10})
	photo := owner.PhotoIDs[0]
	path := "/photos/" + photo + "/content"

	e.must(e.raw("GET", path, owner.Access, "", nil), 200)
	// discoverable owner: another user may view
	e.must(e.raw("GET", path, stranger.Access, "", nil), 200)

	// another user cannot delete/replace/reorder with it
	if r := stranger.call("DELETE", "/me/photos/"+photo, nil); r.Status != 404 {
		t.Errorf("delete other's photo: %d", r.Status)
	}
	if r := e.upload("PUT", "/me/photos/"+photo, stranger.Access, "x.jpg", testutil.JPEG(20, 20)); r.Status != 404 {
		t.Errorf("replace other's photo: %d", r.Status)
	}
	if r := stranger.call("PUT", "/me/photos/order", map[string]any{"photoIds": []string{photo}}); r.Status != 400 {
		t.Errorf("reorder with other's photo: %d", r.Status)
	}

	// hidden profile: strangers get 404, the owner still sees it
	e.must(owner.call("PUT", "/me/profile", map[string]any{"firstName": "O", "birthDate": birthFor(30), "gender": "woman", "bio": "", "interestIds": []int{}, "isDiscoverable": false}), 200)
	if r := e.raw("GET", path, stranger.Access, "", nil); r.Status != 404 {
		t.Errorf("hidden profile photo must be 404 for strangers: %d", r.Status)
	}
	e.must(e.raw("GET", path, owner.Access, "", nil), 200)
	// ... but a match may still see it
	e.matchUsersHidden(owner, stranger)
	e.must(e.raw("GET", path, stranger.Access, "", nil), 200)

	// blocked either way => 404
	third := e.person(spec{Lat: 10, Lon: 10})
	e.must(e.raw("GET", "/photos/"+stranger.PhotoIDs[0]+"/content", third.Access, "", nil), 200)
	e.must(third.call("POST", "/blocks", map[string]string{"userId": stranger.ID}), 204)
	if r := e.raw("GET", "/photos/"+stranger.PhotoIDs[0]+"/content", third.Access, "", nil); r.Status != 404 {
		t.Errorf("blocker viewing blocked user's photo: %d", r.Status)
	}
	if r := e.raw("GET", "/photos/"+third.PhotoIDs[0]+"/content", stranger.Access, "", nil); r.Status != 404 {
		t.Errorf("blocked viewing blocker's photo: %d", r.Status)
	}
	if r := e.raw("GET", "/photos/not-a-uuid/content", owner.Access, "", nil); r.Status != 404 {
		t.Errorf("bad id: %d", r.Status)
	}
}

// matchUsersHidden creates a match directly (used when one side is not discoverable).
func (e *env) matchUsersHidden(a, b *user) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), `
		WITH m AS (INSERT INTO matches (user_a, user_b) VALUES (LEAST($1::uuid,$2::uuid), GREATEST($1::uuid,$2::uuid)) RETURNING id)
		INSERT INTO match_conversations (match_id) SELECT id FROM m`, a.ID, b.ID); err != nil {
		e.t.Fatal(err)
	}
}
