package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := "Auth.User@Test.Local"

	e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": email, "password": "short"}), 400)
	e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": "not-an-email", "password": "longenough1"}), 400)
	reg := e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": "  " + email + " ", "password": "Sup3rSecret!"}), 201)
	if reg.str("user", "email") != strings.ToLower(email) {
		t.Fatalf("email should be normalized: %s", reg.Raw)
	}
	if reg.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("auth responses must not be cached")
	}
	if r := e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": strings.ToLower(email), "password": "Sup3rSecret!"}), 409); r.str("code") != "email_taken" {
		t.Fatalf("expected email_taken: %s", r.Raw)
	}
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": email, "password": "wrong-password"}), 401)
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": "nobody@test.local", "password": "wrong-password"}), 401)

	login := e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": email, "password": "Sup3rSecret!"}), 200)
	refresh1 := login.str("refreshToken")
	rotated := e.expect(e.call("POST", "/auth/refresh", "", map[string]any{"refreshToken": refresh1}), 200)
	if rotated.str("refreshToken") == refresh1 {
		t.Fatal("refresh token must rotate")
	}
	// The old refresh token is single-use.
	e.expect(e.call("POST", "/auth/refresh", "", map[string]any{"refreshToken": refresh1}), 401)
	e.expect(e.call("GET", "/me", rotated.str("accessToken"), nil), 200)

	e.expect(e.call("POST", "/auth/logout", "", map[string]any{"refreshToken": rotated.str("refreshToken")}), 204)
	e.expect(e.call("POST", "/auth/refresh", "", map[string]any{"refreshToken": rotated.str("refreshToken")}), 401)

	// Forged / malformed tokens.
	e.expect(e.call("GET", "/me", "", nil), 401)
	e.expect(e.call("GET", "/me", "garbage", nil), 401)
	uid := reg.str("user", "id")
	forge := func(method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	valid := jwt.MapClaims{"uid": uid, "sid": "x", "exp": time.Now().Add(time.Hour).Unix()}
	e.expect(e.call("GET", "/me", forge(jwt.SigningMethodHS256, []byte("another-secret-another-secret-xx"), valid), nil), 401)
	e.expect(e.call("GET", "/me", forge(jwt.SigningMethodHS512, []byte(testSecret), valid), nil), 401)
	e.expect(e.call("GET", "/me", forge(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid), nil), 401)
	e.expect(e.call("GET", "/me", forge(jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{"uid": uid, "sid": "x"}), nil), 401) // no exp
	e.expect(e.call("GET", "/me", forge(jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{"uid": uid, "sid": "x", "exp": time.Now().Add(-time.Minute).Unix()}), nil), 401)
	e.expect(e.call("GET", "/me", forge(jwt.SigningMethodHS256, []byte(testSecret), jwt.MapClaims{"uid": uid, "exp": time.Now().Add(time.Hour).Unix()}), nil), 401) // no sid
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	u := e.onboard(userSpec{Name: "Rec", Gender: "woman", Age: 30, InterestedIn: []string{"man"}})

	// Unknown email: same 202, no mail (no account enumeration).
	e.expect(e.call("POST", "/auth/forgot-password", "", map[string]any{"email": "ghost@test.local"}), 202)

	e.expect(e.call("POST", "/auth/forgot-password", "", map[string]any{"email": u.Email}), 202)
	code := e.mail.lastCode(t, u.Email)

	// Wrong code is rejected; the right one works once and revokes sessions.
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"email": u.Email, "code": "AAAAAAAA", "password": "NewPassw0rd!"}), 400)
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"email": u.Email, "code": strings.ToLower(code), "password": "short"}), 400)
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"email": u.Email, "code": strings.ToLower(code), "password": "NewPassw0rd!"}), 204)
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"email": u.Email, "code": code, "password": "Another0ne!!"}), 400) // single use
	e.expect(e.call("POST", "/auth/refresh", "", map[string]any{"refreshToken": u.Refresh}), 401)
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": u.Email, "password": u.Password}), 401)
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": u.Email, "password": "NewPassw0rd!"}), 200)

	// Brute force: after 5 wrong guesses even the right code is refused.
	e.expect(e.call("POST", "/auth/forgot-password", "", map[string]any{"email": u.Email}), 202)
	time.Sleep(50 * time.Millisecond)
	code2 := e.mail.lastCode(t, u.Email)
	if code2 == code {
		t.Skip("improbable code collision")
	}
	for i := 0; i < 5; i++ {
		e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"email": u.Email, "code": fmt.Sprintf("ZZZZZZZ%d", i+2), "password": "NewPassw0rd!2"}), 400)
	}
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"email": u.Email, "code": code2, "password": "NewPassw0rd!2"}), 400)
}

func TestChangePasswordAndDeleteAccount(t *testing.T) {
	e := newEnv(t)
	a := e.onboard(userSpec{Name: "Ann", Gender: "woman", Age: 28, InterestedIn: []string{"man"}})
	b := e.onboard(userSpec{Name: "Ben", Gender: "man", Age: 30, InterestedIn: []string{"woman"}})

	e.expect(e.call("POST", "/me/password", a.Token, map[string]any{"currentPassword": "bad-current", "newPassword": "Brand-new-pass1"}), 403)
	e.expect(e.call("POST", "/me/password", a.Token, map[string]any{"currentPassword": a.Password, "newPassword": "short"}), 400)
	e.expect(e.call("POST", "/me/password", a.Token, map[string]any{"currentPassword": a.Password, "newPassword": "Brand-new-pass1"}), 204)
	e.expect(e.call("POST", "/auth/refresh", "", map[string]any{"refreshToken": a.Refresh}), 401)
	a.Password = "Brand-new-pass1"

	// Build a match with a message, then delete Ann's account.
	sw := e.expect(e.call("POST", "/swipes", b.Token, map[string]any{"targetUserId": a.ID, "action": "like"}), 200)
	_ = sw
	m := e.expect(e.call("POST", "/swipes", a.Token, map[string]any{"targetUserId": b.ID, "action": "like"}), 200)
	conv := m.str("conversation", "id")
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "hi"}), 201)
	e.expect(e.call("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam"}), 201)

	var keys []string
	rows, err := e.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) != 1 {
		t.Fatalf("expected 1 photo before deletion, got %d", len(keys))
	}

	e.expect(e.call("DELETE", "/me", a.Token, map[string]any{"password": "wrong"}), 403)
	e.expect(e.call("DELETE", "/me", a.Token, map[string]any{"password": a.Password}), 204)

	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": a.Email, "password": a.Password}), 401)
	if list := e.expect(e.call("GET", "/conversations", b.Token, nil), 200); len(list.list("conversations")) != 0 {
		t.Fatal("deleted user's conversations must vanish for the peer")
	}
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "swiper_id", "messages": "sender_id", "match_participants": "user_id", "preferences": "user_id", "sessions": "user_id"} {
		var n int
		if err := e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s = $1`, table, col), a.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s still has %d rows for the deleted user", table, n)
		}
	}
	// Deleting the reporter removes their reports; the swipe others made on them is gone via cascade as well.
	var reports int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE reporter_id = $1`, a.ID).Scan(&reports)
	if reports != 0 {
		t.Fatal("reports of the deleted reporter should be removed")
	}
	// Their old access token still parses but can no longer act as a user.
	e.expect(e.call("GET", "/me", a.Token, nil), 401)
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": "pv@test.local", "password": "Sup3rSecret!"}), 201)
	tok := u.str("accessToken")

	e.expect(e.call("GET", "/profile", tok, nil), 404)
	base := func() map[string]any {
		return map[string]any{"firstName": "Zoé", "birthDate": birthDateForAge(25), "gender": "woman", "bio": "", "city": "", "interestIds": []int{1}}
	}
	with := func(k string, v any) map[string]any { m := base(); m[k] = v; return m }

	if r := e.expect(e.call("PUT", "/profile", tok, with("birthDate", time.Now().AddDate(-17, 0, 0).Format("2006-01-02"))), 400); r.str("code") != "underage" {
		t.Fatalf("expected underage: %s", r.Raw)
	}
	// Turning 18 tomorrow is still underage today.
	e.expect(e.call("PUT", "/profile", tok, with("birthDate", time.Now().AddDate(-18, 0, 1).Format("2006-01-02"))), 400)
	e.expect(e.call("PUT", "/profile", tok, with("birthDate", "2000-13-45")), 400)
	e.expect(e.call("PUT", "/profile", tok, with("birthDate", "1850-01-01")), 400)
	e.expect(e.call("PUT", "/profile", tok, with("firstName", "   ")), 400)
	e.expect(e.call("PUT", "/profile", tok, with("firstName", strings.Repeat("x", 41))), 400)
	e.expect(e.call("PUT", "/profile", tok, with("firstName", "Bad\x00Name")), 400)
	e.expect(e.call("PUT", "/profile", tok, with("gender", "robot")), 400)
	e.expect(e.call("PUT", "/profile", tok, with("bio", strings.Repeat("é", 501))), 400)
	e.expect(e.call("PUT", "/profile", tok, with("interestIds", []int{9999})), 400)
	e.expect(e.call("PUT", "/profile", tok, with("interestIds", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11})), 400)

	ok := e.expect(e.call("PUT", "/profile", tok, with("firstName", "  Zoé  ")), 200)
	if ok.str("firstName") != "Zoé" || ok.Body["showDistance"] != true || ok.Body["discoverable"] != true {
		t.Fatalf("defaults/trim wrong: %s", ok.Raw)
	}
	// Birth date is locked after creation; other fields stay editable.
	if r := e.expect(e.call("PUT", "/profile", tok, with("birthDate", birthDateForAge(40))), 400); r.str("code") != "birth_date_locked" {
		t.Fatalf("expected birth_date_locked: %s", r.Raw)
	}
	hide := base()
	hide["discoverable"] = false
	if r := e.expect(e.call("PUT", "/profile", tok, hide), 200); r.Body["discoverable"] != false {
		t.Fatal("discoverable should update")
	}
	// Omitting the flags keeps stored values.
	keep := e.expect(e.call("PUT", "/profile", tok, base()), 200)
	if keep.Body["discoverable"] != false {
		t.Fatal("omitted flags must keep the stored value")
	}

	// Location is coarsened server-side (never stored precisely).
	e.expect(e.call("PUT", "/location", tok, map[string]any{"latitude": 91, "longitude": 0}), 400)
	e.expect(e.call("PUT", "/location", tok, map[string]any{"latitude": 48.85661234, "longitude": 2.35219876, "city": "Paris"}), 200)
	var lat, lon float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.str("user", "id")).Scan(&lat, &lon); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lon != 2.35 {
		t.Fatalf("coordinates must be stored on a ~1 km grid, got %v,%v", lat, lon)
	}

	e.expect(e.call("PUT", "/preferences", tok, map[string]any{"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(e.call("PUT", "/preferences", tok, map[string]any{"interestedIn": []string{"woman"}, "minAge": 17, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(e.call("PUT", "/preferences", tok, map[string]any{"interestedIn": []string{"woman"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(e.call("PUT", "/preferences", tok, map[string]any{"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(e.call("PUT", "/preferences", tok, map[string]any{"interestedIn": []string{"woman"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 1000}), 400)
	dedup := e.expect(e.call("PUT", "/preferences", tok, map[string]any{"interestedIn": []string{"woman", "woman"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}), 200)
	if len(dedup.list("interestedIn")) != 1 {
		t.Fatal("duplicate genders should be collapsed")
	}

	// Incomplete profile (no photo): cannot discover or swipe.
	if r := e.expect(e.call("GET", "/discover", tok, nil), 403); r.str("code") != "profile_incomplete" {
		t.Fatalf("expected profile_incomplete: %s", r.Raw)
	}
	me := e.expect(e.call("GET", "/me", tok, nil), 200)
	if me.Body["profileComplete"] != false {
		t.Fatal("profile without photo is incomplete")
	}
	e.expect(e.call("GET", "/interests", tok, nil), 200)
}

func TestPhotoRules(t *testing.T) {
	e := newEnv(t)
	noProfile := e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": "np@test.local", "password": "Sup3rSecret!"}), 201)
	if r := e.upload(noProfile.str("accessToken"), "photo", "a.jpg", "image/jpeg", testJPEG(400, 400)); r.Status != 404 {
		t.Fatalf("photo without profile should be 404, got %d", r.Status)
	}

	u := e.onboard(userSpec{Name: "Pho", Gender: "woman", Age: 30, InterestedIn: []string{"man"}, NoPhoto: true})
	other := e.onboard(userSpec{Name: "Oth", Gender: "man", Age: 30, InterestedIn: []string{"woman"}})

	// Content sniffing, not the declared type / extension, decides.
	if r := e.upload(u.Token, "photo", "evil.jpg", "image/jpeg", []byte("<?php echo 1; ?>")); r.Status != 415 {
		t.Fatalf("non image should be 415, got %d %s", r.Status, r.Raw)
	}
	gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
	if r := e.upload(u.Token, "photo", "a.gif", "image/gif", gif); r.Status != 415 {
		t.Fatalf("gif should be 415, got %d", r.Status)
	}
	// Valid JPEG header followed by garbage.
	broken := append(testJPEG(300, 300)[:400], []byte("garbage-garbage-garbage")...)
	if r := e.upload(u.Token, "photo", "b.jpg", "image/jpeg", broken); r.Status != 400 {
		t.Fatalf("truncated image should be 400, got %d", r.Status)
	}
	if r := e.upload(u.Token, "photo", "tiny.jpg", "image/jpeg", testJPEG(50, 50)); r.Status != 400 {
		t.Fatalf("tiny image should be 400, got %d", r.Status)
	}
	if r := e.upload(u.Token, "wrongfield", "a.jpg", "image/jpeg", testJPEG(300, 300)); r.Status != 400 {
		t.Fatalf("wrong field should be 400, got %d", r.Status)
	}
	huge := append(testJPEG(300, 300), make([]byte, 9<<20)...)
	if r := e.upload(u.Token, "photo", "huge.jpg", "image/jpeg", huge); r.Status != 413 {
		t.Fatalf("oversized upload should be 413, got %d", r.Status)
	}

	// PNG with transparency and a large JPEG are both normalised to bounded JPEGs.
	p1 := e.expect(e.upload(u.Token, "photo", "a.png", "image/png", testPNG(500, 500)), 201)
	p2 := e.expect(e.upload(u.Token, "photo", "b.jpg", "image/jpeg", testJPEG(3000, 2000)), 201)
	if p2.Body["width"].(float64) != 1600 || p2.Body["height"].(float64) != 1066 {
		t.Fatalf("large image should be scaled to 1600px, got %v x %v", p2.Body["width"], p2.Body["height"])
	}
	if p1.Body["position"].(float64) != 0 || p2.Body["position"].(float64) != 1 {
		t.Fatal("positions should follow upload order")
	}
	for i := 0; i < 4; i++ {
		e.expect(e.upload(u.Token, "photo", "c.jpg", "image/jpeg", testJPEG(300, 300)), 201)
	}
	if r := e.upload(u.Token, "photo", "seventh.jpg", "image/jpeg", testJPEG(300, 300)); r.Status != 409 {
		t.Fatalf("7th photo should be 409, got %d", r.Status)
	}

	prof := e.expect(e.call("GET", "/profile", u.Token, nil), 200)
	photos := prof.list("photos")
	ids := make([]string, len(photos))
	for i, p := range photos {
		ids[i] = p.(map[string]any)["id"].(string)
	}
	// Reordering: make the last photo the main one.
	reordered := append([]string{ids[5]}, ids[:5]...)
	r := e.expect(e.call("PUT", "/photos/order", u.Token, map[string]any{"photoIds": reordered}), 200)
	if r.list("photos")[0].(map[string]any)["id"] != ids[5] {
		t.Fatal("first id should be the main photo")
	}
	e.expect(e.call("PUT", "/photos/order", u.Token, map[string]any{"photoIds": reordered[:5]}), 400)                   // incomplete
	e.expect(e.call("PUT", "/photos/order", u.Token, map[string]any{"photoIds": append(reordered[:5:5], ids[5])}), 400) // duplicate
	foreign := e.expect(e.call("GET", "/profile", other.Token, nil), 200).list("photos")[0].(map[string]any)["id"].(string)
	e.expect(e.call("PUT", "/photos/order", u.Token, map[string]any{"photoIds": append(reordered[:5:5], foreign)}), 400) // foreign photo

	// Others cannot delete my photos; I can; positions are compacted.
	e.expect(e.call("DELETE", "/photos/"+ids[0], other.Token, nil), 404)
	e.expect(e.call("DELETE", "/photos/"+ids[5], u.Token, nil), 204)
	e.expect(e.call("DELETE", "/photos/"+ids[5], u.Token, nil), 404)
	after := e.expect(e.call("GET", "/profile", u.Token, nil), 200).list("photos")
	for i, p := range after {
		if int(p.(map[string]any)["position"].(float64)) != i {
			t.Fatalf("positions not compacted: %v", after)
		}
	}
	e.expect(e.call("DELETE", "/photos/not-a-uuid", u.Token, nil), 404)
}

func TestConversationPermissionsAndBlocking(t *testing.T) {
	e := newEnv(t)
	a := e.onboard(userSpec{Name: "A", Gender: "woman", Age: 30, InterestedIn: []string{"man"}})
	b := e.onboard(userSpec{Name: "B", Gender: "man", Age: 30, InterestedIn: []string{"woman"}})
	c := e.onboard(userSpec{Name: "C", Gender: "man", Age: 30, InterestedIn: []string{"woman"}})

	e.expect(e.call("POST", "/swipes", a.Token, map[string]any{"targetUserId": b.ID, "action": "like"}), 200)
	m := e.expect(e.call("POST", "/swipes", b.Token, map[string]any{"targetUserId": a.ID, "action": "like"}), 200)
	conv := m.str("conversation", "id")
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "secret"}), 201)

	// Outsider C: every conversation endpoint answers 404 (no existence leak).
	e.expect(e.call("GET", "/conversations/"+conv+"/messages", c.Token, nil), 404)
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", c.Token, map[string]any{"body": "intrusion"}), 404)
	e.expect(e.call("POST", "/conversations/"+conv+"/read", c.Token, nil), 404)
	e.expect(e.call("DELETE", "/conversations/"+conv, c.Token, nil), 404)
	e.expect(e.call("POST", "/conversations/"+conv+"/unmatch", c.Token, nil), 404)
	e.expect(e.call("GET", "/conversations/not-a-uuid/messages", a.Token, nil), 404)
	if l := e.expect(e.call("GET", "/conversations", c.Token, nil), 200); len(l.list("conversations")) != 0 {
		t.Fatal("outsider must not list others' conversations")
	}
	// C cannot report a conversation they are not part of.
	e.expect(e.call("POST", "/reports", c.Token, map[string]any{"userId": a.ID, "reason": "spam", "conversationId": conv}), 400)

	// Message validation.
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "   "}), 400)
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": strings.Repeat("a", 2001)}), 400)
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "bad\x07bell"}), 400)

	// Block: self-block refused, unknown user 404, idempotent.
	e.expect(e.call("POST", "/blocks", a.Token, map[string]any{"userId": a.ID}), 400)
	e.expect(e.call("POST", "/blocks", a.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000"}), 404)
	e.expect(e.call("POST", "/blocks", a.Token, map[string]any{"userId": b.ID}), 204)
	e.expect(e.call("POST", "/blocks", a.Token, map[string]any{"userId": b.ID}), 204)

	if r := e.expect(e.call("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "hello?"}), 403); r.str("code") != "blocked" {
		t.Fatalf("blocker should get 'blocked': %s", r.Raw)
	}
	// The blocked person learns nothing: the conversation is simply gone.
	e.expect(e.call("POST", "/conversations/"+conv+"/messages", b.Token, map[string]any{"body": "hello?"}), 404)
	e.expect(e.call("GET", "/conversations/"+conv+"/messages", b.Token, nil), 404)
	for _, tok := range []string{a.Token, b.Token} {
		if l := e.expect(e.call("GET", "/conversations", tok, nil), 200); len(l.list("conversations")) != 0 {
			t.Fatal("blocked pair must not see the conversation")
		}
	}
	e.expect(e.call("GET", "/profiles/"+b.ID, a.Token, nil), 404)
	e.expect(e.call("GET", "/profiles/"+a.ID, b.Token, nil), 404)
	bl := e.expect(e.call("GET", "/blocks", a.Token, nil), 200)
	if len(bl.list("blocks")) != 1 || bl.list("blocks")[0].(map[string]any)["firstName"] != "B" {
		t.Fatalf("blocks list wrong: %s", bl.Raw)
	}
	// Blocked users are not swipe-able and their photos are not served.
	e.expect(e.call("POST", "/swipes", b.Token, map[string]any{"targetUserId": a.ID, "action": "like"}), 404)
	aPhoto := e.expect(e.call("GET", "/profile", a.Token, nil), 200).list("photos")[0].(map[string]any)["url"].(string)
	req, _ := http.NewRequest("GET", e.srv.URL+aPhoto, nil)
	req.Header.Set("Authorization", "Bearer "+b.Token)
	if r := e.do(req); r.Status != 404 {
		t.Fatalf("blocked user must not fetch photos, got %d", r.Status)
	}

	// Unblock restores profile visibility (the match itself is back too since it was only hidden).
	e.expect(e.call("DELETE", "/blocks/"+b.ID, a.Token, nil), 204)
	e.expect(e.call("GET", "/profiles/"+b.ID, a.Token, nil), 200)

	// Reports.
	e.expect(e.call("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "bogus"}), 400)
	e.expect(e.call("POST", "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"}), 400)
	e.expect(e.call("POST", "/reports", a.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}), 404)
	e.expect(e.call("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": strings.Repeat("x", 1001)}), 400)
	e.expect(e.call("POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "rude", "conversationId": conv}), 201)
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open' AND match_id = $3`, a.ID, b.ID, conv).Scan(&n)
	if n != 1 {
		t.Fatal("report should be stored")
	}
}

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	me := e.onboard(userSpec{Name: "Me", Gender: "man", Age: 30, InterestedIn: []string{"woman", "nonbinary"}, MinAge: 25, MaxAge: 35, MaxDistance: 50, HasLocation: true, Lat: 48.85, Lon: 2.35})
	near := e.onboard(userSpec{Name: "Near", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, HasLocation: true, Lat: 48.90, Lon: 2.35})
	far := e.onboard(userSpec{Name: "Far", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, HasLocation: true, Lat: 45.76, Lon: 4.84}) // Lyon, ~390 km
	tooOld := e.onboard(userSpec{Name: "Old", Gender: "woman", Age: 50, InterestedIn: []string{"man"}, HasLocation: true, Lat: 48.86, Lon: 2.34})
	notMe := e.onboard(userSpec{Name: "NotMe", Gender: "woman", Age: 28, InterestedIn: []string{"woman"}, HasLocation: true, Lat: 48.86, Lon: 2.34})
	tooYoungForMe := e.onboard(userSpec{Name: "Picky", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, MinAge: 18, MaxAge: 25, HasLocation: true, Lat: 48.86, Lon: 2.34})
	nb := e.onboard(userSpec{Name: "Enby", Gender: "nonbinary", Age: 29, InterestedIn: []string{"man"}, HasLocation: true, Lat: 48.86, Lon: 2.34})
	hidden := e.onboard(userSpec{Name: "Hidden", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, HasLocation: true, Lat: 48.86, Lon: 2.34})
	hid := e.expect(e.call("GET", "/profile", hidden.Token, nil), 200)
	e.expect(e.call("PUT", "/profile", hidden.Token, map[string]any{"firstName": "Hidden", "birthDate": hid.str("birthDate"), "gender": "woman", "discoverable": false, "interestIds": []int{}}), 200)
	noPhoto := e.onboard(userSpec{Name: "NoPhoto", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, NoPhoto: true})

	got := map[string]bool{}
	for _, p := range e.expect(e.call("GET", "/discover?limit=20", me.Token, nil), 200).list("profiles") {
		got[p.(map[string]any)["userId"].(string)] = true
	}
	for name, want := range map[string]struct {
		id   string
		show bool
	}{
		"near": {near.ID, true}, "enby": {nb.ID, true},
		"far": {far.ID, false}, "too old for me": {tooOld.ID, false}, "not into men": {notMe.ID, false},
		"I am too old for them": {tooYoungForMe.ID, false}, "hidden": {hidden.ID, false}, "no photo": {noPhoto.ID, false}, "myself": {me.ID, false},
	} {
		if got[want.id] != want.show {
			t.Errorf("%s: visible=%v, want %v", name, got[want.id], want.show)
		}
	}

	// Server-side rule enforcement on swipes: a filtered-out profile cannot be liked directly.
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": far.ID, "action": "like"}), 404)
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": hidden.ID, "action": "like"}), 404)
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": me.ID, "action": "like"}), 400)
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": near.ID, "action": "superlike"}), 400)
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": "nope", "action": "like"}), 400)
	e.expect(e.call("GET", "/profiles/"+hidden.ID, me.Token, nil), 404)

	// Pass hides a profile; reset-passes brings it back.
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": near.ID, "action": "pass"}), 200)
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": nb.ID, "action": "pass"}), 200)
	if l := e.expect(e.call("GET", "/discover", me.Token, nil), 200); len(l.list("profiles")) != 0 {
		t.Fatalf("passed profiles must not reappear: %s", l.Raw)
	}
	e.expect(e.call("POST", "/swipes/reset-passes", me.Token, nil), 204)
	if l := e.expect(e.call("GET", "/discover", me.Token, nil), 200); len(l.list("profiles")) != 2 {
		t.Fatalf("reset should restore 2 profiles: %s", l.Raw)
	}
	// A pass never produces a match even if the other side liked first.
	e.expect(e.call("POST", "/swipes", near.Token, map[string]any{"targetUserId": me.ID, "action": "like"}), 200)
	e.expect(e.call("POST", "/swipes", me.Token, map[string]any{"targetUserId": near.ID, "action": "pass"}), 200)
	var matches int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches`).Scan(&matches)
	if matches != 0 {
		t.Fatal("pass must not create a match")
	}
}

// Two people liking each other at the same instant must produce exactly one match.
func TestConcurrentMutualLikes(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.onboard(userSpec{Name: fmt.Sprintf("A%d", round), Gender: "woman", Age: 30, InterestedIn: []string{"man"}})
		b := e.onboard(userSpec{Name: fmt.Sprintf("B%d", round), Gender: "man", Age: 30, InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]result, 2)
		for i, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to user) {
				defer wg.Done()
				results[i] = e.call("POST", "/swipes", from.Token, map[string]any{"targetUserId": to.ID, "action": "like"})
			}(i, pair[0], pair[1])
		}
		wg.Wait()
		for _, r := range results {
			if r.Status != 200 {
				t.Fatalf("swipe failed: %d %s", r.Status, r.Raw)
			}
		}
		if results[0].Body["matched"] != true && results[1].Body["matched"] != true {
			t.Fatalf("round %d: simultaneous mutual likes produced no match", round)
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, n)
		}
	}
}

func TestRateLimitAndHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.call("GET", "/health", "", nil), 200)
	var limited bool
	for i := 0; i < 30; i++ {
		if e.call("POST", "/auth/login", "", map[string]any{"email": "x@test.local", "password": "whatever-pass"}).Status == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("login must be rate limited per IP")
	}
}

func TestWebSocketAuth(t *testing.T) {
	e := newEnv(t)
	u := e.onboard(userSpec{Name: "Ws", Gender: "woman", Age: 30, InterestedIn: []string{"man"}})
	conn, events := e.ws(u.Token)
	_ = conn.WriteJSON(map[string]string{"type": "ping"})
	waitEvent(t, events, "pong")

	// A bad token is rejected with the application close code.
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	bad, _, err := websocketDial(url)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "nope"})
	_, _, err = bad.ReadMessage()
	if err == nil {
		t.Fatal("connection with an invalid token must be closed")
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if err := db.RunMigrations(ctx, e.pool); err != nil {
		t.Fatalf("second migration run must be a no-op: %v", err)
	}
	var applied int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil || applied < 1 {
		t.Fatalf("migrations should be tracked: %d %v", applied, err)
	}
	// Required indexes from the project rules exist.
	for _, idx := range []string{"idx_messages_match_created", "idx_notifications_user_created", "idx_photos_owner_created", "idx_match_participants_user", "idx_swipes_target_likes"} {
		var ok bool
		if err := e.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1 AND schemaname = current_schema())`, idx).Scan(&ok); err != nil || !ok {
			t.Errorf("missing index %s", idx)
		}
	}
}
