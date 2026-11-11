package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealth(t *testing.T) {
	e := setupEnv(t, false)
	e.expect(e.do("GET", "/health", "", nil), http.StatusOK, "health")
}

func TestAuthValidationAndSessions(t *testing.T) {
	e := setupEnv(t, false)

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}), http.StatusBadRequest, "invalid email")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": uniqueEmail("weak"), "password": "password"}), http.StatusBadRequest, "weak password")

	a := e.register("auth")
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": strings.ToUpper(a.Email), "password": "Password123"}), http.StatusConflict, "duplicate email (case-insensitive)")

	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized, "me without token")
	e.expect(e.do("GET", "/me", "garbage", nil), http.StatusUnauthorized, "me with garbage token")
	me := e.do("GET", "/me", a.Access, nil)
	e.expect(me, http.StatusOK, "me")
	mustNotContain(t, me.Body, "password")

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Wrong12345"}), http.StatusUnauthorized, "wrong password")
	login := e.do("POST", "/auth/login", "", map[string]string{"email": "  " + strings.ToUpper(a.Email), "password": a.Password})
	e.expect(login, http.StatusOK, "login with normalized email")

	refreshed := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": a.Refresh})
	e.expect(refreshed, http.StatusOK, "refresh")
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": a.Refresh}), http.StatusUnauthorized, "refresh token reuse")

	var tokens struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	refreshed.decode(t, &tokens)
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": tokens.RefreshToken}), http.StatusNoContent, "logout")
	// Logout revokes the session: its access token stops working immediately.
	e.expect(e.do("GET", "/me", tokens.AccessToken, nil), http.StatusUnauthorized, "access token after logout")
}

func TestProfileValidationAndPhotos(t *testing.T) {
	e := setupEnv(t, false)
	a := e.register("profile")

	mine := e.do("GET", "/me/profile", a.Access, nil)
	e.expect(mine, http.StatusOK, "empty profile")
	var empty struct {
		Profile    *json.RawMessage `json:"profile"`
		IsComplete bool             `json:"isComplete"`
	}
	mine.decode(t, &empty)
	if empty.Profile != nil || empty.IsComplete {
		t.Fatalf("new account should have no profile: %s", mine.Body)
	}

	e.expect(e.upload("/me/photos", "POST", a.Access, pngImage(t, 400, 400, 10)), http.StatusConflict, "photo before profile")
	e.expect(e.do("PUT", "/me/preferences", a.Access, map[string]any{"interestedIn": []string{"man"}, "ageMin": 20, "ageMax": 30, "maxDistanceKm": 10}), http.StatusConflict, "preferences before profile")

	underage := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	r := e.do("PUT", "/me/profile", a.Access, map[string]any{"firstName": "Lou", "birthdate": underage, "gender": "woman"})
	e.expect(r, http.StatusBadRequest, "underage")
	if !strings.Contains(string(r.Body), "underage") {
		t.Fatalf("expected underage code: %s", r.Body)
	}
	e.expect(e.do("PUT", "/me/profile", a.Access, map[string]any{"firstName": "Lou", "birthdate": "1990-01-01", "gender": "alien"}), http.StatusBadRequest, "gender")
	e.expect(e.do("PUT", "/me/profile", a.Access, map[string]any{"firstName": "Lou", "birthdate": "1990-01-01", "gender": "woman", "bio": strings.Repeat("a", 501)}), http.StatusBadRequest, "bio too long")
	e.expect(e.do("PUT", "/me/profile", a.Access, map[string]any{"firstName": "Lou", "birthdate": "1990-01-01", "gender": "woman"}), http.StatusOK, "valid profile")

	e.expect(e.do("PUT", "/me/preferences", a.Access, map[string]any{"interestedIn": []string{"man"}, "ageMin": 40, "ageMax": 30, "maxDistanceKm": 10}), http.StatusBadRequest, "inverted ages")
	e.expect(e.do("PUT", "/me/interests", a.Access, map[string]any{"interests": []string{"hiking", "not-a-real-interest"}}), http.StatusBadRequest, "unknown interest")
	e.expect(e.do("PUT", "/me/location", a.Access, map[string]any{"latitude": 91, "longitude": 0}), http.StatusBadRequest, "invalid latitude")

	// Location is stored rounded (~1 km), never the exact GPS position.
	e.expect(e.do("PUT", "/me/location", a.Access, map[string]any{"latitude": 48.856613, "longitude": 2.352222}), http.StatusOK, "location")
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be rounded, got %v,%v", lat, lng)
	}

	// Dangerous uploads are rejected by content sniffing.
	e.expect(e.upload("/me/photos", "POST", a.Access, []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>")), http.StatusUnsupportedMediaType, "svg upload")
	e.expect(e.upload("/me/photos", "POST", a.Access, []byte("#!/bin/sh\nrm -rf /")), http.StatusUnsupportedMediaType, "script upload")
	e.expect(e.upload("/me/photos", "POST", a.Access, pngImage(t, 100, 100, 1)), http.StatusBadRequest, "too small")

	var photoIDs []string
	for i := 0; i < 6; i++ {
		r := e.upload("/me/photos", "POST", a.Access, pngImage(t, 500, 700, uint8(i*30)))
		e.expect(r, http.StatusCreated, "upload photo")
		var out struct {
			Photos []struct {
				ID  string `json:"id"`
				URL string `json:"url"`
			} `json:"photos"`
		}
		r.decode(t, &out)
		photoIDs = append(photoIDs, out.Photos[len(out.Photos)-1].ID)
	}
	e.expect(e.upload("/me/photos", "POST", a.Access, pngImage(t, 500, 700, 9)), http.StatusConflict, "7th photo")

	// Reorder: reverse, then the first id becomes the main photo.
	reversed := make([]string, len(photoIDs))
	for i, id := range photoIDs {
		reversed[len(photoIDs)-1-i] = id
	}
	r = e.do("PUT", "/me/photos/order", a.Access, map[string]any{"photoIds": reversed})
	e.expect(r, http.StatusOK, "reorder")
	var ordered struct {
		Photos []struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			Position int    `json:"position"`
		} `json:"photos"`
		IsComplete bool `json:"isComplete"`
	}
	r.decode(t, &ordered)
	if ordered.Photos[0].ID != photoIDs[5] || ordered.Photos[0].Position != 0 {
		t.Fatalf("reorder not applied: %s", r.Body)
	}
	e.expect(e.do("PUT", "/me/photos/order", a.Access, map[string]any{"photoIds": photoIDs[:3]}), http.StatusBadRequest, "partial order")

	// Signed media URLs: valid works, tampered does not.
	photoURL := ordered.Photos[0].URL
	media := e.do("GET", photoURL, "", nil)
	e.expect(media, http.StatusOK, "signed media")
	if media.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("media must be re-encoded jpeg, got %s", media.Header.Get("Content-Type"))
	}
	e.expect(e.do("GET", photoURL[:len(photoURL)-3]+"abc", "", nil), http.StatusForbidden, "tampered signature")
	e.expect(e.do("GET", "/media/../../etc/passwd", "", nil), http.StatusNotFound, "traversal attempt")

	// Replace keeps the position, delete compacts the rest.
	e.expect(e.upload("/me/photos/"+photoIDs[5], "PUT", a.Access, pngImage(t, 640, 640, 200)), http.StatusOK, "replace")
	e.expect(e.do("GET", photoURL, "", nil), http.StatusNotFound, "old file of replaced photo")
	r = e.do("DELETE", "/me/photos/"+photoIDs[0], a.Access, nil)
	e.expect(r, http.StatusOK, "delete photo")
	r.decode(t, &ordered)
	if len(ordered.Photos) != 5 {
		t.Fatalf("expected 5 photos after delete, got %d", len(ordered.Photos))
	}
	for i, p := range ordered.Photos {
		if p.Position != i {
			t.Fatalf("positions must stay contiguous: %s", r.Body)
		}
	}

	other := e.register("intruder")
	e.expect(e.do("DELETE", "/me/photos/"+photoIDs[1], other.Access, nil), http.StatusNotFound, "delete someone else's photo")
}

func TestDatingFlowMatchChatSafety(t *testing.T) {
	e := setupEnv(t, false)
	lat, lng := randomArea()

	alice := e.register("alice")
	bob := e.register("bob")
	carl := e.register("carl")
	dana := e.register("dana")

	e.expect(e.do("GET", "/discover", alice.Access, nil), http.StatusConflict, "discover before onboarding")

	e.onboard(alice, onboarding{"Alice", "1994-05-10", "woman", []string{"man"}, lat, lng, []string{"hiking", "cooking", "music"}})
	e.onboard(bob, onboarding{"Bob", "1992-02-20", "man", []string{"woman"}, lat + 0.02, lng, []string{"hiking", "music"}})
	// Carl only wants men: never shown to Alice, and Alice never shown to him.
	e.onboard(carl, onboarding{"Carl", "1990-09-09", "man", []string{"man"}, lat, lng + 0.01, []string{"tech"}})
	// Dana wants women but lives 300 km away.
	e.onboard(dana, onboarding{"Dana", "1993-03-03", "nonbinary", []string{"woman"}, lat + 3, lng, nil})

	feed := e.discover(alice)
	if !containsCard(feed, bob.ID) || containsCard(feed, carl.ID) || containsCard(feed, dana.ID) || containsCard(feed, alice.ID) {
		t.Fatalf("unexpected discovery feed for alice: %+v", feed)
	}
	raw := e.do("GET", "/discover", alice.Access, nil)
	mustNotContain(t, raw.Body, bob.Email, "birthdate", "latitude", "longitude", "email")
	for _, c := range feed {
		if c.UserID == bob.ID {
			if c.DistanceKm == nil || *c.DistanceKm < 1 || *c.DistanceKm > 4 {
				t.Fatalf("expected approximate distance ~2 km, got %v", c.DistanceKm)
			}
			shared := 0
			for _, i := range c.Interests {
				if i.Shared {
					shared++
				}
			}
			if shared != 2 {
				t.Fatalf("expected 2 shared interests, got %d", shared)
			}
		}
	}

	e.expect(e.do("POST", "/swipes", alice.Access, map[string]string{"targetUserId": alice.ID, "direction": "like"}), http.StatusBadRequest, "self like")
	e.expect(e.do("POST", "/swipes", alice.Access, map[string]string{"targetUserId": bob.ID, "direction": "superlike"}), http.StatusBadRequest, "unknown direction")

	r := e.do("POST", "/swipes", alice.Access, map[string]string{"targetUserId": bob.ID, "direction": "like"})
	e.expect(r, http.StatusOK, "alice likes bob")
	if strings.Contains(string(r.Body), `"matched":true`) {
		t.Fatal("one-sided like must not match")
	}
	e.expect(e.do("POST", "/swipes", alice.Access, map[string]string{"targetUserId": bob.ID, "direction": "like"}), http.StatusConflict, "repeated like")
	e.expect(e.do("POST", "/swipes", alice.Access, map[string]string{"targetUserId": bob.ID, "direction": "pass"}), http.StatusConflict, "pass after like")
	if containsCard(e.discover(alice), bob.ID) {
		t.Fatal("already swiped profile must not reappear")
	}

	// Bob connects to the realtime channel before liking back.
	ticketRes := e.do("POST", "/realtime/ticket", bob.Access, nil)
	e.expect(ticketRes, http.StatusOK, "ticket")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	ticketRes.decode(t, &ticket)
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/realtime?ticket=" + url.QueryEscape(ticket.Ticket)
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(e.server.URL, "http")+"/realtime?ticket="+bob.Access, nil); err == nil {
		t.Fatal("an access token must not be accepted as a realtime ticket")
	}
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer ws.Close()
	aliceWS := e.dialRealtime(alice)
	defer aliceWS.Close()

	if !containsCard(e.discover(bob), alice.ID) {
		t.Fatal("bob should discover alice")
	}
	r = e.do("POST", "/swipes", bob.Access, map[string]string{"targetUserId": alice.ID, "direction": "like"})
	e.expect(r, http.StatusCreated, "bob likes alice back")
	var swipe struct {
		Matched bool `json:"matched"`
		Match   struct {
			MatchID string `json:"matchId"`
			User    card   `json:"user"`
		} `json:"match"`
	}
	r.decode(t, &swipe)
	if !swipe.Matched || swipe.Match.User.UserID != alice.ID || swipe.Match.MatchID == "" {
		t.Fatalf("expected match with alice: %s", r.Body)
	}
	matchID := swipe.Match.MatchID
	// The other member (alice) is told in real time; bob got the HTTP response.
	if evt := expectEvent(t, aliceWS, "match.created"); !strings.Contains(string(evt), bob.ID) {
		t.Fatalf("alice's match event should carry bob's card: %s", evt)
	}
	expectEvent(t, ws, "notification.created")

	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE id = $1`, matchID).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}

	// Conversations and permissions.
	r = e.do("GET", "/conversations", alice.Access, nil)
	e.expect(r, http.StatusOK, "alice conversations")
	if !strings.Contains(string(r.Body), matchID) {
		t.Fatalf("alice should see the match conversation: %s", r.Body)
	}
	e.expect(e.do("GET", "/conversations/"+matchID+"/messages", carl.Access, nil), http.StatusNotFound, "outsider reads messages")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", carl.Access, map[string]string{"body": "intrusion"}), http.StatusNotFound, "outsider sends message")
	e.expect(e.do("POST", "/conversations/"+matchID+"/read", carl.Access, nil), http.StatusNotFound, "outsider marks read")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", alice.Access, map[string]string{"body": "   "}), http.StatusBadRequest, "blank message")

	r = e.do("POST", "/conversations/"+matchID+"/messages", alice.Access, map[string]string{"body": "Salut Bob ! 👋"})
	e.expect(r, http.StatusCreated, "alice sends")
	evt := expectEvent(t, ws, "message.created")
	if !strings.Contains(string(evt), "Salut Bob") {
		t.Fatalf("realtime event should carry the message: %s", evt)
	}
	for i := 0; i < 4; i++ {
		e.expect(e.do("POST", "/conversations/"+matchID+"/messages", alice.Access, map[string]string{"body": "message " + string(rune('A'+i))}), http.StatusCreated, "alice sends more")
	}

	var convs struct {
		Conversations []struct {
			MatchID     string `json:"matchId"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.do("GET", "/conversations", bob.Access, nil).decode(t, &convs)
	if len(convs.Conversations) == 0 || convs.Conversations[0].UnreadCount != 5 || convs.Conversations[0].LastMessage.Body != "message D" {
		t.Fatalf("unexpected bob conversations: %+v", convs)
	}

	// Pagination (newest page first, chronological inside a page).
	var page struct {
		Messages []struct {
			Body string `json:"body"`
		} `json:"messages"`
		NextCursor      string     `json:"nextCursor"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	e.do("GET", "/conversations/"+matchID+"/messages?limit=3", bob.Access, nil).decode(t, &page)
	if len(page.Messages) != 3 || page.Messages[2].Body != "message D" || page.NextCursor == "" {
		t.Fatalf("unexpected first page: %+v", page)
	}
	e.do("GET", "/conversations/"+matchID+"/messages?limit=3&cursor="+page.NextCursor, bob.Access, nil).decode(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Body != "Salut Bob ! 👋" || page.NextCursor != "" {
		t.Fatalf("unexpected second page: %+v", page)
	}

	// Notifications: bob got a match notification and one coalesced message notification.
	var notifs struct {
		Notifications []struct {
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.do("GET", "/notifications", bob.Access, nil).decode(t, &notifs)
	if notifs.UnreadCount != 2 {
		t.Fatalf("expected 2 unread notifications (match + coalesced messages), got %+v", notifs)
	}

	// Read receipts.
	e.expect(e.do("POST", "/conversations/"+matchID+"/read", bob.Access, nil), http.StatusNoContent, "bob reads")
	e.do("GET", "/conversations", bob.Access, nil).decode(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread count should reset after read")
	}
	e.do("GET", "/conversations/"+matchID+"/messages", alice.Access, nil).decode(t, &page)
	if page.OtherLastReadAt == nil {
		t.Fatal("alice should see bob's read receipt")
	}
	e.do("GET", "/notifications", bob.Access, nil).decode(t, &notifs)
	if notifs.UnreadCount != 1 {
		t.Fatalf("reading the conversation should clear its message notification, got %d", notifs.UnreadCount)
	}
	e.expect(e.do("POST", "/notifications/read-all", bob.Access, nil), http.StatusNoContent, "read all")

	// Local delete hides the conversation until a new message arrives.
	e.expect(e.do("DELETE", "/conversations/"+matchID, bob.Access, nil), http.StatusNoContent, "hide")
	e.do("GET", "/conversations", bob.Access, nil).decode(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("hidden conversation should disappear")
	}
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", alice.Access, map[string]string{"body": "Toujours là ?"}), http.StatusCreated, "new message")
	e.do("GET", "/conversations", bob.Access, nil).decode(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("new message should resurface the hidden conversation")
	}

	// Profile view permissions.
	e.expect(e.do("GET", "/profiles/"+alice.ID, bob.Access, nil), http.StatusOK, "view match profile")

	// Block: dissolves the match and hides both members from each other.
	e.expect(e.do("POST", "/blocks", alice.Access, map[string]string{"userId": bob.ID}), http.StatusNoContent, "block")
	expectEvent(t, ws, "match.removed")
	e.expect(e.do("POST", "/conversations/"+matchID+"/messages", bob.Access, map[string]string{"body": "?"}), http.StatusNotFound, "message after block")
	e.expect(e.do("GET", "/profiles/"+alice.ID, bob.Access, nil), http.StatusNotFound, "view blocker profile")
	e.do("GET", "/conversations", alice.Access, nil).decode(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("blocked match must disappear")
	}
	var blocks struct {
		Blocks []struct {
			UserID string `json:"userId"`
		} `json:"blocks"`
	}
	e.do("GET", "/blocks", alice.Access, nil).decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != bob.ID {
		t.Fatalf("unexpected block list: %+v", blocks)
	}
	e.expect(e.do("DELETE", "/blocks/"+bob.ID, alice.Access, nil), http.StatusNoContent, "unblock")

	// Report: stores evidence and blocks automatically.
	e.expect(e.do("POST", "/reports", alice.Access, map[string]string{"userId": dana.ID, "reason": "nope"}), http.StatusBadRequest, "invalid reason")
	e.expect(e.do("POST", "/reports", alice.Access, map[string]string{"userId": dana.ID, "reason": "spam", "details": "Liens suspects"}), http.StatusCreated, "report")
	e.expect(e.do("POST", "/reports", alice.Access, map[string]string{"userId": dana.ID, "reason": "harassment"}), http.StatusCreated, "duplicate report is idempotent")
	var reports, blocked int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, alice.ID, dana.ID).Scan(&reports)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, alice.ID, dana.ID).Scan(&blocked)
	if reports != 1 || blocked != 1 {
		t.Fatalf("expected 1 report and 1 block, got %d/%d", reports, blocked)
	}

	// Privacy: hidden profiles leave discovery.
	e.expect(e.do("PUT", "/me/visibility", carl.Access, map[string]any{"discoverable": false}), http.StatusOK, "hide carl")
	var carlCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM profiles WHERE user_id = $1 AND discoverable`, carl.ID).Scan(&carlCount)
	if carlCount != 0 {
		t.Fatal("visibility not saved")
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	e := setupEnv(t, false)
	lat, lng := randomArea()
	a := e.register("race_a")
	b := e.register("race_b")
	e.onboard(a, onboarding{"Ana", "1995-01-01", "woman", []string{"man"}, lat, lng, nil})
	e.onboard(b, onboarding{"Ben", "1995-01-01", "man", []string{"woman"}, lat, lng, nil})

	done := make(chan int, 2)
	go func() {
		done <- e.do("POST", "/swipes", a.Access, map[string]string{"targetUserId": b.ID, "direction": "like"}).Status
	}()
	go func() {
		done <- e.do("POST", "/swipes", b.Access, map[string]string{"targetUserId": a.ID, "direction": "like"}).Status
	}()
	statuses := []int{<-done, <-done}
	created := 0
	for _, s := range statuses {
		if s == http.StatusCreated {
			created++
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
	`, a.ID, b.ID).Scan(&n)
	if n != 1 || created != 1 {
		t.Fatalf("expected exactly one match, got rows=%d created=%d statuses=%v", n, created, statuses)
	}
}

func TestPasswordResetAndAccountDeletion(t *testing.T) {
	e := setupEnv(t, false)
	lat, lng := randomArea()
	a := e.register("reset")
	e.onboard(a, onboarding{"Rita", "1991-07-07", "woman", []string{"man"}, lat, lng, nil})

	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": uniqueEmail("ghost")}), http.StatusAccepted, "forgot unknown email does not leak")
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": a.Email}), http.StatusAccepted, "forgot")
	code := e.mail.code(t, a.Email)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": a.Email, "code": wrong, "newPassword": "NewPassword1"}), http.StatusBadRequest, "wrong code")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": a.Email, "code": code, "newPassword": "NewPassword1"}), http.StatusNoContent, "reset")
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"email": a.Email, "code": code, "newPassword": "OtherPass1"}), http.StatusBadRequest, "code reuse")
	e.expect(e.do("GET", "/me", a.Access, nil), http.StatusUnauthorized, "old sessions revoked after reset")
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": a.Password}), http.StatusUnauthorized, "old password")
	login := e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "NewPassword1"})
	e.expect(login, http.StatusOK, "login with new password")
	var tokens struct {
		AccessToken string `json:"accessToken"`
	}
	login.decode(t, &tokens)

	var profile struct {
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
	}
	e.do("GET", "/me/profile", tokens.AccessToken, nil).decode(t, &profile)
	photoURL := profile.Photos[0].URL
	var key string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&key)

	e.expect(e.do("DELETE", "/me", tokens.AccessToken, map[string]string{"password": "wrong-password1"}), http.StatusForbidden, "delete with wrong password")
	e.expect(e.do("DELETE", "/me", tokens.AccessToken, map[string]string{"password": "NewPassword1"}), http.StatusNoContent, "delete account")
	e.expect(e.do("GET", "/me", tokens.AccessToken, nil), http.StatusUnauthorized, "token after deletion")
	e.expect(e.do("GET", photoURL, "", nil), http.StatusNotFound, "photo after deletion")
	if _, err := os.Stat(filepath.Join(e.dir, key[:2], key)); !os.IsNotExist(err) {
		t.Fatalf("photo file should be deleted from storage, stat err=%v", err)
	}
	var count int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM profiles WHERE user_id = $1`, a.ID).Scan(&count)
	if count != 0 {
		t.Fatal("profile should be deleted with the account")
	}
}

func TestRateLimitOnCredentialEndpoints(t *testing.T) {
	e := setupEnv(t, true)
	limited := false
	for i := 0; i < 15; i++ {
		r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "nobody@example-test.dev"})
		if r.Status == http.StatusTooManyRequests {
			limited = r.Header.Get("Retry-After") != ""
			break
		}
	}
	if !limited {
		t.Fatal("credential endpoints should be rate limited with a Retry-After header")
	}
}

func (e *testEnv) dialRealtime(m member) *websocket.Conn {
	e.t.Helper()
	r := e.do("POST", "/realtime/ticket", m.Access, nil)
	e.expect(r, http.StatusOK, "ticket")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	r.decode(e.t, &ticket)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(e.server.URL, "http")+"/realtime?ticket="+url.QueryEscape(ticket.Ticket), nil)
	if err != nil {
		e.t.Fatalf("websocket dial: %v", err)
	}
	return conn
}

// expectEvent reads realtime events until one of the given type arrives,
// skipping unrelated ones (e.g. notification.created).
func expectEvent(t *testing.T, ws *websocket.Conn, eventType string) []byte {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = ws.SetReadDeadline(deadline)
		_, msg, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %s: %v", eventType, err)
		}
		var env struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(msg, &env)
		if env.Type == eventType {
			return msg
		}
	}
}
