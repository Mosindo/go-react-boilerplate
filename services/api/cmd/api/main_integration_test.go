package main

import (
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	p := e.register("auth")

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": p.Email, "password": "Sup3rSecret!"}), 409)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "short@test.invalid", "password": "short"}), 400)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": p.Email, "password": "wrong-password"}), 401)
	if code := e.do("POST", "/auth/login", "", map[string]string{"email": "nobody@test.invalid", "password": "whatever123"}).json(t)["code"]; code != "unauthorized" {
		t.Fatalf("unexpected code %v", code)
	}

	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	me := e.expect(e.do("GET", "/me", p.Token, nil), 200).json(t)
	if me["hasProfile"] != false || me["profileComplete"] != false {
		t.Fatalf("fresh account must have no profile: %v", me)
	}

	// Refresh rotates; the old refresh token is dead.
	rotated := e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": p.Refresh}), 200).json(t)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": p.Refresh}), 401)

	// Logout kills the session immediately, even though the access token has not expired.
	newRefresh := rotated["refreshToken"].(string)
	newAccess := rotated["accessToken"].(string)
	e.expect(e.do("GET", "/me", newAccess, nil), 200)
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": newRefresh}), 204)
	e.expect(e.do("GET", "/me", newAccess, nil), 401)
}

func TestPasswordReset(t *testing.T) {
	e := newEnv(t)
	p := e.register("reset")

	// Unknown e-mails get the same answer and no mail.
	e.expect(e.do("POST", "/auth/password-reset/request", "", map[string]string{"email": "ghost@test.invalid"}), 202)
	if e.mail.last("ghost@test.invalid") != "" {
		t.Fatal("no mail expected for unknown address")
	}

	e.expect(e.do("POST", "/auth/password-reset/request", "", map[string]string{"email": p.Email}), 202)
	body := e.mail.last(p.Email)
	code := regexp.MustCompile(`[A-Z2-9]{4}-[A-Z2-9]{4}-[A-Z2-9]{4}`).FindString(body)
	if code == "" {
		t.Fatalf("no code in mail: %q", body)
	}

	e.expect(e.do("POST", "/auth/password-reset/confirm", "", map[string]string{"token": "AAAA-BBBB-CCCC", "newPassword": "BrandNewPass1"}), 400)
	e.expect(e.do("POST", "/auth/password-reset/confirm", "", map[string]string{"token": strings.ToLower(code), "newPassword": "BrandNewPass1"}), 204)
	// Single use, and old sessions/passwords are gone.
	e.expect(e.do("POST", "/auth/password-reset/confirm", "", map[string]string{"token": code, "newPassword": "AnotherPass22"}), 400)
	e.expect(e.do("GET", "/me", p.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": p.Email, "password": "Sup3rSecret!"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": p.Email, "password": "BrandNewPass1"}), 200)
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	p := e.register("valid")
	put := func(m map[string]any) resp { return e.do("PUT", "/profile/me", p.Token, m) }
	base := func() map[string]any {
		return map[string]any{"firstName": "Sam", "birthDate": "1990-01-01", "gender": "man", "bio": "", "city": "", "interests": []string{}}
	}

	e.expect(e.do("GET", "/profile/me", p.Token, nil), 404)

	underage := base()
	underage["birthDate"] = time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	if code := e.expect(put(underage), 400).json(t)["code"]; code != "underage" {
		t.Fatalf("expected underage code, got %v", code)
	}
	exactly18 := base()
	exactly18["birthDate"] = time.Now().AddDate(-18, 0, 0).Format("2006-01-02")
	e.expect(put(exactly18), 200)

	for name, mut := range map[string]func(map[string]any){
		"bad date":    func(m map[string]any) { m["birthDate"] = "not-a-date" },
		"bad gender":  func(m map[string]any) { m["gender"] = "robot" },
		"html name":   func(m map[string]any) { m["firstName"] = "<b>x</b>" },
		"long bio":    func(m map[string]any) { m["bio"] = strings.Repeat("a", 501) },
		"unknown int": func(m map[string]any) { m["interests"] = []string{"hiking", "not-a-real-interest"} },
		"too many int": func(m map[string]any) {
			m["interests"] = []string{"hiking", "travel", "cooking", "coffee", "wine", "music", "concerts", "movies", "series", "reading", "gaming"}
		},
		"future born":    func(m map[string]any) { m["birthDate"] = "2999-01-01" },
		"ancient":        func(m map[string]any) { m["birthDate"] = "1800-01-01" },
		"control in bio": func(m map[string]any) { m["bio"] = "hi\x00" },
	} {
		m := base()
		mut(m)
		if r := put(m); r.Status != 400 {
			t.Errorf("%s: expected 400, got %d %s", name, r.Status, r.Body)
		}
	}

	e.expect(e.do("PUT", "/profile/me/location", p.Token, map[string]float64{"latitude": 95, "longitude": 0}), 400)

	// Preferences.
	prefs := e.expect(e.do("GET", "/preferences", p.Token, nil), 200).json(t)
	if prefs["maxDistanceKm"].(float64) != 50 || prefs["discoverable"] != true {
		t.Fatalf("unexpected defaults: %v", prefs)
	}
	for name, body := range map[string]map[string]any{
		"empty genders": {"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 10, "discoverable": true, "showDistance": true},
		"min>max":       {"interestedIn": []string{"woman"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10, "discoverable": true, "showDistance": true},
		"under 18":      {"interestedIn": []string{"woman"}, "minAge": 16, "maxAge": 30, "maxDistanceKm": 10, "discoverable": true, "showDistance": true},
		"distance":      {"interestedIn": []string{"woman"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 0, "discoverable": true, "showDistance": true},
		"bad gender":    {"interestedIn": []string{"x"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 5, "discoverable": true, "showDistance": true},
	} {
		if r := e.do("PUT", "/preferences", p.Token, body); r.Status != 400 {
			t.Errorf("%s: expected 400, got %d", name, r.Status)
		}
	}
	e.expect(e.do("PUT", "/preferences", p.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 25, "maxAge": 35, "maxDistanceKm": 20, "discoverable": true, "showDistance": false}), 200)
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	p := e.register("photo")
	other := e.register("photo2")

	// Hostile or malformed uploads are refused.
	e.expect(e.upload("POST", "/photos", p.Token, []byte("<?php echo 1;"), "x.php.jpg"), 400)
	e.expect(e.upload("POST", "/photos", p.Token, []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), "x.png"), 400)
	e.expect(e.upload("POST", "/photos", p.Token, pngBytes(40, 40, 1), "tiny.png"), 400)
	e.expect(e.upload("POST", "/photos", p.Token, make([]byte, 6<<20), "huge.png"), 400)
	e.expect(e.do("POST", "/photos", "", nil), 401)

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.expect(e.upload("POST", "/photos", p.Token, pngBytes(300+i, 300, uint8(i*20)), "p.png"), 201).json(t)
		ids = append(ids, r["id"].(string))
		if int(r["position"].(float64)) != i {
			t.Fatalf("expected position %d, got %v", i, r["position"])
		}
	}
	e.expect(e.upload("POST", "/photos", p.Token, pngBytes(300, 300, 99), "p.png"), 400) // 7th

	// Reorder: reverse. Partial lists or foreign ids fail.
	rev := make([]string, 0, 6)
	for i := len(ids) - 1; i >= 0; i-- {
		rev = append(rev, ids[i])
	}
	e.expect(e.do("PUT", "/photos/order", p.Token, map[string]any{"photoIds": ids[:3]}), 400)
	e.expect(e.do("PUT", "/photos/order", other.Token, map[string]any{"photoIds": rev}), 400)
	items := e.expect(e.do("PUT", "/photos/order", p.Token, map[string]any{"photoIds": rev}), 200).items(t)
	if items[0]["id"] != rev[0] {
		t.Fatalf("main photo should be %s, got %v", rev[0], items[0]["id"])
	}

	// Signed URL serves a JPEG; tampering does not.
	u := items[0]["url"].(string)
	img := e.expect(e.do("GET", u, "", nil), 200)
	if ct := img.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("content type %q", ct)
	}
	e.expect(e.do("GET", strings.Replace(u, "sig=", "sig=00", 1), "", nil), 404)
	e.expect(e.do("GET", "/media/"+rev[0], "", nil), 404)

	// Others cannot delete or replace my photos.
	e.expect(e.do("DELETE", "/photos/"+rev[0], other.Token, nil), 404)
	e.expect(e.upload("PUT", "/photos/"+rev[0], other.Token, pngBytes(300, 300, 5), "x.png"), 404)

	// Replace keeps the position with a new id.
	replaced := e.expect(e.upload("PUT", "/photos/"+rev[2], p.Token, pngBytes(310, 310, 3), "x.png"), 200).json(t)
	if replaced["id"] == rev[2] || int(replaced["position"].(float64)) != 2 {
		t.Fatalf("unexpected replace result: %v", replaced)
	}

	// Delete compacts so positions stay 0..n-1 and the new main photo is first.
	e.expect(e.do("DELETE", "/photos/"+rev[0], p.Token, nil), 204)
	items = e.expect(e.do("GET", "/photos", p.Token, nil), 200).items(t)
	if len(items) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(items))
	}
	for i, it := range items {
		if int(it["position"].(float64)) != i {
			t.Fatalf("positions not compact: %v", items)
		}
	}
}

func TestMatchingAndChatJourney(t *testing.T) {
	e := newEnv(t)
	ana := e.onboard("ana", "woman", "hiking", "coffee")
	ben := e.onboard("ben", "man", "hiking", "music")
	cleo := e.onboard("cleo", "woman", "music")

	// Incomplete profile cannot browse.
	bare := e.register("bare")
	if code := e.expect(e.do("GET", "/discover", bare.Token, nil), 403).json(t)["code"]; code != "profile_incomplete" {
		t.Fatalf("unexpected code %v", code)
	}

	// Ana (default: interested in everyone) sees Ben and Cleo, with approximate distance and shared interests.
	list := e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t)
	if !containsUser(list, ben.ID) || !containsUser(list, cleo.ID) || containsUser(list, ana.ID) {
		t.Fatalf("unexpected discover list: %v", list)
	}
	for _, it := range list {
		if it["userId"] == ben.ID {
			shared, _ := it["sharedInterests"].([]any)
			if len(shared) != 1 || shared[0] != "hiking" {
				t.Fatalf("shared interests wrong: %v", it["sharedInterests"])
			}
			if it["distanceKm"] == nil {
				t.Fatal("distance expected")
			}
			if _, leaked := it["latitude"]; leaked {
				t.Fatal("coordinates must never be exposed")
			}
			if _, leaked := it["email"]; leaked {
				t.Fatal("email must never be exposed")
			}
			if len(it["photos"].([]any)) != 1 {
				t.Fatal("photos expected")
			}
		}
	}

	// Validation and dedup.
	e.expect(e.swipe(ana, ana, "like"), 400)
	e.expect(e.do("POST", "/swipes", ana.Token, map[string]string{"targetUserId": ben.ID, "action": "superlike"}), 400)
	e.expect(e.do("POST", "/swipes", ana.Token, map[string]string{"targetUserId": "11111111-1111-1111-1111-111111111111", "action": "like"}), 404)

	first := e.expect(e.swipe(ana, ben, "like"), 200).json(t)
	if first["matched"] != false {
		t.Fatal("one-sided like must not match")
	}
	if code := e.expect(e.swipe(ana, ben, "like"), 409).json(t)["code"]; code != "already_swiped" {
		t.Fatalf("unexpected code %v", code)
	}
	e.expect(e.swipe(ana, ben, "pass"), 409)

	// Ben no longer shows up for Ana; Ana (who liked Ben) shows up first-class for Ben.
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t), ben.ID) {
		t.Fatal("swiped profile reappeared")
	}
	if !containsUser(e.expect(e.do("GET", "/discover?limit=20", ben.Token, nil), 200).items(t), ana.ID) {
		t.Fatal("ben should see ana")
	}

	// No match yet means no conversation and no access to profile via matches.
	if n := len(e.expect(e.do("GET", "/matches", ana.Token, nil), 200).items(t)); n != 0 {
		t.Fatalf("expected no matches, got %d", n)
	}

	// Mutual like: Ben gets the match immediately.
	second := e.expect(e.swipe(ben, ana, "like"), 200).json(t)
	if second["matched"] != true {
		t.Fatalf("expected a match: %v", second)
	}
	match := second["match"].(map[string]any)
	convID := match["conversationId"].(string)
	if match["user"].(map[string]any)["userId"] != ana.ID {
		t.Fatalf("match should show the other person: %v", match)
	}

	anaMatches := e.expect(e.do("GET", "/matches", ana.Token, nil), 200).items(t)
	benMatches := e.expect(e.do("GET", "/matches", ben.Token, nil), 200).items(t)
	if len(anaMatches) != 1 || len(benMatches) != 1 || anaMatches[0]["id"] != benMatches[0]["id"] {
		t.Fatalf("both sides should see the same single match: %v %v", anaMatches, benMatches)
	}

	// Notifications for both.
	for _, who := range []person{ana, ben} {
		n := e.expect(e.do("GET", "/notifications", who.Token, nil), 200).json(t)
		if n["unreadCount"].(float64) != 1 {
			t.Fatalf("expected one unread match notification: %v", n)
		}
	}

	// Chat: permissions first. Cleo (not part of it) sees nothing and can do nothing.
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", cleo.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", cleo.Token, map[string]string{"content": "intruder"}), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", cleo.Token, nil), 404)
	e.expect(e.do("DELETE", "/conversations/"+convID, cleo.Token, nil), 404)
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", "", nil), 401)

	// Realtime: Ben listens while Ana writes.
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	ticket := e.expect(e.do("POST", "/realtime/ticket", ben.Token, nil), 200).json(t)["ticket"].(string)
	wsURL := url.URL{Scheme: "ws", Host: strings.TrimPrefix(srv.URL, "http://"), Path: "/ws", RawQuery: "ticket=" + ticket}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	if _, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil); err == nil {
		t.Fatal("tickets must be single use")
	}

	for _, bad := range []string{"", "   ", strings.Repeat("x", 2001)} {
		e.expect(e.do("POST", "/conversations/"+convID+"/messages", ana.Token, map[string]string{"content": bad}), 400)
	}
	sent := e.expect(e.do("POST", "/conversations/"+convID+"/messages", ana.Token, map[string]string{"content": "  Hi Ben 👋  "}), 201).json(t)
	if sent["content"] != "Hi Ben 👋" || sent["senderId"] != ana.ID {
		t.Fatalf("unexpected message: %v", sent)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got := map[string]bool{}
	for len(got) < 2 {
		var ev struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("ws read (have %v): %v", got, err)
		}
		got[ev.Type] = true
		if ev.Type == "message" && ev.Data["content"] != "Hi Ben 👋" {
			t.Fatalf("unexpected ws message: %v", ev.Data)
		}
	}
	if !got["message"] || !got["notification"] {
		t.Fatalf("expected message and notification events, got %v", got)
	}

	// Unread state and read receipts.
	benConvs := e.expect(e.do("GET", "/conversations", ben.Token, nil), 200).items(t)
	if len(benConvs) != 1 || benConvs[0]["unreadCount"].(float64) != 1 {
		t.Fatalf("ben should have 1 unread: %v", benConvs)
	}
	if benConvs[0]["lastMessage"].(map[string]any)["content"] != "Hi Ben 👋" {
		t.Fatalf("last message missing: %v", benConvs[0])
	}
	if anaConvs := e.expect(e.do("GET", "/conversations", ana.Token, nil), 200).items(t); anaConvs[0]["unreadCount"].(float64) != 0 || anaConvs[0]["peerLastReadAt"] != nil {
		t.Fatalf("ana: no unread, not yet read by ben: %v", anaConvs[0])
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/read", ben.Token, nil), 204)
	if benConvs = e.expect(e.do("GET", "/conversations", ben.Token, nil), 200).items(t); benConvs[0]["unreadCount"].(float64) != 0 {
		t.Fatal("unread should be cleared")
	}
	if anaConvs := e.expect(e.do("GET", "/conversations", ana.Token, nil), 200).items(t); anaConvs[0]["peerLastReadAt"] == nil {
		t.Fatal("ana should see ben's read receipt")
	}

	// Pagination of history with a cursor.
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", "/conversations/"+convID+"/messages", ben.Token, map[string]string{"content": "msg " + string(rune('A'+i))}), 201)
	}
	page1 := e.expect(e.do("GET", "/conversations/"+convID+"/messages?limit=4", ana.Token, nil), 200).json(t)
	if len(page1["items"].([]any)) != 4 || page1["nextCursor"] == nil {
		t.Fatalf("page1: %v", page1)
	}
	page2 := e.expect(e.do("GET", "/conversations/"+convID+"/messages?limit=4&before="+page1["nextCursor"].(string), ana.Token, nil), 200).json(t)
	if len(page2["items"].([]any)) != 2 || page2["nextCursor"] != nil {
		t.Fatalf("page2: %v", page2)
	}
	if first := page1["items"].([]any)[0].(map[string]any)["content"]; first != "msg E" {
		t.Fatalf("newest first expected, got %v", first)
	}

	// Hiding is per-user and undone by a new message.
	e.expect(e.do("DELETE", "/conversations/"+convID, ana.Token, nil), 204)
	if len(e.expect(e.do("GET", "/conversations", ana.Token, nil), 200).items(t)) != 0 {
		t.Fatal("hidden conversation should be gone for ana")
	}
	if len(e.expect(e.do("GET", "/conversations", ben.Token, nil), 200).items(t)) != 1 {
		t.Fatal("ben still has it")
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", ben.Token, map[string]string{"content": "still there?"}), 201)
	if len(e.expect(e.do("GET", "/conversations", ana.Token, nil), 200).items(t)) != 1 {
		t.Fatal("new message should resurface the conversation")
	}

	// Blocking ends everything: no messages, no discovery, no match, no profile.
	e.expect(e.do("POST", "/blocks", ana.Token, map[string]string{"userId": ben.ID}), 204)
	e.expect(e.do("POST", "/blocks", ana.Token, map[string]string{"userId": ben.ID}), 204) // idempotent
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", ben.Token, map[string]string{"content": "hello?"}), 404)
	if len(e.expect(e.do("GET", "/matches", ben.Token, nil), 200).items(t)) != 0 {
		t.Fatal("block should remove the match")
	}
	e.expect(e.do("GET", "/users/"+ben.ID+"/profile", ana.Token, nil), 404)
	e.expect(e.do("GET", "/users/"+ana.ID+"/profile", ben.Token, nil), 404)
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", ben.Token, nil), 200).items(t), ana.ID) {
		t.Fatal("blocker must be invisible to the blocked")
	}
	blocked := e.expect(e.do("GET", "/blocks", ana.Token, nil), 200).items(t)
	if len(blocked) != 1 || blocked[0]["userId"] != ben.ID {
		t.Fatalf("block list: %v", blocked)
	}
	e.expect(e.do("POST", "/blocks", ana.Token, map[string]string{"userId": ana.ID}), 400)
	e.expect(e.do("DELETE", "/blocks/"+ben.ID, ana.Token, nil), 204)
}

func TestPassNeverReturnsAndPreferencesFilter(t *testing.T) {
	e := newEnv(t)
	ana := e.onboard("ana", "woman")
	ben := e.onboard("ben", "man")
	dan := e.onboard("dan", "man")

	e.expect(e.swipe(ana, ben, "pass"), 200)
	list := e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t)
	if containsUser(list, ben.ID) || !containsUser(list, dan.ID) {
		t.Fatalf("pass must hide ben only: %v", list)
	}
	// A passed person who likes you back later still cannot create a match through you.
	if r := e.expect(e.swipe(ben, ana, "like"), 200).json(t); r["matched"] != false {
		t.Fatal("ben liked ana but ana passed: no match")
	}

	// Gender preference: Ana only wants women now -> no men.
	e.expect(e.do("PUT", "/preferences", ana.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": true}), 200)
	if list = e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t); containsUser(list, dan.ID) {
		t.Fatal("gender preference ignored")
	}
	// ... and she can no longer swipe on him either (server-side enforcement).
	e.expect(e.swipe(ana, dan, "like"), 404)

	// Mutual: Dan wants only women, and Ana now is a woman -> visible to Dan. Ben only wants men -> hidden from Dan.
	e.expect(e.do("PUT", "/preferences", dan.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": true}), 200)
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", dan.Token, nil), 200).items(t), ana.ID) {
		t.Fatal("dan does not want women: ana must not be shown")
	}
	e.expect(e.do("PUT", "/preferences", dan.Token, map[string]any{"interestedIn": []string{"woman", "man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": false, "showDistance": true}), 200)
	// Paused accounts disappear from discovery.
	e.expect(e.do("PUT", "/preferences", ana.Token, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": true}), 200)
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t), dan.ID) {
		t.Fatal("paused account must be hidden")
	}

	// Age filter: Ana asks for 18-25, Ben is ~30.
	e.expect(e.do("PUT", "/preferences", ana.Token, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 18, "maxAge": 25, "maxDistanceKm": 50, "discoverable": true, "showDistance": true}), 200)
	e.expect(e.do("PUT", "/preferences", dan.Token, map[string]any{"interestedIn": []string{"woman", "man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": true}), 200)
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t), dan.ID) {
		t.Fatal("age filter ignored")
	}

	// Distance filter + privacy toggle.
	e.expect(e.do("PUT", "/preferences", ana.Token, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": true}), 200)
	e.expect(e.do("PUT", "/profile/me/location", dan.Token, map[string]float64{"latitude": e.lat + 0.3, "longitude": e.lng}), 204) // ~33 km
	e.expect(e.do("PUT", "/preferences", dan.Token, map[string]any{"interestedIn": []string{"woman", "man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": false}), 200)
	for _, it := range e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t) {
		if it["userId"] == dan.ID && it["distanceKm"] != nil {
			t.Fatal("distance must be hidden when showDistance=false")
		}
	}
	e.expect(e.do("PUT", "/preferences", ana.Token, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 10, "discoverable": true, "showDistance": true}), 200)
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", ana.Token, nil), 200).items(t), dan.ID) {
		t.Fatal("distance filter ignored")
	}
}

func TestSimultaneousLikesProduceExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.onboard("ana", "woman")
		b := e.onboard("ben", "man")
		var wg sync.WaitGroup
		results := make([]bool, 2)
		for i, pair := range [][2]person{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to person) {
				defer wg.Done()
				r := e.swipe(from, to, "like")
				if r.Status != 200 {
					t.Errorf("swipe status %d: %s", r.Status, r.Body)
					return
				}
				results[i], _ = r.json(t)["matched"].(bool)
			}(i, pair[0], pair[1])
		}
		wg.Wait()
		if results[0] == results[1] {
			t.Fatalf("round %d: exactly one of the two swipes must report the match, got %v", round, results)
		}
		var matches, conversations int
		_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&matches)
		_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM conversations c JOIN matches m ON m.id=c.match_id WHERE m.user_a IN ($1,$2) AND m.user_b IN ($1,$2)`, a.ID, b.ID).Scan(&conversations)
		if matches != 1 || conversations != 1 {
			t.Fatalf("round %d: matches=%d conversations=%d", round, matches, conversations)
		}
	}
}

func TestReportAndUnmatch(t *testing.T) {
	e := newEnv(t)
	ana := e.onboard("ana", "woman")
	ben := e.onboard("ben", "man")
	e.expect(e.swipe(ana, ben, "like"), 200)
	m := e.expect(e.swipe(ben, ana, "like"), 200).json(t)["match"].(map[string]any)
	convID := m["conversationId"].(string)
	e.expect(e.do("POST", "/conversations/"+convID+"/messages", ben.Token, map[string]string{"content": "rude message"}), 201)

	e.expect(e.do("POST", "/reports", ana.Token, map[string]string{"targetUserId": ben.ID, "reason": "nonsense"}), 400)
	e.expect(e.do("POST", "/reports", ana.Token, map[string]string{"targetUserId": ana.ID, "reason": "spam"}), 400)
	e.expect(e.do("POST", "/reports", ana.Token, map[string]string{"targetUserId": ben.ID, "reason": "harassment", "details": strings.Repeat("x", 1001)}), 400)
	id := e.expect(e.do("POST", "/reports", ana.Token, map[string]string{"targetUserId": ben.ID, "reason": "harassment", "details": "insults"}), 201).json(t)["id"].(string)

	var reason, snapshot string
	if err := e.pool.QueryRow(t.Context(), `SELECT reason, context::text FROM reports WHERE id=$1`, id).Scan(&reason, &snapshot); err != nil {
		t.Fatal(err)
	}
	if reason != "harassment" || !strings.Contains(snapshot, "rude message") {
		t.Fatalf("report should snapshot the conversation: %s %s", reason, snapshot)
	}

	// Unmatch: only a participant may do it; it removes the chat, and the pair never reappears.
	e.expect(e.do("DELETE", "/matches/"+m["id"].(string), e.register("stranger").Token, nil), 404)
	e.expect(e.do("DELETE", "/matches/"+m["id"].(string), ana.Token, nil), 204)
	e.expect(e.do("GET", "/conversations/"+convID+"/messages", ben.Token, nil), 404)
	if containsUser(e.expect(e.do("GET", "/discover?limit=20", ben.Token, nil), 200).items(t), ana.ID) {
		t.Fatal("unmatched pair must not reappear")
	}
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := newEnv(t)
	ana := e.onboard("ana", "woman", "hiking")
	ben := e.onboard("ben", "man")
	e.expect(e.swipe(ana, ben, "like"), 200)
	m := e.expect(e.swipe(ben, ana, "like"), 200).json(t)["match"].(map[string]any)
	e.expect(e.do("POST", "/conversations/"+m["conversationId"].(string)+"/messages", ana.Token, map[string]string{"content": "bye soon"}), 201)
	photoURL := e.expect(e.do("GET", "/photos", ana.Token, nil), 200).items(t)[0]["url"].(string)

	e.expect(e.do("DELETE", "/me", ana.Token, map[string]string{"password": "wrong-password"}), 403)
	e.expect(e.do("DELETE", "/me", "", map[string]string{"password": "Sup3rSecret!"}), 401)
	e.expect(e.do("DELETE", "/me", ana.Token, map[string]string{"password": "Sup3rSecret!"}), 204)

	e.expect(e.do("GET", "/me", ana.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": ana.Email, "password": "Sup3rSecret!"}), 401)
	e.expect(e.do("GET", photoURL, "", nil), 404)
	if n := len(e.expect(e.do("GET", "/matches", ben.Token, nil), 200).items(t)); n != 0 {
		t.Fatalf("ben still has %d matches", n)
	}
	for table, col := range map[string]string{"profiles": "user_id", "preferences": "user_id", "photos": "user_id", "user_interests": "user_id", "swipes": "swiper_id", "sessions": "user_id", "messages": "sender_id", "notifications": "user_id"} {
		var n int
		if err := e.pool.QueryRow(t.Context(), `SELECT count(*) FROM `+table+` WHERE `+col+`=$1`, ana.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s still holds %d rows for the deleted user", table, n)
		}
	}
}

func TestNotificationsAndRateLimit(t *testing.T) {
	e := newEnv(t)
	p := e.register("notif")
	e.expect(e.do("POST", "/notifications/read-all", p.Token, nil), 204)
	e.expect(e.do("POST", "/notifications/11111111-1111-1111-1111-111111111111/read", p.Token, nil), 404)
	e.expect(e.do("POST", "/notifications/not-a-uuid/read", p.Token, nil), 404)

	// Credential endpoints are throttled per IP.
	var limited bool
	for i := 0; i < 30 && !limited; i++ {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"x@test.invalid","password":"whatever12"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.77:5555"
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		if rec.Code == 429 {
			limited = true
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After expected")
			}
		}
	}
	if !limited {
		t.Fatal("expected login to be rate limited")
	}
}
