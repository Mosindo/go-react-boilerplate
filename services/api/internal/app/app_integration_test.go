package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealth(t *testing.T) {
	e := newEnv(t, generousLimits())
	r := e.do("GET", "/health", "", nil)
	e.expect(r, http.StatusOK, "health")
	if r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers missing")
	}
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t, generousLimits())

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("weak"), "password": "password"}), http.StatusBadRequest, "weak password")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}), http.StatusBadRequest, "invalid email")

	u := e.register("auth")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": "Password123"}), http.StatusConflict, "duplicate normalized email")

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "WrongPass1"}), http.StatusUnauthorized, "wrong password")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": uniqueEmail("ghost"), "password": "Password123"}), http.StatusUnauthorized, "unknown email")
	login := e.do("POST", "/auth/login", "", map[string]string{"email": strings.ToUpper(u.Email), "password": "Password123"})
	e.expect(login, http.StatusOK, "login")

	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized, "me without token")
	e.expect(e.do("GET", "/me", "garbage", nil), http.StatusUnauthorized, "me with garbage token")
	me := e.do("GET", "/me", u.Token, nil)
	e.expect(me, http.StatusOK, "me")
	if !strings.Contains(string(me.Body), u.Email) {
		t.Fatalf("unexpected me payload %s", me.Body)
	}

	refreshed := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh})
	e.expect(refreshed, http.StatusOK, "refresh")
	var rp authPayload
	refreshed.json(t, &rp)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), http.StatusUnauthorized, "refresh token reuse")

	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": rp.RefreshToken}), http.StatusNoContent, "logout")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": rp.RefreshToken}), http.StatusUnauthorized, "refresh after logout")

	// Password reset by emailed code.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": uniqueEmail("nobody")}), http.StatusAccepted, "forgot unknown email (no enumeration)")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), http.StatusAccepted, "forgot")
	code := e.mailer.code(t, u.Email)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "NewPassword456"}), http.StatusBadRequest, "wrong code")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), http.StatusNoContent, "reset")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "OtherPassword789"}), http.StatusBadRequest, "code reuse")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"}), http.StatusUnauthorized, "old password after reset")
	newLogin := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456"})
	e.expect(newLogin, http.StatusOK, "login with new password")
	var nl authPayload
	newLogin.json(t, &nl)

	e.expect(e.do("POST", "/me/password", nl.AccessToken, map[string]string{"currentPassword": "bad", "newPassword": "Another123"}), http.StatusForbidden, "change password wrong current")
	e.expect(e.do("POST", "/me/password", nl.AccessToken, map[string]string{"currentPassword": "NewPassword456", "newPassword": "Another123"}), http.StatusNoContent, "change password")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Another123"}), http.StatusOK, "login after change")
}

func TestResetCodeBruteForceIsCapped(t *testing.T) {
	e := newEnv(t, generousLimits())
	u := e.register("brute")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), http.StatusAccepted, "forgot")
	code := e.mailer.code(t, u.Email)
	for i := 0; i < 5; i++ {
		wrong := "00000" + string(rune('0'+i))
		if wrong == code {
			continue
		}
		e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "NewPassword456"}), http.StatusBadRequest, "wrong attempt")
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}), http.StatusBadRequest, "correct code after too many attempts")
}

func TestCredentialRateLimit(t *testing.T) {
	limits := generousLimits()
	limits.Credentials = [2]int{1, 3}
	e := newEnv(t, limits)
	var last resp
	for i := 0; i < 4; i++ {
		last = e.do("POST", "/auth/login", "", map[string]string{"email": uniqueEmail("rl"), "password": "Password123"})
	}
	e.expect(last, http.StatusTooManyRequests, "4th login attempt")
	if last.Header.Get("Retry-After") == "" {
		t.Fatal("Retry-After header expected")
	}
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t, generousLimits())
	u := e.register("profile")

	r := e.do("GET", "/profile", u.Token, nil)
	e.expect(r, http.StatusOK, "empty profile")
	var own struct {
		Profile    *map[string]any `json:"profile"`
		Completion struct {
			Complete bool     `json:"complete"`
			Missing  []string `json:"missing"`
		} `json:"completion"`
	}
	r.json(t, &own)
	if own.Profile != nil || own.Completion.Complete || len(own.Completion.Missing) != 4 {
		t.Fatalf("unexpected empty profile %s", r.Body)
	}

	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Minor", "birthdate": birthdateYearsAgo(17), "gender": "woman"}), http.StatusBadRequest, "underage")
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Zoé", "birthdate": birthdateYearsAgo(30), "gender": "dragon"}), http.StatusBadRequest, "bad gender")
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Zoé", "birthdate": birthdateYearsAgo(30), "gender": "woman", "interestIds": []int{9999}}), http.StatusBadRequest, "unknown interest")
	e.expect(e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": 10, "longitude": 10}), http.StatusConflict, "location before profile")

	bd := birthdateYearsAgo(30)
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Zoé", "birthdate": bd, "gender": "woman", "interestIds": []int{1, 2}}), http.StatusOK, "create profile")
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Zoé", "birthdate": birthdateYearsAgo(25), "gender": "woman"}), http.StatusBadRequest, "birthdate locked")
	e.expect(e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": 91, "longitude": 10}), http.StatusBadRequest, "invalid latitude")
	e.expect(e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": 48.856613, "longitude": 2.352222}), http.StatusNoContent, "location")

	var lat, lon float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lon); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lon != 2.35 {
		t.Fatalf("location must be stored coarsened, got %v,%v", lat, lon)
	}
	r = e.do("GET", "/profile", u.Token, nil)
	if strings.Contains(string(r.Body), "48.8") || strings.Contains(string(r.Body), "latitude") {
		t.Fatalf("raw coordinates must never be returned: %s", r.Body)
	}

	e.expect(e.do("PUT", "/profile/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10}), http.StatusBadRequest, "inverted ages")
	e.expect(e.do("PUT", "/profile/privacy", u.Token, map[string]any{"isDiscoverable": false, "showDistance": false}), http.StatusNoContent, "privacy")
	e.expect(e.do("GET", "/interests", u.Token, nil), http.StatusOK, "interests catalog")
}

type matchPayload struct {
	ID      string `json:"id"`
	Profile struct {
		UserID    string `json:"userId"`
		FirstName string `json:"firstName"`
	} `json:"profile"`
	UnreadCount int `json:"unreadCount"`
	LastMessage *struct {
		Body string `json:"body"`
	} `json:"lastMessage"`
}

// TestCriticalFlow: register → profile → discovery → like → match → chat.
func TestCriticalFlow(t *testing.T) {
	e := newEnv(t, generousLimits())
	alice := e.register("alice")
	bob := e.register("bob")
	carl := e.register("carl")   // man looking for men: not compatible with alice
	dana := e.register("dana")   // incomplete profile
	eve := e.register("eve")     // compatible but blocks later
	young := e.register("young") // outside bob's age range

	e.onboard(alice, "Alice", "woman", []string{"man"}, birthdateYearsAgo(29))
	e.onboard(bob, "Bob", "man", []string{"woman"}, birthdateYearsAgo(31))
	e.onboard(carl, "Carl", "man", []string{"man"}, birthdateYearsAgo(33))
	e.onboard(eve, "Eve", "woman", []string{"man"}, birthdateYearsAgo(27))
	e.onboard(young, "Jade", "woman", []string{"man"}, birthdateYearsAgo(19))
	e.expect(e.do("PUT", "/profile/preferences", bob.Token, map[string]any{
		"interestedIn": []string{"woman"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 10,
	}), http.StatusNoContent, "bob age range")

	e.expect(e.do("GET", "/discovery", dana.Token, nil), http.StatusConflict, "deck with incomplete profile")

	aliceDeck := e.deck(alice)
	if !containsCard(aliceDeck, bob.ID) || containsCard(aliceDeck, carl.ID) || containsCard(aliceDeck, alice.ID) || containsCard(aliceDeck, eve.ID) {
		t.Fatalf("alice deck must contain bob only among men looking for women: %+v", aliceDeck)
	}
	bobDeck := e.deck(bob)
	if !containsCard(bobDeck, alice.ID) || !containsCard(bobDeck, eve.ID) || containsCard(bobDeck, young.ID) {
		t.Fatalf("bob deck must respect age range: %+v", bobDeck)
	}
	for _, c := range bobDeck {
		if c.UserID == alice.ID {
			if c.Age != 29 || c.DistanceKm == nil || *c.DistanceKm > 3 || len(c.Photos) != 1 || len(c.Interests) != 3 {
				t.Fatalf("unexpected card %+v", c)
			}
			if !strings.HasPrefix(c.Photos[0].URL, "/media/") || !strings.Contains(c.Photos[0].URL, "sig=") {
				t.Fatalf("photo URL must be signed: %s", c.Photos[0].URL)
			}
		}
	}

	// Server enforces eligibility: carl cannot like alice (preferences), self-like is rejected.
	e.expect(e.swipe(carl, alice.ID, "like"), http.StatusNotFound, "incompatible like")
	e.expect(e.swipe(alice, alice.ID, "like"), http.StatusBadRequest, "self like")
	e.expect(e.swipe(alice, bob.ID, "superlike"), http.StatusBadRequest, "invalid action")

	// Realtime: alice listens for events.
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	ticketResp := e.do("POST", "/realtime/ticket", alice.Token, nil)
	e.expect(ticketResp, http.StatusOK, "ticket")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	ticketResp.json(t, &ticket)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/realtime?ticket=" + ticket.Ticket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	if _, _, err := websocket.DefaultDialer.Dial(wsURL, nil); err == nil {
		t.Fatal("ticket must be single use")
	}
	events := make(chan string, 20)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				close(events)
				return
			}
			var ev struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(data, &ev)
			events <- ev.Type
		}
	}()
	waitEvent := func(want string) {
		t.Helper()
		deadline := time.After(3 * time.Second)
		for {
			select {
			case got, ok := <-events:
				if !ok {
					t.Fatalf("socket closed while waiting for %s", want)
				}
				if got == want {
					return
				}
			case <-deadline:
				t.Fatalf("timeout waiting for realtime event %s", want)
			}
		}
	}
	for e.hub.ConnectionCount(alice.ID) == 0 {
		time.Sleep(10 * time.Millisecond)
	}

	// Alice likes bob: no match yet. Repeated like is rejected.
	r := e.swipe(alice, bob.ID, "like")
	e.expect(r, http.StatusOK, "alice likes bob")
	if strings.Contains(string(r.Body), `"matched":true`) {
		t.Fatal("one-sided like must not match")
	}
	e.expect(e.swipe(alice, bob.ID, "like"), http.StatusConflict, "duplicate like")
	e.expect(e.swipe(alice, bob.ID, "pass"), http.StatusConflict, "changing answer")
	if containsCard(e.deck(alice), bob.ID) {
		t.Fatal("swiped profile must not reappear")
	}

	// Bob passes on eve, likes alice → match.
	e.expect(e.swipe(bob, eve.ID, "pass"), http.StatusOK, "bob passes eve")
	r = e.swipe(bob, alice.ID, "like")
	e.expect(r, http.StatusOK, "bob likes alice")
	var sw struct {
		Matched bool   `json:"matched"`
		MatchID string `json:"matchId"`
	}
	r.json(t, &sw)
	if !sw.Matched || sw.MatchID == "" {
		t.Fatalf("expected a match, got %s", r.Body)
	}
	waitEvent("match.new")
	waitEvent("notification.new")
	var count int
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match row, got %d", count)
	}

	// Notifications for both.
	for _, u := range []user{alice, bob} {
		nr := e.do("GET", "/notifications", u.Token, nil)
		e.expect(nr, http.StatusOK, "notifications")
		if !strings.Contains(string(nr.Body), `"type":"match"`) {
			t.Fatalf("expected match notification: %s", nr.Body)
		}
	}

	// Conversation list.
	lr := e.do("GET", "/matches", alice.Token, nil)
	e.expect(lr, http.StatusOK, "alice matches")
	var list struct {
		Matches []matchPayload `json:"matches"`
	}
	lr.json(t, &list)
	if len(list.Matches) != 1 || list.Matches[0].Profile.UserID != bob.ID {
		t.Fatalf("unexpected matches %s", lr.Body)
	}
	base := "/matches/" + sw.MatchID

	// Permissions: carl can neither read nor write nor unmatch.
	e.expect(e.do("GET", base+"/messages", carl.Token, nil), http.StatusNotFound, "outsider reads messages")
	e.expect(e.do("POST", base+"/messages", carl.Token, map[string]string{"body": "hi"}), http.StatusNotFound, "outsider sends message")
	e.expect(e.do("GET", base, carl.Token, nil), http.StatusNotFound, "outsider gets match")
	e.expect(e.do("DELETE", base, carl.Token, nil), http.StatusNotFound, "outsider unmatches")
	e.expect(e.do("POST", base+"/read", carl.Token, nil), http.StatusNotFound, "outsider marks read")
	e.expect(e.do("GET", "/matches/not-a-uuid/messages", alice.Token, nil), http.StatusBadRequest, "invalid match id")

	// Chat.
	e.expect(e.do("POST", base+"/messages", bob.Token, map[string]string{"body": "   "}), http.StatusBadRequest, "empty message")
	e.expect(e.do("POST", base+"/messages", bob.Token, map[string]string{"body": strings.Repeat("a", 2001)}), http.StatusBadRequest, "long message")
	sent := e.do("POST", base+"/messages", bob.Token, map[string]string{"body": "  Salut Alice !  "})
	e.expect(sent, http.StatusCreated, "bob sends")
	waitEvent("message.new")
	e.expect(e.do("POST", base+"/messages", bob.Token, map[string]string{"body": "Tu aimes la randonnée ?"}), http.StatusCreated, "bob sends again")

	lr = e.do("GET", "/matches", alice.Token, nil)
	lr.json(t, &list)
	if list.Matches[0].UnreadCount != 2 || list.Matches[0].LastMessage == nil || list.Matches[0].LastMessage.Body != "Tu aimes la randonnée ?" {
		t.Fatalf("unexpected conversation summary %s", lr.Body)
	}
	unread := e.do("GET", "/notifications/unread-count", alice.Token, nil)
	if !strings.Contains(string(unread.Body), `"unreadCount":2`) { // 1 match + 1 collapsed message notification
		t.Fatalf("expected collapsed message notification: %s", unread.Body)
	}

	mr := e.do("GET", base+"/messages?limit=1", alice.Token, nil)
	e.expect(mr, http.StatusOK, "alice reads page 1")
	var page struct {
		Messages []struct {
			Body   string     `json:"body"`
			ReadAt *time.Time `json:"readAt"`
		} `json:"messages"`
		NextCursor *string `json:"nextCursor"`
	}
	mr.json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Tu aimes la randonnée ?" || page.NextCursor == nil {
		t.Fatalf("unexpected page 1 %s", mr.Body)
	}
	mr = e.do("GET", base+"/messages?limit=1&cursor="+*page.NextCursor, alice.Token, nil)
	mr.json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Salut Alice !" {
		t.Fatalf("unexpected page 2 %s", mr.Body)
	}

	e.expect(e.do("POST", base+"/read", alice.Token, nil), http.StatusNoContent, "alice marks read")
	mr = e.do("GET", base+"/messages", bob.Token, nil)
	mr.json(t, &page)
	for _, m := range page.Messages {
		if m.ReadAt == nil {
			t.Fatalf("messages must be marked read: %s", mr.Body)
		}
	}
	unread = e.do("GET", "/notifications/unread-count", alice.Token, nil)
	if !strings.Contains(string(unread.Body), `"unreadCount":0`) {
		t.Fatalf("conversation notifications must be cleared when reading it: %s", unread.Body)
	}
	e.expect(e.do("POST", "/notifications/read-all", alice.Token, nil), http.StatusNoContent, "read all")

	// Alice can view bob's full profile (they are matched).
	e.expect(e.do("GET", "/profiles/"+bob.ID, alice.Token, nil), http.StatusOK, "view match profile")

	// Unmatch removes the conversation for both, and nobody reappears in decks.
	e.expect(e.do("DELETE", base, alice.Token, nil), http.StatusNoContent, "unmatch")
	waitEvent("match.removed")
	e.expect(e.do("GET", base+"/messages", bob.Token, nil), http.StatusNotFound, "messages after unmatch")
	if containsCard(e.deck(bob), alice.ID) || containsCard(e.deck(alice), bob.ID) {
		t.Fatal("unmatched profiles must not reappear")
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t, generousLimits())
	for i := 0; i < 5; i++ {
		a := e.register("racea")
		b := e.register("raceb")
		e.onboard(a, "Ana", "woman", []string{"man"}, birthdateYearsAgo(30))
		e.onboard(b, "Ben", "man", []string{"woman"}, birthdateYearsAgo(30))

		var wg sync.WaitGroup
		results := make([]resp, 2)
		wg.Add(2)
		go func() { defer wg.Done(); results[0] = e.swipe(a, b.ID, "like") }()
		go func() { defer wg.Done(); results[1] = e.swipe(b, a.ID, "like") }()
		wg.Wait()

		matched := 0
		for _, r := range results {
			e.expect(r, http.StatusOK, "concurrent like")
			if strings.Contains(string(r.Body), `"matched":true`) {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("expected exactly one swipe to report the match, got %d", matched)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t, generousLimits())
	a := e.register("blocka")
	b := e.register("blockb")
	c := e.register("blockc")
	e.onboard(a, "Inès", "woman", []string{"man", "woman"}, birthdateYearsAgo(28))
	e.onboard(b, "Hugo", "man", []string{"woman"}, birthdateYearsAgo(28))
	e.onboard(c, "Léo", "man", []string{"woman"}, birthdateYearsAgo(28))

	e.expect(e.swipe(a, b.ID, "like"), http.StatusOK, "a likes b")
	r := e.swipe(b, a.ID, "like")
	var sw struct {
		MatchID string `json:"matchId"`
	}
	r.json(t, &sw)
	e.expect(e.do("POST", "/matches/"+sw.MatchID+"/messages", b.Token, map[string]string{"body": "hello"}), http.StatusCreated, "message before block")

	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": a.ID}), http.StatusBadRequest, "self block")
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), http.StatusNoContent, "block")
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), http.StatusNoContent, "block is idempotent")
	e.expect(e.do("GET", "/matches/"+sw.MatchID+"/messages", b.Token, nil), http.StatusNotFound, "blocked user loses conversation")
	e.expect(e.do("POST", "/matches/"+sw.MatchID+"/messages", b.Token, map[string]string{"body": "still there?"}), http.StatusNotFound, "blocked user cannot message")
	e.expect(e.do("GET", "/profiles/"+a.ID, b.Token, nil), http.StatusNotFound, "blocked user cannot view profile")
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), http.StatusNotFound, "blocker cannot view blocked profile")

	lr := e.do("GET", "/blocks", a.Token, nil)
	if !strings.Contains(string(lr.Body), b.ID) {
		t.Fatalf("blocked list must contain b: %s", lr.Body)
	}
	e.expect(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), http.StatusNoContent, "unblock")

	// Report: blocks automatically, hides the reported user from the reporter.
	if !containsCard(e.deck(a), c.ID) {
		t.Fatal("c should be in a's deck before the report")
	}
	e.expect(e.do("POST", "/reports", a.Token, map[string]string{"userId": c.ID, "reason": "nope"}), http.StatusBadRequest, "invalid reason")
	e.expect(e.do("POST", "/reports", a.Token, map[string]string{"userId": c.ID, "reason": "harassment", "details": "insults"}), http.StatusCreated, "report")
	if containsCard(e.deck(a), c.ID) {
		t.Fatal("reported user must disappear from reporter's deck")
	}

	// Three distinct reporters put the profile under review (hidden from everyone).
	for i := 0; i < 2; i++ {
		rep := e.register("reporter")
		e.onboard(rep, "Rita", "woman", []string{"man"}, birthdateYearsAgo(30))
		e.expect(e.do("POST", "/reports", rep.Token, map[string]string{"userId": c.ID, "reason": "fake_profile"}), http.StatusCreated, "extra report")
	}
	var status string
	e.pool.QueryRow(context.Background(), `SELECT moderation_status FROM profiles WHERE user_id = $1`, c.ID).Scan(&status)
	if status != "under_review" {
		t.Fatalf("expected profile under review, got %s", status)
	}
	other := e.register("viewer")
	e.onboard(other, "Lou", "woman", []string{"man"}, birthdateYearsAgo(30))
	if containsCard(e.deck(other), c.ID) {
		t.Fatal("profile under review must be hidden from discovery")
	}
}

func TestPhotosSecurity(t *testing.T) {
	e := newEnv(t, generousLimits())
	a := e.register("photoa")
	b := e.register("photob")

	e.expect(e.upload("POST", "/profile/photos", a.Token, []byte("<?php echo 1; ?>")), http.StatusUnprocessableEntity, "php upload")
	e.expect(e.upload("POST", "/profile/photos", a.Token, []byte("<svg onload=alert(1)>")), http.StatusUnprocessableEntity, "svg upload")

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.upload("POST", "/profile/photos", a.Token, testPNG(t, uint8(i*30)))
		e.expect(r, http.StatusCreated, "upload")
		var out struct {
			Photos []struct {
				ID  string `json:"id"`
				URL string `json:"url"`
			} `json:"photos"`
		}
		r.json(t, &out)
		ids = append(ids, out.Photos[len(out.Photos)-1].ID)
	}
	e.expect(e.upload("POST", "/profile/photos", a.Token, testPNG(t, 7)), http.StatusConflict, "7th photo")

	// Reorder: last becomes main.
	order := append([]string{ids[5]}, ids[:5]...)
	r := e.do("PUT", "/profile/photos/order", a.Token, map[string]any{"photoIds": order})
	e.expect(r, http.StatusOK, "reorder")
	var listed struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
			URL      string `json:"url"`
		} `json:"photos"`
	}
	r.json(t, &listed)
	if listed.Photos[0].ID != ids[5] {
		t.Fatalf("expected reordered main photo: %s", r.Body)
	}
	e.expect(e.do("PUT", "/profile/photos/order", a.Token, map[string]any{"photoIds": ids[:3]}), http.StatusBadRequest, "partial reorder")
	e.expect(e.do("PUT", "/profile/photos/order", b.Token, map[string]any{"photoIds": order}), http.StatusBadRequest, "reorder someone else's photos")

	// Other users can't delete or replace my photos.
	e.expect(e.do("DELETE", "/profile/photos/"+ids[0], b.Token, nil), http.StatusNotFound, "delete foreign photo")
	e.expect(e.upload("PUT", "/profile/photos/"+ids[0], b.Token, testPNG(t, 9)), http.StatusNotFound, "replace foreign photo")

	// Signed media URLs.
	url := listed.Photos[0].URL
	media := e.do("GET", url, "", nil)
	e.expect(media, http.StatusOK, "signed media")
	if media.Header.Get("Content-Type") != "image/jpeg" || len(media.Body) < 100 {
		t.Fatalf("unexpected media response %v", media.Header)
	}
	if media.Body[0] != 0xFF || media.Body[1] != 0xD8 {
		t.Fatal("served file must be the re-encoded JPEG")
	}
	e.expect(e.do("GET", trimQuery(url), "", nil), http.StatusForbidden, "unsigned media")
	e.expect(e.do("GET", strings.Replace(url, "sig=", "sig=0", 1), "", nil), http.StatusForbidden, "tampered signature")
	if r := e.do("GET", "/media/..%2f..%2fetc%2fpasswd?exp=9999999999&sig=x", "", nil); r.Status == http.StatusOK {
		t.Fatal("path traversal must never succeed")
	}

	// Replace keeps position, delete compacts positions and removes the file.
	e.expect(e.upload("PUT", "/profile/photos/"+ids[5], a.Token, testPNG(t, 200)), http.StatusOK, "replace")
	e.expect(e.do("GET", url, "", nil), http.StatusNotFound, "old file removed after replace")
	r = e.do("DELETE", "/profile/photos/"+ids[1], a.Token, nil)
	e.expect(r, http.StatusOK, "delete")
	r.json(t, &listed)
	if len(listed.Photos) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(listed.Photos))
	}
	for i, p := range listed.Photos {
		if p.Position != i {
			t.Fatalf("positions must be compacted: %s", r.Body)
		}
	}
}

func TestDeleteAccountRemovesData(t *testing.T) {
	e := newEnv(t, generousLimits())
	a := e.register("dela")
	b := e.register("delb")
	e.onboard(a, "Nina", "woman", []string{"man"}, birthdateYearsAgo(26))
	e.onboard(b, "Omar", "man", []string{"woman"}, birthdateYearsAgo(26))
	e.swipe(a, b.ID, "like")
	r := e.swipe(b, a.ID, "like")
	var sw struct {
		MatchID string `json:"matchId"`
	}
	r.json(t, &sw)
	e.expect(e.do("POST", "/matches/"+sw.MatchID+"/messages", a.Token, map[string]string{"body": "bye"}), http.StatusCreated, "message")

	prof := e.do("GET", "/profile", a.Token, nil)
	var own struct {
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
	}
	prof.json(t, &own)
	photoURL := own.Photos[0].URL

	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "nope"}), http.StatusForbidden, "delete with wrong password")
	e.expect(e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"}), http.StatusNoContent, "delete account")

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}), http.StatusUnauthorized, "login after deletion")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": a.Refresh}), http.StatusUnauthorized, "refresh after deletion")
	e.expect(e.do("GET", "/me", a.Token, nil), http.StatusUnauthorized, "me after deletion")
	e.expect(e.do("GET", "/matches/"+sw.MatchID+"/messages", b.Token, nil), http.StatusNotFound, "conversation removed")
	e.expect(e.do("GET", photoURL, "", nil), http.StatusNotFound, "photo file removed")

	var leftovers int
	e.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM profiles WHERE user_id = $1) + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1)
		     + (SELECT COUNT(*) FROM messages WHERE sender_id = $1)
		     + (SELECT COUNT(*) FROM sessions WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM notifications WHERE user_id = $1 OR actor_id = $1)
	`, a.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("expected all personal data to be deleted, %d rows left", leftovers)
	}
}
