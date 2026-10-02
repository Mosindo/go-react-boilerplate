package app_test

import (
	"strings"
	"testing"
	"time"
)

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	u := e.register("auth")

	// Duplicate (case/space-insensitive), invalid and weak inputs.
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": " " + strings.ToUpper(u.Email), "password": "Password123"}), 409)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "x-" + u.Email, "password": "short"}), 400)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "WrongPass1"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@integration.test", "password": "WrongPass1"}), 401)

	// Protected routes need a valid access token.
	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	me := e.expect(e.do("GET", "/me", u.Token, nil), 200)
	if me.str("email") != u.Email {
		t.Fatalf("me: %s", me.Raw)
	}
	if strings.Contains(string(me.Raw), "password") {
		t.Fatal("me leaks password data")
	}

	// Refresh rotates: the old refresh token is dead, the new one works.
	r := e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 200)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), 401)
	newRefresh := r.str("refreshToken")
	if newRefresh == "" || newRefresh == u.Refresh {
		t.Fatal("refresh token not rotated")
	}

	// Logout kills the session immediately, including its access tokens.
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": newRefresh}), 204)
	e.expect(e.do("GET", "/me", r.str("accessToken"), nil), 401)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": newRefresh}), 401)
}

func TestPasswordRecoveryAndChange(t *testing.T) {
	e := newEnv(t)
	u := e.register("pw")

	// Unknown email: same answer, nothing sent.
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": "ghost@integration.test"}), 202)
	if e.mail.body("ghost@integration.test") != "" {
		t.Fatal("no email must be sent for unknown accounts")
	}
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": u.Email}), 202)
	body := e.mail.body(u.Email)
	idx := strings.Index(body, ": ")
	code := strings.Fields(body[idx+2:])[0]
	if len(code) != 10 {
		t.Fatalf("unexpected code %q in %q", code, body)
	}

	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": "AAAAAAAAAA", "newPassword": "BrandNew123"}), 400)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": code, "newPassword": "short"}), 400)
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": strings.ToLower(code), "newPassword": "BrandNew123"}), 204)
	// Single use + all sessions revoked + new password active.
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another1234"}), 400)
	e.expect(e.do("GET", "/me", u.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"}), 401)
	l := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "BrandNew123"}), 200)

	// Brute force: after 5 wrong tries the code is dead even if then correct.
	e.expect(e.do("POST", "/auth/forgot-password", "", map[string]string{"email": u.Email}), 202)
	body = e.mail.body(u.Email)
	code = strings.Fields(body[strings.Index(body, ": ")+2:])[0]
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": "ZZZZZZZZZZ", "newPassword": "Whatever123"}), 400)
	}
	e.expect(e.do("POST", "/auth/reset-password", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Whatever123"}), 400)

	// Change password keeps this session, drops the others.
	other := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "BrandNew123"}), 200)
	tok := l.str("accessToken")
	e.expect(e.do("POST", "/me/password", tok, map[string]string{"currentPassword": "nope", "newPassword": "Changed12345"}), 403)
	e.expect(e.do("POST", "/me/password", tok, map[string]string{"currentPassword": "BrandNew123", "newPassword": "Changed12345"}), 204)
	e.expect(e.do("GET", "/me", tok, nil), 200)
	e.expect(e.do("GET", "/me", other.str("accessToken"), nil), 401)
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("val")
	base := func() map[string]any {
		return map[string]any{"firstName": "Val", "birthDate": time.Now().AddDate(-30, 0, 0).Format("2006-01-02"),
			"gender": "woman", "bio": "hi", "city": "X", "latitude": e.lat, "longitude": e.lng}
	}
	with := func(k string, v any) map[string]any { m := base(); m[k] = v; return m }

	e.expect(e.do("PUT", "/profile", u.Token, with("birthDate", time.Now().AddDate(-17, 0, 1).Format("2006-01-02"))), 422) // 17 years old
	e.expect(e.do("PUT", "/profile", u.Token, with("birthDate", "01/02/1990")), 400)
	e.expect(e.do("PUT", "/profile", u.Token, with("gender", "robot")), 400)
	e.expect(e.do("PUT", "/profile", u.Token, with("firstName", "   ")), 400)
	e.expect(e.do("PUT", "/profile", u.Token, with("bio", strings.Repeat("a", 501))), 400)
	e.expect(e.do("PUT", "/profile", u.Token, with("latitude", 123.0)), 400)
	e.expect(e.do("PUT", "/profile", u.Token, with("interests", []string{"nonexistent"})), 400)
	e.expect(e.do("PUT", "/profile", u.Token, with("interests", []string{"travel", "cooking", "hiking", "music", "cinema", "reading", "sport", "yoga", "art", "wine", "coffee"})), 400)
	e.expect(e.do("PUT", "/profile", "", base()), 401)

	// Coordinates are stored coarsely (2 decimals) and never returned to others.
	m := with("latitude", 48.856613)
	m["longitude"] = 2.352222
	r := e.expect(e.do("PUT", "/profile", u.Token, m), 200)
	if lat := r.JSON["profile"].(map[string]any)["latitude"].(float64); lat != 48.86 {
		t.Fatalf("latitude should be rounded, got %v", lat)
	}
	if r.JSON["complete"] != false {
		t.Fatal("profile without photo must not be complete")
	}
	// Birth date is locked after first save.
	e.expect(e.do("PUT", "/profile", u.Token, with("birthDate", time.Now().AddDate(-40, 0, 0).Format("2006-01-02"))), 409)
	// Discovery is closed until onboarding is complete.
	e.expect(e.do("GET", "/discover", u.Token, nil), 409)

	// Preferences validation.
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40}), 400)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30}), 400)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30}), 400)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 0}), 400)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{"interestedIn": []string{"man", "man"}, "minAge": 20, "maxAge": 30}), 200)
}

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	me := e.register("dm")
	e.onboard(me, profileOpts{name: "Me", gender: "man", interestedIn: []string{"woman"}, minAge: 25, maxAge: 35, maxDistance: 30})

	young := e.register("dyoung")
	e.onboard(young, profileOpts{name: "Young", gender: "woman", interestedIn: []string{"man"}, birth: time.Now().AddDate(-22, 0, -5).Format("2006-01-02")})
	far := e.register("dfar")
	e.onboard(far, profileOpts{name: "Far", gender: "woman", interestedIn: []string{"man"}, dLat: 0.9}) // ~100 km
	hidden := e.register("dhid")
	e.onboard(hidden, profileOpts{name: "Hidden", gender: "woman", interestedIn: []string{"man"}})
	e.expect(e.do("PUT", "/profile", hidden.Token, map[string]any{
		"firstName": "Hidden", "birthDate": time.Now().AddDate(-28, 0, -10).Format("2006-01-02"), "gender": "woman",
		"city": "T", "latitude": e.lat, "longitude": e.lng, "discoverable": false}), 200)
	picky := e.register("dpicky") // wants women only: I am a man
	e.onboard(picky, profileOpts{name: "Picky", gender: "woman", interestedIn: []string{"woman"}})
	nophoto := e.register("dnophoto")
	e.onboard(nophoto, profileOpts{name: "NoPhoto", gender: "woman", interestedIn: []string{"man"}, noPhoto: true})
	ok := e.register("dok")
	e.onboard(ok, profileOpts{name: "Ok", gender: "woman", interestedIn: []string{"man"}, dLat: 0.1, interests: []string{"yoga"}})

	ids := e.discoverIDs(me)
	if !has(ids, ok.ID) {
		t.Fatalf("eligible profile missing: %v", ids)
	}
	for name, id := range map[string]string{"too young": young.ID, "too far": far.ID, "hidden": hidden.ID, "orientation": picky.ID, "no photo": nophoto.ID, "self": me.ID} {
		if has(ids, id) {
			t.Fatalf("%s profile must be filtered out", name)
		}
	}
	// Hidden profiles cannot be opened or swiped either (server-side, not just UI).
	e.expect(e.do("GET", "/profiles/"+hidden.ID, me.Token, nil), 404)
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"targetId": hidden.ID, "action": "like"}), 409)
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"targetId": me.ID, "action": "like"}), 400)
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"targetId": ok.ID, "action": "super"}), 400)
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"targetId": "not-a-uuid", "action": "like"}), 409)
	// A pass removes the profile for good.
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"targetId": ok.ID, "action": "pass"}), 200)
	if has(e.discoverIDs(me), ok.ID) {
		t.Fatal("passed profile must not come back")
	}
	e.expect(e.do("GET", "/discover?limit=0", me.Token, nil), 400)
	e.expect(e.do("GET", "/discover?limit=21", me.Token, nil), 400)
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	u := e.register("ph")
	other := e.register("ph2")

	// Dangerous or invalid uploads are refused whatever the filename claims.
	e.expect(e.upload("POST", "/photos", u.Token, []byte("<?php system($_GET['c']); ?>"), "shell.jpg"), 415)
	e.expect(e.upload("POST", "/photos", u.Token, []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>"), "x.png"), 415)
	e.expect(e.upload("POST", "/photos", u.Token, append([]byte("\xff\xd8\xff\xe0"), make([]byte, 100)...), "broken.jpg"), 415)
	e.expect(e.upload("POST", "/photos", u.Token, pngBytes(50, 50), "tiny.png"), 400)
	e.expect(e.upload("POST", "/photos", "", pngBytes(400, 400), "a.png"), 401)

	// A valid PNG is re-encoded to a bounded JPEG.
	r1 := e.expect(e.upload("POST", "/photos", u.Token, pngBytes(2000, 1000), "big.png"), 201)
	r2 := e.expect(e.upload("POST", "/photos", u.Token, pngBytes(400, 400), "b.png"), 201)
	r3 := e.expect(e.upload("POST", "/photos", u.Token, pngBytes(400, 400), "c.png"), 201)
	url := r1.str("url")
	req := httptestGet(e, url)
	if req.Code != 200 || req.Header().Get("Content-Type") != "image/jpeg" || !strings.HasPrefix(req.Body.String(), "\xff\xd8\xff") {
		t.Fatalf("photo not served as jpeg: %d %s", req.Code, req.Header().Get("Content-Type"))
	}
	if req.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("nosniff header missing")
	}
	// Tampered or foreign signatures are rejected.
	if c := httptestGet(e, strings.Replace(url, "sig=", "sig=00", 1)).Code; c != 403 {
		t.Fatalf("tampered signature must be 403, got %d", c)
	}
	if c := httptestGet(e, "/photos/"+r2.str("id")+"/file?"+url[strings.Index(url, "?")+1:]).Code; c != 403 {
		t.Fatalf("signature of another photo must be 403, got %d", c)
	}

	// Ownership: nobody else can touch or reorder my photos.
	e.expect(e.do("DELETE", "/photos/"+r1.str("id"), other.Token, nil), 404)
	e.expect(e.upload("PUT", "/photos/"+r1.str("id"), other.Token, pngBytes(400, 400), "x.png"), 404)
	e.expect(e.do("PUT", "/photos/order", other.Token, map[string]any{"ids": []string{r1.str("id")}}), 400)
	e.expect(e.do("PUT", "/photos/order", u.Token, map[string]any{"ids": []string{r1.str("id")}}), 400) // incomplete list

	// Primary = position 0; reorder; replace; delete closes the gap.
	ph := e.expect(e.do("PUT", "/photos/"+r3.str("id")+"/primary", u.Token, nil), 200)
	if ph.JSON["photos"].([]any)[0].(map[string]any)["id"] != r3.str("id") {
		t.Fatalf("primary not first: %s", ph.Raw)
	}
	ph = e.expect(e.do("PUT", "/photos/order", u.Token, map[string]any{"ids": []string{r2.str("id"), r1.str("id"), r3.str("id")}}), 200)
	if ph.JSON["photos"].([]any)[0].(map[string]any)["id"] != r2.str("id") {
		t.Fatalf("reorder failed: %s", ph.Raw)
	}
	e.expect(e.upload("PUT", "/photos/"+r1.str("id"), u.Token, pngBytes(300, 300), "new.png"), 200)
	e.expect(e.do("DELETE", "/photos/"+r2.str("id"), u.Token, nil), 204)
	ph = e.expect(e.do("GET", "/photos", u.Token, nil), 200)
	list := ph.JSON["photos"].([]any)
	if len(list) != 2 || list[0].(map[string]any)["position"].(float64) != 0 || list[1].(map[string]any)["position"].(float64) != 1 {
		t.Fatalf("positions should be compact: %s", ph.Raw)
	}

	// Limit of 6.
	for i := 0; i < 4; i++ {
		e.expect(e.upload("POST", "/photos", u.Token, pngBytes(300, 300), "n.png"), 201)
	}
	e.expect(e.upload("POST", "/photos", u.Token, pngBytes(300, 300), "n.png"), 409)
}
