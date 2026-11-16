package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealth(t *testing.T) {
	env := newTestEnv(t)
	r := env.expect(env.do("GET", "/health", "", nil), 200)
	if !contains(r.Body, `"ok"`) {
		t.Fatalf("unexpected health body %s", r.Body)
	}
	if r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("expected security headers")
	}
}

func TestAuthLifecycle(t *testing.T) {
	env := newTestEnv(t)
	email := env.uniqueEmail("auth")

	env.expect(env.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "short"}), 400)
	env.expect(env.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "onlyletters"}), 400)
	env.expect(env.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": testPassword}), 400)

	var reg authPayload
	env.expect(env.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": testPassword}), 201).decode(t, &reg)
	if reg.User.Email != email || reg.User.HasProfile {
		t.Fatalf("unexpected user %+v", reg.User)
	}
	env.expect(env.do("POST", "/auth/register", "", map[string]string{"email": email, "password": testPassword}), 409)

	env.expect(env.do("POST", "/auth/login", "", map[string]string{"email": email, "password": "WrongPass1"}), 401)
	env.expect(env.do("POST", "/auth/login", "", map[string]string{"email": "nobody@it.test", "password": testPassword}), 401)
	var login authPayload
	env.expect(env.do("POST", "/auth/login", "", map[string]string{"email": email, "password": testPassword}), 200).decode(t, &login)

	env.expect(env.do("GET", "/me", "", nil), 401)
	env.expect(env.do("GET", "/me", "garbage", nil), 401)
	env.expect(env.do("GET", "/me", login.AccessToken, nil), 200)

	// Refresh rotates: the old refresh token stops working.
	var refreshed authPayload
	env.expect(env.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 200).decode(t, &refreshed)
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	env.expect(env.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), 401)

	// Logout revokes the session immediately, including its access tokens.
	env.expect(env.do("POST", "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}), 204)
	env.expect(env.do("GET", "/me", refreshed.AccessToken, nil), 401)
	env.expect(env.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refreshed.RefreshToken}), 401)

	// The registration session is independent and still valid.
	env.expect(env.do("GET", "/me", reg.AccessToken, nil), 200)
}

func TestPasswordReset(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("reset")

	// Unknown emails get the same answer (no account enumeration).
	env.expect(env.do("POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@it.test"}), 202)
	env.expect(env.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	code := env.mailer.code(t, u.Email)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "NewPassword9"}), 400)
	env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "weak"}), 400)
	env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword9"}), 204)
	// Codes are single use and every existing session is revoked.
	env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "Another9pass"}), 400)
	env.expect(env.do("GET", "/me", u.Token, nil), 401)
	env.expect(env.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": testPassword}), 401)
	env.expect(env.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword9"}), 200)
}

func TestResetCodeBruteForceIsCapped(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("brute")
	env.expect(env.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	code := env.mailer.code(t, u.Email)
	wrong := "123456"
	if code == wrong {
		wrong = "654321"
	}
	for i := 0; i < 5; i++ {
		env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "NewPassword9"}), 400)
	}
	// After 5 failures even the right code is refused.
	env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword9"}), 400)
}

func TestProfileOnboardingAndValidation(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("profile")

	env.expect(env.do("GET", "/discover", u.Token, nil), 409)
	env.expect(env.upload("POST", "/me/photos", u.Token, testPNG(300, 300, 10)), 409)

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	r := env.expect(env.do("PUT", "/me/profile", u.Token, map[string]any{"firstName": "Léa", "birthdate": minor, "gender": "woman"}), 400)
	if !contains(r.Body, "18") {
		t.Fatalf("expected age message, got %s", r.Body)
	}
	env.expect(env.do("PUT", "/me/profile", u.Token, map[string]any{"firstName": "Léa", "birthdate": "1996-02-29", "gender": "robot"}), 400)

	var me struct {
		Profile *struct {
			FirstName string `json:"firstName"`
			Age       int    `json:"age"`
			Interests []struct {
				ID int `json:"id"`
			} `json:"interests"`
		} `json:"profile"`
		Completeness struct {
			IsComplete bool `json:"isComplete"`
			HasPhotos  bool `json:"hasPhotos"`
		} `json:"completeness"`
	}
	env.expect(env.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": "  Léa  ", "birthdate": "1996-02-29", "gender": "woman", "bio": "Hello", "interestIds": []int{1, 2, 3, 9999},
	}), 200).decode(t, &me)
	if me.Profile == nil || me.Profile.FirstName != "Léa" || len(me.Profile.Interests) != 3 || me.Completeness.IsComplete {
		t.Fatalf("unexpected profile %+v", me)
	}

	env.expect(env.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 20, "maxDistanceKm": 10}), 400)
	env.expect(env.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 25, "ageMax": 40, "maxDistanceKm": 30}), 200)
	env.expect(env.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 123.0, "longitude": 2.0}), 400)
	env.expect(env.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 48.856613, "longitude": 2.352222}), 204)

	var lat, lng float64
	if err := env.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("location must be stored coarsened, got %v,%v", lat, lng)
	}

	env.expect(env.upload("POST", "/me/photos", u.Token, testPNG(300, 300, 50)), 201)
	env.expect(env.do("GET", "/me/profile", u.Token, nil), 200).decode(t, &me)
	if !me.Completeness.IsComplete {
		t.Fatal("profile with photo and preferences should be complete")
	}
	var meAuth authPayload
	env.expect(env.do("GET", "/me", u.Token, nil), 200).decode(t, &meAuth.User)
	if !meAuth.User.HasProfile {
		t.Fatal("expected hasProfile=true")
	}
}

func TestPhotoManagement(t *testing.T) {
	env := newTestEnv(t)
	u := env.onboard("photos", profileSpec{Name: "Inès", Gender: "woman", InterestedIn: []string{"man"}})

	env.expect(env.upload("POST", "/me/photos", u.Token, []byte("GIF89a not really")), 400)
	env.expect(env.upload("POST", "/me/photos", u.Token, []byte("<script>alert(1)</script>")), 400)
	env.expect(env.upload("POST", "/me/photos", u.Token, testPNG(100, 100, 1)), 400)

	for i := 0; i < 5; i++ {
		env.expect(env.upload("POST", "/me/photos", u.Token, testPNG(300, 300, uint8(i*30))), 201)
	}
	env.expect(env.upload("POST", "/me/photos", u.Token, testPNG(300, 300, 7)), 409) // 7th photo

	type photo struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Position int    `json:"position"`
	}
	var list struct {
		Photos []photo `json:"photos"`
	}
	env.expect(env.do("GET", "/me/photos", u.Token, nil), 200).decode(t, &list)
	if len(list.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(list.Photos))
	}

	// Signed media URL works without auth; a tampered one does not.
	media := env.expect(env.do("GET", list.Photos[0].URL, "", nil), 200)
	if media.Header.Get("Content-Type") != "image/jpeg" || len(media.Body) < 100 {
		t.Fatalf("expected jpeg body, got %s", media.Header.Get("Content-Type"))
	}
	if !strings.HasPrefix(string(media.Body), "\xff\xd8") {
		t.Fatal("stored photo must be re-encoded JPEG")
	}
	env.expect(env.do("GET", list.Photos[0].URL+"x", "", nil), 404)
	env.expect(env.do("GET", "/media/photos/"+list.Photos[0].ID, "", nil), 404)

	// Reorder: reverse the list; the last photo becomes the main one.
	ids := make([]string, len(list.Photos))
	for i, p := range list.Photos {
		ids[len(ids)-1-i] = p.ID
	}
	env.expect(env.do("PUT", "/me/photos/order", u.Token, map[string]any{"photoIds": ids[:3]}), 400)
	env.expect(env.do("PUT", "/me/photos/order", u.Token, map[string]any{"photoIds": ids}), 200).decode(t, &list)
	if list.Photos[0].ID != ids[0] || list.Photos[5].ID != ids[5] {
		t.Fatal("reorder not applied")
	}

	// Replace keeps the slot and swaps the file.
	var oldKey string
	_ = env.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[0]).Scan(&oldKey)
	env.expect(env.upload("PUT", "/me/photos/"+ids[0], u.Token, testPNG(400, 300, 200)), 200)
	if _, err := os.Stat(filepath.Join(env.dir, oldKey)); !os.IsNotExist(err) {
		t.Fatal("replaced file must be deleted")
	}

	// Delete compacts positions.
	env.expect(env.do("DELETE", "/me/photos/"+ids[0], u.Token, nil), 204)
	env.expect(env.do("GET", "/me/photos", u.Token, nil), 200).decode(t, &list)
	if len(list.Photos) != 5 || list.Photos[0].Position != 0 || list.Photos[4].Position != 4 || list.Photos[0].ID != ids[1] {
		t.Fatalf("unexpected photos after delete: %+v", list.Photos)
	}

	// Another user cannot touch these photos.
	other := env.onboard("photos-other", profileSpec{Name: "Max", Gender: "man", InterestedIn: []string{"woman"}})
	env.expect(env.do("DELETE", "/me/photos/"+ids[1], other.Token, nil), 404)
	env.expect(env.upload("PUT", "/me/photos/"+ids[1], other.Token, testPNG(300, 300, 3)), 404)
}

func TestDiscoveryFilters(t *testing.T) {
	env := newTestEnv(t)
	paris := profileSpec{Lat: ptr(48.8566), Lng: ptr(2.3522)}

	alice := env.onboard("alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, AgeMin: 25, AgeMax: 45, Distance: 50, Lat: paris.Lat, Lng: paris.Lng, Interests: []int{1, 2}, Birthdate: "1992-01-01"})
	bob := env.onboard("bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Lat: ptr(48.80), Lng: ptr(2.30), Interests: []int{2, 3}, Birthdate: "1990-01-01"})
	gayMan := env.onboard("carl", profileSpec{Name: "Carl", Gender: "man", InterestedIn: []string{"man"}, Lat: paris.Lat, Lng: paris.Lng})
	tooYoung := env.onboard("dan", profileSpec{Name: "Dan", Gender: "man", InterestedIn: []string{"woman"}, Lat: paris.Lat, Lng: paris.Lng, Birthdate: time.Now().AddDate(-20, 0, 0).Format("2006-01-02")})
	lyon := env.onboard("eric", profileSpec{Name: "Eric", Gender: "man", InterestedIn: []string{"woman"}, Lat: ptr(45.7640), Lng: ptr(4.8357)})
	picky := env.onboard("fred", profileSpec{Name: "Fred", Gender: "man", InterestedIn: []string{"woman"}, AgeMin: 18, AgeMax: 25, Lat: paris.Lat, Lng: paris.Lng})
	paused := env.onboard("gus", profileSpec{Name: "Gus", Gender: "man", InterestedIn: []string{"woman"}, Lat: paris.Lat, Lng: paris.Lng})
	env.expect(env.do("PUT", "/me/visibility", paused.Token, map[string]bool{"isDiscoverable": false}), 204)
	noPhoto := env.register("hugo")
	env.expect(env.do("PUT", "/me/profile", noPhoto.Token, map[string]any{"firstName": "Hugo", "birthdate": "1990-01-01", "gender": "man"}), 200)
	env.expect(env.do("PUT", "/me/preferences", noPhoto.Token, map[string]any{"interestedIn": []string{"woman"}, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 100}), 200)

	d := env.deck(alice)
	if !d.has(bob.ID) {
		t.Fatal("bob matches every criterion and should be in alice's deck")
	}
	for name, u := range map[string]testUser{"carl (orientation)": gayMan, "dan (age)": tooYoung, "eric (distance)": lyon, "fred (his age range)": picky, "gus (paused)": paused, "hugo (no photo)": noPhoto, "alice (self)": alice} {
		if d.has(u.ID) {
			t.Errorf("%s must not be in alice's deck", name)
		}
	}
	for _, c := range d.Profiles {
		if c.UserID != bob.ID {
			continue
		}
		if c.DistanceKm == nil || *c.DistanceKm < 5 || *c.DistanceKm > 10 {
			t.Fatalf("expected approximate distance around 7 km, got %v", c.DistanceKm)
		}
		shared := 0
		for _, i := range c.Interests {
			if i.Shared {
				shared++
			}
		}
		if shared != 1 {
			t.Fatalf("expected one shared interest flagged, got %d", shared)
		}
		if contains(mustJSON(t, c), "latitude") || contains(mustJSON(t, c), "email") || contains(mustJSON(t, c), "birthdate") {
			t.Fatal("cards must not expose private fields")
		}
	}
	raw := env.do("GET", "/discover", alice.Token, nil).Body
	for _, secret := range []string{"latitude", "longitude", "email", "birthdate", "48.8"} {
		if contains(raw, secret) {
			t.Fatalf("discover response leaks %q", secret)
		}
	}

	// Profile view follows the same visibility rules.
	env.expect(env.do("GET", "/profiles/"+bob.ID, alice.Token, nil), 200)
	env.expect(env.do("GET", "/profiles/"+paused.ID, alice.Token, nil), 404)
	env.expect(env.do("GET", "/profiles/not-a-uuid", alice.Token, nil), 400)
}

func mustJSON(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestLikeMatchAndChat(t *testing.T) {
	env := newTestEnv(t)
	alice := env.onboard("m-alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := env.onboard("m-bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	mallory := env.onboard("m-mallory", profileSpec{Name: "Mallory", Gender: "man", InterestedIn: []string{"woman"}})

	env.expect(env.do("POST", "/swipes", alice.Token, map[string]string{"targetUserId": alice.ID, "action": "like"}), 400)
	env.expect(env.do("POST", "/swipes", alice.Token, map[string]string{"targetUserId": bob.ID, "action": "superlike"}), 400)

	if s := env.swipe(alice, bob.ID, "like"); s.Matched {
		t.Fatal("one-sided like must not match")
	}
	// Repeated like is idempotent, switching to pass is refused.
	if s := env.swipe(alice, bob.ID, "like"); s.Matched || s.Action != "like" {
		t.Fatal("repeated like should be a no-op")
	}
	env.expect(env.do("POST", "/swipes", alice.Token, map[string]string{"targetUserId": bob.ID, "action": "pass"}), 409)
	if env.deck(alice).has(bob.ID) {
		t.Fatal("already-swiped profiles must not reappear")
	}
	if n := env.countRows(`SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 AND target_id = $2`, alice.ID, bob.ID); n != 1 {
		t.Fatalf("expected exactly one swipe row, got %d", n)
	}

	s := env.swipe(bob, alice.ID, "like")
	if !s.Matched || s.Match == nil || s.Match.User.UserID != alice.ID {
		t.Fatalf("mutual like must match: %+v", s)
	}
	matchID := s.Match.ID
	if again := env.swipe(bob, alice.ID, "like"); !again.Matched || again.Match.ID != matchID {
		t.Fatal("repeating the like must return the same match")
	}
	if n := env.countRows(`SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID); n != 1 {
		t.Fatalf("expected one match row, got %d", n)
	}

	// Mallory passes on alice: no match, no reappearance.
	env.swipe(mallory, alice.ID, "pass")
	if env.deck(mallory).has(alice.ID) {
		t.Fatal("passed profile reappeared")
	}

	type matchList struct {
		Matches []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			User        card   `json:"user"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"matches"`
	}
	var ml matchList
	env.expect(env.do("GET", "/matches", alice.Token, nil), 200).decode(t, &ml)
	if len(ml.Matches) != 1 || ml.Matches[0].User.UserID != bob.ID {
		t.Fatalf("alice should have bob as match: %+v", ml)
	}

	// Chat.
	path := "/matches/" + matchID + "/messages"
	env.expect(env.do("POST", path, alice.Token, map[string]string{"body": "   "}), 400)
	env.expect(env.do("POST", path, alice.Token, map[string]string{"body": strings.Repeat("a", 2001)}), 400)
	clientID := "6f1c2c5e-6a0f-4b8a-9a43-0d4a3a6c1b2e"
	var m1, m1again struct {
		ID string `json:"id"`
	}
	env.expect(env.do("POST", path, alice.Token, map[string]string{"body": "Salut Bob 👋", "clientMessageId": clientID}), 201).decode(t, &m1)
	env.expect(env.do("POST", path, alice.Token, map[string]string{"body": "Salut Bob 👋", "clientMessageId": clientID}), 201).decode(t, &m1again)
	if m1.ID != m1again.ID {
		t.Fatal("retried send with same clientMessageId must not duplicate")
	}
	for i := 0; i < 4; i++ {
		env.expect(env.do("POST", path, alice.Token, map[string]string{"body": "message " + string(rune('A'+i))}), 201)
	}

	env.expect(env.do("GET", "/matches", bob.Token, nil), 200).decode(t, &ml)
	if ml.Matches[0].UnreadCount != 5 || ml.Matches[0].LastMessage == nil || ml.Matches[0].LastMessage.Body != "message D" {
		t.Fatalf("unexpected bob match summary: %+v", ml.Matches[0])
	}

	// Pagination newest-first with a cursor.
	var page struct {
		Messages []struct {
			ID     string  `json:"id"`
			Body   string  `json:"body"`
			ReadAt *string `json:"readAt"`
		} `json:"messages"`
		NextBefore *string `json:"nextBefore"`
	}
	env.expect(env.do("GET", path+"?limit=3", bob.Token, nil), 200).decode(t, &page)
	if len(page.Messages) != 3 || page.Messages[0].Body != "message D" || page.NextBefore == nil {
		t.Fatalf("unexpected first page %+v", page)
	}
	env.expect(env.do("GET", path+"?limit=3&before="+*page.NextBefore, bob.Token, nil), 200).decode(t, &page)
	if len(page.Messages) != 2 || page.Messages[1].Body != "Salut Bob 👋" || page.NextBefore != nil {
		t.Fatalf("unexpected second page %+v", page)
	}

	// Outsiders can neither read nor write, nor mark read.
	env.expect(env.do("GET", path, mallory.Token, nil), 404)
	env.expect(env.do("POST", path, mallory.Token, map[string]string{"body": "intrusion"}), 404)
	env.expect(env.do("POST", "/matches/"+matchID+"/read", mallory.Token, nil), 404)
	env.expect(env.do("GET", "/matches/"+matchID, mallory.Token, nil), 404)

	// Read receipts.
	env.expect(env.do("POST", "/matches/"+matchID+"/read", bob.Token, nil), 204)
	env.expect(env.do("GET", path+"?limit=1", alice.Token, nil), 200).decode(t, &page)
	if page.Messages[0].ReadAt == nil {
		t.Fatal("message should be marked read for the sender")
	}
	env.expect(env.do("GET", "/matches", bob.Token, nil), 200).decode(t, &ml)
	if ml.Matches[0].UnreadCount != 0 {
		t.Fatal("unread count must reset after read")
	}

	// Notifications: a match notification for both, one collapsed message notification for bob (now read).
	var notif struct {
		Notifications []struct {
			ID     string `json:"id"`
			Type   string `json:"type"`
			Body   string `json:"body"`
			IsRead bool   `json:"isRead"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	env.expect(env.do("GET", "/notifications", bob.Token, nil), 200).decode(t, &notif)
	types := map[string]int{}
	for _, n := range notif.Notifications {
		types[n.Type]++
		if strings.Contains(n.Body, "Salut") {
			t.Fatal("notifications must not leak message content")
		}
	}
	if types["match"] != 1 || types["message"] != 1 || notif.UnreadCount != 1 {
		t.Fatalf("unexpected bob notifications: %+v", notif)
	}
	env.expect(env.do("POST", "/notifications/read-all", bob.Token, nil), 204)
	env.expect(env.do("GET", "/notifications", bob.Token, nil), 200).decode(t, &notif)
	if notif.UnreadCount != 0 {
		t.Fatal("read-all must clear unread count")
	}
	env.expect(env.do("POST", "/notifications/"+notif.Notifications[0].ID+"/read", alice.Token, nil), 404)

	// Unmatch closes the conversation for both.
	env.expect(env.do("DELETE", "/matches/"+matchID, bob.Token, nil), 204)
	env.expect(env.do("GET", path, alice.Token, nil), 404)
	env.expect(env.do("GET", "/matches", alice.Token, nil), 200).decode(t, &ml)
	if len(ml.Matches) != 0 {
		t.Fatal("unmatched conversation must disappear")
	}
	if s := env.swipe(alice, bob.ID, "like"); s.Matched {
		t.Fatal("an ended match must not be revived by a repeated like")
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	env := newTestEnv(t)
	for round := 0; round < 5; round++ {
		a := env.onboard("race-a", profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}})
		b := env.onboard("race-b", profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		wg.Add(2)
		go func() { defer wg.Done(); results[0] = env.swipe(a, b.ID, "like") }()
		go func() { defer wg.Done(); results[1] = env.swipe(b, a.ID, "like") }()
		wg.Wait()
		if n := env.countRows(`SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID); n != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, n)
		}
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: at least one swiper must see the match", round)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	env := newTestEnv(t)
	alice := env.onboard("b-alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := env.onboard("b-bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	carl := env.onboard("b-carl", profileSpec{Name: "Carl", Gender: "man", InterestedIn: []string{"woman"}})
	matchID := env.matchUsers(alice, bob)
	env.expect(env.do("POST", "/matches/"+matchID+"/messages", bob.Token, map[string]string{"body": "hello"}), 201)

	env.expect(env.do("POST", "/blocks", alice.Token, map[string]string{"userId": alice.ID}), 400)
	env.expect(env.do("POST", "/blocks", alice.Token, map[string]string{"userId": "6f1c2c5e-6a0f-4b8a-9a43-0d4a3a6c1b2e"}), 404)
	env.expect(env.do("POST", "/blocks", alice.Token, map[string]string{"userId": bob.ID}), 204)
	env.expect(env.do("POST", "/blocks", alice.Token, map[string]string{"userId": bob.ID}), 204) // idempotent

	env.expect(env.do("GET", "/matches/"+matchID+"/messages", bob.Token, nil), 404)
	env.expect(env.do("POST", "/matches/"+matchID+"/messages", bob.Token, map[string]string{"body": "still there?"}), 404)
	env.expect(env.do("GET", "/profiles/"+alice.ID, bob.Token, nil), 404)
	env.expect(env.do("GET", "/profiles/"+bob.ID, alice.Token, nil), 404)
	env.expect(env.do("POST", "/swipes", bob.Token, map[string]string{"targetUserId": alice.ID, "action": "like"}), 404)

	var blocks struct {
		Blocks []struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	env.expect(env.do("GET", "/blocks", alice.Token, nil), 200).decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].FirstName != "Bob" {
		t.Fatalf("unexpected blocks %+v", blocks)
	}
	env.expect(env.do("DELETE", "/blocks/"+bob.ID, alice.Token, nil), 204)
	env.expect(env.do("DELETE", "/blocks/"+bob.ID, alice.Token, nil), 404)
	// Unblocking does not resurrect the ended match.
	env.expect(env.do("GET", "/matches/"+matchID+"/messages", alice.Token, nil), 404)

	// Report: stored for moderation, and reporter is protected by an automatic block.
	env.expect(env.do("POST", "/reports", alice.Token, map[string]string{"userId": carl.ID, "reason": "nonsense"}), 400)
	env.expect(env.do("POST", "/reports", alice.Token, map[string]string{"userId": carl.ID, "reason": "harassment", "details": "messages insistants"}), 204)
	env.expect(env.do("POST", "/reports", alice.Token, map[string]string{"userId": carl.ID, "reason": "spam"}), 204)
	if n := env.countRows(`SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND reason = 'spam'`, alice.ID, carl.ID); n != 1 {
		t.Fatalf("expected a single upserted report, got %d", n)
	}
	if env.deck(alice).has(carl.ID) || env.deck(carl).has(alice.ID) {
		t.Fatal("reported user must be hidden both ways")
	}
}

func TestDeleteAccountPurgesData(t *testing.T) {
	env := newTestEnv(t)
	alice := env.onboard("d-alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := env.onboard("d-bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	matchID := env.matchUsers(alice, bob)
	env.expect(env.do("POST", "/matches/"+matchID+"/messages", alice.Token, map[string]string{"body": "bye"}), 201)
	env.expect(env.do("POST", "/reports", bob.Token, map[string]string{"userId": alice.ID, "reason": "spam"}), 204)

	var keys []string
	rows, _ := env.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, alice.ID)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) == 0 {
		t.Fatal("expected photos")
	}

	env.expect(env.do("DELETE", "/me", alice.Token, map[string]string{"password": "WrongPass1"}), 401)
	env.expect(env.do("DELETE", "/me", alice.Token, map[string]string{"password": testPassword}), 204)

	env.expect(env.do("GET", "/me", alice.Token, nil), 401)
	env.expect(env.do("POST", "/auth/login", "", map[string]string{"email": alice.Email, "password": testPassword}), 401)
	for _, k := range keys {
		if _, err := os.Stat(filepath.Join(env.dir, k)); !os.IsNotExist(err) {
			t.Fatalf("photo file %s must be deleted", k)
		}
	}
	for table, q := range map[string]string{
		"profiles":   `SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
		"photos":     `SELECT COUNT(*) FROM photos WHERE user_id = $1`,
		"swipes":     `SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1`,
		"matches":    `SELECT COUNT(*) FROM matches WHERE user_a_id = $1 OR user_b_id = $1`,
		"messages":   `SELECT COUNT(*) FROM messages WHERE sender_id = $1`,
		"sessions":   `SELECT COUNT(*) FROM sessions WHERE user_id = $1`,
		"interests":  `SELECT COUNT(*) FROM user_interests WHERE user_id = $1`,
		"preference": `SELECT COUNT(*) FROM preferences WHERE user_id = $1`,
	} {
		if n := env.countRows(q, alice.ID); n != 0 {
			t.Errorf("%s still has %d rows for deleted user", table, n)
		}
	}
	// Moderation history survives with the reference nulled.
	if n := env.countRows(`SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL`, bob.ID); n != 1 {
		t.Fatalf("expected report to be kept anonymized, got %d", n)
	}
	var ml struct {
		Matches []any `json:"matches"`
	}
	env.expect(env.do("GET", "/matches", bob.Token, nil), 200).decode(t, &ml)
	if len(ml.Matches) != 0 {
		t.Fatal("bob's match with a deleted user must disappear")
	}
}

func TestRealtimeDelivery(t *testing.T) {
	env := newTestEnv(t)
	srv := httptest.NewServer(env.router)
	defer srv.Close()

	alice := env.onboard("rt-alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := env.onboard("rt-bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/realtime?ticket="
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL+"forged", nil); err == nil || resp.StatusCode != 401 {
		t.Fatal("connection without a valid ticket must be refused")
	}

	var ticket struct {
		Ticket string `json:"ticket"`
	}
	env.expect(env.do("POST", "/realtime/ticket", alice.Token, nil), 200).decode(t, &ticket)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+ticket.Ticket, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL+ticket.Ticket, nil); err == nil || resp.StatusCode != 401 {
		t.Fatal("tickets must be single use")
	}

	deadline := time.Now().Add(2 * time.Second)
	for env.hub.ConnectionCount(alice.ID) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	env.swipe(alice, bob.ID, "like")
	match := env.swipe(bob, alice.ID, "like")
	env.expect(env.do("POST", "/matches/"+match.Match.ID+"/messages", bob.Token, map[string]string{"body": "temps réel"}), 201)

	want := map[string]bool{"match.created": false, "message.created": false}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for !(want["match.created"] && want["message.created"]) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for events %v: %v", want, err)
		}
		var ev struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(data, &ev)
		if _, ok := want[ev.Type]; ok {
			want[ev.Type] = true
		}
		if ev.Type == "message.created" && !contains(ev.Data, "temps réel") {
			t.Fatalf("unexpected message payload %s", ev.Data)
		}
	}
}

func TestRateLimitOnLogin(t *testing.T) {
	env := newTestEnv(t)
	cfg := testConfig()
	router := buildRouter(cfg, Deps{Pool: env.pool, Storage: env.storage, Mailer: env.mailer})
	limited := 0
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"x@it.test","password":"Password1"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == 429 {
			limited++
			if w.Header().Get("Retry-After") == "" {
				t.Fatal("429 must include Retry-After")
			}
		}
	}
	if limited == 0 {
		t.Fatal("expected brute-force logins to be rate limited")
	}
}
