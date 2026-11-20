package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHealth(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	res := e.do(http.MethodGet, "/health", "", nil)
	e.expect(res, http.StatusOK, "health")
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	email := uniqueEmail("auth")

	res := e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": testPassword})
	e.expect(res, http.StatusCreated, "register")
	var s session
	res.decode(t, &s)
	if s.User.Email != email || s.User.ProfileComplete {
		t.Fatalf("unexpected registered user %+v", s.User)
	}

	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": email, "password": testPassword}), http.StatusConflict, "duplicate")
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": uniqueEmail("weak"), "password": "onlyletters"}), http.StatusBadRequest, "weak password")
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "not-an-email", "password": testPassword}), http.StatusBadRequest, "bad email")
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": "Wrong12345"}), http.StatusUnauthorized, "wrong password")
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": uniqueEmail("ghost"), "password": testPassword}), http.StatusUnauthorized, "unknown email")

	res = e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": email, "password": testPassword})
	e.expect(res, http.StatusOK, "login")
	var login session
	res.decode(t, &login)

	e.expect(e.do(http.MethodGet, "/me", login.AccessToken, nil), http.StatusOK, "me")
	e.expect(e.do(http.MethodGet, "/me", "", nil), http.StatusUnauthorized, "me without token")
	e.expect(e.do(http.MethodGet, "/me", "garbage", nil), http.StatusUnauthorized, "me with garbage token")

	res = e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken})
	e.expect(res, http.StatusOK, "refresh")
	var refreshed session
	res.decode(t, &refreshed)
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), http.StatusUnauthorized, "reused refresh token")

	// Logging out revokes the session server-side: the access token dies immediately.
	e.expect(e.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}), http.StatusNoContent, "logout")
	e.expect(e.do(http.MethodGet, "/me", refreshed.AccessToken, nil), http.StatusUnauthorized, "me after logout")
	e.expect(e.do(http.MethodGet, "/me", s.AccessToken, nil), http.StatusOK, "other session still valid")

	// A realtime ticket is not an access token.
	res = e.do(http.MethodPost, "/realtime/ticket", s.AccessToken, nil)
	e.expect(res, http.StatusOK, "ticket")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	res.decode(t, &ticket)
	e.expect(e.do(http.MethodGet, "/me", ticket.Ticket, nil), http.StatusUnauthorized, "ticket used as bearer")
}

func TestPasswordResetAndChange(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	s := e.register("reset")

	e.expect(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": uniqueEmail("nobody")}), http.StatusAccepted, "forgot unknown")
	e.expect(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": s.User.Email}), http.StatusAccepted, "forgot")
	code := e.mail.lastCodeFor(s.User.Email)
	if code == "" {
		t.Fatal("expected a reset code e-mail")
	}

	bad := map[string]string{"email": s.User.Email, "code": "AAAAAAAAAA", "password": "NewPassword9"}
	e.expect(e.do(http.MethodPost, "/auth/password/reset", "", bad), http.StatusBadRequest, "wrong code")

	good := map[string]string{"email": s.User.Email, "code": strings.ToLower(code[:5]) + "-" + code[5:], "password": "NewPassword9"}
	res := e.do(http.MethodPost, "/auth/password/reset", "", good)
	e.expect(res, http.StatusOK, "reset")
	var afterReset session
	res.decode(t, &afterReset)

	e.expect(e.do(http.MethodGet, "/me", s.AccessToken, nil), http.StatusUnauthorized, "old session revoked by reset")
	e.expect(e.do(http.MethodPost, "/auth/password/reset", "", good), http.StatusBadRequest, "code is single use")
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": s.User.Email, "password": "NewPassword9"}), http.StatusOK, "login new password")

	e.expect(e.do(http.MethodPost, "/me/password", afterReset.AccessToken, map[string]string{"currentPassword": "nope12345", "newPassword": "Another123"}), http.StatusForbidden, "change wrong current")
	e.expect(e.do(http.MethodPost, "/me/password", afterReset.AccessToken, map[string]string{"currentPassword": "NewPassword9", "newPassword": "Another123"}), http.StatusNoContent, "change password")
	e.expect(e.do(http.MethodGet, "/me", afterReset.AccessToken, nil), http.StatusOK, "current session kept after change")
}

func TestResetCodeLockedAfterTooManyAttempts(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	s := e.register("brute")
	e.expect(e.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": s.User.Email}), http.StatusAccepted, "forgot")
	code := e.mail.lastCodeFor(s.User.Email)
	for i := 0; i < 5; i++ {
		e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": s.User.Email, "code": "ZZZZZZZZZZ", "password": "NewPassword9"})
	}
	res := e.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"email": s.User.Email, "code": code, "password": "NewPassword9"})
	e.expect(res, http.StatusBadRequest, "valid code after lockout")
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	s := e.register("profile")

	e.expect(e.do(http.MethodGet, "/profile", s.AccessToken, nil), http.StatusNotFound, "profile before creation")
	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	res := e.do(http.MethodPut, "/profile", s.AccessToken, map[string]any{"firstName": "Lou", "birthdate": minor, "gender": "woman"})
	e.expect(res, http.StatusUnprocessableEntity, "underage")
	if res.code(t) != "underage" {
		t.Fatalf("expected underage code, got %s", res.Body)
	}
	e.expect(e.do(http.MethodPut, "/profile", s.AccessToken, map[string]any{"firstName": "Lou", "birthdate": "1990-01-01", "gender": "alien"}), http.StatusBadRequest, "bad gender")
	e.expect(e.do(http.MethodPut, "/profile", s.AccessToken, map[string]any{"firstName": "Lou", "birthdate": "1990-01-01", "gender": "woman", "interestIds": []int{9999}}), http.StatusBadRequest, "unknown interest")

	res = e.do(http.MethodGet, "/interests", s.AccessToken, nil)
	e.expect(res, http.StatusOK, "interests")
	var catalog struct {
		Interests []struct {
			ID int `json:"id"`
		} `json:"interests"`
	}
	res.decode(t, &catalog)
	if len(catalog.Interests) < 20 {
		t.Fatalf("expected seeded interests, got %d", len(catalog.Interests))
	}

	res = e.do(http.MethodPut, "/profile", s.AccessToken, map[string]any{
		"firstName": "  Lou  ", "birthdate": "1990-01-01", "gender": "woman", "bio": "Hi", "interestIds": []int{catalog.Interests[0].ID, catalog.Interests[1].ID},
	})
	e.expect(res, http.StatusOK, "create profile")
	var profile struct {
		FirstName string `json:"firstName"`
		Interests []any  `json:"interests"`
	}
	res.decode(t, &profile)
	if profile.FirstName != "Lou" || len(profile.Interests) != 2 {
		t.Fatalf("unexpected profile %s", res.Body)
	}
	e.expect(e.do(http.MethodPut, "/profile", s.AccessToken, map[string]any{"firstName": "Lou", "birthdate": "1985-01-01", "gender": "woman"}), http.StatusUnprocessableEntity, "birthdate locked")

	e.expect(e.do(http.MethodPut, "/profile/location", s.AccessToken, map[string]any{"latitude": 95, "longitude": 2}), http.StatusBadRequest, "invalid latitude")
	e.expect(e.do(http.MethodPut, "/profile/location", s.AccessToken, map[string]any{"latitude": 48.856613, "longitude": 2.352222}), http.StatusNoContent, "location")
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT location_lat, location_lng FROM profiles WHERE user_id = $1`, s.User.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("exact coordinates must never be stored, got %v,%v", lat, lng)
	}

	e.expect(e.do(http.MethodPut, "/profile/preferences", s.AccessToken, map[string]any{"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 20, "maxDistanceKm": 10}), http.StatusBadRequest, "inverted ages")

	var me session
	res = e.do(http.MethodGet, "/me", s.AccessToken, nil)
	res.decode(t, &me.User)
	if me.User.ProfileComplete {
		t.Fatal("profile without preferences and photo must be incomplete")
	}
}

func TestPhotosLifecycle(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	s := e.register("photos")

	e.expect(e.upload(http.MethodPost, "/profile/photos", s.AccessToken, []byte("#!/bin/sh\nrm -rf /")), http.StatusUnsupportedMediaType, "script upload")
	e.expect(e.upload(http.MethodPost, "/profile/photos", s.AccessToken, pngBytes(t, 50, 50, 1)), http.StatusUnprocessableEntity, "tiny image")

	ids := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		res := e.upload(http.MethodPost, "/profile/photos", s.AccessToken, pngBytes(t, 400, 500, uint8(i*30)))
		e.expect(res, http.StatusCreated, "upload")
		var p struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			Position int    `json:"position"`
		}
		res.decode(t, &p)
		if p.Position != i {
			t.Fatalf("expected position %d, got %d", i, p.Position)
		}
		ids = append(ids, p.ID)
	}
	e.expect(e.upload(http.MethodPost, "/profile/photos", s.AccessToken, pngBytes(t, 400, 500, 1)), http.StatusConflict, "seventh photo")

	reversed := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	res := e.do(http.MethodPut, "/profile/photos/order", s.AccessToken, map[string]any{"photoIds": reversed})
	e.expect(res, http.StatusOK, "reorder")
	var list struct {
		Photos []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"photos"`
	}
	res.decode(t, &list)
	if list.Photos[0].ID != ids[5] {
		t.Fatal("reorder must change the primary photo")
	}
	e.expect(e.do(http.MethodPut, "/profile/photos/order", s.AccessToken, map[string]any{"photoIds": ids[:2]}), http.StatusBadRequest, "partial order")

	// Signed media URL works without a bearer token; tampering fails.
	media := e.do(http.MethodGet, list.Photos[0].URL, "", nil)
	e.expect(media, http.StatusOK, "media")
	if !strings.HasPrefix(http.DetectContentType(media.Body), "image/jpeg") {
		t.Fatal("photos must be served as re-encoded JPEG")
	}
	e.expect(e.do(http.MethodGet, strings.Replace(list.Photos[0].URL, "sig=", "sig=00", 1), "", nil), http.StatusForbidden, "tampered signature")
	e.expect(e.do(http.MethodGet, "/media/photos/"+ids[0], "", nil), http.StatusForbidden, "unsigned media")

	res = e.upload(http.MethodPut, "/profile/photos/"+ids[5], s.AccessToken, pngBytes(t, 600, 600, 200))
	e.expect(res, http.StatusOK, "replace")
	var replaced struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
	}
	res.decode(t, &replaced)
	if replaced.ID == ids[5] || replaced.Position != 0 {
		t.Fatalf("replace must keep position and rotate id: %s", res.Body)
	}
	e.expect(e.do(http.MethodGet, list.Photos[0].URL, "", nil), http.StatusNotFound, "old media after replace")

	other := e.register("intruder")
	e.expect(e.do(http.MethodDelete, "/profile/photos/"+ids[0], other.AccessToken, nil), http.StatusNotFound, "delete someone else's photo")

	for _, id := range ids[:5] {
		e.expect(e.do(http.MethodDelete, "/profile/photos/"+id, s.AccessToken, nil), http.StatusNoContent, "delete")
	}
	e.expect(e.do(http.MethodDelete, "/profile/photos/"+replaced.ID, s.AccessToken, nil), http.StatusConflict, "delete last photo")
}

func TestDiscoveryRules(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	parisLat, parisLng := coords(48.8566, 2.3522)
	nearLat, nearLng := coords(48.80, 2.30)
	lyonLat, lyonLng := coords(45.764, 4.8357)

	alice := e.onboard(profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng, Interests: []int{1, 2, 3}})
	bob := e.onboard(profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Lat: nearLat, Lng: nearLng, Interests: []int{1, 2}})
	far := e.onboard(profileSpec{Name: "Farid", Gender: "man", InterestedIn: []string{"woman"}, Lat: lyonLat, Lng: lyonLng})
	notInterested := e.onboard(profileSpec{Name: "Marc", Gender: "man", InterestedIn: []string{"man"}, Lat: nearLat, Lng: nearLng})
	woman := e.onboard(profileSpec{Name: "Eve", Gender: "woman", InterestedIn: []string{"man"}, Lat: nearLat, Lng: nearLng})
	young := e.onboard(profileSpec{Name: "Hugo", Gender: "man", InterestedIn: []string{"woman"}, Lat: nearLat, Lng: nearLng, Birthdate: time.Now().AddDate(-19, 0, 0).Format("2006-01-02")})
	_ = e.register("noprofile")

	// Alice only accepts 25-60 year olds.
	e.expect(e.do(http.MethodPut, "/profile/preferences", alice.AccessToken, map[string]any{"interestedIn": []string{"man"}, "ageMin": 25, "ageMax": 60, "maxDistanceKm": 50}), http.StatusOK, "alice prefs")

	cards := e.discover(alice.AccessToken)
	if !containsCard(cards, bob.User.ID) {
		t.Fatal("Bob matches every criterion and must be suggested")
	}
	for _, excluded := range []member{far, notInterested, woman, young} {
		if containsCard(cards, excluded.User.ID) {
			t.Fatalf("%s must not be suggested to Alice", excluded.Name)
		}
	}
	for _, c := range cards {
		if c.UserID == bob.User.ID {
			if c.DistanceKm == nil || *c.DistanceKm < 1 || *c.DistanceKm > 15 {
				t.Fatalf("expected an approximate distance for Bob, got %v", c.DistanceKm)
			}
			if len(c.Photos) != 1 || !strings.HasPrefix(c.Photos[0].URL, "/media/photos/") {
				t.Fatal("cards must expose signed photo URLs")
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
	var raw map[string]any
	res := e.do(http.MethodGet, "/profiles/"+bob.User.ID, alice.AccessToken, nil)
	e.expect(res, http.StatusOK, "view profile")
	res.decode(t, &raw)
	for _, forbidden := range []string{"birthdate", "email", "latitude", "longitude", "locationLat"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("public profile leaks %q", forbidden)
		}
	}

	// Likes are only possible between mutually compatible members.
	resp, _ := e.swipe(alice.AccessToken, woman.User.ID, "like")
	e.expect(resp, http.StatusNotFound, "like outside Alice's preferences")
	resp, _ = e.swipe(notInterested.AccessToken, alice.User.ID, "like")
	e.expect(resp, http.StatusNotFound, "like from someone Alice is not compatible with")
	resp, _ = e.swipe(young.AccessToken, alice.User.ID, "like")
	e.expect(resp, http.StatusNotFound, "like outside Alice's age range")

	// A swiped profile never comes back.
	resp, _ = e.swipe(alice.AccessToken, bob.User.ID, "pass")
	e.expect(resp, http.StatusOK, "pass")
	if containsCard(e.discover(alice.AccessToken), bob.User.ID) {
		t.Fatal("passed profile must not reappear")
	}
	resp, _ = e.swipe(alice.AccessToken, bob.User.ID, "like")
	e.expect(resp, http.StatusConflict, "second decision")

	// Pausing the profile hides it.
	e.expect(e.do(http.MethodPut, "/profile/visibility", woman.AccessToken, map[string]any{"discoverable": false}), http.StatusNoContent, "pause")
	if containsCard(e.discover(bob.AccessToken), woman.User.ID) {
		t.Fatal("paused profile must be hidden")
	}

	incomplete := e.register("incomplete")
	res = e.do(http.MethodGet, "/discovery", incomplete.AccessToken, nil)
	e.expect(res, http.StatusConflict, "discovery without profile")
	e.expect(e.do(http.MethodGet, "/discovery?limit=abc", alice.AccessToken, nil), http.StatusBadRequest, "invalid limit")
	e.expect(e.do(http.MethodGet, "/discovery?limit=1000", alice.AccessToken, nil), http.StatusBadRequest, "limit too large")
}

func TestMatchChatAndPermissions(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	alice := e.onboard(profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.onboard(profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	mallory := e.onboard(profileSpec{Name: "Mallory", Gender: "man", InterestedIn: []string{"woman"}})

	res, _ := e.swipe(alice.AccessToken, alice.User.ID, "like")
	e.expect(res, http.StatusBadRequest, "self like")
	res, _ = e.swipe(alice.AccessToken, "00000000-0000-0000-0000-000000000000", "like")
	e.expect(res, http.StatusNotFound, "like unknown user")
	res, _ = e.swipe(alice.AccessToken, bob.User.ID, "superlike")
	e.expect(res, http.StatusBadRequest, "unknown action")

	// Bob sees Alice's like for free.
	e.swipe(alice.AccessToken, bob.User.ID, "like")
	res = e.do(http.MethodGet, "/likes/received", bob.AccessToken, nil)
	e.expect(res, http.StatusOK, "likes received")
	if !strings.Contains(string(res.Body), alice.User.ID) {
		t.Fatal("Bob must see that Alice liked him")
	}
	res, result := e.swipe(bob.AccessToken, alice.User.ID, "like")
	e.expect(res, http.StatusOK, "match like")
	if !result.Matched {
		t.Fatal("expected a match")
	}
	conversationID, matchID := *result.ConversationID, *result.MatchID

	var matches int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches`).Scan(&matches)
	if matches != 1 {
		t.Fatalf("expected exactly one match, got %d", matches)
	}

	res = e.do(http.MethodGet, "/conversations?filter=new", alice.AccessToken, nil)
	e.expect(res, http.StatusOK, "new matches")
	if !strings.Contains(string(res.Body), conversationID) {
		t.Fatal("new match must be listed")
	}

	path := "/conversations/" + conversationID + "/messages"
	e.expect(e.do(http.MethodPost, path, alice.AccessToken, map[string]string{"body": "   "}), http.StatusBadRequest, "empty message")
	e.expect(e.do(http.MethodPost, path, alice.AccessToken, map[string]string{"body": "Salut Bob !"}), http.StatusCreated, "send")
	e.expect(e.do(http.MethodPost, path, alice.AccessToken, map[string]string{"body": "Ça va ?"}), http.StatusCreated, "send 2")

	// Mallory is not a participant: every access is denied.
	e.expect(e.do(http.MethodGet, path, mallory.AccessToken, nil), http.StatusNotFound, "intruder read")
	e.expect(e.do(http.MethodPost, path, mallory.AccessToken, map[string]string{"body": "hi"}), http.StatusNotFound, "intruder send")
	e.expect(e.do(http.MethodPost, "/conversations/"+conversationID+"/read", mallory.AccessToken, nil), http.StatusNotFound, "intruder read receipt")
	e.expect(e.do(http.MethodDelete, "/matches/"+matchID, mallory.AccessToken, nil), http.StatusNotFound, "intruder unmatch")

	res = e.do(http.MethodGet, "/conversations?filter=active", bob.AccessToken, nil)
	e.expect(res, http.StatusOK, "bob conversations")
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
			User struct {
				FirstName string `json:"firstName"`
			} `json:"user"`
		} `json:"conversations"`
	}
	res.decode(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 2 || convs.Conversations[0].LastMessage.Body != "Ça va ?" || convs.Conversations[0].User.FirstName != "Alice" {
		t.Fatalf("unexpected conversations %s", res.Body)
	}

	res = e.do(http.MethodGet, "/notifications", bob.AccessToken, nil)
	var notifs struct {
		Notifications []struct {
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	res.decode(t, &notifs)
	if notifs.UnreadCount != 2 {
		t.Fatalf("expected 1 match + 1 collapsed message notification, got %s", res.Body)
	}

	res = e.do(http.MethodPost, "/conversations/"+conversationID+"/read", bob.AccessToken, nil)
	e.expect(res, http.StatusOK, "mark read")
	res = e.do(http.MethodGet, path, alice.AccessToken, nil)
	var msgs struct {
		Messages []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"messages"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	res.decode(t, &msgs)
	if len(msgs.Messages) != 2 || msgs.Messages[0].Body != "Ça va ?" || msgs.OtherLastReadAt == nil {
		t.Fatalf("expected newest-first messages with read receipt, got %s", res.Body)
	}
	res = e.do(http.MethodGet, path+"?limit=1&before="+msgs.Messages[0].ID, alice.AccessToken, nil)
	e.expect(res, http.StatusOK, "paginate")
	if !strings.Contains(string(res.Body), "Salut Bob") || strings.Contains(string(res.Body), "Ça va") {
		t.Fatalf("unexpected page %s", res.Body)
	}

	res = e.do(http.MethodGet, "/notifications", bob.AccessToken, nil)
	res.decode(t, &notifs)
	if notifs.UnreadCount != 1 {
		t.Fatalf("reading the conversation must clear its message notification: %s", res.Body)
	}
	e.expect(e.do(http.MethodPost, "/notifications/read-all", bob.AccessToken, nil), http.StatusNoContent, "read all")

	// Local deletion hides history for Bob only.
	e.expect(e.do(http.MethodDelete, "/conversations/"+conversationID, bob.AccessToken, nil), http.StatusNoContent, "clear")
	res = e.do(http.MethodGet, path, bob.AccessToken, nil)
	if strings.Contains(string(res.Body), "Salut") {
		t.Fatal("cleared history must be hidden for Bob")
	}
	res = e.do(http.MethodGet, path, alice.AccessToken, nil)
	if !strings.Contains(string(res.Body), "Salut") {
		t.Fatal("Alice keeps her history")
	}

	// Unmatch removes the conversation for both.
	e.expect(e.do(http.MethodDelete, "/matches/"+matchID, alice.AccessToken, nil), http.StatusNoContent, "unmatch")
	e.expect(e.do(http.MethodGet, path, bob.AccessToken, nil), http.StatusNotFound, "conversation after unmatch")
	if containsCard(e.discover(alice.AccessToken), bob.User.ID) {
		t.Fatal("unmatched profile must not reappear")
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	for round := 0; round < 5; round++ {
		a := e.onboard(profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}})
		b := e.onboard(profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]member{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to member) {
				defer wg.Done()
				_, results[i] = e.swipe(from.AccessToken, to.User.ID, "like")
			}(i, pair[0], pair[1])
		}
		wg.Wait()
		if results[0].Matched == results[1].Matched {
			t.Fatalf("exactly one of the two simultaneous likes must report the match: %+v", results)
		}
		var count int
		_ = e.pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		`, a.User.ID, b.User.ID).Scan(&count)
		if count != 1 {
			t.Fatalf("expected one match, got %d", count)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	alice := e.onboard(profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.onboard(profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	_, conversationID := e.match(alice, bob)

	e.expect(e.do(http.MethodPost, "/blocks", alice.AccessToken, map[string]string{"userId": alice.User.ID}), http.StatusBadRequest, "self block")
	e.expect(e.do(http.MethodPost, "/blocks", alice.AccessToken, map[string]string{"userId": bob.User.ID}), http.StatusNoContent, "block")
	e.expect(e.do(http.MethodPost, "/conversations/"+conversationID+"/messages", bob.AccessToken, map[string]string{"body": "hey"}), http.StatusNotFound, "message after block")
	e.expect(e.do(http.MethodGet, "/profiles/"+alice.User.ID, bob.AccessToken, nil), http.StatusNotFound, "view blocker")
	e.expect(e.do(http.MethodGet, "/profiles/"+bob.User.ID, alice.AccessToken, nil), http.StatusNotFound, "view blocked")

	res := e.do(http.MethodGet, "/blocks", alice.AccessToken, nil)
	if !strings.Contains(string(res.Body), bob.User.ID) {
		t.Fatal("blocked list must include Bob")
	}
	e.expect(e.do(http.MethodDelete, "/blocks/"+bob.User.ID, alice.AccessToken, nil), http.StatusNoContent, "unblock")

	// Reports: invalid reason rejected, valid report auto-blocks.
	carl := e.onboard(profileSpec{Name: "Carl", Gender: "man", InterestedIn: []string{"woman"}})
	e.expect(e.do(http.MethodPost, "/reports", alice.AccessToken, map[string]string{"userId": carl.User.ID, "reason": "ugly"}), http.StatusBadRequest, "bad reason")
	e.expect(e.do(http.MethodPost, "/reports", alice.AccessToken, map[string]string{"userId": carl.User.ID, "reason": "fake_profile", "details": "photos volées"}), http.StatusCreated, "report")
	if containsCard(e.discover(alice.AccessToken), carl.User.ID) {
		t.Fatal("a reported profile must disappear for the reporter")
	}

	// Three distinct reporters put the account under review: hidden for everyone.
	for i := 0; i < 2; i++ {
		reporter := e.onboard(profileSpec{Name: "Rita", Gender: "woman", InterestedIn: []string{"man"}})
		e.expect(e.do(http.MethodPost, "/reports", reporter.AccessToken, map[string]string{"userId": carl.User.ID, "reason": "spam"}), http.StatusCreated, "report")
	}
	witness := e.onboard(profileSpec{Name: "Wendy", Gender: "woman", InterestedIn: []string{"man"}})
	if containsCard(e.discover(witness.AccessToken), carl.User.ID) {
		t.Fatal("an account under review must be hidden from discovery")
	}
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	alice := e.onboard(profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.onboard(profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	_, conversationID := e.match(alice, bob)
	e.expect(e.do(http.MethodPost, "/conversations/"+conversationID+"/messages", alice.AccessToken, map[string]string{"body": "bye"}), http.StatusCreated, "send")

	var key string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, alice.User.ID).Scan(&key)

	e.expect(e.do(http.MethodDelete, "/me", alice.AccessToken, map[string]string{"password": "wrong-password1"}), http.StatusForbidden, "delete wrong password")
	e.expect(e.do(http.MethodDelete, "/me", alice.AccessToken, map[string]string{"password": testPassword}), http.StatusNoContent, "delete account")
	e.expect(e.do(http.MethodGet, "/me", alice.AccessToken, nil), http.StatusUnauthorized, "token after deletion")
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": alice.User.Email, "password": testPassword}), http.StatusUnauthorized, "login after deletion")

	for _, table := range []string{"profiles", "photos", "swipes", "messages", "notifications", "sessions"} {
		var count int
		column := "user_id"
		switch table {
		case "swipes":
			column = "swiper_id"
		case "messages":
			column = "sender_id"
		}
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM `+table+` WHERE `+column+` = $1`, alice.User.ID).Scan(&count)
		if count != 0 {
			t.Fatalf("%s still holds %d rows for the deleted account", table, count)
		}
	}
	e.expect(e.do(http.MethodGet, "/conversations/"+conversationID+"/messages", bob.AccessToken, nil), http.StatusNotFound, "conversation after deletion")
	if key == "" {
		t.Fatal("expected a stored photo key")
	}
	if _, err := os.Stat(filepath.Join(e.uploadDir, filepath.FromSlash(key))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("photo file must be deleted from storage, stat error: %v", err)
	}
}

func TestRealtimeDelivery(t *testing.T) {
	e := newEnv(t, relaxedLimits())
	alice := e.onboard(profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.onboard(profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})

	res := e.do(http.MethodPost, "/realtime/ticket", bob.AccessToken, nil)
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	res.decode(t, &ticket)
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/realtime?ticket="
	if _, _, err := websocket.DefaultDialer.Dial(wsURL+"forged", nil); err == nil {
		t.Fatal("forged ticket must be refused")
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+ticket.Ticket, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	next := func(eventType string) map[string]any {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("waiting for %s: %v", eventType, err)
			}
			var event struct {
				Type string         `json:"type"`
				Data map[string]any `json:"data"`
			}
			_ = json.Unmarshal(raw, &event)
			if event.Type == eventType {
				return event.Data
			}
		}
	}
	next("ready")

	_, conversationID := e.match(bob, alice)
	if data := next("match.created"); data["conversationId"] != conversationID {
		t.Fatalf("unexpected match event %v", data)
	}
	e.do(http.MethodPost, "/conversations/"+conversationID+"/messages", alice.AccessToken, map[string]string{"body": "Coucou en temps réel"})
	if data := next("message.created"); data["body"] != "Coucou en temps réel" {
		t.Fatalf("unexpected message event %v", data)
	}
	next("notification.created")

	// Deleting the account closes its sockets.
	e.expect(e.do(http.MethodDelete, "/me", bob.AccessToken, map[string]string{"password": testPassword}), http.StatusNoContent, "delete")
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	if e.app.Hub.ConnectionCount(bob.User.ID) != 0 {
		t.Fatal("connections must be closed after account deletion")
	}
}

func TestAuthRateLimit(t *testing.T) {
	limits := relaxedLimits()
	limits.Auth = rate{Count: 3, Window: time.Minute, Burst: 3}
	e := newEnv(t, limits)
	statuses := []int{}
	for i := 0; i < 5; i++ {
		res := e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "x@lueur.test", "password": "Password123"})
		statuses = append(statuses, res.Status)
	}
	if statuses[4] != http.StatusTooManyRequests {
		t.Fatalf("expected login to be rate limited, got %v", statuses)
	}
}
