package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealthAndAuthRequired(t *testing.T) {
	h := newHarness(t)
	h.expect(200, "GET", "/health", "", nil, nil)
	for _, path := range []string{"/me", "/discover", "/matches", "/conversations", "/notifications", "/blocks", "/interests"} {
		if code, _ := h.do("GET", path, "", nil); code != 401 {
			t.Errorf("GET %s without token: expected 401, got %d", path, code)
		}
	}
	if code, _ := h.do("GET", "/me", "not.a.jwt", nil); code != 401 {
		t.Errorf("garbage token must be 401, got %d", code)
	}
}

func TestAuthFlowOverHTTP(t *testing.T) {
	h := newHarness(t)
	h.expect(400, "POST", "/auth/register", "", map[string]string{"email": "bad", "password": "Passw0rd!x"}, nil)
	h.expect(400, "POST", "/auth/register", "", map[string]string{"email": "ok@example.com", "password": "short"}, nil)

	u := h.signup("auth")
	h.expect(409, "POST", "/auth/register", "", map[string]string{"email": strings.ToUpper(u.Email), "password": "Passw0rd!x"}, nil)
	h.expect(401, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "WrongPass1!"}, nil)

	var login struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	h.expect(200, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Passw0rd!x"}, &login)

	var me struct {
		User       struct{ Email string }  `json:"user"`
		Onboarding struct{ Complete bool } `json:"onboarding"`
		Profile    *struct{}               `json:"profile"`
	}
	h.expect(200, "GET", "/me", login.AccessToken, nil, &me)
	if me.User.Email != u.Email || me.Onboarding.Complete || me.Profile != nil {
		t.Fatalf("fresh account should have no profile: %+v", me)
	}

	var refreshed struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	h.expect(200, "POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}, &refreshed)
	h.expect(401, "POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}, nil)

	// Password recovery end to end through the HTTP layer.
	h.expect(202, "POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}, nil)
	i := strings.Index(h.mail.last, ": ")
	code := h.mail.last[i+2 : i+10]
	h.expect(204, "POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Brand-new-pass1"}, nil)
	h.expect(200, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Brand-new-pass1"}, nil)

	h.expect(204, "POST", "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}, nil)
}

func TestLoginRateLimitPerAccount(t *testing.T) {
	h := newHarness(t)
	u := h.signup("rl")
	var last int
	for i := 0; i < 12; i++ {
		last, _ = h.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "WrongPass1!"})
	}
	if last != 429 {
		t.Fatalf("expected 429 after repeated failures, got %d", last)
	}
}

func TestProfileValidation(t *testing.T) {
	h := newHarness(t)
	u := h.signup("pv")
	adult := time.Now().AddDate(-30, 0, 0).Format("2006-01-02")
	base := func() map[string]any {
		return map[string]any{"firstName": "Léa-Rose", "birthdate": adult, "gender": "woman", "bio": "Bonjour"}
	}
	with := func(k string, v any) map[string]any { m := base(); m[k] = v; return m }

	underage := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	code, body := h.do("PUT", "/me/profile", u.Token, with("birthdate", underage))
	if code != 403 || !strings.Contains(string(body), "underage") {
		t.Fatalf("under 18 must be refused: %d %s", code, body)
	}
	for name, bad := range map[string]map[string]any{
		"digits in name":   with("firstName", "R2D2"),
		"empty name":       with("firstName", "   "),
		"bad date":         with("birthdate", "31/12/1990"),
		"too old":          with("birthdate", "1900-01-01"),
		"unknown gender":   with("gender", "alien"),
		"long bio":         with("bio", strings.Repeat("x", 501)),
		"unknown interest": with("interests", []string{"not-a-thing"}),
		"too many":         with("interests", []string{"travel", "cooking", "hiking", "music", "cinema", "reading", "sport", "yoga", "art", "coffee", "wine"}),
		"bad goal":         with("lookingFor", "marriage-now"),
	} {
		if code, _ := h.do("PUT", "/me/profile", u.Token, bad); code != 400 {
			t.Errorf("%s: expected 400, got %d", name, code)
		}
	}

	var me struct {
		Profile struct {
			FirstName string                  `json:"firstName"`
			Age       int                     `json:"age"`
			Interests []struct{ Slug string } `json:"interests"`
		} `json:"profile"`
	}
	h.expect(200, "PUT", "/me/profile", u.Token, with("interests", []string{"travel", "travel", "yoga"}), &me)
	if me.Profile.FirstName != "Léa-Rose" || me.Profile.Age != 30 || len(me.Profile.Interests) != 2 {
		t.Fatalf("unexpected profile: %+v", me.Profile)
	}

	for name, bad := range map[string]map[string]any{
		"no genders":    {"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "isVisible": true, "showDistance": true},
		"min below 18":  {"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 40, "isVisible": true, "showDistance": true},
		"min over max":  {"interestedIn": []string{"man"}, "minAge": 50, "maxAge": 40, "isVisible": true, "showDistance": true},
		"bad distance":  {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 0, "isVisible": true, "showDistance": true},
		"unknown value": {"interestedIn": []string{"robot"}, "minAge": 18, "maxAge": 40, "isVisible": true, "showDistance": true},
	} {
		if code, _ := h.do("PUT", "/me/preferences", u.Token, bad); code != 400 {
			t.Errorf("prefs %s: expected 400, got %d", name, code)
		}
	}
	h.expect(400, "PUT", "/me/location", u.Token, map[string]any{"latitude": 91, "longitude": 0}, nil)
	h.expect(400, "PUT", "/me/location", u.Token, map[string]any{"longitude": 0}, nil)
}

func TestLocationIsSnappedAndDistanceIsCoarse(t *testing.T) {
	h := newHarness(t)
	a := h.onboard(spec{Name: "Ana", Gender: "woman", AgeYears: 28, Lat: f(48.856614), Lng: f(2.352222)})
	b := h.onboard(spec{Name: "Ben", Gender: "man", AgeYears: 30, Lat: f(48.8738), Lng: f(2.2950)})

	var lat, lng float64
	if err := h.pool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be snapped to 0.01 degrees, stored %v,%v", lat, lng)
	}

	var raw map[string]json.RawMessage
	_, body := h.do("GET", "/profiles/"+b.ID, a.Token, nil)
	_ = json.Unmarshal(body, &raw)
	for _, forbidden := range []string{"latitude", "longitude", "birthdate", "email"} {
		if _, leaked := raw[forbidden]; leaked {
			t.Errorf("public profile leaks %q", forbidden)
		}
	}
	var pub card
	_ = json.Unmarshal(body, &pub)
	if pub.DistanceKm == nil || *pub.DistanceKm%5 != 0 {
		t.Fatalf("distance must be bucketed in 5 km steps, got %v", pub.DistanceKm)
	}

	// Hiding distance removes it entirely.
	h.expect(200, "PUT", "/me/preferences", b.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 99, "isVisible": true, "showDistance": false}, nil)
	_, body = h.do("GET", "/profiles/"+b.ID, a.Token, nil)
	_ = json.Unmarshal(body, &pub)
	if pub.DistanceKm != nil {
		t.Fatal("showDistance=false must hide the distance")
	}
}

func TestPhotos(t *testing.T) {
	h := newHarness(t)
	u := h.signup("ph")
	other := h.signup("ph2")

	type photosRes struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	upload := func(data []byte, name, ct string) (int, []byte) {
		return h.upload("POST", "/me/photos", u.Token, data, name, ct)
	}

	// Rejections: content is checked, not the filename or declared type.
	if code, _ := upload([]byte("<?php echo 1; ?>"), "shell.jpg", "image/jpeg"); code != 415 {
		t.Errorf("non-image disguised as jpg: expected 415, got %d", code)
	}
	if code, _ := upload([]byte("GIF89a....."), "a.gif", "image/gif"); code != 415 {
		t.Errorf("gif: expected 415, got %d", code)
	}
	if code, _ := upload(jpegBytes(100, 100), "tiny.jpg", "image/jpeg"); code != 400 {
		t.Errorf("tiny image: expected 400, got %d", code)
	}
	if code, _ := upload(append(jpegBytes(300, 300), make([]byte, 9<<20)...), "big.jpg", "image/jpeg"); code != 413 {
		t.Errorf("oversize: expected 413, got %d", code)
	}
	if code, _ := h.do("POST", "/me/photos", u.Token, map[string]string{"x": "y"}); code != 400 {
		t.Errorf("missing file: expected 400, got %d", code)
	}

	var first photosRes
	code, body := upload(jpegBytes(2000, 1500), "big.jpg", "image/jpeg")
	if code != 201 {
		t.Fatalf("upload: %d %s", code, body)
	}
	_ = json.Unmarshal(body, &first)
	if len(first.Photos) != 1 {
		t.Fatalf("expected 1 photo, got %+v", first)
	}

	// Stored file is a downscaled JPEG, served only to authorised viewers.
	req := httptest.NewRequest("GET", "/photos/"+first.Photos[0].ID, nil)
	req.Header.Set("Authorization", "Bearer "+u.Token)
	status, img, hdr := h.raw(req)
	if status != 200 || hdr.Get("Content-Type") != "image/jpeg" || hdr.Get("Cache-Control") == "" || !strings.HasPrefix(hdr.Get("Cache-Control"), "private") {
		t.Fatalf("serve own photo: %d %v", status, hdr)
	}
	if len(img) >= len(jpegBytes(2000, 1500)) {
		t.Fatal("served image should be recompressed/downscaled")
	}
	if code, _ := h.do("GET", "/photos/"+first.Photos[0].ID+"?size=thumb", u.Token, nil); code != 200 {
		t.Fatalf("thumb: %d", code)
	}
	if code, _ := h.do("GET", "/photos/"+first.Photos[0].ID, "", nil); code != 401 {
		t.Fatalf("photos require auth, got %d", code)
	}
	// No profile yet => not discoverable => other users cannot fetch it.
	if code, _ := h.do("GET", "/photos/"+first.Photos[0].ID, other.Token, nil); code != 404 {
		t.Fatalf("photo of a user without a profile must be hidden, got %d", code)
	}
	if code, _ := h.do("GET", "/photos/not-a-uuid", u.Token, nil); code != 404 {
		t.Fatalf("malformed id: %d", code)
	}

	// Cannot delete the only photo.
	h.expect(409, "DELETE", "/me/photos/"+first.Photos[0].ID, u.Token, nil, nil)

	// Fill to the limit of 6.
	for i := 0; i < 5; i++ {
		if code, body := upload(jpegBytes(400, 400), "p.jpg", "image/jpeg"); code != 201 {
			t.Fatalf("upload %d: %d %s", i, code, body)
		}
	}
	if code, _ := upload(jpegBytes(400, 400), "p.jpg", "image/jpeg"); code != 409 {
		t.Fatalf("7th photo must be refused, got %d", code)
	}
}

func TestPhotoOrderReplaceDeleteAndFiles(t *testing.T) {
	h := newHarness(t)
	u := h.signup("po")
	type photosRes struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	var res photosRes
	for i := 0; i < 3; i++ {
		code, body := h.upload("POST", "/me/photos", u.Token, jpegBytes(400+i*10, 400), "p.jpg", "image/jpeg")
		if code != 201 {
			t.Fatalf("upload: %d %s", code, body)
		}
		_ = json.Unmarshal(body, &res)
	}
	a, b, c := res.Photos[0].ID, res.Photos[1].ID, res.Photos[2].ID

	// Make the last photo the main one.
	h.expect(200, "PUT", "/me/photos/order", u.Token, map[string]any{"ids": []string{c, a, b}}, &res)
	if res.Photos[0].ID != c || res.Photos[1].ID != a || res.Photos[2].ID != b {
		t.Fatalf("order not applied: %+v", res)
	}
	// Order must name each photo exactly once.
	h.expect(400, "PUT", "/me/photos/order", u.Token, map[string]any{"ids": []string{c, a}}, nil)
	h.expect(400, "PUT", "/me/photos/order", u.Token, map[string]any{"ids": []string{c, c, a}}, nil)

	// Another user cannot touch these photos.
	intruder := h.signup("intruder")
	h.expect(404, "DELETE", "/me/photos/"+a, intruder.Token, nil, nil)
	if code, _ := h.upload("PUT", "/me/photos/"+a, intruder.Token, jpegBytes(400, 400), "x.jpg", "image/jpeg"); code != 404 {
		t.Fatalf("replace by non-owner: %d", code)
	}
	h.expect(400, "PUT", "/me/photos/order", intruder.Token, map[string]any{"ids": []string{a, b, c}}, nil)

	// Replace keeps the slot but issues a new id and removes the old files.
	filesBefore := countFiles(t, h.uploadDir)
	code, body := h.upload("PUT", "/me/photos/"+c, u.Token, jpegBytes(500, 500), "n.jpg", "image/jpeg")
	if code != 200 {
		t.Fatalf("replace: %d %s", code, body)
	}
	_ = json.Unmarshal(body, &res)
	if res.Photos[0].ID == c || len(res.Photos) != 3 {
		t.Fatalf("replace should keep position 0 with a new id: %+v", res)
	}
	if got := countFiles(t, h.uploadDir); got != filesBefore {
		t.Fatalf("replace leaked files: before %d after %d", filesBefore, got)
	}

	h.expect(200, "DELETE", "/me/photos/"+a, u.Token, nil, &res)
	if len(res.Photos) != 2 || res.Photos[0].Position != 0 || res.Photos[1].Position != 1 {
		t.Fatalf("positions must be compacted: %+v", res)
	}
	if got := countFiles(t, h.uploadDir); got != 4 {
		t.Fatalf("expected 2 photos x (full+thumb) on disk, found %d files", got)
	}
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jpg" {
			n++
		}
	}
	return n
}

func TestDiscoveryRequiresCompleteProfile(t *testing.T) {
	h := newHarness(t)
	u := h.signup("dr")
	h.expect(409, "GET", "/discover", u.Token, nil, nil)
	noPhoto := h.onboard(spec{Name: "Nina", Gender: "woman", AgeYears: 25, NoPhoto: true})
	h.expect(409, "GET", "/discover", noPhoto.Token, nil, nil)
	target := h.onboard(spec{Name: "Tom", Gender: "man", AgeYears: 25})
	h.expect(409, "POST", "/swipes", noPhoto.Token, map[string]string{"userId": target.ID, "action": "like"}, nil)
	h.expect(400, "GET", "/discover?limit=500", target.Token, nil, nil)
}

func TestDiscoveryFilters(t *testing.T) {
	h := newHarness(t)
	paris := spec{Lat: f(48.8566), Lng: f(2.3522)}

	me := h.onboard(spec{Name: "Me", Gender: "woman", AgeYears: 30, InterestedIn: []string{"man"}, MinAge: 25, MaxAge: 40, MaxDistance: ptr(50), Lat: paris.Lat, Lng: paris.Lng})
	ok := h.onboard(spec{Name: "Okay", Gender: "man", AgeYears: 33, InterestedIn: []string{"woman"}, Lat: f(48.87), Lng: f(2.30)})
	wrongGender := h.onboard(spec{Name: "Wg", Gender: "woman", AgeYears: 33, Lat: paris.Lat, Lng: paris.Lng})
	tooYoung := h.onboard(spec{Name: "Young", Gender: "man", AgeYears: 22, Lat: paris.Lat, Lng: paris.Lng})
	tooOld := h.onboard(spec{Name: "Old", Gender: "man", AgeYears: 55, Lat: paris.Lat, Lng: paris.Lng})
	tooFar := h.onboard(spec{Name: "Far", Gender: "man", AgeYears: 33, Lat: f(45.76), Lng: f(4.84)})
	notInterested := h.onboard(spec{Name: "Ni", Gender: "man", AgeYears: 33, InterestedIn: []string{"man"}, Lat: paris.Lat, Lng: paris.Lng})
	theyWantYounger := h.onboard(spec{Name: "Ty", Gender: "man", AgeYears: 33, InterestedIn: []string{"woman"}, MinAge: 18, MaxAge: 25, Lat: paris.Lat, Lng: paris.Lng})
	paused := h.onboard(spec{Name: "Paused", Gender: "man", AgeYears: 33, Lat: paris.Lat, Lng: paris.Lng})
	h.expect(200, "PUT", "/me/preferences", paused.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 99, "isVisible": false, "showDistance": true}, nil)
	noPhoto := h.onboard(spec{Name: "Nophoto", Gender: "man", AgeYears: 33, NoPhoto: true})
	noLocation := h.onboard(spec{Name: "Noloc", Gender: "man", AgeYears: 33})

	got := h.discover(me)
	if !contains(got, ok.ID) || !contains(got, noLocation.ID) {
		t.Fatalf("expected compatible users, got %v", ids(got))
	}
	for name, u := range map[string]user{"gender": wrongGender, "too young": tooYoung, "too old": tooOld, "too far": tooFar,
		"they dislike my gender": notInterested, "outside their age range": theyWantYounger, "paused": paused, "no photo": noPhoto, "self": me} {
		if contains(got, u.ID) {
			t.Errorf("%s must be excluded from discovery", name)
		}
	}
	for _, c := range got {
		if c.Age < 25 || c.Age > 40 {
			t.Errorf("age %d outside range", c.Age)
		}
	}

	// Treated profiles never come back; pass and like behave the same for the deck.
	h.swipe(me, ok, "pass")
	if contains(h.discover(me), ok.ID) {
		t.Fatal("a passed profile must not reappear")
	}
	h.swipe(me, noLocation, "like")
	if contains(h.discover(me), noLocation.ID) {
		t.Fatal("a liked profile must not reappear")
	}
}

func ptr(v int) *int { return &v }

func TestRankingPrefersSharedInterestsAndAdmirers(t *testing.T) {
	h := newHarness(t)
	me := h.onboard(spec{Name: "Me", Gender: "woman", AgeYears: 30, Interests: []string{"travel", "music", "yoga"}})
	none := h.onboard(spec{Name: "None", Gender: "man", AgeYears: 30, Interests: []string{"gaming"}})
	two := h.onboard(spec{Name: "Two", Gender: "man", AgeYears: 30, Interests: []string{"travel", "music"}})
	admirer := h.onboard(spec{Name: "Admirer", Gender: "man", AgeYears: 30, Interests: []string{"gaming"}})
	h.swipe(admirer, me, "like")

	got := ids(h.discover(me))
	if len(got) != 3 || got[0] != admirer.ID || got[1] != two.ID || got[2] != none.ID {
		t.Fatalf("expected admirer, shared-interests, others; got %v (admirer=%s two=%s none=%s)", got, admirer.ID, two.ID, none.ID)
	}
}

func TestMatchingRules(t *testing.T) {
	h := newHarness(t)
	a := h.onboard(spec{Name: "Alice", Gender: "woman", AgeYears: 28})
	b := h.onboard(spec{Name: "Bob", Gender: "man", AgeYears: 30})
	c := h.onboard(spec{Name: "Carl", Gender: "man", AgeYears: 30})

	if r := h.swipe(a, b, "like"); r.Matched {
		t.Fatal("one-sided like must not match")
	}
	// Repeating the same swipe is idempotent.
	if r := h.swipe(a, b, "like"); r.Matched || !r.AlreadyDone {
		t.Fatalf("repeat swipe should be flagged as already processed: %+v", r)
	}
	// A pass is final: a later like on the same profile does not flip it.
	h.swipe(a, c, "pass")
	if r := h.swipe(a, c, "like"); !r.AlreadyDone || r.Matched {
		t.Fatalf("pass then like must not change the decision: %+v", r)
	}
	if r := h.swipe(c, a, "like"); r.Matched {
		t.Fatal("a passed profile must not create a match")
	}

	r := h.swipe(b, a, "like")
	if !r.Matched || r.MatchID == "" || r.ConversationID == "" {
		t.Fatalf("mutual like must match: %+v", r)
	}
	again := h.swipe(b, a, "like")
	if again.MatchID != r.MatchID || !again.AlreadyDone {
		t.Fatalf("repeat must return the same match: %+v vs %+v", again, r)
	}
	// Exactly one match / one conversation / two participants.
	var matches, convs, parts int
	_ = h.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM matches), (SELECT count(*) FROM conversations), (SELECT count(*) FROM conversation_participants)`).Scan(&matches, &convs, &parts)
	if matches != 1 || convs != 1 || parts != 2 {
		t.Fatalf("expected 1/1/2, got %d/%d/%d", matches, convs, parts)
	}

	// Both sides see the match; each got a notification.
	for _, u := range []user{a, b} {
		var res struct {
			Matches []struct {
				ID          string `json:"id"`
				HasMessages bool   `json:"hasMessages"`
				User        card   `json:"user"`
			} `json:"matches"`
		}
		h.expect(200, "GET", "/matches", u.Token, nil, &res)
		if len(res.Matches) != 1 || res.Matches[0].ID != r.MatchID {
			t.Fatalf("matches list: %+v", res)
		}
		var n struct {
			Notifications []struct {
				ID, Type string
				IsRead   bool `json:"isRead"`
			} `json:"notifications"`
		}
		h.expect(200, "GET", "/notifications", u.Token, nil, &n)
		if len(n.Notifications) != 1 || n.Notifications[0].Type != "match" || n.Notifications[0].IsRead {
			t.Fatalf("notifications: %+v", n)
		}
		var cnt struct{ Unread int }
		h.expect(200, "GET", "/notifications/unread-count", u.Token, nil, &cnt)
		if cnt.Unread != 1 {
			t.Fatalf("unread count %d", cnt.Unread)
		}
		h.expect(204, "POST", "/notifications/"+n.Notifications[0].ID+"/read", u.Token, nil, nil)
		h.expect(200, "GET", "/notifications/unread-count", u.Token, nil, &cnt)
		if cnt.Unread != 0 {
			t.Fatal("notification should be read")
		}
	}

	// Rules on the endpoint itself.
	h.expect(400, "POST", "/swipes", a.Token, map[string]string{"userId": a.ID, "action": "like"}, nil)
	h.expect(400, "POST", "/swipes", a.Token, map[string]string{"userId": b.ID, "action": "superlike"}, nil)
	h.expect(400, "POST", "/swipes", a.Token, map[string]string{"userId": "nope", "action": "like"}, nil)
	h.expect(404, "POST", "/swipes", a.Token, map[string]string{"userId": "11111111-1111-4111-8111-111111111111", "action": "like"}, nil)
}

func TestSwipeCannotBypassPreferences(t *testing.T) {
	h := newHarness(t)
	a := h.onboard(spec{Name: "Alice", Gender: "woman", AgeYears: 28, InterestedIn: []string{"woman"}})
	b := h.onboard(spec{Name: "Bob", Gender: "man", AgeYears: 30})
	h.expect(404, "POST", "/swipes", a.Token, map[string]string{"userId": b.ID, "action": "like"}, nil)
	h.expect(404, "POST", "/swipes", b.Token, map[string]string{"userId": a.ID, "action": "like"}, nil)
}

func TestSimultaneousMutualLikesCreateOneMatch(t *testing.T) {
	h := newHarness(t)
	for round := 0; round < 5; round++ {
		a := h.onboard(spec{Name: "Ann", Gender: "woman", AgeYears: 28})
		b := h.onboard(spec{Name: "Bo", Gender: "man", AgeYears: 30})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				code, body := h.do("POST", "/swipes", pair[0].Token, map[string]string{"userId": pair[1].ID, "action": "like"})
				if code != 200 {
					t.Errorf("swipe: %d %s", code, body)
					return
				}
				_ = json.Unmarshal(body, &results[i])
			}()
		}
		wg.Wait()
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: simultaneous mutual likes produced no match", round)
		}
		var n int
		_ = h.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly 1 match, got %d", round, n)
		}
	}
}

type msg struct {
	ID       string    `json:"id"`
	SenderID string    `json:"senderId"`
	Body     string    `json:"body"`
	Created  time.Time `json:"createdAt"`
}

func matched(t *testing.T, h *harness) (a, b user, conv string, matchID string) {
	t.Helper()
	a = h.onboard(spec{Name: "Alice", Gender: "woman", AgeYears: 28})
	b = h.onboard(spec{Name: "Bob", Gender: "man", AgeYears: 30})
	h.swipe(a, b, "like")
	r := h.swipe(b, a, "like")
	return a, b, r.ConversationID, r.MatchID
}

func TestChatFlowAndPermissions(t *testing.T) {
	h := newHarness(t)
	a, b, conv, _ := matched(t, h)
	stranger := h.onboard(spec{Name: "Eve", Gender: "woman", AgeYears: 28})

	// Matches without messages do not clutter the conversation list.
	var list struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage msg    `json:"lastMessage"`
			User        card   `json:"user"`
		} `json:"conversations"`
	}
	h.expect(200, "GET", "/conversations", a.Token, nil, &list)
	if len(list.Conversations) != 0 {
		t.Fatalf("no messages yet, expected empty list: %+v", list)
	}

	// Non-participants get 404 everywhere, indistinguishable from a missing conversation.
	h.expect(404, "GET", "/conversations/"+conv+"/messages", stranger.Token, nil, nil)
	h.expect(404, "POST", "/conversations/"+conv+"/messages", stranger.Token, map[string]string{"body": "hi"}, nil)
	h.expect(404, "POST", "/conversations/"+conv+"/read", stranger.Token, nil, nil)
	h.expect(404, "DELETE", "/conversations/"+conv, stranger.Token, nil, nil)
	h.expect(404, "GET", "/conversations/11111111-1111-4111-8111-111111111111/messages", a.Token, nil, nil)

	// Message validation.
	h.expect(400, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "   "}, nil)
	h.expect(400, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": strings.Repeat("é", 2001)}, nil)

	var m1 msg
	h.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "  Salut Bob !  "}, &m1)
	if m1.Body != "Salut Bob !" || m1.SenderID != a.ID {
		t.Fatalf("unexpected message: %+v", m1)
	}
	h.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Ça va ?"}, nil)

	h.expect(200, "GET", "/conversations", b.Token, nil, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 2 || list.Conversations[0].LastMessage.Body != "Ça va ?" || list.Conversations[0].User.ID != a.ID {
		t.Fatalf("recipient view: %+v", list)
	}
	h.expect(200, "GET", "/conversations", a.Token, nil, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 0 {
		t.Fatalf("sender has no unread: %+v", list)
	}
	var unread struct{ UnreadMessages, UnreadConversations int }
	h.expect(200, "GET", "/conversations/unread", b.Token, nil, &unread)
	if unread.UnreadMessages != 2 || unread.UnreadConversations != 1 {
		t.Fatalf("unread totals: %+v", unread)
	}

	// Read state: sender sees the receipt only after the other reads.
	var history struct {
		Messages        []msg     `json:"messages"`
		OtherLastReadAt time.Time `json:"otherLastReadAt"`
		NextCursor      string    `json:"nextCursor"`
	}
	h.expect(200, "GET", "/conversations/"+conv+"/messages", a.Token, nil, &history)
	if len(history.Messages) != 2 || history.Messages[0].Body != "Ça va ?" {
		t.Fatalf("history newest first: %+v", history)
	}
	if !history.Messages[1].Created.After(history.OtherLastReadAt) {
		t.Fatal("message must be unread by the other side before they open it")
	}
	h.expect(204, "POST", "/conversations/"+conv+"/read", b.Token, nil, nil)
	h.expect(200, "GET", "/conversations/"+conv+"/messages", a.Token, nil, &history)
	if history.Messages[0].Created.After(history.OtherLastReadAt) {
		t.Fatal("message should now count as read")
	}
	h.expect(200, "GET", "/conversations/unread", b.Token, nil, &unread)
	if unread.UnreadMessages != 0 {
		t.Fatalf("unread after read: %+v", unread)
	}

	// Pagination with cursor.
	for i := 0; i < 5; i++ {
		h.expect(201, "POST", "/conversations/"+conv+"/messages", b.Token, map[string]string{"body": "m"}, nil)
	}
	var page1, page2 struct {
		Messages   []msg  `json:"messages"`
		NextCursor string `json:"nextCursor"`
	}
	h.expect(200, "GET", "/conversations/"+conv+"/messages?limit=4", a.Token, nil, &page1)
	if len(page1.Messages) != 4 || page1.NextCursor == "" {
		t.Fatalf("page1: %+v", page1)
	}
	h.expect(200, "GET", "/conversations/"+conv+"/messages?limit=4&cursor="+page1.NextCursor, a.Token, nil, &page2)
	if len(page2.Messages) != 3 || page2.NextCursor != "" || page2.Messages[0].ID == page1.Messages[3].ID {
		t.Fatalf("page2: %+v", page2)
	}
	h.expect(400, "GET", "/conversations/"+conv+"/messages?cursor=garbage", a.Token, nil, nil)

	// Local deletion hides it for one side only, and a new message brings it back.
	h.expect(204, "DELETE", "/conversations/"+conv, a.Token, nil, nil)
	h.expect(200, "GET", "/conversations", a.Token, nil, &list)
	if len(list.Conversations) != 0 {
		t.Fatal("hidden conversation must disappear for its owner")
	}
	h.expect(200, "GET", "/conversations", b.Token, nil, &list)
	if len(list.Conversations) != 1 {
		t.Fatal("other side keeps the conversation")
	}
	h.expect(201, "POST", "/conversations/"+conv+"/messages", b.Token, map[string]string{"body": "Tu es là ?"}, nil)
	h.expect(200, "GET", "/conversations", a.Token, nil, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 1 {
		t.Fatalf("new message should resurface the conversation: %+v", list)
	}
}

func TestRealtimeDelivery(t *testing.T) {
	h := newHarness(t)
	a, b, conv, _ := matched(t, h)
	srv := httptest.NewServer(h.app.Engine)
	defer srv.Close()

	var ticket struct {
		Ticket string `json:"ticket"`
	}
	h.expect(200, "POST", "/ws/ticket", b.Token, nil, &ticket)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?ticket="
	conn, _, err := websocket.DefaultDialer.Dial(url+ticket.Ticket, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Tickets are single-use and the endpoint needs one.
	if _, resp, err := websocket.DefaultDialer.Dial(url+ticket.Ticket, nil); err == nil || resp == nil || resp.StatusCode != 401 {
		t.Fatalf("reused ticket must be refused: %v %v", err, resp)
	}
	if _, resp, err := websocket.DefaultDialer.Dial(url+"forged", nil); err == nil || resp == nil || resp.StatusCode != 401 {
		t.Fatalf("forged ticket must be refused")
	}
	// A browser from an unlisted origin cannot open the socket even with a ticket.
	h.expect(200, "POST", "/ws/ticket", b.Token, nil, &ticket)
	if _, _, err := websocket.DefaultDialer.Dial(url+ticket.Ticket, http.Header{"Origin": {"https://evil.test"}}); err == nil {
		t.Fatal("foreign origin must be refused")
	}

	next := func() (typ string, data json.RawMessage) {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read event: %v", err)
		}
		return ev.Type, ev.Data
	}

	h.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "en direct"}, nil)
	typ, data := next()
	var m msg
	_ = json.Unmarshal(data, &m)
	if typ != "message.new" || m.Body != "en direct" || m.SenderID != a.ID {
		t.Fatalf("expected message.new, got %s %s", typ, data)
	}

	// Alice reading triggers a read receipt event for Bob only when Bob reads Alice's... verify the reverse direction.
	h.expect(204, "POST", "/conversations/"+conv+"/read", a.Token, nil, nil)
	typ, _ = next()
	if typ != "conversation.read" {
		t.Fatalf("expected conversation.read, got %s", typ)
	}

	// A third user receives nothing from this conversation.
	c := h.onboard(spec{Name: "Cleo", Gender: "woman", AgeYears: 28})
	h.expect(200, "POST", "/ws/ticket", c.Token, nil, &ticket)
	cc, _, err := websocket.DefaultDialer.Dial(url+ticket.Ticket, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cc.Close()
	h.expect(201, "POST", "/conversations/"+conv+"/messages", b.Token, map[string]string{"body": "privé"}, nil)
	_ = cc.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	if _, _, err := cc.ReadMessage(); err == nil {
		t.Fatal("an outsider received a conversation event")
	}
}

func TestBlockDissolvesMatchAndHidesUsers(t *testing.T) {
	h := newHarness(t)
	a, b, conv, matchID := matched(t, h)
	h.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "hello"}, nil)

	h.expect(400, "POST", "/blocks", a.Token, map[string]string{"userId": a.ID}, nil)
	h.expect(404, "POST", "/blocks", a.Token, map[string]string{"userId": "11111111-1111-4111-8111-111111111111"}, nil)
	h.expect(204, "POST", "/blocks", a.Token, map[string]string{"userId": b.ID}, nil)
	h.expect(204, "POST", "/blocks", a.Token, map[string]string{"userId": b.ID}, nil) // idempotent

	// Match, conversation, messages and the match notification are gone for both.
	for _, u := range []user{a, b} {
		var ms struct{ Matches []any }
		h.expect(200, "GET", "/matches", u.Token, nil, &ms)
		if len(ms.Matches) != 0 {
			t.Fatal("block must dissolve the match")
		}
		h.expect(404, "GET", "/conversations/"+conv+"/messages", u.Token, nil, nil)
		var n struct{ Notifications []any }
		h.expect(200, "GET", "/notifications", u.Token, nil, &n)
		if len(n.Notifications) != 0 {
			t.Fatal("match notification must be removed")
		}
	}
	var count int
	_ = h.pool.QueryRow(t.Context(), `SELECT count(*) FROM messages`).Scan(&count)
	if count != 0 {
		t.Fatal("messages must be deleted with the match")
	}
	_ = matchID

	// Neither can see the other: profile, photos, discovery, swipes.
	h.expect(404, "GET", "/profiles/"+b.ID, a.Token, nil, nil)
	h.expect(404, "GET", "/profiles/"+a.ID, b.Token, nil, nil)
	var bProfile struct{ Photos []struct{ URL string } }
	_ = bProfile
	var photoID string
	_ = h.pool.QueryRow(t.Context(), `SELECT id FROM photos WHERE user_id = $1`, b.ID).Scan(&photoID)
	h.expect(404, "GET", "/photos/"+photoID, a.Token, nil, nil)
	if contains(h.discover(a), b.ID) || contains(h.discover(b), a.ID) {
		t.Fatal("blocked users must not appear in each other's deck")
	}
	h.expect(404, "POST", "/swipes", b.Token, map[string]string{"userId": a.ID, "action": "like"}, nil)

	var bl struct {
		Blocked []struct{ UserID, FirstName string } `json:"blocked"`
	}
	h.expect(200, "GET", "/blocks", a.Token, nil, &bl)
	if len(bl.Blocked) != 1 || bl.Blocked[0].UserID != b.ID || bl.Blocked[0].FirstName != "Bob" {
		t.Fatalf("blocked list: %+v", bl)
	}
	// Unblocking restores visibility but never revives the deck entry or the match.
	h.expect(204, "DELETE", "/blocks/"+b.ID, a.Token, nil, nil)
	h.expect(200, "GET", "/profiles/"+b.ID, a.Token, nil, nil)
	if contains(h.discover(a), b.ID) {
		t.Fatal("unblocked user should not reappear in the deck")
	}
}

func TestUnmatch(t *testing.T) {
	h := newHarness(t)
	a, b, conv, matchID := matched(t, h)
	stranger := h.onboard(spec{Name: "Eve", Gender: "woman", AgeYears: 28})
	h.expect(404, "DELETE", "/matches/"+matchID, stranger.Token, nil, nil)
	h.expect(204, "DELETE", "/matches/"+matchID, a.Token, nil, nil)
	h.expect(404, "DELETE", "/matches/"+matchID, a.Token, nil, nil)
	h.expect(404, "GET", "/conversations/"+conv+"/messages", b.Token, nil, nil)
	if contains(h.discover(a), b.ID) {
		t.Fatal("unmatched user must not return to the deck")
	}
}

func TestReports(t *testing.T) {
	h := newHarness(t)
	a, b, _, _ := matched(t, h)
	h.expect(400, "POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "dislike"}, nil)
	h.expect(400, "POST", "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"}, nil)
	h.expect(404, "POST", "/reports", a.Token, map[string]any{"userId": "11111111-1111-4111-8111-111111111111", "reason": "spam"}, nil)
	h.expect(202, "POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "  rude  "}, nil)
	h.expect(202, "POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam"}, nil) // duplicate is absorbed

	var n int
	var details, status string
	_ = h.pool.QueryRow(t.Context(), `SELECT count(*), max(details), max(status) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n, &details, &status)
	if n != 1 || details != "rude" || status != "open" {
		t.Fatalf("report rows: n=%d details=%q status=%q", n, details, status)
	}

	// Report + block in one step.
	c := h.onboard(spec{Name: "Cleo", Gender: "woman", AgeYears: 28})
	h.expect(202, "POST", "/reports", c.Token, map[string]any{"userId": b.ID, "reason": "fake_profile", "alsoBlock": true}, nil)
	h.expect(404, "GET", "/profiles/"+b.ID, c.Token, nil, nil)
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	h := newHarness(t)
	a, b, conv, _ := matched(t, h)
	h.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "bye"}, nil)
	h.expect(202, "POST", "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "spam"}, nil)
	h.expect(202, "POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam"}, nil)
	if got := countFiles(t, h.uploadDir); got != 4 {
		t.Fatalf("setup: expected 4 files, got %d", got)
	}

	h.expect(403, "DELETE", "/me", a.Token, map[string]string{"password": "wrong-Pass1"}, nil)
	h.expect(204, "DELETE", "/me", a.Token, map[string]string{"password": "Passw0rd!x"}, nil)

	for table, col := range map[string]string{
		"users": "id", "profiles": "user_id", "preferences": "user_id", "photos": "user_id", "sessions": "user_id",
		"swipes": "from_user_id", "user_interests": "user_id", "notifications": "user_id", "conversation_participants": "user_id",
	} {
		var n int
		_ = h.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n)
		if n != 0 {
			t.Errorf("%s still holds %d rows for the deleted user", table, n)
		}
	}
	var n int
	_ = h.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM swipes WHERE to_user_id = $1) + (SELECT count(*) FROM matches) + (SELECT count(*) FROM messages) + (SELECT count(*) FROM conversations)`, a.ID).Scan(&n)
	if n != 0 {
		t.Errorf("shared rows remain after deletion: %d", n)
	}
	if got := countFiles(t, h.uploadDir); got != 2 {
		t.Errorf("photo files of the deleted user must be removed; %d files remain (expected Bob's 2)", got)
	}
	// Report filed against the deleted user survives, detached, for moderators; reports they filed are gone.
	var kept, filedByDeleted int
	_ = h.pool.QueryRow(t.Context(), `SELECT count(*) FROM reports WHERE reported_id IS NULL`).Scan(&kept)
	_ = h.pool.QueryRow(t.Context(), `SELECT count(*) FROM reports WHERE reporter_id = $1`, a.ID).Scan(&filedByDeleted)
	if kept != 1 || filedByDeleted != 0 {
		t.Errorf("report retention: kept=%d filedByDeleted=%d", kept, filedByDeleted)
	}

	// The old token no longer opens anything, login is impossible, Bob is unaffected.
	h.expect(401, "GET", "/me", a.Token, nil, nil)
	h.expect(401, "POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Passw0rd!x"}, nil)
	h.expect(401, "POST", "/auth/refresh", "", map[string]string{"refreshToken": a.Refresh}, nil)
	h.expect(200, "GET", "/me", b.Token, nil, nil)
	h.expect(404, "GET", "/profiles/"+a.ID, b.Token, nil, nil)
}

func TestChangePasswordKeepsCurrentSessionOnly(t *testing.T) {
	h := newHarness(t)
	u := h.signup("cp")
	var second struct {
		RefreshToken string `json:"refreshToken"`
		AccessToken  string `json:"accessToken"`
	}
	h.expect(200, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Passw0rd!x"}, &second)

	h.expect(403, "POST", "/me/password", u.Token, map[string]string{"currentPassword": "Nope-wrong1", "newPassword": "Another-pass1"}, nil)
	h.expect(400, "POST", "/me/password", u.Token, map[string]string{"currentPassword": "Passw0rd!x", "newPassword": "weak"}, nil)
	h.expect(204, "POST", "/me/password", u.Token, map[string]string{"currentPassword": "Passw0rd!x", "newPassword": "Another-pass1"}, nil)

	h.expect(401, "POST", "/auth/refresh", "", map[string]string{"refreshToken": second.RefreshToken}, nil)
	h.expect(200, "POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}, nil)
}

func TestMalformedAndOversizedInputs(t *testing.T) {
	h := newHarness(t)
	u := h.signup("mal")
	for _, path := range []string{"/profiles/xyz", "/photos/xyz", "/conversations/xyz/messages", "/matches/xyz"} {
		method := "GET"
		if strings.HasPrefix(path, "/matches") {
			method = "DELETE"
		}
		if code, _ := h.do(method, path, u.Token, nil); code != 404 {
			t.Errorf("%s %s: expected 404, got %d", method, path, code)
		}
	}
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"`+strings.Repeat("a", 2<<20)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	if code, _, _ := h.raw(req); code != 400 {
		t.Errorf("oversized JSON body should be rejected with 400, got %d", code)
	}
	req = httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	if code, _, _ := h.raw(req); code != 400 {
		t.Errorf("invalid JSON: %d", code)
	}
}

func TestSecurityHeadersAndTokenValidation(t *testing.T) {
	h := newHarness(t)
	_, _, hdr := h.raw(httptest.NewRequest("GET", "/health", nil))
	if hdr.Get("X-Content-Type-Options") != "nosniff" || hdr.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing security headers: %v", hdr)
	}
	// A token signed with another secret, or with alg=none, is rejected.
	forged := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiIxMSIsInNpZCI6IjIyIiwiZXhwIjo5OTk5OTk5OTk5fQ."
	if code, _ := h.do("GET", "/me", forged, nil); code != 401 {
		t.Fatalf("alg=none token accepted: %d", code)
	}
}
