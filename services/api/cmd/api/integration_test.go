package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
}

func (e *env) discover(u testUser) []card {
	e.t.Helper()
	var res struct {
		Profiles []card `json:"profiles"`
	}
	e.call("GET", "/discover?limit=20", nil, u.Token, 200, &res)
	return res.Profiles
}

func contains(cs []card, id string) bool {
	for _, c := range cs {
		if c.UserID == id {
			return true
		}
	}
	return false
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.call("GET", "/health", nil, "", 200, nil)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail("auth")
	e.emails = append(e.emails, email)
	body := map[string]any{"email": "  " + strings.ToUpper(email) + " ", "password": "correct-horse", "firstName": "Ana", "birthDate": "1990-01-01"}

	var reg struct {
		AccessToken  string                 `json:"accessToken"`
		RefreshToken string                 `json:"refreshToken"`
		User         struct{ Email string } `json:"user"`
	}
	e.call("POST", "/auth/register", body, "", 201, &reg)
	if reg.User.Email != email {
		t.Fatalf("email must be trimmed and lower-cased, got %q", reg.User.Email)
	}
	e.call("POST", "/auth/register", body, "", 409, nil)

	// Validation.
	for name, payload := range map[string]map[string]any{
		"underage":      {"email": uniqueEmail("kid"), "password": "correct-horse", "firstName": "Kid", "birthDate": time.Now().AddDate(-17, 0, 0).Format("2006-01-02")},
		"short pwd":     {"email": uniqueEmail("p"), "password": "short", "firstName": "A", "birthDate": "1990-01-01"},
		"bad email":     {"email": "nope", "password": "correct-horse", "firstName": "A", "birthDate": "1990-01-01"},
		"bad birthdate": {"email": uniqueEmail("b"), "password": "correct-horse", "firstName": "A", "birthDate": "01/01/1990"},
		"long pwd":      {"email": uniqueEmail("l"), "password": strings.Repeat("x", 80), "firstName": "A", "birthDate": "1990-01-01"},
	} {
		status, _ := e.do("POST", "/auth/register", payload, "")
		if status != 400 && status != 403 {
			t.Errorf("%s: expected 400/403, got %d", name, status)
		}
	}
	// A user under 18 gets exactly 403.
	status, _ := e.do("POST", "/auth/register", map[string]any{"email": uniqueEmail("kid"), "password": "correct-horse", "firstName": "Kid", "birthDate": time.Now().AddDate(-17, 0, 0).Format("2006-01-02")}, "")
	if status != 403 {
		t.Errorf("underage expected 403, got %d", status)
	}

	e.call("GET", "/me", nil, "", 401, nil)
	e.call("GET", "/me", nil, "garbage", 401, nil)
	e.call("GET", "/me", nil, reg.AccessToken, 200, nil)

	e.call("POST", "/auth/login", map[string]any{"email": email, "password": "wrong-password"}, "", 401, nil)
	e.call("POST", "/auth/login", map[string]any{"email": "ghost@test.invalid", "password": "whatever-123"}, "", 401, nil)
	var login struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	e.call("POST", "/auth/login", map[string]any{"email": email, "password": "correct-horse"}, "", 200, &login)

	// Refresh rotates: the old refresh token stops working.
	var refreshed struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	e.call("POST", "/auth/refresh", map[string]any{"refreshToken": login.RefreshToken}, "", 200, &refreshed)
	e.call("POST", "/auth/refresh", map[string]any{"refreshToken": login.RefreshToken}, "", 401, nil)

	// Logout revokes the session immediately, even though the access token has not expired.
	e.call("POST", "/auth/logout", map[string]any{"refreshToken": refreshed.RefreshToken}, "", 204, nil)
	e.call("GET", "/me", nil, refreshed.AccessToken, 401, nil)

	// Password recovery.
	e.call("POST", "/auth/forgot-password", map[string]any{"email": "ghost@test.invalid"}, "", 202, nil)
	if _, ok := e.mail.Last(); ok {
		t.Fatal("no e-mail must be sent for unknown accounts")
	}
	e.call("POST", "/auth/forgot-password", map[string]any{"email": email}, "", 202, nil)
	msg, ok := e.mail.Last()
	if !ok || msg.To != email {
		t.Fatalf("recovery mail not sent: %+v", msg)
	}
	var code string
	for _, line := range strings.Split(msg.Body, "\n") {
		if strings.Contains(line, "code de récupération :") {
			code = strings.TrimSpace(line[strings.LastIndex(line, ":")+1:])
		}
	}
	if len(code) != 10 {
		t.Fatalf("could not extract code from %q", msg.Body)
	}
	e.call("POST", "/auth/reset-password", map[string]any{"code": "AAAAAAAAAA", "newPassword": "brand-new-pass"}, "", 400, nil)
	e.call("POST", "/auth/reset-password", map[string]any{"code": code, "newPassword": "short"}, "", 400, nil)
	e.call("POST", "/auth/reset-password", map[string]any{"code": strings.ToLower(code), "newPassword": "brand-new-pass"}, "", 204, nil)
	e.call("POST", "/auth/reset-password", map[string]any{"code": code, "newPassword": "another-pass-1"}, "", 400, nil) // single use
	e.call("POST", "/auth/login", map[string]any{"email": email, "password": "correct-horse"}, "", 401, nil)
	e.call("POST", "/auth/login", map[string]any{"email": email, "password": "brand-new-pass"}, "", 200, &login)

	// Change password requires the current one.
	e.call("PUT", "/auth/password", map[string]any{"currentPassword": "nope-nope", "newPassword": "yet-another-1"}, login.AccessToken, 401, nil)
	e.call("PUT", "/auth/password", map[string]any{"currentPassword": "brand-new-pass", "newPassword": "yet-another-1"}, login.AccessToken, 204, nil)
}

func TestProfilePreferencesAndValidation(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(spec{name: "Lea", gender: "woman", interestedIn: []string{"man"}, lat: 48.8566, lng: 2.3522})

	var me struct {
		Complete    bool     `json:"complete"`
		Missing     []string `json:"missing"`
		HasLocation bool     `json:"hasLocation"`
		Interests   []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
		Age int `json:"age"`
	}
	e.call("GET", "/profile", nil, u.Token, 200, &me)
	if !me.Complete || len(me.Missing) != 0 || !me.HasLocation || len(me.Interests) != 2 || me.Age < 30 {
		t.Fatalf("unexpected profile: %+v", me)
	}

	// Raw coordinates are never stored beyond ~1km precision.
	var lat float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 {
		t.Fatalf("latitude should be rounded to 2 decimals, got %v", lat)
	}

	bad := []map[string]any{
		{"gender": "robot"},
		{"bio": strings.Repeat("a", 501)},
		{"firstName": ""},
		{"interests": []string{"does-not-exist"}},
		{"interests": []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9"}},
	}
	for _, b := range bad {
		if status, _ := e.do("PUT", "/profile", b, u.Token); status != 400 {
			t.Errorf("PUT /profile %v expected 400, got %d", b, status)
		}
	}
	e.call("PUT", "/profile/location", map[string]any{"latitude": 120, "longitude": 0}, u.Token, 400, nil)
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{}, "ageMin": 18, "ageMax": 40, "maxDistanceKm": 30}, u.Token, 400, nil)
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 17, "ageMax": 40, "maxDistanceKm": 30}, u.Token, 400, nil)
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 25, "maxDistanceKm": 30}, u.Token, 400, nil)
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 25, "ageMax": 40, "maxDistanceKm": 0}, u.Token, 400, nil)
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{"man", "non_binary"}, "ageMin": 25, "ageMax": 40, "maxDistanceKm": 30}, u.Token, 200, nil)

	var interests struct {
		Interests []map[string]string `json:"interests"`
	}
	e.call("GET", "/interests", nil, u.Token, 200, &interests)
	if len(interests.Interests) < 10 {
		t.Fatal("interest catalogue missing")
	}
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	owner := e.newUser(spec{name: "Owner", gender: "woman", interestedIn: []string{"man"}, lat: -33.87, lng: 151.21, noPhoto: true})
	stranger := e.newUser(spec{name: "Stranger", gender: "man", interestedIn: []string{"woman"}, lat: -33.9, lng: 151.2})

	// Invalid uploads.
	if status, _ := e.upload("POST", "/photos", owner.Token, []byte("<?php echo 1; ?>")); status != http.StatusUnsupportedMediaType {
		t.Errorf("script upload expected 415, got %d", status)
	}
	if status, _ := e.upload("POST", "/photos", owner.Token, []byte("GIF89a....")); status != http.StatusUnsupportedMediaType {
		t.Errorf("gif upload expected 415, got %d", status)
	}
	if status, _ := e.upload("POST", "/photos", owner.Token, append([]byte("\x89PNG\r\n\x1a\n"), []byte("corrupt")...)); status != 400 {
		t.Errorf("corrupt png expected 400, got %d", status)
	}
	if status, _ := e.upload("POST", "/photos", owner.Token, pngBytes(50, 50)); status != 400 {
		t.Errorf("tiny image expected 400, got %d", status)
	}
	if status, _ := e.upload("POST", "/photos", owner.Token, make([]byte, 9<<20)); status != 413 && status != 400 {
		t.Errorf("oversized upload expected 413/400, got %d", status)
	}
	e.call("GET", "/photos", nil, owner.Token, 200, nil)

	// Large PNG is converted to a downscaled JPEG.
	var p1, p2, p3 struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	status, body := e.upload("POST", "/photos", owner.Token, pngBytes(2400, 1800))
	if status != 201 {
		t.Fatalf("upload: %d %s", status, body)
	}
	_ = json.Unmarshal(body, &p1)
	for _, p := range []*struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}{&p2, &p3} {
		status, body := e.upload("POST", "/photos", owner.Token, pngBytes(400, 400))
		if status != 201 {
			t.Fatalf("upload: %d %s", status, body)
		}
		_ = json.Unmarshal(body, p)
	}
	var w, h int
	_ = e.pool.QueryRow(context.Background(), `SELECT width, height FROM photos WHERE id = $1`, p1.ID).Scan(&w, &h)
	if w != 1440 || h != 1080 {
		t.Errorf("expected 1440x1080 after downscale, got %dx%d", w, h)
	}

	// The owner can fetch the file; it's a JPEG and not cacheable by shared caches.
	status, data := e.do("GET", p1.URL, nil, owner.Token)
	if status != 200 || len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatalf("expected a jpeg, got status=%d", status)
	}
	// Unauthenticated: refused.
	e.call("GET", p1.URL, nil, "", 401, nil)

	// Max 6 photos.
	for i := 0; i < 3; i++ {
		e.upload("POST", "/photos", owner.Token, pngBytes(300, 300))
	}
	if status, _ := e.upload("POST", "/photos", owner.Token, pngBytes(300, 300)); status != 409 {
		t.Errorf("7th photo expected 409, got %d", status)
	}

	// Reorder: make p3 the main photo. Must list every photo exactly once.
	var list struct {
		Photos []struct{ ID string } `json:"photos"`
	}
	e.call("GET", "/photos", nil, owner.Token, 200, &list)
	ids := []string{p3.ID}
	for _, p := range list.Photos {
		if p.ID != p3.ID {
			ids = append(ids, p.ID)
		}
	}
	e.call("PUT", "/photos/order", map[string]any{"ids": ids[:3]}, owner.Token, 400, nil)
	e.call("PUT", "/photos/order", map[string]any{"ids": ids}, owner.Token, 200, &list)
	if list.Photos[0].ID != p3.ID {
		t.Fatal("p3 should be first after reorder")
	}
	// Another user's ids are rejected.
	e.call("PUT", "/photos/order", map[string]any{"ids": []string{p3.ID}}, stranger.Token, 400, nil)

	// Replace keeps position and changes the file version.
	var replaced struct{ ID, URL string }
	status, body = e.upload("PUT", "/photos/"+p3.ID, owner.Token, pngBytes(640, 640))
	if status != 200 {
		t.Fatalf("replace: %d %s", status, body)
	}
	_ = json.Unmarshal(body, &replaced)
	if replaced.ID != p3.ID || replaced.URL == p3.URL {
		t.Errorf("replace should keep id and change url version: %+v vs %+v", replaced, p3)
	}
	if status, _ := e.upload("PUT", "/photos/"+p3.ID, stranger.Token, pngBytes(640, 640)); status != 404 {
		t.Errorf("replacing someone else's photo expected 404, got %d", status)
	}

	// Delete: only the owner, positions are compacted.
	e.call("DELETE", "/photos/"+p2.ID, nil, stranger.Token, 404, nil)
	e.call("DELETE", "/photos/"+p2.ID, nil, owner.Token, 204, nil)
	e.call("GET", "/photos", nil, owner.Token, 200, &list)
	var maxPos int
	_ = e.pool.QueryRow(context.Background(), `SELECT MAX(position) FROM photos WHERE user_id = $1`, owner.ID).Scan(&maxPos)
	if maxPos != len(list.Photos)-1 {
		t.Errorf("positions should be contiguous, max=%d count=%d", maxPos, len(list.Photos))
	}
}

func TestDiscoveryFiltersAndOrder(t *testing.T) {
	e := newEnv(t)
	// Region isolated from other tests: around (60N, 25E).
	me := e.newUser(spec{name: "Me", gender: "man", interestedIn: []string{"woman"}, lat: 60.17, lng: 24.94})
	near := e.newUser(spec{name: "Near", gender: "woman", interestedIn: []string{"man"}, lat: 60.20, lng: 24.96})
	far := e.newUser(spec{name: "Far", gender: "woman", interestedIn: []string{"man"}, lat: 62.0, lng: 28.0}) // >300km
	wrongOrientation := e.newUser(spec{name: "NotInterested", gender: "woman", interestedIn: []string{"woman"}, lat: 60.18, lng: 24.95})
	wrongGender := e.newUser(spec{name: "Man", gender: "man", interestedIn: []string{"man"}, lat: 60.18, lng: 24.95})
	noPhoto := e.newUser(spec{name: "NoPhoto", gender: "woman", interestedIn: []string{"man"}, lat: 60.18, lng: 24.95, noPhoto: true})
	hidden := e.newUser(spec{name: "Hidden", gender: "woman", interestedIn: []string{"man"}, lat: 60.18, lng: 24.95})
	e.call("PUT", "/profile", map[string]any{"discoverable": false}, hidden.Token, 200, nil)
	blocker := e.newUser(spec{name: "Blocker", gender: "woman", interestedIn: []string{"man"}, lat: 60.18, lng: 24.95})
	e.call("POST", "/blocks", map[string]any{"userId": me.ID}, blocker.Token, 204, nil)
	// Their own age limits: she only wants 18-25 and Me is ~32.
	young := e.newUser(spec{name: "Young", gender: "woman", interestedIn: []string{"man"}, lat: 60.18, lng: 24.95})
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 25, "maxDistanceKm": 100}, young.Token, 200, nil)
	// Their own distance limit is 1km.
	closeOnly := e.newUser(spec{name: "CloseOnly", gender: "woman", interestedIn: []string{"man"}, lat: 60.25, lng: 25.1})
	e.call("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 1}, closeOnly.Token, 200, nil)

	got := e.discover(me)
	if !contains(got, near.ID) {
		t.Fatalf("near candidate missing: %+v", got)
	}
	for name, u := range map[string]testUser{"far": far, "orientation": wrongOrientation, "gender": wrongGender, "no photo": noPhoto, "hidden": hidden, "blocker": blocker, "age": young, "their distance": closeOnly, "self": me} {
		if contains(got, u.ID) {
			t.Errorf("%s must not be proposed", name)
		}
	}
	for _, c := range got {
		if c.DistanceKm == nil || *c.DistanceKm < 1 {
			t.Errorf("distance must be approximate and present: %+v", c)
		}
		if len(c.Photos) == 0 {
			t.Errorf("candidate without photos: %+v", c)
		}
	}

	// A raw response never exposes exact coordinates or private fields.
	_, raw := e.do("GET", "/discover", nil, me.Token)
	for _, forbidden := range []string{"latitude", "longitude", "email", "birthDate", "password"} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("discover response leaks %q", forbidden)
		}
	}

	// Once handled, a profile never comes back.
	e.swipe(me, near, "pass")
	if contains(e.discover(me), near.ID) {
		t.Fatal("passed profile must not reappear")
	}

	// An incomplete profile cannot use discovery.
	incomplete := e.newUser(spec{name: "Inc", gender: "man", interestedIn: []string{"woman"}, lat: 60.17, lng: 24.94, noPhoto: true})
	e.call("GET", "/discover", nil, incomplete.Token, 403, nil)
	e.call("POST", "/swipes", map[string]any{"userId": near.ID, "action": "like"}, incomplete.Token, 403, nil)
}

func TestMatchingAndChat(t *testing.T) {
	e := newEnv(t)
	// Isolated region around (35N, 139E).
	alice := e.newUser(spec{name: "Alice", gender: "woman", interestedIn: []string{"man"}, lat: 35.68, lng: 139.69})
	bob := e.newUser(spec{name: "Bob", gender: "man", interestedIn: []string{"woman"}, lat: 35.70, lng: 139.70})
	carol := e.newUser(spec{name: "Carol", gender: "woman", interestedIn: []string{"man"}, lat: 35.69, lng: 139.71})

	// Validation of the swipe API.
	e.call("POST", "/swipes", map[string]any{"userId": bob.ID, "action": "superlike"}, alice.Token, 400, nil)
	e.call("POST", "/swipes", map[string]any{"userId": "not-a-uuid", "action": "like"}, alice.Token, 400, nil)
	e.call("POST", "/swipes", map[string]any{"userId": alice.ID, "action": "like"}, alice.Token, 400, nil)
	e.call("POST", "/swipes", map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "action": "like"}, alice.Token, 404, nil)
	e.call("POST", "/swipes", map[string]any{"userId": bob.ID, "action": "like"}, "", 401, nil)

	// One-sided like: no match, no conversation.
	if matched, _, _ := e.swipe(alice, bob, "like"); matched {
		t.Fatal("one-sided like must not match")
	}
	// Repeating the same like is idempotent; contradicting it is refused.
	if matched, _, _ := e.swipe(alice, bob, "like"); matched {
		t.Fatal("repeated like must not match")
	}
	e.call("POST", "/swipes", map[string]any{"userId": bob.ID, "action": "pass"}, alice.Token, 409, nil)
	var count int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 AND target_id = $2`, alice.ID, bob.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("duplicate swipe rows: %d", count)
	}

	// Bob sees Alice first (she liked him) and likes back: MATCH.
	if got := e.discover(bob); len(got) == 0 || got[0].UserID != alice.ID {
		t.Fatalf("liked-me profile should rank first: %+v", got)
	}
	matched, matchID, convID := e.swipe(bob, alice, "like")
	if !matched || matchID == "" || convID == "" {
		t.Fatal("mutual like must match")
	}
	// Idempotent: replaying creates no second match/conversation.
	_, matchID2, convID2 := e.swipe(bob, alice, "like")
	if matchID2 != matchID || convID2 != convID {
		t.Fatal("replaying a mutual like must return the same match")
	}
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match row, got %d", count)
	}

	// Both get exactly one match notification.
	for _, u := range []testUser{alice, bob} {
		var n struct {
			Notifications []struct {
				Type  string `json:"type"`
				Actor *struct {
					FirstName string `json:"firstName"`
				} `json:"actor"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		e.call("GET", "/notifications", nil, u.Token, 200, &n)
		if len(n.Notifications) != 1 || n.Notifications[0].Type != "match" || n.UnreadCount != 1 || n.Notifications[0].Actor == nil {
			t.Fatalf("expected one match notification for %s: %+v", u.Email, n)
		}
	}

	// Matches list.
	var ml struct {
		Matches []struct {
			ID          string `json:"id"`
			HasMessages bool   `json:"hasMessages"`
			User        card   `json:"user"`
		} `json:"matches"`
	}
	e.call("GET", "/matches", nil, alice.Token, 200, &ml)
	if len(ml.Matches) != 1 || ml.Matches[0].User.UserID != bob.ID || ml.Matches[0].HasMessages {
		t.Fatalf("unexpected matches: %+v", ml)
	}

	// --- chat ---
	send := func(u testUser, body string, want int) {
		t.Helper()
		e.call("POST", "/conversations/"+convID+"/messages", map[string]any{"body": body}, u.Token, want, nil)
	}
	send(alice, "Salut Bob !", 201)
	send(alice, "   ", 400)
	send(alice, strings.Repeat("x", 2001), 400)
	send(bob, "Salut Alice", 201)
	send(alice, "Ça va ?", 201)

	// A user who is not in the conversation can neither read, write, clear nor mark it read.
	e.call("GET", "/conversations/"+convID+"/messages", nil, carol.Token, 404, nil)
	send(carol, "intruder", 404)
	e.call("POST", "/conversations/"+convID+"/read", nil, carol.Token, 404, nil)
	e.call("DELETE", "/conversations/"+convID, nil, carol.Token, 404, nil)
	e.call("GET", "/conversations/not-a-uuid/messages", nil, alice.Token, 404, nil)

	// Bob: one unread (Alice's last), conversation listed with the last message.
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			Peer        card   `json:"peer"`
			LastMessage struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.call("GET", "/conversations", nil, bob.Token, 200, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].LastMessage.Body != "Ça va ?" || convs.Conversations[0].Peer.UserID != alice.ID {
		t.Fatalf("unexpected conversations for bob: %+v", convs)
	}
	var unread struct {
		UnreadCount int `json:"unreadCount"`
	}
	e.call("GET", "/conversations/unread-count", nil, bob.Token, 200, &unread)
	if unread.UnreadCount != 1 {
		t.Fatalf("unread total: %d", unread.UnreadCount)
	}

	// Message pagination (newest first) and read state.
	var page struct {
		Messages []struct {
			Body string `json:"body"`
		} `json:"messages"`
		NextBefore *time.Time `json:"nextBefore"`
	}
	e.call("GET", "/conversations/"+convID+"/messages?limit=2", nil, bob.Token, 200, &page)
	if len(page.Messages) != 2 || page.Messages[0].Body != "Ça va ?" || page.NextBefore == nil {
		t.Fatalf("unexpected page: %+v", page)
	}
	e.call("GET", "/conversations/"+convID+"/messages?before="+page.NextBefore.Format(time.RFC3339Nano), nil, bob.Token, 200, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Salut Bob !" {
		t.Fatalf("unexpected older page: %+v", page)
	}
	e.call("GET", "/conversations/"+convID+"/messages?before=yesterday", nil, bob.Token, 400, nil)

	e.call("POST", "/conversations/"+convID+"/read", nil, bob.Token, 204, nil)
	e.call("GET", "/conversations/unread-count", nil, bob.Token, 200, &unread)
	if unread.UnreadCount != 0 {
		t.Fatalf("unread after read: %d", unread.UnreadCount)
	}
	var aliceConvs struct {
		Conversations []struct {
			PeerLastReadAt time.Time `json:"peerLastReadAt"`
			LastMessage    struct {
				CreatedAt time.Time `json:"createdAt"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.call("GET", "/conversations", nil, alice.Token, 200, &aliceConvs)
	if aliceConvs.Conversations[0].PeerLastReadAt.Before(aliceConvs.Conversations[0].LastMessage.CreatedAt) {
		t.Fatal("alice should see her last message as read by bob")
	}

	// Local deletion hides the history for one side only.
	e.call("DELETE", "/conversations/"+convID, nil, bob.Token, 204, nil)
	e.call("GET", "/conversations", nil, bob.Token, 200, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("cleared conversation should disappear from bob's list")
	}
	e.call("GET", "/conversations", nil, alice.Token, 200, &aliceConvs)
	if len(aliceConvs.Conversations) != 1 {
		t.Fatal("alice keeps her history")
	}
	time.Sleep(5 * time.Millisecond)
	send(alice, "Tu es là ?", 201)
	e.call("GET", "/conversations/"+convID+"/messages", nil, bob.Token, 200, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Tu es là ?" {
		t.Fatalf("bob should only see messages after clearing: %+v", page)
	}

	// Block: conversation, matches, discovery and profile all disappear for both sides, sending is refused.
	e.call("POST", "/blocks", map[string]any{"userId": alice.ID}, bob.Token, 204, nil)
	e.call("POST", "/blocks", map[string]any{"userId": alice.ID}, bob.Token, 204, nil) // idempotent
	send(bob, "hello?", 403)
	send(alice, "hello?", 403)
	e.call("GET", "/conversations/"+convID+"/messages", nil, alice.Token, 404, nil)
	e.call("GET", "/profiles/"+bob.ID, nil, alice.Token, 404, nil)
	e.call("GET", "/profiles/"+alice.ID, nil, bob.Token, 404, nil)
	e.call("GET", "/matches", nil, alice.Token, 200, &ml)
	if len(ml.Matches) != 0 {
		t.Fatal("blocked match must be hidden")
	}
	e.call("GET", "/conversations", nil, alice.Token, 200, &aliceConvs)
	if len(aliceConvs.Conversations) != 0 {
		t.Fatal("blocked conversation must be hidden")
	}
	var blocks struct {
		Blocks []struct{ UserID string } `json:"blocks"`
	}
	e.call("GET", "/blocks", nil, bob.Token, 200, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != alice.ID {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	// Photos of a blocked user are not downloadable either.
	var ph struct {
		Photos []struct{ URL string } `json:"photos"`
	}
	e.call("GET", "/photos", nil, bob.Token, 200, &ph)
	e.call("GET", ph.Photos[0].URL, nil, alice.Token, 404, nil)
	e.call("POST", "/swipes", map[string]any{"userId": bob.ID, "action": "like"}, alice.Token, 404, nil)

	e.call("DELETE", "/blocks/"+alice.ID, nil, bob.Token, 204, nil)
	e.call("DELETE", "/blocks/"+alice.ID, nil, bob.Token, 404, nil)
	e.call("GET", "/profiles/"+bob.ID, nil, alice.Token, 200, nil)
	send(bob, "back again", 201)

	// Unmatch removes the conversation.
	e.call("DELETE", "/matches/"+matchID, nil, carol.Token, 404, nil)
	e.call("DELETE", "/matches/"+matchID, nil, alice.Token, 204, nil)
	e.call("GET", "/conversations/"+convID+"/messages", nil, bob.Token, 404, nil)
	// ...and they never get proposed again.
	if contains(e.discover(alice), bob.ID) {
		t.Fatal("unmatched profile must not reappear")
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{name: "Ra", gender: "woman", interestedIn: []string{"man"}, lat: -22.9, lng: -43.2})
	b := e.newUser(spec{name: "Rb", gender: "man", interestedIn: []string{"woman"}, lat: -22.95, lng: -43.2})
	done := make(chan int, 2)
	for _, pair := range [][2]testUser{{a, b}, {b, a}} {
		go func(from, to testUser) {
			status, _ := e.do("POST", "/swipes", map[string]any{"userId": to.ID, "action": "like"}, from.Token)
			done <- status
		}(pair[0], pair[1])
	}
	for i := 0; i < 2; i++ {
		if s := <-done; s != 200 {
			t.Fatalf("concurrent swipe returned %d", s)
		}
	}
	var n, c int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a = LEAST($1::uuid,$2::uuid) AND m.user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&c)
	if n != 1 || c != 1 {
		t.Fatalf("simultaneous mutual likes must yield exactly one match and conversation, got matches=%d conversations=%d", n, c)
	}
}

func TestReportsAndAccountDeletion(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{name: "Del", gender: "woman", interestedIn: []string{"man"}, lat: 64.1, lng: -21.9})
	b := e.newUser(spec{name: "Other", gender: "man", interestedIn: []string{"woman"}, lat: 64.15, lng: -21.9})

	e.call("POST", "/reports", map[string]any{"userId": b.ID, "reason": "nonsense"}, a.Token, 400, nil)
	e.call("POST", "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, a.Token, 400, nil)
	e.call("POST", "/reports", map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}, a.Token, 404, nil)
	e.call("POST", "/reports", map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude"}, a.Token, 201, nil)
	e.call("POST", "/reports", map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude again"}, a.Token, 201, nil)
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("identical open reports should be merged, got %d", n)
	}

	_, _, convID := e.swipe(a, b, "like")
	_ = convID
	e.swipe(b, a, "like")

	var keys []string
	rows, _ := e.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) == 0 {
		t.Fatal("expected a stored photo")
	}

	e.call("DELETE", "/me", map[string]any{"password": "wrong-password"}, a.Token, 401, nil)
	e.call("DELETE", "/me", map[string]any{"password": "correct-horse"}, a.Token, 204, nil)

	// Token is dead, every dependent row is gone, files are removed.
	e.call("GET", "/me", nil, a.Token, 401, nil)
	e.call("POST", "/auth/login", map[string]any{"email": a.Email, "password": "correct-horse"}, "", 401, nil)
	for _, table := range []string{"profiles", "preferences", "photos", "sessions", "swipes", "notifications", "reports"} {
		col := "user_id"
		switch table {
		case "swipes":
			col = "swiper_id"
		case "reports":
			col = "reporter_id"
		}
		var c int
		if err := e.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&c); err != nil {
			t.Fatal(err)
		}
		if c != 0 {
			t.Errorf("%s still holds %d rows of the deleted user", table, c)
		}
	}
	var m int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = $1 OR user_b = $1`, a.ID).Scan(&m)
	if m != 0 {
		t.Error("matches of the deleted user must be removed")
	}
	for _, k := range keys {
		if f, err := e.store.Open(context.Background(), k); err == nil {
			f.Close()
			t.Errorf("photo file %s should be deleted", k)
		}
	}
	// The other user is unaffected and no longer sees the deleted account.
	var ml struct {
		Matches []any `json:"matches"`
	}
	e.call("GET", "/matches", nil, b.Token, 200, &ml)
	if len(ml.Matches) != 0 {
		t.Error("deleted user's match must disappear")
	}
}

func TestNotificationsFlow(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{name: "Na", gender: "woman", interestedIn: []string{"man"}, lat: -54.8, lng: -68.3})
	b := e.newUser(spec{name: "Nb", gender: "man", interestedIn: []string{"woman"}, lat: -54.85, lng: -68.3})
	_, _, conv := e.swipe(a, b, "like")
	_, _, conv = e.swipe(b, a, "like")
	// Three messages produce one coalesced message notification.
	for i := 0; i < 3; i++ {
		e.call("POST", "/conversations/"+conv+"/messages", map[string]any{"body": "hey"}, a.Token, 201, nil)
	}
	var n struct {
		Notifications []struct {
			ID     string     `json:"id"`
			Type   string     `json:"type"`
			ReadAt *time.Time `json:"readAt"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.call("GET", "/notifications", nil, b.Token, 200, &n)
	if n.UnreadCount != 2 || len(n.Notifications) != 2 {
		t.Fatalf("expected match + one message notification: %+v", n)
	}
	// Others' notifications cannot be marked.
	e.call("POST", "/notifications/"+n.Notifications[0].ID+"/read", nil, a.Token, 404, nil)
	e.call("POST", "/notifications/"+n.Notifications[0].ID+"/read", nil, b.Token, 204, nil)
	e.call("GET", "/notifications", nil, b.Token, 200, &n)
	if n.UnreadCount != 1 {
		t.Fatalf("unread should be 1: %+v", n)
	}
	e.call("POST", "/notifications/read-all", nil, b.Token, 204, nil)
	e.call("GET", "/notifications", nil, b.Token, 200, &n)
	if n.UnreadCount != 0 {
		t.Fatalf("unread should be 0: %+v", n)
	}
	e.call("GET", "/notifications?before=bad", nil, b.Token, 400, nil)
	e.call("GET", "/notifications?limit=1", nil, b.Token, 200, &n)
	if len(n.Notifications) != 1 {
		t.Fatalf("limit not honoured: %+v", n)
	}
}

func TestRateLimiting(t *testing.T) {
	e := newEnv(t)
	limited, err := NewRouter(Deps{DB: e.pool, JWTSecret: []byte("integration-test-secret-0123456789"), Storage: e.store, Mailer: e.mail,
		Limits: Limits{Public: [2]int{1, 3}, Swipes: [2]int{6000, 6000}, Messages: [2]int{6000, 6000}, Uploads: [2]int{6000, 6000}, Reports: [2]int{6000, 6000}}})
	if err != nil {
		t.Fatal(err)
	}
	e.router = limited
	var last int
	for i := 0; i < 6; i++ {
		last, _ = e.do("POST", "/auth/login", map[string]any{"email": "x@test.invalid", "password": "whatever-123"}, "")
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("login brute force should be rate limited, last status %d", last)
	}
}
