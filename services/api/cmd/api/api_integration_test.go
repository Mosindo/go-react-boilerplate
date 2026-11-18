package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/ratelimit"
)

func TestHealthEndpoint(t *testing.T) {
	env := newTestEnv(t)
	resp := env.do(http.MethodGet, "/health", nil, "")
	env.expect(resp, http.StatusOK, "health")
	if !strings.Contains(string(resp.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health body: %s", resp.Body)
	}
}

func TestHealthPropagatesRequestID(t *testing.T) {
	env := newTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "req-integration-fixed-id")
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "req-integration-fixed-id" {
		t.Fatalf("expected request id to be echoed, got %q", got)
	}
}

func TestHealthReturns503WhenDBUnavailable(t *testing.T) {
	env := newTestEnv(t)
	env.pool.Close()
	env.expect(env.do(http.MethodGet, "/health", nil, ""), http.StatusServiceUnavailable, "health without db")
}

func TestAuthFlowAndSessionRevocation(t *testing.T) {
	env := newTestEnv(t)

	weak := env.do(http.MethodPost, "/auth/register", map[string]string{"email": uniqueEmail("weak"), "password": "password"}, "")
	env.expect(weak, http.StatusBadRequest, "weak password")

	s := env.register("auth")
	env.expect(env.do(http.MethodPost, "/auth/register", map[string]string{"email": strings.ToUpper(s.Email), "password": testPassword}, ""), http.StatusConflict, "duplicate e-mail (case-insensitive)")
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": s.Email, "password": "WrongPass1"}, ""), http.StatusUnauthorized, "wrong password")

	login := env.do(http.MethodPost, "/auth/login", map[string]string{"email": "  " + s.Email + " ", "password": testPassword}, "")
	env.expect(login, http.StatusOK, "login")
	var p authPayload
	login.decode(t, &p)

	me := env.do(http.MethodGet, "/me", nil, p.AccessToken)
	env.expect(me, http.StatusOK, "me")
	if strings.Contains(string(me.Body), "password") {
		t.Fatal("password hash must never be returned")
	}

	refreshed := env.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": p.RefreshToken}, "")
	env.expect(refreshed, http.StatusOK, "refresh")
	var r authPayload
	refreshed.decode(t, &r)
	env.expect(env.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": p.RefreshToken}, ""), http.StatusUnauthorized, "rotated refresh token reuse")

	env.expect(env.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": r.RefreshToken}, ""), http.StatusNoContent, "logout")
	env.expect(env.do(http.MethodGet, "/me", nil, r.AccessToken), http.StatusUnauthorized, "access token after logout")

	// Password change keeps the current session and revokes the others.
	env.expect(env.do(http.MethodPut, "/me/password", map[string]string{"currentPassword": "nope", "newPassword": "NewPassword9"}, s.Token), http.StatusForbidden, "wrong current password")
	env.expect(env.do(http.MethodPut, "/me/password", map[string]string{"currentPassword": testPassword, "newPassword": "NewPassword9"}, s.Token), http.StatusNoContent, "change password")
	env.expect(env.do(http.MethodGet, "/me", nil, s.Token), http.StatusOK, "current session survives password change")

	env.expect(env.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": "nobody@lueur.test"}, ""), http.StatusAccepted, "forgot unknown e-mail")
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": s.Email, "code": "123456", "password": "Another123"}, ""), http.StatusBadRequest, "reset without code")
}

func TestProtectedEndpointsRequireAuth(t *testing.T) {
	env := newTestEnv(t)
	paths := []struct{ method, path string }{
		{http.MethodGet, "/me"},
		{http.MethodGet, "/me/profile"},
		{http.MethodPut, "/me/profile"},
		{http.MethodPost, "/me/photos"},
		{http.MethodGet, "/discovery"},
		{http.MethodGet, "/profiles/00000000-0000-0000-0000-000000000000"},
		{http.MethodPost, "/swipes"},
		{http.MethodGet, "/matches"},
		{http.MethodGet, "/chats"},
		{http.MethodGet, "/chats/00000000-0000-0000-0000-000000000000/messages"},
		{http.MethodPost, "/blocks"},
		{http.MethodPost, "/reports"},
		{http.MethodGet, "/notifications"},
		{http.MethodDelete, "/me"},
		{http.MethodPost, "/realtime/ticket"},
	}
	for _, p := range paths {
		env.expect(env.do(p.method, p.path, nil, ""), http.StatusUnauthorized, p.method+" "+p.path)
		env.expect(env.do(p.method, p.path, nil, "not-a-jwt"), http.StatusUnauthorized, p.method+" "+p.path+" bad token")
	}
}

func TestProfileOnboardingAndPhotos(t *testing.T) {
	env := newTestEnv(t)
	s := env.register("profile")

	var me struct {
		IsComplete bool     `json:"isComplete"`
		Missing    []string `json:"missing"`
		Profile    *struct {
			FirstName   string `json:"firstName"`
			Age         int    `json:"age"`
			HasLocation bool   `json:"hasLocation"`
		} `json:"profile"`
		Photos []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"photos"`
		Interests []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
	}
	resp := env.do(http.MethodGet, "/me/profile", nil, s.Token)
	env.expect(resp, http.StatusOK, "empty profile")
	resp.decode(t, &me)
	if me.IsComplete || len(me.Missing) != 3 {
		t.Fatalf("new account must be incomplete: %+v", me)
	}

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	under := env.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "Léa", "birthdate": minor, "gender": "woman"}, s.Token)
	env.expect(under, http.StatusBadRequest, "underage")
	if !strings.Contains(string(under.Body), "underage") {
		t.Fatalf("expected underage code: %s", under.Body)
	}
	env.expect(env.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "<b>x</b>", "birthdate": "1995-01-01", "gender": "woman"}, s.Token), http.StatusBadRequest, "invalid first name")
	env.expect(env.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "Léa", "birthdate": "1995-01-01", "gender": "robot"}, s.Token), http.StatusBadRequest, "invalid gender")
	env.expect(env.do(http.MethodPut, "/me/location", map[string]any{"latitude": 48.8, "longitude": 2.3}, s.Token), http.StatusConflict, "location before profile")

	env.expect(env.do(http.MethodPut, "/me/profile", map[string]any{"firstName": " Léa ", "birthdate": "1995-01-01", "gender": "woman", "bio": "Café & montagne", "heightCm": 168}, s.Token), http.StatusOK, "profile")
	env.expect(env.do(http.MethodPut, "/me/profile", map[string]any{"firstName": "Léa", "birthdate": "2000-01-01", "gender": "woman"}, s.Token), http.StatusBadRequest, "birthdate locked")
	env.expect(env.do(http.MethodPut, "/me/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10}, s.Token), http.StatusBadRequest, "invalid age range")
	env.expect(env.do(http.MethodPut, "/me/preferences", map[string]any{"interestedIn": []string{"man", "nonbinary"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 30}, s.Token), http.StatusOK, "preferences")
	env.expect(env.do(http.MethodPut, "/me/interests", map[string]any{"slugs": []string{"hiking", "unknown-thing"}}, s.Token), http.StatusBadRequest, "unknown interest")
	env.expect(env.do(http.MethodPut, "/me/interests", map[string]any{"slugs": []string{"hiking", "coffee", "hiking"}}, s.Token), http.StatusOK, "interests")

	// Precise coordinates are rounded before storage.
	env.expect(env.do(http.MethodPut, "/me/location", map[string]any{"latitude": 48.856613, "longitude": 2.352222, "city": "Paris"}, s.Token), http.StatusOK, "location")
	var lat, lng float64
	if err := env.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, s.UserID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be stored rounded, got %v,%v", lat, lng)
	}

	// Uploads are validated by content.
	env.expect(env.uploadPhoto(s.Token, http.MethodPost, "/me/photos", []byte("<?php system($_GET['c']); ?>")), http.StatusUnsupportedMediaType, "script upload")
	env.expect(env.uploadPhoto(s.Token, http.MethodPost, "/me/photos", testImage(t, 100, 100)), http.StatusBadRequest, "tiny image")
	for i := 0; i < 3; i++ {
		env.expect(env.uploadPhoto(s.Token, http.MethodPost, "/me/photos", testImage(t, 300+i, 400)), http.StatusCreated, "upload")
	}

	resp = env.do(http.MethodGet, "/me/profile", nil, s.Token)
	resp.decode(t, &me)
	if !me.IsComplete || len(me.Photos) != 3 || len(me.Interests) != 2 || me.Profile.FirstName != "Léa" || !me.Profile.HasLocation {
		t.Fatalf("unexpected completed profile: %+v", me)
	}
	if strings.Contains(string(resp.Body), "48.86") || strings.Contains(string(resp.Body), "latitude") {
		t.Fatal("coordinates must never be returned")
	}

	// Signed media URLs work, tampered ones do not.
	photoURL := me.Photos[0].URL
	env.expect(env.do(http.MethodGet, photoURL, nil, ""), http.StatusOK, "signed photo")
	env.expect(env.do(http.MethodGet, strings.Replace(photoURL, "sig=", "sig=x", 1), nil, ""), http.StatusNotFound, "tampered signature")
	env.expect(env.do(http.MethodGet, "/media/photos/"+me.Photos[1].ID+photoURL[strings.Index(photoURL, "?"):], nil, ""), http.StatusNotFound, "signature reused for another photo")

	// Reorder: last photo becomes primary.
	order := []string{me.Photos[2].ID, me.Photos[0].ID, me.Photos[1].ID}
	env.expect(env.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order[:2]}, s.Token), http.StatusBadRequest, "partial order")
	reordered := env.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order}, s.Token)
	env.expect(reordered, http.StatusOK, "reorder")
	var photos struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	reordered.decode(t, &photos)
	if photos.Photos[0].ID != order[0] || photos.Photos[0].Position != 0 {
		t.Fatalf("unexpected order %+v", photos)
	}

	// Replace keeps the position, delete compacts positions.
	env.expect(env.uploadPhoto(s.Token, http.MethodPut, "/me/photos/"+order[1], testImage(t, 500, 500)), http.StatusOK, "replace")
	deleted := env.do(http.MethodDelete, "/me/photos/"+order[0], nil, s.Token)
	env.expect(deleted, http.StatusOK, "delete")
	deleted.decode(t, &photos)
	if len(photos.Photos) != 2 || photos.Photos[0].Position != 0 || photos.Photos[1].Position != 1 || photos.Photos[0].ID != order[1] {
		t.Fatalf("unexpected photos after delete %+v", photos)
	}

	// Another member cannot touch these photos.
	other := env.register("intruder")
	env.expect(env.do(http.MethodDelete, "/me/photos/"+order[1], nil, other.Token), http.StatusNotFound, "foreign photo delete")

	for i := 0; i < 4; i++ {
		env.expect(env.uploadPhoto(s.Token, http.MethodPost, "/me/photos", testImage(t, 300, 300)), http.StatusCreated, "fill photos")
	}
	env.expect(env.uploadPhoto(s.Token, http.MethodPost, "/me/photos", testImage(t, 300, 300)), http.StatusConflict, "seventh photo")
}

func TestDiscoveryLikeMatchAndChat(t *testing.T) {
	env := newTestEnv(t)
	alice := env.onboard("alice", "Alice", "woman", []string{"man"}, 48.8566, 2.3522)
	bruno := env.onboard("bruno", "Bruno", "man", []string{"woman"}, 48.87, 2.33)
	carl := env.onboard("carl", "Carl", "man", []string{"man"}, 48.86, 2.34)
	far := env.onboard("far", "Farid", "man", []string{"woman"}, 43.2965, 5.3698) // Marseille, ~660 km

	cards := env.discover(alice.Token)
	if !containsCard(cards, bruno.UserID) {
		t.Fatal("Alice should discover Bruno")
	}
	if containsCard(cards, carl.UserID) {
		t.Fatal("Carl is not interested in women: preferences must be mutual")
	}
	if containsCard(cards, far.UserID) {
		t.Fatal("members outside the distance radius must be excluded")
	}
	if containsCard(cards, alice.UserID) {
		t.Fatal("members never see themselves")
	}
	for _, c := range cards {
		if c.UserID == bruno.UserID && (c.DistanceKm == nil || *c.DistanceKm < 1 || *c.DistanceKm > 5) {
			t.Fatalf("expected an approximate distance for Bruno, got %v", c.DistanceKm)
		}
	}

	// Alice likes Bruno: no match yet, Bruno disappears from her feed.
	resp := env.swipe(alice.Token, bruno.UserID, "like")
	env.expect(resp, http.StatusOK, "like")
	if strings.Contains(string(resp.Body), `"matched":true`) {
		t.Fatal("a one-sided like must not match")
	}
	env.expect(env.swipe(alice.Token, bruno.UserID, "like"), http.StatusConflict, "repeated like")
	env.expect(env.swipe(alice.Token, bruno.UserID, "pass"), http.StatusConflict, "changing an existing decision")
	env.expect(env.swipe(alice.Token, alice.UserID, "like"), http.StatusBadRequest, "self like")
	env.expect(env.swipe(alice.Token, carl.UserID, "superlike"), http.StatusBadRequest, "unknown decision")
	if containsCard(env.discover(alice.Token), bruno.UserID) {
		t.Fatal("processed profiles must not reappear")
	}

	// No conversation before the match.
	env.expect(env.do(http.MethodPost, "/chats/"+bruno.UserID+"/messages", map[string]string{"content": "Salut"}, alice.Token), http.StatusNotFound, "chat before match")

	// Bruno likes back: match.
	matched := env.swipe(bruno.Token, alice.UserID, "like")
	env.expect(matched, http.StatusCreated, "mutual like")
	var swipe struct {
		Matched bool `json:"matched"`
		Match   struct {
			ID   string `json:"id"`
			User struct {
				UserID    string `json:"userId"`
				FirstName string `json:"firstName"`
			} `json:"user"`
		} `json:"match"`
	}
	matched.decode(t, &swipe)
	if !swipe.Matched || swipe.Match.User.UserID != alice.UserID || swipe.Match.User.FirstName != "Alice" {
		t.Fatalf("unexpected match payload %s", matched.Body)
	}
	var matchCount int
	if err := env.pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
	`, alice.UserID, bruno.UserID).Scan(&matchCount); err != nil || matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d (%v)", matchCount, err)
	}

	for _, m := range []member{alice, bruno} {
		list := env.do(http.MethodGet, "/matches", nil, m.Token)
		env.expect(list, http.StatusOK, "matches")
		if !strings.Contains(string(list.Body), swipe.Match.ID) || strings.Contains(string(list.Body), "@lueur.test") {
			t.Fatalf("matches list must contain the match and never e-mails: %s", list.Body)
		}
		notifs := env.do(http.MethodGet, "/notifications", nil, m.Token)
		if !strings.Contains(string(notifs.Body), "match.created") {
			t.Fatalf("expected a match notification: %s", notifs.Body)
		}
	}

	// Full profile of a match is visible.
	profile := env.do(http.MethodGet, "/profiles/"+alice.UserID, nil, bruno.Token)
	env.expect(profile, http.StatusOK, "match profile")
	if !strings.Contains(string(profile.Body), `"isMatch":true`) || strings.Contains(string(profile.Body), "birthdate") {
		t.Fatalf("unexpected profile payload: %s", profile.Body)
	}

	// Chat.
	env.expect(env.do(http.MethodPost, "/chats/"+bruno.UserID+"/messages", map[string]string{"content": "   "}, alice.Token), http.StatusBadRequest, "empty message")
	env.expect(env.do(http.MethodPost, "/chats/"+bruno.UserID+"/messages", map[string]string{"content": strings.Repeat("a", 2001)}, alice.Token), http.StatusBadRequest, "too long message")
	first := env.do(http.MethodPost, "/chats/"+bruno.UserID+"/messages", map[string]string{"content": "Salut Bruno !"}, alice.Token)
	env.expect(first, http.StatusCreated, "send")
	env.expect(env.do(http.MethodPost, "/chats/"+bruno.UserID+"/messages", map[string]string{"content": "Tu es dispo ce week-end ?"}, alice.Token), http.StatusCreated, "send 2")

	chats := env.do(http.MethodGet, "/chats", nil, bruno.Token)
	env.expect(chats, http.StatusOK, "chats")
	var chatList struct {
		Chats []struct {
			User struct {
				UserID string `json:"userId"`
			} `json:"user"`
			LastMessage struct {
				Content string `json:"content"`
			} `json:"lastMessage"`
			UnreadCount int `json:"unreadCount"`
		} `json:"chats"`
	}
	chats.decode(t, &chatList)
	if len(chatList.Chats) != 1 || chatList.Chats[0].UnreadCount != 2 || chatList.Chats[0].LastMessage.Content != "Tu es dispo ce week-end ?" {
		t.Fatalf("unexpected chat list %s", chats.Body)
	}

	// Message notifications are coalesced while unread.
	var messageNotifications int
	if err := env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND type = 'message.received'`, bruno.UserID).Scan(&messageNotifications); err != nil || messageNotifications != 1 {
		t.Fatalf("expected one coalesced message notification, got %d (%v)", messageNotifications, err)
	}

	env.expect(env.do(http.MethodPost, "/chats/"+alice.UserID+"/read", nil, bruno.Token), http.StatusOK, "mark read")
	chats = env.do(http.MethodGet, "/chats", nil, bruno.Token)
	chats.decode(t, &chatList)
	if chatList.Chats[0].UnreadCount != 0 {
		t.Fatalf("expected unread count reset, got %d", chatList.Chats[0].UnreadCount)
	}
	history := env.do(http.MethodGet, "/chats/"+bruno.UserID+"/messages?limit=1", nil, alice.Token)
	env.expect(history, http.StatusOK, "history")
	var page struct {
		Messages []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"messages"`
		HasMore         bool       `json:"hasMore"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	history.decode(t, &page)
	if len(page.Messages) != 1 || !page.HasMore || page.OtherLastReadAt == nil || page.Messages[0].Content != "Tu es dispo ce week-end ?" {
		t.Fatalf("unexpected history page %s", history.Body)
	}
	older := env.do(http.MethodGet, "/chats/"+bruno.UserID+"/messages?limit=10&before="+page.Messages[0].ID, nil, alice.Token)
	older.decode(t, &page)
	if len(page.Messages) != 1 || page.HasMore || page.Messages[0].Content != "Salut Bruno !" {
		t.Fatalf("unexpected older page %s", older.Body)
	}

	// Outsiders can never read or write into the conversation.
	env.expect(env.do(http.MethodGet, "/chats/"+bruno.UserID+"/messages", nil, carl.Token), http.StatusNotFound, "outsider read")
	env.expect(env.do(http.MethodPost, "/chats/"+alice.UserID+"/messages", map[string]string{"content": "hey"}, carl.Token), http.StatusNotFound, "outsider send")
	env.expect(env.do(http.MethodGet, "/chats", nil, carl.Token), http.StatusOK, "outsider list")

	// Unmatch removes the conversation for both members.
	env.expect(env.do(http.MethodDelete, "/matches/"+bruno.UserID, nil, alice.Token), http.StatusNoContent, "unmatch")
	env.expect(env.do(http.MethodGet, "/chats/"+alice.UserID+"/messages", nil, bruno.Token), http.StatusNotFound, "chat after unmatch")
	env.expect(env.do(http.MethodDelete, "/matches/"+bruno.UserID, nil, alice.Token), http.StatusNotFound, "double unmatch")
	var messages int
	if err := env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE sender_user_id = $1`, alice.UserID).Scan(&messages); err != nil || messages != 0 {
		t.Fatalf("messages must be deleted with the conversation, got %d (%v)", messages, err)
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	env := newTestEnv(t)
	a := env.onboard("race_a", "Anna", "woman", []string{"man"}, 45.76, 4.83)
	b := env.onboard("race_b", "Basile", "man", []string{"woman"}, 45.76, 4.84)

	results := make(chan int, 2)
	go func() { results <- env.swipe(a.Token, b.UserID, "like").Status }()
	go func() { results <- env.swipe(b.Token, a.UserID, "like").Status }()
	s1, s2 := <-results, <-results
	if !((s1 == http.StatusCreated && s2 == http.StatusOK) || (s1 == http.StatusOK && s2 == http.StatusCreated)) {
		t.Fatalf("expected exactly one request to create the match, got %d and %d", s1, s2)
	}
	var count int
	if err := env.pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid, $2::uuid) AND user_high_id = GREATEST($1::uuid, $2::uuid)
	`, a.UserID, b.UserID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected one match, got %d (%v)", count, err)
	}
}

func TestBlockReportAndModeration(t *testing.T) {
	env := newTestEnv(t)
	emma := env.onboard("emma", "Emma", "woman", []string{"man"}, 44.84, -0.58)
	marc := env.onboard("marc", "Marc", "man", []string{"woman"}, 44.84, -0.57)
	env.match(emma, marc)
	env.expect(env.do(http.MethodPost, "/chats/"+marc.UserID+"/messages", map[string]string{"content": "Coucou"}, emma.Token), http.StatusCreated, "send")

	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": emma.UserID}, emma.Token), http.StatusBadRequest, "self block")
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": marc.UserID}, emma.Token), http.StatusNoContent, "block")
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": marc.UserID}, emma.Token), http.StatusNoContent, "block is idempotent")
	env.expect(env.do(http.MethodGet, "/chats/"+emma.UserID+"/messages", nil, marc.Token), http.StatusNotFound, "blocked chat")
	env.expect(env.do(http.MethodGet, "/profiles/"+emma.UserID, nil, marc.Token), http.StatusNotFound, "blocked profile view")
	env.expect(env.do(http.MethodGet, "/profiles/"+marc.UserID, nil, emma.Token), http.StatusNotFound, "blocker profile view")
	if strings.Contains(string(env.do(http.MethodGet, "/matches", nil, marc.Token).Body), emma.UserID) {
		t.Fatal("blocking must remove the match")
	}

	blocks := env.do(http.MethodGet, "/blocks", nil, emma.Token)
	if !strings.Contains(string(blocks.Body), marc.UserID) {
		t.Fatalf("blocked list must contain Marc: %s", blocks.Body)
	}
	env.expect(env.do(http.MethodDelete, "/blocks/"+marc.UserID, nil, emma.Token), http.StatusNoContent, "unblock")
	env.expect(env.do(http.MethodDelete, "/blocks/"+marc.UserID, nil, emma.Token), http.StatusNotFound, "unblock twice")
	// Previous swipes remain, so they do not reappear to each other.
	if containsCard(env.discover(emma.Token), marc.UserID) {
		t.Fatal("unblocking must not resurrect a processed profile")
	}

	// Blocked members never see each other in discovery.
	nina := env.onboard("nina", "Nina", "woman", []string{"man"}, 44.84, -0.58)
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": nina.UserID}, marc.Token), http.StatusNoContent, "marc blocks nina")
	if containsCard(env.discover(nina.Token), marc.UserID) || containsCard(env.discover(marc.Token), nina.UserID) {
		t.Fatal("blocks must hide profiles in both directions")
	}
	env.expect(env.swipe(nina.Token, marc.UserID, "like"), http.StatusNotFound, "like a member who blocked you")

	// Reports: validation, then moderation threshold.
	suspect := env.onboard("suspect", "Sam", "man", []string{"woman"}, 44.84, -0.58)
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": suspect.UserID, "reason": "because"}, emma.Token), http.StatusBadRequest, "invalid reason")
	reporters := []member{emma, nina, env.onboard("zoe", "Zoé", "woman", []string{"man"}, 44.84, -0.58)}
	// Reports from accounts younger than 24 h are stored but do not hide anyone.
	newcomer := env.onboard("newcomer", "Nadia", "woman", []string{"man"}, 44.84, -0.58)
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": suspect.UserID, "reason": "spam"}, newcomer.Token), http.StatusCreated, "newcomer report")
	for _, r := range reporters {
		if _, err := env.pool.Exec(context.Background(), `UPDATE users SET created_at = NOW() - INTERVAL '3 days' WHERE id = $1`, r.UserID); err != nil {
			t.Fatal(err)
		}
	}
	for i, r := range reporters {
		env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": suspect.UserID, "reason": "harassment", "details": "Messages insistants"}, r.Token), http.StatusCreated, "report")
		var hidden bool
		if err := env.pool.QueryRow(context.Background(), `SELECT moderation_hidden FROM profiles WHERE user_id = $1`, suspect.UserID).Scan(&hidden); err != nil {
			t.Fatal(err)
		}
		if want := i+1 >= 3; hidden != want {
			t.Fatalf("after %d reports moderation_hidden=%v, want %v", i+1, hidden, want)
		}
	}
	// Duplicate open reports from the same member are ignored.
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": suspect.UserID, "reason": "spam"}, emma.Token), http.StatusCreated, "repeat report")
	var reports int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reported_user_id = $1`, suspect.UserID).Scan(&reports)
	if reports != 4 {
		t.Fatalf("expected 4 reports, got %d", reports)
	}
	fresh := env.onboard("fresh", "Fanny", "woman", []string{"man"}, 44.84, -0.58)
	if containsCard(env.discover(fresh.Token), suspect.UserID) {
		t.Fatal("profiles under moderation review must be hidden")
	}
}

func TestPrivacyControls(t *testing.T) {
	env := newTestEnv(t)
	ines := env.onboard("ines", "Inès", "woman", []string{"man"}, 50.63, 3.06)
	hugo := env.onboard("hugo", "Hugo", "man", []string{"woman"}, 50.63, 3.07)

	env.expect(env.do(http.MethodPut, "/me/privacy", map[string]any{"showDistance": false}, ines.Token), http.StatusOK, "hide distance")
	for _, c := range env.discover(hugo.Token) {
		if c.UserID == ines.UserID && c.DistanceKm != nil {
			t.Fatal("distance must be hidden when the member opted out")
		}
	}
	env.expect(env.do(http.MethodPut, "/me/privacy", map[string]any{"isDiscoverable": false}, ines.Token), http.StatusOK, "pause profile")
	if containsCard(env.discover(hugo.Token), ines.UserID) {
		t.Fatal("paused profiles must not be discoverable")
	}
	env.expect(env.do(http.MethodGet, "/profiles/"+ines.UserID, nil, hugo.Token), http.StatusNotFound, "paused profile view by stranger")
	env.expect(env.do(http.MethodDelete, "/me/location", nil, ines.Token), http.StatusOK, "clear location")
}

func TestTenantIsolation(t *testing.T) {
	env := newTestEnv(t)
	other := newTestEnv(t)
	a := env.onboard("tenant_a", "Alma", "woman", []string{"man"}, 47.21, -1.55)
	b := other.onboard("tenant_b", "Boris", "man", []string{"woman"}, 47.21, -1.55)
	if containsCard(env.discover(a.Token), b.UserID) {
		t.Fatal("members of another organisation must never be discoverable")
	}
	env.expect(env.swipe(a.Token, b.UserID, "like"), http.StatusNotFound, "cross-tenant like")
	env.expect(env.do(http.MethodGet, "/profiles/"+b.UserID, nil, a.Token), http.StatusNotFound, "cross-tenant profile")
}

func TestAccountDeletion(t *testing.T) {
	env := newTestEnv(t)
	lou := env.onboard("lou", "Lou", "woman", []string{"man"}, 43.6, 1.44)
	max := env.onboard("max", "Max", "man", []string{"woman"}, 43.6, 1.44)
	env.match(lou, max)
	env.expect(env.do(http.MethodPost, "/chats/"+max.UserID+"/messages", map[string]string{"content": "Bonjour"}, lou.Token), http.StatusCreated, "send")

	var keys []string
	rows, err := env.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, lou.UserID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()

	env.expect(env.do(http.MethodDelete, "/me", map[string]string{"password": "wrong"}, lou.Token), http.StatusForbidden, "delete with wrong password")
	env.expect(env.do(http.MethodDelete, "/me", map[string]string{"password": testPassword}, lou.Token), http.StatusNoContent, "delete")
	env.expect(env.do(http.MethodGet, "/me", nil, lou.Token), http.StatusUnauthorized, "token after deletion")
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": lou.Email, "password": testPassword}, ""), http.StatusUnauthorized, "login after deletion")

	if strings.Contains(string(env.do(http.MethodGet, "/matches", nil, max.Token).Body), lou.UserID) {
		t.Fatal("deleted member must disappear from matches")
	}
	var leftovers int
	_ = env.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM messages WHERE sender_user_id = $1 OR recipient_user_id = $1)
		     + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM swipes WHERE actor_user_id = $1 OR target_user_id = $1)
		     + (SELECT COUNT(*) FROM notifications WHERE data->>'userId' = $1::text)
	`, lou.UserID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("expected no personal data left, found %d rows", leftovers)
	}
	for _, k := range keys {
		path, _ := env.app.storage.Path(k)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("photo file %s must be deleted", k)
		}
	}
}

func TestRateLimitingOnAnonymousEndpoints(t *testing.T) {
	l := generousLimits()
	l.anonymous = ratelimit.New(3, time.Hour, 3)
	env := newTestEnvWithLimits(t, l)
	payload := map[string]string{"email": "nobody@lueur.test", "password": "Password123"}
	for i := 0; i < 3; i++ {
		env.expect(env.do(http.MethodPost, "/auth/login", payload, ""), http.StatusUnauthorized, "login attempt")
	}
	limited := env.do(http.MethodPost, "/auth/login", payload, "")
	env.expect(limited, http.StatusTooManyRequests, "rate limited login")
	if limited.Header.Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}
