package app

import (
	"context"
	"encoding/json"
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
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := e.email("Auth")

	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": testPassword}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": "weakpassword", "acceptedTerms": true}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": "not-an-email", "password": testPassword, "acceptedTerms": true}), 400)

	var reg struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": "  " + strings.ToUpper(email) + " ", "password": testPassword, "acceptedTerms": true}), 201).json(t, &reg)
	if reg.User.Email != strings.ToLower(email) {
		t.Fatalf("email not normalised: %q", reg.User.Email)
	}
	e.expect(e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": testPassword, "acceptedTerms": true}), 409)

	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	e.expect(e.do("GET", "/me", reg.AccessToken, nil), 200)

	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": email, "password": "WrongPass1"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": "nobody@integration.test", "password": testPassword}), 401)
	var login struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": email, "password": testPassword}), 200).json(t, &login)

	// Refresh tokens rotate; replaying a used one fails.
	var refreshed struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	e.expect(e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": login.RefreshToken}), 200).json(t, &refreshed)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": login.RefreshToken}), 401)

	// Logout revokes the session immediately, including its access token.
	e.expect(e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": refreshed.RefreshToken}), 204)
	e.expect(e.do("GET", "/me", refreshed.AccessToken, nil), 401)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": refreshed.RefreshToken}), 401)

	// Password change keeps the current session and revokes the others.
	e.expect(e.do("PUT", "/me/password", reg.AccessToken, map[string]any{"currentPassword": "WrongPass1", "newPassword": "NewPassword456"}), 403)
	var other struct {
		AccessToken string `json:"accessToken"`
	}
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": email, "password": testPassword}), 200).json(t, &other)
	e.expect(e.do("PUT", "/me/password", reg.AccessToken, map[string]any{"currentPassword": testPassword, "newPassword": "NewPassword456"}), 204)
	e.expect(e.do("GET", "/me", reg.AccessToken, nil), 200)
	e.expect(e.do("GET", "/me", other.AccessToken, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": email, "password": "NewPassword456"}), 200)
}

func TestPasswordReset(t *testing.T) {
	e := newEnv(t)
	s := e.register("reset")

	// Unknown addresses get the same answer (no account enumeration).
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]any{"email": "ghost@integration.test"}), 202)
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]any{"email": s.Email}), 202)
	code := e.mailer.code(s.Email)
	if code == "" {
		t.Fatal("expected a reset code to be mailed")
	}

	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": s.Email, "code": "WRONGCODE1", "password": "Brandnew789"}), 400)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": "other@integration.test", "code": code, "password": "Brandnew789"}), 400)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": s.Email, "code": strings.ToLower(code), "password": "Brandnew789"}), 204)
	// Single use, and every session is revoked.
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]any{"email": s.Email, "code": code, "password": "Another789x"}), 400)
	e.expect(e.do("GET", "/me", s.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": s.Email, "password": "Brandnew789"}), 200)
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t)
	s := e.register("profile")

	var mine struct {
		Completion struct {
			Profile, Preferences, Photos, Complete bool
		} `json:"completion"`
		Birthdate   string `json:"birthdate"`
		HasLocation bool   `json:"hasLocation"`
	}
	e.expect(e.do("GET", "/profile", s.Token, nil), 200).json(t, &mine)
	if mine.Completion.Complete || mine.Completion.Profile {
		t.Fatal("new account must not be complete")
	}
	e.expect(e.do("GET", "/discover", s.Token, nil), 409)

	base := map[string]any{"firstName": "Léa", "birthdate": birthdateForAge(29), "gender": "woman", "bio": "Hello", "interests": []string{"hiking"}}
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for key, val := range base {
			m[key] = val
		}
		m[k] = v
		return m
	}
	e.expect(e.do("PUT", "/profile", s.Token, with("birthdate", time.Now().AddDate(-17, 0, 0).Format("2006-01-02"))), 400)
	e.expect(e.do("PUT", "/profile", s.Token, with("gender", "robot")), 400)
	e.expect(e.do("PUT", "/profile", s.Token, with("firstName", "<b>x</b>")), 400)
	e.expect(e.do("PUT", "/profile", s.Token, with("bio", strings.Repeat("a", 501))), 400)
	e.expect(e.do("PUT", "/profile", s.Token, with("interests", []string{"not_a_real_interest"})), 400)
	e.expect(e.do("PUT", "/profile", s.Token, with("interests", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"})), 400)
	e.expect(e.do("PUT", "/profile", s.Token, base), 200)
	// Birthdate is locked once set (age fraud prevention).
	e.expect(e.do("PUT", "/profile", s.Token, with("birthdate", birthdateForAge(25))), 409)

	e.expect(e.do("PUT", "/profile/preferences", s.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10}), 400)
	e.expect(e.do("PUT", "/profile/preferences", s.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 30}), 200)

	e.expect(e.do("PUT", "/profile/location", s.Token, map[string]any{"latitude": 123.0, "longitude": 2.0}), 400)
	e.expect(e.do("PUT", "/profile/location", s.Token, map[string]any{"latitude": 48.856613, "longitude": 2.352222}), 204)

	// Coordinates are coarsened before storage.
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, s.UserID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be coarsened, got %v,%v", lat, lng)
	}

	// The public profile never leaks coordinates, birthdate or e-mail.
	r := e.expect(e.do("GET", "/profiles/"+s.UserID, s.Token, nil), 200)
	for _, forbidden := range []string{"latitude", "longitude", "birthdate", "email", "48.86"} {
		if strings.Contains(string(r.Body), forbidden) {
			t.Fatalf("public profile leaks %q: %s", forbidden, r.Body)
		}
	}

	e.expect(e.do("PATCH", "/profile/privacy", s.Token, map[string]any{"discoverable": false}), 200)
	e.expect(e.do("GET", "/interests", "", nil), 200)
}

func TestPhotosLifecycle(t *testing.T) {
	e := newEnv(t)
	s := e.register("photos")

	e.uploadPhoto(s.Token, []byte("<html><script>alert(1)</script></html>"), 415)
	e.uploadPhoto(s.Token, testImage(t, 100, 100), 400)

	var list photoList
	for i := 0; i < 6; i++ {
		list = e.uploadPhoto(s.Token, testImage(t, 300+i, 300), 201)
	}
	if len(list.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(list.Photos))
	}
	e.uploadPhoto(s.Token, testImage(t, 300, 300), 409)

	// Signed media URL works; tampering or another key does not.
	url := list.Photos[0].URL
	media := e.do("GET", url, "", nil)
	if media.Status != 200 || len(media.Body) < 100 || string(media.Body[:2]) != "\xff\xd8" {
		t.Fatalf("expected a JPEG, got %d", media.Status)
	}
	e.expect(e.do("GET", strings.Replace(url, "sig=", "sig=x", 1), "", nil), 404)
	e.expect(e.do("GET", "/media/../../etc/passwd?exp=9999999999&sig=x", "", nil), 404)

	// Reorder: reversed order makes the last photo the main one.
	ids := make([]string, len(list.Photos))
	for i, p := range list.Photos {
		ids[len(ids)-1-i] = p.ID
	}
	var reordered photoList
	e.expect(e.do("PUT", "/photos/order", s.Token, map[string]any{"photoIds": ids}), 200).json(t, &reordered)
	if reordered.Photos[0].ID != list.Photos[5].ID {
		t.Fatal("reorder did not apply")
	}
	e.expect(e.do("PUT", "/photos/order", s.Token, map[string]any{"photoIds": ids[:2]}), 400)
	dup := append([]string{ids[0]}, ids[:5]...)
	e.expect(e.do("PUT", "/photos/order", s.Token, map[string]any{"photoIds": dup}), 400)

	// Replace keeps the position, delete compacts positions and removes the file.
	var replaced photoList
	e.expect(e.multipart("PUT", "/photos/"+ids[0], s.Token, testImage(t, 640, 480)), 200).json(t, &replaced)
	if replaced.Photos[0].ID != ids[0] || replaced.Photos[0].URL == reordered.Photos[0].URL {
		t.Fatal("replace should keep id/position and change the file")
	}
	var after photoList
	e.expect(e.do("DELETE", "/photos/"+ids[0], s.Token, nil), 200).json(t, &after)
	if len(after.Photos) != 5 || after.Photos[0].Position != 0 {
		t.Fatalf("unexpected photos after delete: %+v", after.Photos)
	}
	e.expect(e.do("GET", replaced.Photos[0].URL, "", nil), 404)

	// Another member cannot touch these photos.
	intruder := e.register("intruder")
	e.expect(e.do("DELETE", "/photos/"+ids[1], intruder.Token, nil), 404)
}

func dialRealtime(t *testing.T, e *env, token string) (*websocket.Conn, <-chan map[string]any) {
	t.Helper()
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	e.expect(e.do("POST", "/realtime/ticket", token, nil), 201).json(t, &ticket)
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/realtime?ticket=" + ticket.Ticket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	events := make(chan map[string]any, 32)
	go func() {
		defer close(events)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ev map[string]any
			if json.Unmarshal(raw, &ev) == nil {
				events <- ev
			}
		}
	}()
	// Wait for the ready frame so the subscription is registered before we continue.
	waitEvent(t, events, "ready")
	// A ticket is single use.
	if _, _, err := websocket.DefaultDialer.Dial(wsURL, nil); err == nil {
		t.Fatal("ticket reuse must fail")
	}
	return conn, events
}

func waitEvent(t *testing.T, events <-chan map[string]any, eventType string) map[string]any {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("socket closed while waiting for %s", eventType)
			}
			if ev["type"] == eventType {
				return ev
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s", eventType)
		}
	}
}

func TestDatingFlowLikeMatchChat(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard(memberSpec{Name: "Alice", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng, Interests: []string{"hiking", "coffee", "jazz"}})
	bob := e.onboard(memberSpec{Name: "Bob", Gender: "man", Age: 31, InterestedIn: []string{"woman"}, Lat: nearbyLat, Lng: nearbyLng, Interests: []string{"hiking", "jazz"}})
	carol := e.onboard(memberSpec{Name: "Carol", Gender: "woman", Age: 30, InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng})

	// Discovery: mutual preferences, approximate distance.
	d := e.deck(alice.Token)
	if !d.has(bob.UserID) || d.has(carol.UserID) || d.has(alice.UserID) {
		t.Fatalf("unexpected deck for Alice: %+v", d.Profiles)
	}
	for _, p := range d.Profiles {
		if p.UserID == bob.UserID {
			if p.DistanceKm == nil || *p.DistanceKm < 5 || *p.DistanceKm > 12 {
				t.Fatalf("unexpected approximate distance %v", p.DistanceKm)
			}
			if p.Age != 31 || len(p.Photos) != 1 {
				t.Fatalf("unexpected card %+v", p)
			}
		}
	}

	bobSocket, bobEvents := dialRealtime(t, e, bob.Token)
	defer bobSocket.Close()

	// Validation of swipes.
	e.swipe(alice.Token, alice.UserID, "like", 400)
	e.swipe(alice.Token, bob.UserID, "superlike", 400)
	e.swipe(alice.Token, "00000000-0000-0000-0000-000000000000", "like", 404)

	// Alice likes Bob: no match yet, Bob is notified anonymously and sees Alice in his likes.
	if res := e.swipe(alice.Token, bob.UserID, "like", 201); res.Matched {
		t.Fatal("one-sided like must not match")
	}
	e.swipe(alice.Token, bob.UserID, "like", 409)
	e.swipe(alice.Token, bob.UserID, "pass", 409)
	if e.deck(alice.Token).has(bob.UserID) {
		t.Fatal("swiped profile must not reappear")
	}
	waitEvent(t, bobEvents, "notification.created")
	var likes struct {
		Profiles []struct {
			UserID string `json:"userId"`
		} `json:"profiles"`
	}
	e.expect(e.do("GET", "/likes/received", bob.Token, nil), 200).json(t, &likes)
	if len(likes.Profiles) != 1 || likes.Profiles[0].UserID != alice.UserID {
		t.Fatalf("Bob should see Alice's like: %+v", likes.Profiles)
	}

	// Bob likes back: match + conversation, announced in real time.
	res := e.swipe(bob.Token, alice.UserID, "like", 201)
	if !res.Matched || res.Match == nil || res.Match.ConversationID == "" || res.Match.Profile.UserID != alice.UserID {
		t.Fatalf("expected a match, got %+v", res)
	}
	waitEvent(t, bobEvents, "match.created")
	var matchCount int
	if err := e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = ANY($1) AND user_b_id = ANY($1)`, []string{alice.UserID, bob.UserID}).Scan(&matchCount); err != nil {
		t.Fatal(err)
	}
	if matchCount != 1 {
		t.Fatalf("expected exactly one match, got %d", matchCount)
	}
	conv := res.Match.ConversationID

	var matches struct {
		Matches []struct {
			ID          string `json:"id"`
			HasMessages bool   `json:"hasMessages"`
		} `json:"matches"`
	}
	e.expect(e.do("GET", "/matches", alice.Token, nil), 200).json(t, &matches)
	if len(matches.Matches) != 1 || matches.Matches[0].HasMessages {
		t.Fatalf("unexpected matches %+v", matches)
	}

	// Chat: send, receive in real time, read receipts, unread counts.
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": "   "}), 400)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": strings.Repeat("x", 2001)}), 400)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": "Salut Bob !"}), 201)
	ev := waitEvent(t, bobEvents, "message.created")
	if data, _ := ev["data"].(map[string]any); data["body"] != "Salut Bob !" {
		t.Fatalf("unexpected realtime payload %+v", ev)
	}

	var unread struct {
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do("GET", "/conversations/unread-count", bob.Token, nil), 200).json(t, &unread)
	if unread.UnreadCount != 1 {
		t.Fatalf("expected 1 unread conversation, got %d", unread.UnreadCount)
	}
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", bob.Token, nil), 200).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].LastMessage.Body != "Salut Bob !" {
		t.Fatalf("unexpected conversations %+v", convs)
	}

	aliceSocket, aliceEvents := dialRealtime(t, e, alice.Token)
	defer aliceSocket.Close()
	e.expect(e.do("POST", "/conversations/"+conv+"/read", bob.Token, nil), 200)
	waitEvent(t, aliceEvents, "conversation.read")
	e.expect(e.do("GET", "/conversations/unread-count", bob.Token, nil), 200).json(t, &unread)
	if unread.UnreadCount != 0 {
		t.Fatal("conversation should be read")
	}

	e.expect(e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]any{"body": "Hello Alice"}), 201)
	var page struct {
		Messages []struct {
			Body     string `json:"body"`
			SenderID string `json:"senderId"`
		} `json:"messages"`
		NextCursor      *string    `json:"nextCursor"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=1", alice.Token, nil), 200).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Hello Alice" || page.NextCursor == nil || page.OtherLastReadAt == nil {
		t.Fatalf("unexpected first page %+v", page)
	}
	e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=1&cursor="+*page.NextCursor, alice.Token, nil), 200).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "Salut Bob !" {
		t.Fatalf("unexpected second page %+v", page)
	}

	// Permissions: an outsider can neither read nor write, nor learn the conversation exists.
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", carol.Token, nil), 404)
	e.expect(e.do("GET", "/conversations/"+conv, carol.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", carol.Token, map[string]any{"body": "intrusion"}), 404)
	e.expect(e.do("POST", "/conversations/"+conv+"/read", carol.Token, nil), 404)
	e.expect(e.do("DELETE", "/matches/"+res.Match.ID, carol.Token, nil), 404)

	// Notifications: Alice got a match notification; message notifications coalesce.
	var notifs struct {
		Notifications []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	e.expect(e.do("GET", "/notifications", alice.Token, nil), 200).json(t, &notifs)
	types := map[string]int{}
	for _, n := range notifs.Notifications {
		types[n.Type]++
	}
	if types["match"] != 1 || types["message"] != 1 {
		t.Fatalf("unexpected notifications %+v", notifs.Notifications)
	}
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", bob.Token, nil), 404)
	e.expect(e.do("POST", "/notifications/"+notifs.Notifications[0].ID+"/read", alice.Token, nil), 204)
	e.expect(e.do("POST", "/notifications/read-all", alice.Token, nil), 204)
	e.expect(e.do("GET", "/notifications", alice.Token, nil), 200).json(t, &notifs)
	if notifs.UnreadCount != 0 {
		t.Fatal("expected all notifications read")
	}

	// Unmatch removes the conversation for both, and neither reappears in discovery.
	e.expect(e.do("DELETE", "/matches/"+res.Match.ID, alice.Token, nil), 204)
	waitEvent(t, bobEvents, "match.removed")
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", bob.Token, nil), 404)
	if e.deck(bob.Token).has(alice.UserID) || e.deck(alice.Token).has(bob.UserID) {
		t.Fatal("unmatched members must not reappear")
	}
	e.swipe(bob.Token, alice.UserID, "like", 409)
}

func TestBlockAndReport(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard(memberSpec{Name: "Alice", Gender: "woman", Age: 28, InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng})
	bob := e.onboard(memberSpec{Name: "Bob", Gender: "man", Age: 31, InterestedIn: []string{"woman"}, Lat: parisLat, Lng: parisLng})
	dan := e.onboard(memberSpec{Name: "Dan", Gender: "man", Age: 33, InterestedIn: []string{"woman"}, Lat: parisLat, Lng: parisLng})

	e.swipe(alice.Token, bob.UserID, "like", 201)
	res := e.swipe(bob.Token, alice.UserID, "like", 201)
	conv := res.Match.ConversationID

	// Blocking breaks the match and the conversation, and hides profiles both ways.
	e.expect(e.do("POST", "/blocks", alice.Token, map[string]any{"userId": alice.UserID}), 400)
	e.expect(e.do("POST", "/blocks", alice.Token, map[string]any{"userId": bob.UserID}), 204)
	e.expect(e.do("POST", "/blocks", alice.Token, map[string]any{"userId": bob.UserID}), 204) // idempotent
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]any{"body": "hey"}), 404)
	e.expect(e.do("GET", "/profiles/"+alice.UserID, bob.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+bob.UserID, alice.Token, nil), 404)
	var matches struct {
		Matches []any `json:"matches"`
	}
	e.expect(e.do("GET", "/matches", bob.Token, nil), 200).json(t, &matches)
	if len(matches.Matches) != 0 {
		t.Fatal("block must remove the match")
	}
	var blocked struct {
		Blocked []struct {
			UserID string `json:"userId"`
		} `json:"blocked"`
	}
	e.expect(e.do("GET", "/blocks", alice.Token, nil), 200).json(t, &blocked)
	if len(blocked.Blocked) != 1 || blocked.Blocked[0].UserID != bob.UserID {
		t.Fatalf("unexpected block list %+v", blocked)
	}
	e.expect(e.do("DELETE", "/blocks/"+bob.UserID, alice.Token, nil), 204)
	e.expect(e.do("DELETE", "/blocks/"+bob.UserID, alice.Token, nil), 404)

	// Reporting validates input, blocks the reported member and is not repeatable.
	e.expect(e.do("POST", "/reports", alice.Token, map[string]any{"userId": dan.UserID, "reason": "bad"}), 400)
	e.expect(e.do("POST", "/reports", alice.Token, map[string]any{"userId": dan.UserID, "reason": "harassment", "details": "Messages insistants"}), 204)
	e.expect(e.do("POST", "/reports", alice.Token, map[string]any{"userId": dan.UserID, "reason": "spam"}), 409)
	if e.deck(alice.Token).has(dan.UserID) {
		t.Fatal("reported member must be hidden from the reporter")
	}

	// Enough distinct reporters put the account under review: hidden from discovery.
	for i := 0; i < 2; i++ {
		reporter := e.register("reporter")
		e.expect(e.do("POST", "/reports", reporter.Token, map[string]any{"userId": dan.UserID, "reason": "fake_profile"}), 204)
	}
	var status string
	if err := e.pool.QueryRow(context.Background(), `SELECT moderation_status FROM users WHERE id = $1`, dan.UserID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "under_review" {
		t.Fatalf("expected under_review, got %s", status)
	}
	eve := e.onboard(memberSpec{Name: "Eve", Gender: "woman", Age: 30, InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng})
	if e.deck(eve.Token).has(dan.UserID) {
		t.Fatal("account under review must not be discoverable")
	}
}

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	seeker := e.onboard(memberSpec{Name: "Sam", Gender: "nonbinary", Age: 30, InterestedIn: []string{"woman"}, MinAge: 25, MaxAge: 35, MaxDistance: 20, Lat: parisLat, Lng: parisLng})

	match := e.onboard(memberSpec{Name: "Mia", Gender: "woman", Age: 29, InterestedIn: []string{"nonbinary"}, Lat: nearbyLat, Lng: nearbyLng})
	tooOld := e.onboard(memberSpec{Name: "Olga", Gender: "woman", Age: 40, InterestedIn: []string{"nonbinary"}, Lat: parisLat, Lng: parisLng})
	tooFar := e.onboard(memberSpec{Name: "Fanny", Gender: "woman", Age: 29, InterestedIn: []string{"nonbinary"}, Lat: lyonLat, Lng: lyonLng})
	notInterested := e.onboard(memberSpec{Name: "Nina", Gender: "woman", Age: 29, InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng})
	theirAgeRange := e.onboard(memberSpec{Name: "Rose", Gender: "woman", Age: 29, InterestedIn: []string{"nonbinary"}, MinAge: 18, MaxAge: 25, Lat: parisLat, Lng: parisLng})
	theirDistance := e.onboard(memberSpec{Name: "Dora", Gender: "woman", Age: 29, InterestedIn: []string{"nonbinary"}, MaxDistance: 2, Lat: nearbyLat, Lng: nearbyLng})
	hidden := e.onboard(memberSpec{Name: "Hana", Gender: "woman", Age: 29, InterestedIn: []string{"nonbinary"}, Lat: parisLat, Lng: parisLng})
	e.expect(e.do("PATCH", "/profile/privacy", hidden.Token, map[string]any{"discoverable": false}), 200)
	noPhoto := e.onboard(memberSpec{Name: "Pia", Gender: "woman", Age: 29, InterestedIn: []string{"nonbinary"}, Lat: parisLat, Lng: parisLng})
	var photos photoList
	e.expect(e.do("GET", "/photos", noPhoto.Token, nil), 200).json(t, &photos)
	e.expect(e.do("DELETE", "/photos/"+photos.Photos[0].ID, noPhoto.Token, nil), 200)

	d := e.deck(seeker.Token)
	if !d.has(match.UserID) {
		t.Fatal("compatible member missing from deck")
	}
	for name, m := range map[string]member{"too old": tooOld, "too far": tooFar, "not interested": notInterested, "their age range": theirAgeRange, "their distance": theirDistance, "not discoverable": hidden, "no photo": noPhoto} {
		if d.has(m.UserID) {
			t.Errorf("%s should be filtered out", name)
		}
	}

	// Hiding distance keeps the member discoverable but without a distance.
	e.expect(e.do("PATCH", "/profile/privacy", match.Token, map[string]any{"showDistance": false}), 200)
	for _, p := range e.deck(seeker.Token).Profiles {
		if p.UserID == match.UserID && p.DistanceKm != nil {
			t.Fatal("distance must be hidden")
		}
	}
}

// Two members liking each other at the same instant must produce exactly one match.
func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 8; i++ {
		a := e.onboard(memberSpec{Name: "Ana", Gender: "woman", Age: 27, InterestedIn: []string{"man"}})
		b := e.onboard(memberSpec{Name: "Ben", Gender: "man", Age: 27, InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		start := make(chan struct{})
		for idx, pair := range [][2]member{{a, b}, {b, a}} {
			wg.Add(1)
			go func(idx int, from, to member) {
				defer wg.Done()
				<-start
				results[idx] = e.swipe(from.Token, to.UserID, "like", 201)
			}(idx, pair[0], pair[1])
		}
		close(start)
		wg.Wait()
		if results[0].Matched == results[1].Matched {
			t.Fatalf("exactly one of the two swipes must report the match: %+v", results)
		}
		var n int
		if err := e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.UserID, b.UserID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("expected one match, got %d", n)
		}
	}
}

func TestAccountDeletionErasesData(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard(memberSpec{Name: "Alice", Gender: "woman", Age: 28, InterestedIn: []string{"man"}})
	bob := e.onboard(memberSpec{Name: "Bob", Gender: "man", Age: 31, InterestedIn: []string{"woman"}})
	e.swipe(alice.Token, bob.UserID, "like", 201)
	res := e.swipe(bob.Token, alice.UserID, "like", 201)
	e.expect(e.do("POST", "/conversations/"+res.Match.ConversationID+"/messages", alice.Token, map[string]any{"body": "Bonjour"}), 201)

	countFiles := func() int {
		n := 0
		_ = filepath.Walk(e.uploads, func(_ string, info os.FileInfo, _ error) error {
			if info != nil && !info.IsDir() {
				n++
			}
			return nil
		})
		return n
	}
	before := countFiles()

	e.expect(e.do("DELETE", "/me", alice.Token, map[string]any{"password": "WrongPass1"}), 403)
	e.expect(e.do("DELETE", "/me", alice.Token, map[string]any{"password": testPassword}), 204)

	e.expect(e.do("GET", "/me", alice.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]any{"email": alice.Email, "password": testPassword}), 401)
	if countFiles() != before-1 {
		t.Fatal("photo files must be purged")
	}
	for _, table := range []string{"profiles", "photos", "swipes", "matches", "messages", "notifications", "sessions"} {
		var n int
		query := map[string]string{
			"profiles":      `SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
			"photos":        `SELECT COUNT(*) FROM photos WHERE user_id = $1`,
			"swipes":        `SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1`,
			"matches":       `SELECT COUNT(*) FROM matches WHERE user_a_id = $1 OR user_b_id = $1`,
			"messages":      `SELECT COUNT(*) FROM messages WHERE sender_id = $1`,
			"notifications": `SELECT COUNT(*) FROM notifications WHERE user_id = $1`,
			"sessions":      `SELECT COUNT(*) FROM sessions WHERE user_id = $1`,
		}[table]
		if err := e.pool.QueryRow(context.Background(), query, alice.UserID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s still has %d rows for the deleted account", table, n)
		}
	}
	var convs struct {
		Conversations []any `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", bob.Token, nil), 200).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("conversation with a deleted account must disappear")
	}
}

func TestRateLimitOnLogin(t *testing.T) {
	e := newEnv(t, func(o *Options) { o.DisableRateLimits = false })
	statuses := map[int]int{}
	for i := 0; i < 22; i++ {
		statuses[e.do("POST", "/auth/login", "", map[string]any{"email": "nobody@integration.test", "password": "Whatever1"}).Status]++
	}
	if statuses[http.StatusTooManyRequests] == 0 {
		t.Fatalf("expected rate limiting, got %v", statuses)
	}
}
