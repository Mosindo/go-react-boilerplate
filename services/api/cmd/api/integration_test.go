package main

import (
	"context"
	"encoding/json"
	"image/color"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestHealth(t *testing.T) {
	env := newTestEnv(t)
	r := env.do(http.MethodGet, "/health", nil, "")
	env.expect(r, http.StatusOK)
	if r.Header.Get("X-Request-ID") == "" || r.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing standard headers: %v", r.Header)
	}
}

func TestProtectedEndpointsRequireAuth(t *testing.T) {
	env := newTestEnv(t)
	for _, route := range [][2]string{
		{http.MethodGet, "/me"}, {http.MethodGet, "/profile"}, {http.MethodGet, "/discover"},
		{http.MethodPost, "/swipes"}, {http.MethodGet, "/matches"}, {http.MethodGet, "/notifications"},
		{http.MethodGet, "/conversations/7b1d7f0e-4e43-4a3f-9a5a-1f1f1f1f1f1f/messages"},
		{http.MethodPost, "/blocks"}, {http.MethodPost, "/reports"}, {http.MethodDelete, "/me"},
	} {
		r := env.do(route[0], route[1], nil, "")
		if r.Status != http.StatusUnauthorized {
			t.Errorf("%s %s: expected 401, got %d", route[0], route[1], r.Status)
		}
	}
	if r := env.do(http.MethodGet, "/me", nil, "forged.token.value"); r.Status != http.StatusUnauthorized {
		t.Fatalf("forged token must be rejected, got %d", r.Status)
	}
}

func TestAuthLifecycle(t *testing.T) {
	env := newTestEnv(t)

	weak := env.do(http.MethodPost, "/auth/register", map[string]string{"email": "weak@hearth.test", "password": "password"}, "")
	if weak.Status != http.StatusBadRequest || weak.code(t) != "weak_password" {
		t.Fatalf("weak password must be rejected: %d %s", weak.Status, weak.Body)
	}

	m := env.register("auth")
	dup := env.do(http.MethodPost, "/auth/register", map[string]string{"email": "  " + strings.ToUpper(m.Email) + " ", "password": testPassword}, "")
	if dup.Status != http.StatusConflict {
		t.Fatalf("expected normalized duplicate email to conflict, got %d", dup.Status)
	}

	env.expect(env.do(http.MethodGet, "/me", nil, m.Token), http.StatusOK)
	bad := env.do(http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": "Wrong12345"}, "")
	env.expect(bad, http.StatusUnauthorized)
	unknown := env.do(http.MethodPost, "/auth/login", map[string]string{"email": "nobody@hearth.test", "password": "Wrong12345"}, "")
	if unknown.Status != http.StatusUnauthorized || string(unknown.Body) != string(bad.Body) {
		t.Fatal("unknown email and wrong password must be indistinguishable")
	}

	login := env.do(http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": testPassword}, "")
	env.expect(login, http.StatusOK)
	var session authResult
	login.decode(t, &session)

	refreshed := env.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, "")
	env.expect(refreshed, http.StatusOK)
	var rotated authResult
	refreshed.decode(t, &rotated)
	env.expect(env.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": session.RefreshToken}, ""), http.StatusUnauthorized)

	// Logout revokes the session: its access token stops working immediately.
	env.expect(env.do(http.MethodPost, "/auth/logout", map[string]string{"refreshToken": rotated.RefreshToken}, ""), http.StatusNoContent)
	env.expect(env.do(http.MethodGet, "/me", nil, rotated.AccessToken), http.StatusUnauthorized)

	// Changing the password keeps the current session and revokes the others.
	r := env.do(http.MethodPost, "/me/password", map[string]string{"currentPassword": "nope12345", "newPassword": "NewPass123"}, m.Token)
	env.expect(r, http.StatusForbidden)
	env.expect(env.do(http.MethodPost, "/me/password", map[string]string{"currentPassword": testPassword, "newPassword": "NewPass123"}, m.Token), http.StatusNoContent)
	env.expect(env.do(http.MethodGet, "/me", nil, m.Token), http.StatusOK)
	env.expect(env.do(http.MethodGet, "/me", nil, session.AccessToken), http.StatusUnauthorized)
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": "NewPass123"}, ""), http.StatusOK)
}

func TestPasswordRecovery(t *testing.T) {
	env := newTestEnv(t)
	m := env.register("recover")

	// Unknown emails get the same answer.
	env.expect(env.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": "ghost@hearth.test"}, ""), http.StatusAccepted)
	env.expect(env.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": m.Email}, ""), http.StatusAccepted)
	code := env.mail.lastCodeFor(m.Email)
	if code == "" {
		t.Fatal("expected a reset code email")
	}

	wrong := env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": "AAAAAAAA", "password": "Recovered123"}, "")
	if wrong.Status != http.StatusBadRequest || wrong.code(t) != "invalid_reset_code" {
		t.Fatalf("wrong code must fail: %d %s", wrong.Status, wrong.Body)
	}
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": strings.ToLower(code), "password": "Recovered123"}, ""), http.StatusNoContent)

	// Code is single use, every session is revoked, the new password works.
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "password": "Another123"}, ""), http.StatusBadRequest)
	env.expect(env.do(http.MethodGet, "/me", nil, m.Token), http.StatusUnauthorized)
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": "Recovered123"}, ""), http.StatusOK)
}

func TestPasswordRecoveryLocksAfterAttempts(t *testing.T) {
	env := newTestEnv(t)
	m := env.register("lock")
	env.expect(env.do(http.MethodPost, "/auth/password/forgot", map[string]string{"email": m.Email}, ""), http.StatusAccepted)
	code := env.mail.lastCodeFor(m.Email)
	for i := 0; i < 5; i++ {
		env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": "ZZZZZZZZ", "password": "Recovered123"}, ""), http.StatusBadRequest)
	}
	env.expect(env.do(http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "password": "Recovered123"}, ""), http.StatusBadRequest)
}

func TestProfileOnboardingAndValidation(t *testing.T) {
	env := newTestEnv(t)
	m := env.register("profile")

	var mine struct {
		Missing     []string `json:"missing"`
		IsComplete  bool     `json:"isComplete"`
		Age         *int     `json:"age"`
		HasLocation bool     `json:"hasLocation"`
		Interests   []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
	}
	r := env.do(http.MethodGet, "/profile", nil, m.Token)
	env.expect(r, http.StatusOK)
	r.decode(t, &mine)
	if mine.IsComplete || len(mine.Missing) != 6 {
		t.Fatalf("new profile must be incomplete: %+v", mine)
	}

	underage := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	cases := []struct {
		payload map[string]any
		code    string
	}{
		{map[string]any{"birthdate": underage}, "underage"},
		{map[string]any{"firstName": "<b>x</b>"}, "invalid_first_name"},
		{map[string]any{"gender": "robot"}, "invalid_gender"},
		{map[string]any{"bio": strings.Repeat("a", 501)}, "bio_too_long"},
		{map[string]any{"interests": []string{"not-a-real-interest"}}, "unknown_interest"},
		{map[string]any{"interests": []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}}, "too_many_interests"},
	}
	for _, c := range cases {
		r := env.do(http.MethodPatch, "/profile", c.payload, m.Token)
		if r.Status != http.StatusBadRequest || r.code(t) != c.code {
			t.Errorf("payload %v: expected %s, got %d %s", c.payload, c.code, r.Status, r.Body)
		}
	}
	if r := env.do(http.MethodPut, "/profile/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30, "maxDistanceKm": 10}, m.Token); r.code(t) != "invalid_age_range" {
		t.Errorf("expected invalid_age_range, got %s", r.Body)
	}
	if r := env.do(http.MethodPut, "/profile/location", map[string]any{"latitude": 123.0, "longitude": 2.0}, m.Token); r.code(t) != "invalid_coordinates" {
		t.Errorf("expected invalid_coordinates, got %s", r.Body)
	}

	env.onboard(m, persona{Name: "Inès", Gender: "woman", Birthdate: "1994-02-03", InterestedIn: []string{"man", "woman"}})
	r = env.do(http.MethodGet, "/profile", nil, m.Token)
	r.decode(t, &mine)
	if !mine.IsComplete || len(mine.Missing) != 0 || !mine.HasLocation || mine.Age == nil || len(mine.Interests) != 2 {
		t.Fatalf("profile should be complete: %s", r.Body)
	}
	if strings.Contains(string(r.Body), "latitude") || strings.Contains(string(r.Body), "longitude") {
		t.Fatal("coordinates must never be returned, even to the owner")
	}

	var stored struct{ Lat, Lon float64 }
	if err := env.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, m.ID).Scan(&stored.Lat, &stored.Lon); err != nil {
		t.Fatal(err)
	}
	if stored.Lat*100 != float64(int64(stored.Lat*100)) {
		t.Fatalf("stored latitude must be snapped to 2 decimals, got %v", stored.Lat)
	}

	// Clearing the location removes it entirely.
	env.expect(env.do(http.MethodDelete, "/profile/location", nil, m.Token), http.StatusOK)
	var lat *float64
	_ = env.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, m.ID).Scan(&lat)
	if lat != nil {
		t.Fatal("location must be cleared")
	}

	r = env.do(http.MethodGet, "/interests", nil, m.Token)
	env.expect(r, http.StatusOK)
	if !strings.Contains(string(r.Body), `"hiking"`) {
		t.Fatal("interest catalog must be seeded")
	}
}

func TestPhotosLifecycle(t *testing.T) {
	env := newTestEnv(t)
	m := env.register("photos")
	other := env.register("photos_other")

	type photosRes struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
			URL      string `json:"url"`
		} `json:"photos"`
	}

	bad := env.upload(http.MethodPost, "/profile/photos", []byte("<svg onload=alert(1)>"), m.Token)
	env.expect(bad, http.StatusUnsupportedMediaType)

	var res photosRes
	for i := 0; i < 6; i++ {
		r := env.upload(http.MethodPost, "/profile/photos", testImage(t, 300, 300, color.RGBA{uint8(i * 40), 50, 50, 255}), m.Token)
		env.expect(r, http.StatusCreated)
		r.decode(t, &res)
	}
	if len(res.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(res.Photos))
	}
	env.expect(env.upload(http.MethodPost, "/profile/photos", testImage(t, 300, 300, color.RGBA{1, 2, 3, 255}), m.Token), http.StatusConflict)

	// Reorder: reverse.
	ids := make([]string, 0, 6)
	for i := len(res.Photos) - 1; i >= 0; i-- {
		ids = append(ids, res.Photos[i].ID)
	}
	r := env.do(http.MethodPut, "/profile/photos/order", map[string]any{"photoIds": ids}, m.Token)
	env.expect(r, http.StatusOK)
	r.decode(t, &res)
	if res.Photos[0].ID != ids[0] || res.Photos[5].ID != ids[5] {
		t.Fatal("reorder did not apply")
	}
	env.expect(env.do(http.MethodPut, "/profile/photos/order", map[string]any{"photoIds": ids[:5]}, m.Token), http.StatusBadRequest)

	// Someone else cannot delete or replace my photo.
	env.expect(env.do(http.MethodDelete, "/profile/photos/"+ids[0], nil, other.Token), http.StatusNotFound)
	env.expect(env.upload(http.MethodPut, "/profile/photos/"+ids[0], testImage(t, 300, 300, color.RGBA{9, 9, 9, 255}), other.Token), http.StatusNotFound)

	// Delete compacts positions.
	r = env.do(http.MethodDelete, "/profile/photos/"+ids[2], nil, m.Token)
	env.expect(r, http.StatusOK)
	r.decode(t, &res)
	for i, p := range res.Photos {
		if p.Position != i {
			t.Fatalf("positions not compacted: %+v", res.Photos)
		}
	}

	// Replace keeps the position and removes the old file.
	var oldKey string
	_ = env.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[0]).Scan(&oldKey)
	r = env.upload(http.MethodPut, "/profile/photos/"+ids[0], testImage(t, 500, 400, color.RGBA{9, 9, 9, 255}), m.Token)
	env.expect(r, http.StatusOK)
	if _, err := os.Stat(env.uploads + "/" + oldKey); !os.IsNotExist(err) {
		t.Fatal("old file must be deleted after replacement")
	}

	// Serving requires a valid signature.
	r.decode(t, &res)
	served := env.do(http.MethodGet, res.Photos[0].URL, nil, "")
	env.expect(served, http.StatusOK)
	if served.Header.Get("Content-Type") != "image/jpeg" || len(served.Body) < 100 {
		t.Fatalf("unexpected media response %v", served.Header)
	}
	u, _ := url.Parse(res.Photos[0].URL)
	q := u.Query()
	q.Set("sig", "forged")
	env.expect(env.do(http.MethodGet, u.Path+"?"+q.Encode(), nil, ""), http.StatusNotFound)
	env.expect(env.do(http.MethodGet, u.Path, nil, ""), http.StatusNotFound)
}

func TestDiscoveryRespectsMutualPreferences(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno, carl, dana, eve, far := env.register("alice"), env.register("bruno"), env.register("carl"), env.register("dana"), env.register("eve"), env.register("far")

	env.onboard(alice, persona{Name: "Alice", Gender: "woman", Birthdate: "1995-04-10", InterestedIn: []string{"man"}, MinAge: 25, MaxAge: 40, Interests: []string{"hiking", "wine"}})
	env.onboard(bruno, persona{Name: "Bruno", Gender: "man", Birthdate: "1993-08-21", InterestedIn: []string{"woman"}, LatOffset: 0.1, Interests: []string{"hiking", "wine"}})
	env.onboard(carl, persona{Name: "Carl", Gender: "man", Birthdate: "1992-01-01", InterestedIn: []string{"man"}})                 // not interested in women
	env.onboard(dana, persona{Name: "Dana", Gender: "woman", Birthdate: "1990-01-01", InterestedIn: []string{"man"}})               // woman: Alice not interested
	env.onboard(eve, persona{Name: "Eve", Gender: "man", Birthdate: "1960-01-01", InterestedIn: []string{"woman"}})                 // out of Alice's age range
	env.onboard(far, persona{Name: "Farid", Gender: "man", Birthdate: "1994-01-01", InterestedIn: []string{"woman"}, LatOffset: 3}) // ~330 km away

	deck := env.discover(alice)
	if !containsProfile(deck, bruno.ID) {
		t.Fatal("Bruno must be suggested to Alice")
	}
	for _, excluded := range []member{carl, dana, eve, far, alice} {
		if containsProfile(deck, excluded.ID) {
			t.Fatalf("user %s must not be suggested to Alice", excluded.Email)
		}
	}
	for _, p := range deck {
		if p.UserID == bruno.ID {
			if p.DistanceKm == nil || *p.DistanceKm < 10 || *p.DistanceKm > 13 {
				t.Fatalf("expected approximate distance ~11 km, got %v", p.DistanceKm)
			}
			if len(p.Photos) != 1 || !strings.HasPrefix(p.Photos[0].URL, "/media/photos/") {
				t.Fatal("card must expose signed photo URLs")
			}
		}
	}

	// Hiding the distance and pausing the profile are honored.
	env.expect(env.do(http.MethodPut, "/profile/settings", map[string]bool{"isDiscoverable": true, "hideDistance": true}, bruno.Token), http.StatusOK)
	for _, p := range env.discover(alice) {
		if p.UserID == bruno.ID && p.DistanceKm != nil {
			t.Fatal("distance must be hidden")
		}
	}
	env.expect(env.do(http.MethodPut, "/profile/settings", map[string]bool{"isDiscoverable": false, "hideDistance": true}, bruno.Token), http.StatusOK)
	if containsProfile(env.discover(alice), bruno.ID) {
		t.Fatal("paused profile must not be discoverable")
	}

	// Incomplete viewers are told to finish onboarding.
	newbie := env.register("newbie")
	r := env.do(http.MethodGet, "/discover", nil, newbie.Token)
	if r.Status != http.StatusConflict || r.code(t) != "profile_incomplete" {
		t.Fatalf("expected profile_incomplete, got %d %s", r.Status, r.Body)
	}
}

func TestLikesMatchesAndNoDuplicates(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno := env.register("alice"), env.register("bruno")
	carl := env.register("carl")
	env.onboard(alice, persona{Name: "Alice", Gender: "woman", Birthdate: "1995-04-10", InterestedIn: []string{"man"}})
	env.onboard(bruno, persona{Name: "Bruno", Gender: "man", Birthdate: "1993-08-21", InterestedIn: []string{"woman"}})
	env.onboard(carl, persona{Name: "Carl", Gender: "man", Birthdate: "1991-08-21", InterestedIn: []string{"woman"}})

	r, res := env.swipe(alice, bruno.ID, "like")
	env.expect(r, http.StatusOK)
	if res.Matched {
		t.Fatal("one-sided like must not match")
	}
	if r, _ := env.swipe(alice, bruno.ID, "like"); r.Status != http.StatusConflict || r.code(t) != "already_swiped" {
		t.Fatalf("repeated like must be rejected, got %d %s", r.Status, r.Body)
	}
	if r, _ := env.swipe(alice, bruno.ID, "pass"); r.Status != http.StatusConflict {
		t.Fatal("changing a decision must be rejected")
	}
	if containsProfile(env.discover(alice), bruno.ID) {
		t.Fatal("already swiped profile must not reappear")
	}
	if r, _ := env.swipe(alice, alice.ID, "like"); r.Status != http.StatusBadRequest {
		t.Fatal("self like must be rejected")
	}
	if r, _ := env.swipe(alice, "7b1d7f0e-4e43-4a3f-9a5a-1f1f1f1f1f1f", "like"); r.Status != http.StatusNotFound {
		t.Fatal("unknown target must be rejected")
	}
	if r, _ := env.swipe(alice, carl.ID, "superlike"); r.Status != http.StatusBadRequest {
		t.Fatal("unknown action must be rejected")
	}

	// Pass never matches, even if the other side liked.
	env.swipe(carl, alice.ID, "like")
	r, res = env.swipe(alice, carl.ID, "pass")
	env.expect(r, http.StatusOK)
	if res.Matched {
		t.Fatal("pass must not create a match")
	}

	r, res = env.swipe(bruno, alice.ID, "like")
	env.expect(r, http.StatusOK)
	if !res.Matched || res.Match == nil || res.Match.User.UserID != alice.ID || res.Match.User.FirstName != "Alice" {
		t.Fatalf("expected match with Alice, got %s", r.Body)
	}

	var count int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, alice.ID, bruno.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match, got %d", count)
	}

	for _, m := range []member{alice, bruno} {
		r := env.do(http.MethodGet, "/matches", nil, m.Token)
		env.expect(r, http.StatusOK)
		var list struct {
			Matches []struct {
				ConversationID string `json:"conversationId"`
				User           struct {
					UserID string `json:"userId"`
				} `json:"user"`
			} `json:"matches"`
		}
		r.decode(t, &list)
		if len(list.Matches) != 1 || list.Matches[0].ConversationID != res.Match.ConversationID {
			t.Fatalf("unexpected matches for %s: %s", m.Email, r.Body)
		}
	}
	if r := env.do(http.MethodGet, "/matches", nil, carl.Token); strings.Contains(string(r.Body), res.Match.ID) {
		t.Fatal("third party must not see the match")
	}

	// Both receive a match notification.
	for _, m := range []member{alice, bruno} {
		r := env.do(http.MethodGet, "/notifications", nil, m.Token)
		if !strings.Contains(string(r.Body), `"type":"match"`) {
			t.Fatalf("expected match notification for %s: %s", m.Email, r.Body)
		}
	}
}

func TestSimultaneousLikesCreateOneMatch(t *testing.T) {
	env := newTestEnv(t)
	for round := 0; round < 5; round++ {
		a, b := env.register("race_a"), env.register("race_b")
		env.onboard(a, persona{Name: "Anna", Gender: "woman", Birthdate: "1995-04-10", InterestedIn: []string{"man"}})
		env.onboard(b, persona{Name: "Basile", Gender: "man", Birthdate: "1993-08-21", InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]member{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to member) {
				defer wg.Done()
				_, results[i] = env.swipe(from, to.ID, "like")
			}(i, pair[0], pair[1])
		}
		wg.Wait()
		if results[0].Matched == results[1].Matched {
			t.Fatalf("exactly one of two concurrent likes must report the match: %+v", results)
		}
		var count int
		_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&count)
		if count != 1 {
			t.Fatalf("expected one match, got %d", count)
		}
	}
}

type messagesRes struct {
	Messages []struct {
		ID           string `json:"id"`
		SenderUserID string `json:"senderUserId"`
		Content      string `json:"content"`
	} `json:"messages"`
	HasMore         bool       `json:"hasMore"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
}

func TestChatPermissionsReadStateAndPagination(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno, _, conversationID := env.pair("chat_a", "chat_b")
	intruder := env.register("intruder")
	path := "/conversations/" + conversationID

	env.expect(env.do(http.MethodPost, path+"/messages", map[string]string{"content": "   "}, alice.Token), http.StatusBadRequest)
	env.expect(env.do(http.MethodPost, path+"/messages", map[string]string{"content": strings.Repeat("x", 2001)}, alice.Token), http.StatusBadRequest)

	for i := 0; i < 35; i++ {
		env.expect(env.do(http.MethodPost, path+"/messages", map[string]string{"content": "message " + string(rune('A'+i%26))}, alice.Token), http.StatusCreated)
	}

	// Outsiders get 404 on read, write and read-receipt.
	env.expect(env.do(http.MethodGet, path+"/messages", nil, intruder.Token), http.StatusNotFound)
	env.expect(env.do(http.MethodPost, path+"/messages", map[string]string{"content": "hi"}, intruder.Token), http.StatusNotFound)
	env.expect(env.do(http.MethodPost, path+"/read", nil, intruder.Token), http.StatusNotFound)

	var page messagesRes
	r := env.do(http.MethodGet, path+"/messages?limit=30", nil, bruno.Token)
	env.expect(r, http.StatusOK)
	r.decode(t, &page)
	if len(page.Messages) != 30 || !page.HasMore {
		t.Fatalf("expected first page of 30 with more, got %d %v", len(page.Messages), page.HasMore)
	}
	last := page.Messages[len(page.Messages)-1].ID
	r = env.do(http.MethodGet, path+"/messages?limit=30&before="+last, nil, bruno.Token)
	r.decode(t, &page)
	if len(page.Messages) != 5 || page.HasMore {
		t.Fatalf("expected last page of 5, got %d", len(page.Messages))
	}

	// Unread count and collapsed message notification for Bruno.
	var list struct {
		Matches []struct {
			UnreadCount int `json:"unreadCount"`
			LastMessage *struct {
				Content string `json:"content"`
			} `json:"lastMessage"`
		} `json:"matches"`
	}
	env.do(http.MethodGet, "/matches", nil, bruno.Token).decode(t, &list)
	if len(list.Matches) != 1 || list.Matches[0].UnreadCount != 35 || list.Matches[0].LastMessage == nil {
		t.Fatalf("unexpected unread state %+v", list)
	}
	var notifs struct {
		Notifications []struct {
			Type string `json:"type"`
		} `json:"notifications"`
	}
	env.do(http.MethodGet, "/notifications", nil, bruno.Token).decode(t, &notifs)
	messageNotifs := 0
	for _, n := range notifs.Notifications {
		if n.Type == "message" {
			messageNotifs++
		}
	}
	if messageNotifs != 1 {
		t.Fatalf("message notifications must collapse to 1, got %d", messageNotifs)
	}

	env.expect(env.do(http.MethodPost, path+"/read", nil, bruno.Token), http.StatusOK)
	env.do(http.MethodGet, "/matches", nil, bruno.Token).decode(t, &list)
	if list.Matches[0].UnreadCount != 0 {
		t.Fatal("unread count must reset after read")
	}
	env.do(http.MethodGet, path+"/messages", nil, alice.Token).decode(t, &page)
	if page.OtherLastReadAt == nil {
		t.Fatal("sender must see the read receipt")
	}
	var unread struct {
		UnreadCount int `json:"unreadCount"`
	}
	env.do(http.MethodGet, "/notifications/unread-count", nil, bruno.Token).decode(t, &unread)
	if unread.UnreadCount != 1 { // the match notification remains, message one is cleared
		t.Fatalf("expected 1 unread notification, got %d", unread.UnreadCount)
	}
	env.expect(env.do(http.MethodPost, "/notifications/read-all", nil, bruno.Token), http.StatusNoContent)
	env.do(http.MethodGet, "/notifications/unread-count", nil, bruno.Token).decode(t, &unread)
	if unread.UnreadCount != 0 {
		t.Fatal("read-all must clear notifications")
	}
}

func TestRealtimeDelivery(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno, _, conversationID := env.pair("rt_a", "rt_b")
	server := httptest.NewServer(env.router)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime"

	// Wrong token is rejected.
	bad, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","token":"nope"}`))
	if _, _, err := bad.Read(ctx); err == nil {
		t.Fatal("invalid token must close the socket")
	}

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	auth, _ := json.Marshal(map[string]string{"type": "auth", "token": bruno.Token})
	if err := conn.Write(ctx, websocket.MessageText, auth); err != nil {
		t.Fatal(err)
	}
	readEvent := func() map[string]any {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read event: %v", err)
		}
		var ev map[string]any
		_ = json.Unmarshal(data, &ev)
		return ev
	}
	if ev := readEvent(); ev["type"] != "ready" {
		t.Fatalf("expected ready event, got %v", ev)
	}

	env.expect(env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"content": "Coucou en direct"}, alice.Token), http.StatusCreated)
	seenMessage := false
	for i := 0; i < 3 && !seenMessage; i++ {
		ev := readEvent()
		if ev["type"] == "message.created" {
			data := ev["data"].(map[string]any)
			if data["content"] != "Coucou en direct" || data["conversationId"] != conversationID {
				t.Fatalf("unexpected message payload %v", data)
			}
			seenMessage = true
		}
	}
	if !seenMessage {
		t.Fatal("bruno must receive the message in realtime")
	}
}

func TestBlockingDissolvesMatchAndHidesEveryone(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno, matchID, conversationID := env.pair("block_a", "block_b")
	env.expect(env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"content": "salut"}, bruno.Token), http.StatusCreated)

	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": alice.ID}, alice.Token), http.StatusBadRequest)
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": bruno.ID}, alice.Token), http.StatusNoContent)
	env.expect(env.do(http.MethodPost, "/blocks", map[string]string{"userId": bruno.ID}, alice.Token), http.StatusNoContent) // idempotent

	env.expect(env.do(http.MethodGet, "/conversations/"+conversationID+"/messages", nil, bruno.Token), http.StatusNotFound)
	env.expect(env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"content": "tu es là ?"}, bruno.Token), http.StatusNotFound)
	env.expect(env.do(http.MethodGet, "/profiles/"+alice.ID, nil, bruno.Token), http.StatusNotFound)
	env.expect(env.do(http.MethodGet, "/profiles/"+bruno.ID, nil, alice.Token), http.StatusNotFound)
	if strings.Contains(string(env.do(http.MethodGet, "/matches", nil, bruno.Token).Body), matchID) {
		t.Fatal("match must be removed")
	}
	var messages int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`, conversationID).Scan(&messages)
	if messages != 0 {
		t.Fatal("messages must be deleted with the conversation")
	}

	var blocks struct {
		Blocks []struct {
			UserID string `json:"userId"`
		} `json:"blocks"`
	}
	env.do(http.MethodGet, "/blocks", nil, alice.Token).decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != bruno.ID {
		t.Fatalf("unexpected blocks %+v", blocks)
	}
	env.expect(env.do(http.MethodDelete, "/blocks/"+bruno.ID, nil, alice.Token), http.StatusNoContent)
	if strings.Contains(string(env.do(http.MethodGet, "/matches", nil, alice.Token).Body), matchID) {
		t.Fatal("unblocking must not restore a dissolved match")
	}
}

func TestUnmatch(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno, matchID, conversationID := env.pair("unmatch_a", "unmatch_b")
	intruder := env.register("unmatch_x")
	env.expect(env.do(http.MethodDelete, "/matches/"+matchID, nil, intruder.Token), http.StatusNotFound)
	env.expect(env.do(http.MethodDelete, "/matches/"+matchID, nil, bruno.Token), http.StatusNoContent)
	env.expect(env.do(http.MethodGet, "/conversations/"+conversationID+"/messages", nil, alice.Token), http.StatusNotFound)
	if containsProfile(env.discover(alice), bruno.ID) {
		t.Fatal("unmatched people must not be suggested again")
	}
}

func TestReportsBlockAndSuspendAfterThreshold(t *testing.T) {
	env := newTestEnv(t)
	target := env.register("reported")
	env.onboard(target, persona{Name: "Target", Gender: "man", Birthdate: "1990-01-01", InterestedIn: []string{"woman"}})
	observer := env.register("observer")
	env.onboard(observer, persona{Name: "Olga", Gender: "woman", Birthdate: "1990-01-01", InterestedIn: []string{"man"}})
	if !containsProfile(env.discover(observer), target.ID) {
		t.Fatal("target should be discoverable before reports")
	}

	r := env.do(http.MethodPost, "/reports", map[string]string{"userId": target.ID, "reason": "spam"}, observer.Token)
	if r.Status != http.StatusBadRequest || r.code(t) != "invalid_reason" {
		t.Fatalf("expected invalid_reason, got %s", r.Body)
	}
	env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": observer.ID, "reason": "scam"}, observer.Token), http.StatusBadRequest)

	for i := 0; i < 3; i++ {
		reporter := env.register("reporter")
		env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": target.ID, "reason": "fake_profile", "details": "Photos volées"}, reporter.Token), http.StatusCreated)
		// Reporting twice does not count twice.
		env.expect(env.do(http.MethodPost, "/reports", map[string]string{"userId": target.ID, "reason": "scam"}, reporter.Token), http.StatusCreated)
		var blocked bool
		_ = env.pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM blocks WHERE blocker_id = $1 AND blocked_id = $2)`, reporter.ID, target.ID).Scan(&blocked)
		if !blocked {
			t.Fatal("reporting must block the reported user")
		}
	}
	var suspended bool
	_ = env.pool.QueryRow(context.Background(), `SELECT suspended_at IS NOT NULL FROM users WHERE id = $1`, target.ID).Scan(&suspended)
	if !suspended {
		t.Fatal("account must be suspended after 3 distinct reports")
	}
	if containsProfile(env.discover(observer), target.ID) {
		t.Fatal("suspended account must not be discoverable")
	}
}

func TestAccountDeletionRemovesData(t *testing.T) {
	env := newTestEnv(t)
	alice, bruno, matchID, conversationID := env.pair("delete_a", "delete_b")
	env.expect(env.do(http.MethodPost, "/conversations/"+conversationID+"/messages", map[string]string{"content": "bye"}, alice.Token), http.StatusCreated)
	var keys []string
	rows, _ := env.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, alice.ID)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) == 0 {
		t.Fatal("expected photos to exist")
	}

	env.expect(env.do(http.MethodDelete, "/me", map[string]string{"password": "WrongPass1"}, alice.Token), http.StatusForbidden)
	env.expect(env.do(http.MethodDelete, "/me", map[string]string{"password": testPassword}, alice.Token), http.StatusNoContent)

	env.expect(env.do(http.MethodGet, "/me", nil, alice.Token), http.StatusUnauthorized)
	env.expect(env.do(http.MethodPost, "/auth/login", map[string]string{"email": alice.Email, "password": testPassword}, ""), http.StatusUnauthorized)
	for _, k := range keys {
		if _, err := os.Stat(env.uploads + "/" + k); !os.IsNotExist(err) {
			t.Fatalf("photo file %s must be deleted", k)
		}
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
		`SELECT COUNT(*) FROM photos WHERE user_id = $1`,
		`SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1`,
		`SELECT COUNT(*) FROM messages WHERE sender_user_id = $1 OR recipient_user_id = $1`,
		`SELECT COUNT(*) FROM sessions WHERE user_id = $1`,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`,
	} {
		var n int
		if err := env.pool.QueryRow(context.Background(), q, alice.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: expected 0 rows, got %d (%v)", q, n, err)
		}
	}
	if strings.Contains(string(env.do(http.MethodGet, "/matches", nil, bruno.Token).Body), matchID) {
		t.Fatal("the match must disappear for the other person")
	}
}
