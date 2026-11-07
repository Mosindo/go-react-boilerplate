package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var (
	alice = profileSpec{Name: "Alice", Gender: "woman", BirthDate: "1994-05-10", InterestedIn: []string{"man"}, Lat: 48.8566, Lon: 2.3522, Interests: []string{"travel", "music"}}
	bob   = profileSpec{Name: "Bob", Gender: "man", BirthDate: "1992-03-02", InterestedIn: []string{"woman"}, Lat: 48.8700, Lon: 2.3300, Interests: []string{"music", "cooking"}}
)

type cardResp struct {
	UserID          string                     `json:"userId"`
	FirstName       string                     `json:"firstName"`
	Age             int                        `json:"age"`
	DistanceKm      *int                       `json:"distanceKm"`
	SharedInterests int                        `json:"sharedInterests"`
	Photos          []struct{ ID, URL string } `json:"photos"`
}

func ids(cards []cardResp) []string {
	out := []string{}
	for _, c := range cards {
		out = append(out, c.UserID)
	}
	return out
}

func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", nil, ""), 200)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	u := e.register("auth")

	e.expect(e.do("POST", "/auth/register", map[string]string{"email": strings.ToUpper(u.Email), "password": "correct-horse"}, ""), 409)
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": uniqueEmail("x"), "password": "short"}, ""), 400)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "wrong-password"}, ""), 401)
	e.expect(e.do("GET", "/me", nil, ""), 401)
	e.expect(e.do("GET", "/me", nil, "garbage"), 401)

	login := e.expect(e.do("POST", "/auth/login", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": "correct-horse"}, ""), 200)
	var lr struct{ AccessToken, RefreshToken string }
	login.json(t, &lr)
	e.expect(e.do("GET", "/me", nil, lr.AccessToken), 200)

	// refresh rotates the refresh token: the old one stops working.
	ref := e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": lr.RefreshToken}, ""), 200)
	var rr struct{ AccessToken, RefreshToken string }
	ref.json(t, &rr)
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": lr.RefreshToken}, ""), 401)

	// logout revokes the session, so even a still-valid access token is rejected.
	e.expect(e.do("POST", "/auth/logout", map[string]string{"refreshToken": rr.RefreshToken}, ""), 204)
	e.expect(e.do("GET", "/me", nil, rr.AccessToken), 401)
}

func TestPasswordResetAndChange(t *testing.T) {
	e := newEnv(t)
	u := e.register("reset")

	e.expect(e.do("POST", "/auth/forgot-password", map[string]string{"email": "nobody" + testEmailDomain}, ""), 204)
	e.expect(e.do("POST", "/auth/forgot-password", map[string]string{"email": u.Email}, ""), 204)
	body := e.mail.body(u.Email)
	idx := strings.Index(body, "token=")
	if idx < 0 {
		t.Fatalf("no reset link in mail: %q", body)
	}
	token := strings.Fields(body[idx+len("token="):])[0]

	e.expect(e.do("POST", "/auth/reset-password", map[string]string{"token": "bogus", "newPassword": "brand-new-pass"}, ""), 400)
	e.expect(e.do("POST", "/auth/reset-password", map[string]string{"token": token, "newPassword": "brand-new-pass"}, ""), 204)
	e.expect(e.do("POST", "/auth/reset-password", map[string]string{"token": token, "newPassword": "another-pass-1"}, ""), 400) // single use
	e.expect(e.do("GET", "/me", nil, u.Token), 401)                                                                             // sessions revoked
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "correct-horse"}, ""), 401)
	login := e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "brand-new-pass"}, ""), 200)
	var lr struct{ AccessToken string }
	login.json(t, &lr)

	e.expect(e.do("POST", "/me/password", map[string]string{"currentPassword": "nope-nope", "newPassword": "third-pass-12"}, lr.AccessToken), 403)
	e.expect(e.do("POST", "/me/password", map[string]string{"currentPassword": "brand-new-pass", "newPassword": "third-pass-12"}, lr.AccessToken), 204)
	e.expect(e.do("GET", "/me", nil, lr.AccessToken), 200) // current session kept
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t)
	u := e.register("prof")

	e.expect(e.do("GET", "/profile/me", nil, u.Token), 404)

	minor := alice
	minor.BirthDate = time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(e.setProfile(u, minor), 400)
	bad := alice
	bad.Gender = "robot"
	e.expect(e.setProfile(u, bad), 400)
	unknown := alice
	unknown.Interests = []string{"not-an-interest"}
	e.expect(e.setProfile(u, unknown), 400)

	r := e.expect(e.setProfile(u, alice), 200)
	if strings.Contains(string(r.Body), "latitude") || strings.Contains(string(r.Body), "48.85") {
		t.Fatalf("exact coordinates leaked in own profile: %s", r.Body)
	}
	var mine struct {
		Age         int  `json:"age"`
		HasLocation bool `json:"hasLocation"`
		Complete    bool `json:"complete"`
	}
	r.json(t, &mine)
	if !mine.HasLocation || mine.Complete || mine.Age < 18 {
		t.Fatalf("unexpected profile state: %+v", mine)
	}

	// Coordinates are stored rounded to ~1 km.
	var lat float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 {
		t.Fatalf("latitude not rounded: %v", lat)
	}

	// Incomplete profile (no photo) cannot discover or swipe.
	e.expect(e.do("GET", "/discover", nil, u.Token), 403)
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	u := e.register("photo")
	e.expect(e.setProfile(u, alice), 200)

	e.expect(e.upload(u.Token, "evil.png", []byte("<?php system($_GET['c']); ?>")), 400)
	e.expect(e.upload(u.Token, "tiny.png", samplePNG(50, 50)), 400)
	e.expect(e.upload(u.Token, "huge.png", append(samplePNG(300, 300), make([]byte, 6<<20)...)), 413)

	var photos []struct{ ID, URL string }
	for i := 0; i < 6; i++ {
		var p struct{ ID, URL string }
		e.expect(e.upload(u.Token, "p.png", samplePNG(1600+i, 900)), 201).json(t, &p)
		photos = append(photos, p)
	}
	e.expect(e.upload(u.Token, "p7.png", samplePNG(400, 400)), 400) // limit

	// served bytes are a re-encoded, downscaled JPEG; tampering with the signature fails.
	img := e.expect(e.do("GET", photos[0].URL, nil, ""), 200)
	if len(img.Body) < 3 || img.Body[0] != 0xFF || img.Body[1] != 0xD8 {
		t.Fatal("expected JPEG output")
	}
	e.expect(e.do("GET", photos[0].URL+"0", nil, ""), 404)
	e.expect(e.do("GET", "/media/"+photos[0].ID, nil, ""), 404)

	// reorder: last becomes main.
	order := []string{photos[5].ID, photos[0].ID, photos[1].ID, photos[2].ID, photos[3].ID, photos[4].ID}
	e.expect(e.do("PUT", "/profile/photos/order", map[string]any{"ids": order}, u.Token), 204)
	e.expect(e.do("PUT", "/profile/photos/order", map[string]any{"ids": order[:3]}, u.Token), 400)
	var mine struct{ Photos []struct{ ID string } }
	e.expect(e.do("GET", "/profile/me", nil, u.Token), 200).json(t, &mine)
	if mine.Photos[0].ID != photos[5].ID {
		t.Fatalf("main photo not updated: %+v", mine.Photos)
	}

	// another user cannot delete it; owner can, positions compact.
	other := e.register("photo-other")
	e.expect(e.do("DELETE", "/profile/photos/"+photos[5].ID, nil, other.Token), 404)
	e.expect(e.do("DELETE", "/profile/photos/"+photos[5].ID, nil, u.Token), 204)
	e.expect(e.do("GET", "/profile/me", nil, u.Token), 200).json(t, &mine)
	if len(mine.Photos) != 5 || mine.Photos[0].ID != photos[0].ID {
		t.Fatalf("unexpected photos after delete: %+v", mine.Photos)
	}
}

func TestMatchChatSafetyFlow(t *testing.T) {
	e := newEnv(t)
	a := e.onboarded("alice", alice)
	b := e.onboarded("bob", bob)
	c := e.onboarded("carol", profileSpec{Name: "Carol", Gender: "woman", BirthDate: "1990-01-01", InterestedIn: []string{"man"}, Lat: 48.86, Lon: 2.34})

	// Discovery: mutual orientation, privacy-safe cards.
	var feed struct{ Profiles []cardResp }
	e.expect(e.do("GET", "/discover?limit=20", nil, a.Token), 200).json(t, &feed)
	if !contains(ids(feed.Profiles), b.ID) || contains(ids(feed.Profiles), a.ID) || contains(ids(feed.Profiles), c.ID) {
		t.Fatalf("alice feed wrong: %v", ids(feed.Profiles))
	}
	raw := e.expect(e.do("GET", "/discover?limit=20", nil, a.Token), 200).Body
	if strings.Contains(string(raw), "latitude") || strings.Contains(string(raw), "birthDate") || strings.Contains(string(raw), "email") {
		t.Fatalf("private fields leaked in discovery: %s", raw)
	}
	for _, p := range feed.Profiles {
		if p.UserID == b.ID {
			if p.DistanceKm == nil || *p.DistanceKm < 1 || *p.DistanceKm > 5 || p.SharedInterests != 1 || len(p.Photos) != 1 {
				t.Fatalf("unexpected bob card: %+v", p)
			}
		}
	}

	// Swipes: validation, no duplicates, unknown target.
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": b.ID, "action": "super"}, a.Token), 400)
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, a.Token), 404)
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": "11111111-1111-1111-1111-111111111111", "action": "like"}, a.Token), 404)

	// Connect a websocket for bob to verify realtime delivery.
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	ws := dialWS(t, srv.URL, b.Token)
	defer ws.Close()

	var sw struct {
		Matched bool
		Match   *struct{ ID, ConversationID string }
	}
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": b.ID, "action": "like"}, a.Token), 200).json(t, &sw)
	if sw.Matched {
		t.Fatal("one-sided like must not match")
	}
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": b.ID, "action": "like"}, a.Token), 409) // no repeated likes
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": b.ID, "action": "pass"}, a.Token), 409)

	// alice no longer sees bob; bob sees alice in "likes received" and has a notification.
	e.expect(e.do("GET", "/discover?limit=20", nil, a.Token), 200).json(t, &feed)
	if contains(ids(feed.Profiles), b.ID) {
		t.Fatal("decided profile reappeared in discovery")
	}
	var recv struct{ Profiles []cardResp }
	e.expect(e.do("GET", "/likes/received", nil, b.Token), 200).json(t, &recv)
	if !contains(ids(recv.Profiles), a.ID) {
		t.Fatalf("likes received: %v", ids(recv.Profiles))
	}
	var notifs struct {
		Notifications []struct{ Type, ActorName string }
		UnreadCount   int
	}
	e.expect(e.do("GET", "/notifications", nil, b.Token), 200).json(t, &notifs)
	if notifs.UnreadCount != 1 || notifs.Notifications[0].Type != "like" {
		t.Fatalf("notifications: %+v", notifs)
	}

	// Reciprocal like -> match (and the realtime event reaches alice's counterpart on bob's side too).
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, b.Token), 200).json(t, &sw)
	if !sw.Matched || sw.Match == nil {
		t.Fatalf("expected match: %+v", sw)
	}
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, b.Token), 409)
	waitEvent(t, ws, "notification")

	var matches struct {
		Matches []struct {
			ID   string
			User cardResp
		}
	}
	e.expect(e.do("GET", "/matches", nil, a.Token), 200).json(t, &matches)
	if len(matches.Matches) != 1 || matches.Matches[0].User.UserID != b.ID {
		t.Fatalf("matches: %+v", matches)
	}
	var count int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id IN ($1,$2) AND user_b_id IN ($1,$2)`, a.ID, b.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly one match row, got %d", count)
	}

	// Chat.
	convID := sw.Match.ConversationID
	path := "/conversations/" + convID + "/messages"
	e.expect(e.do("POST", path, map[string]string{"body": "   "}, a.Token), 400)
	e.expect(e.do("POST", path, map[string]string{"body": strings.Repeat("é", 2001)}, a.Token), 400)
	e.expect(e.do("POST", path, map[string]string{"body": "Salut Bob !"}, a.Token), 201)
	ev := waitEvent(t, ws, "message")
	if !strings.Contains(string(ev), "Salut Bob") {
		t.Fatalf("realtime message payload: %s", ev)
	}

	var convs struct {
		Conversations []struct {
			ID          string
			UnreadCount int
			LastMessage *struct{ Body string }
		}
	}
	e.expect(e.do("GET", "/conversations", nil, b.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].LastMessage.Body != "Salut Bob !" {
		t.Fatalf("bob conversations: %+v", convs)
	}
	e.expect(e.do("GET", "/conversations", nil, a.Token), 200).json(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatalf("sender should have 0 unread: %+v", convs)
	}

	// Permissions: carol is not in the conversation.
	e.expect(e.do("GET", path, nil, c.Token), 404)
	e.expect(e.do("POST", path, map[string]string{"body": "intrusion"}, c.Token), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", nil, c.Token), 404)
	e.expect(e.do("DELETE", "/conversations/"+convID, nil, c.Token), 404)
	e.expect(e.do("DELETE", "/matches/"+sw.Match.ID, nil, c.Token), 404)

	// Read receipts.
	var page struct {
		Messages []struct {
			Body      string
			CreatedAt time.Time
		}
		PeerReadAt time.Time
	}
	e.expect(e.do("GET", path, nil, b.Token), 200).json(t, &page)
	if len(page.Messages) != 1 {
		t.Fatalf("bob messages: %+v", page)
	}
	e.expect(e.do("GET", path, nil, a.Token), 200).json(t, &page)
	if !page.PeerReadAt.Before(page.Messages[0].CreatedAt) {
		t.Fatal("message must be unread by bob before he opens the conversation")
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/read", nil, b.Token), 204)
	e.expect(e.do("GET", path, nil, a.Token), 200).json(t, &page)
	if page.PeerReadAt.Before(page.Messages[0].CreatedAt) {
		t.Fatal("message should now be read by bob")
	}
	e.expect(e.do("GET", "/conversations", nil, b.Token), 200).json(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread should be cleared")
	}

	// Pagination of messages.
	for i := 0; i < 5; i++ {
		e.expect(e.do("POST", path, map[string]string{"body": "m"}, b.Token), 201)
	}
	var p1, p2 struct {
		Messages   []struct{ ID int64 }
		NextBefore *int64
	}
	e.expect(e.do("GET", path+"?limit=4", nil, a.Token), 200).json(t, &p1)
	if len(p1.Messages) != 4 || p1.NextBefore == nil {
		t.Fatalf("page1: %+v", p1)
	}
	e.expect(e.do("GET", path+"?limit=4&before="+itoa(*p1.NextBefore), nil, a.Token), 200).json(t, &p2)
	if len(p2.Messages) != 2 || p2.NextBefore != nil || p2.Messages[len(p2.Messages)-1].ID >= p1.Messages[0].ID {
		t.Fatalf("page2: %+v", p2)
	}

	// Local deletion hides the conversation for alice only, until a new message arrives.
	e.expect(e.do("DELETE", "/conversations/"+convID, nil, a.Token), 204)
	e.expect(e.do("GET", "/conversations", nil, a.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("conversation should be hidden for alice")
	}
	e.expect(e.do("GET", "/conversations", nil, b.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("bob must still see the conversation")
	}
	e.expect(e.do("POST", path, map[string]string{"body": "back"}, b.Token), 201)
	e.expect(e.do("GET", "/conversations", nil, a.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("new message should resurface the conversation")
	}
	e.expect(e.do("GET", path, nil, a.Token), 200).json(t, &page)
	if len(page.Messages) != 1 || page.Messages[0].Body != "back" {
		t.Fatalf("history should be cleared locally: %+v", page.Messages)
	}

	// Blocking removes the match and the conversation, hides both users from each other, and is listed.
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": a.ID}, a.Token), 400)
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": b.ID}, a.Token), 204)
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": b.ID}, a.Token), 204) // idempotent
	e.expect(e.do("GET", path, nil, a.Token), 404)
	e.expect(e.do("POST", path, map[string]string{"body": "still there?"}, b.Token), 404)
	e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), 404)
	e.expect(e.do("GET", "/profiles/"+a.ID, nil, b.Token), 404)
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, b.Token), 404)
	var blocks struct{ Blocks []struct{ UserID string } }
	e.expect(e.do("GET", "/blocks", nil, a.Token), 200).json(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != b.ID {
		t.Fatalf("blocks: %+v", blocks)
	}
	e.expect(e.do("DELETE", "/blocks/"+b.ID, nil, a.Token), 204)
	e.expect(e.do("GET", "/blocks", nil, a.Token), 200).json(t, &blocks)
	if len(blocks.Blocks) != 0 {
		t.Fatal("unblock failed")
	}

	// Reports.
	e.expect(e.do("POST", "/reports", map[string]any{"userId": c.ID, "reason": "nope"}, a.Token), 400)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, a.Token), 400)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": c.ID, "reason": "harassment", "details": "rude", "block": true}, a.Token), 204)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, c.ID).Scan(&count)
	if count != 1 {
		t.Fatal("report not stored")
	}
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, a.ID, c.ID).Scan(&count)
	if count != 1 {
		t.Fatal("report with block=true must also block")
	}
}

func TestSimultaneousLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	a := e.onboarded("race-a", alice)
	b := e.onboarded("race-b", bob)
	done := make(chan int, 2)
	go func() {
		done <- e.do("POST", "/swipes", map[string]string{"userId": b.ID, "action": "like"}, a.Token).Status
	}()
	go func() {
		done <- e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, b.Token).Status
	}()
	<-done
	<-done
	var matches, convs int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id IN ($1,$2)`, a.ID, b.ID).Scan(&matches)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a_id IN ($1,$2)`, a.ID, b.ID).Scan(&convs)
	if matches != 1 || convs != 1 {
		t.Fatalf("expected 1 match and 1 conversation, got %d/%d", matches, convs)
	}
}

func TestUnmatchAndDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	a := e.onboarded("flt-a", alice)
	far := bob
	far.Lat, far.Lon = 40.7128, -74.0060 // New York: outside alice's 50 km radius
	farUser := e.onboarded("flt-far", far)
	older := bob
	older.BirthDate = "1950-01-01" // outside alice's 18-60 age range
	olderUser := e.onboarded("flt-old", older)
	hidden := e.onboarded("flt-hidden", bob)
	e.expect(e.do("PUT", "/profile/me", map[string]any{
		"firstName": "Hidden", "birthDate": "1992-03-02", "gender": "man", "discoverable": false,
		"latitude": 48.87, "longitude": 2.33,
		"preferences": map[string]any{"interestedIn": []string{"woman"}, "ageMin": 18, "ageMax": 60, "maxDistanceKm": 50},
	}, hidden.Token), 200)
	near := e.onboarded("flt-near", bob)

	var feed struct{ Profiles []cardResp }
	e.expect(e.do("GET", "/discover?limit=20", nil, a.Token), 200).json(t, &feed)
	got := ids(feed.Profiles)
	if !contains(got, near.ID) {
		t.Fatal("near candidate missing")
	}
	for name, id := range map[string]string{"far": farUser.ID, "older": olderUser.ID, "hidden": hidden.ID} {
		if contains(got, id) {
			t.Fatalf("%s candidate must be filtered out", name)
		}
	}
	e.expect(e.do("GET", "/discover?limit=99", nil, a.Token), 400)

	// match then unmatch: gone for both, never resurfaces.
	var sw struct {
		Match *struct{ ID, ConversationID string }
	}
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": near.ID, "action": "like"}, a.Token), 200)
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, near.Token), 200).json(t, &sw)
	e.expect(e.do("POST", "/conversations/"+sw.Match.ConversationID+"/messages", map[string]string{"body": "hi"}, near.Token), 201)
	e.expect(e.do("DELETE", "/matches/"+sw.Match.ID, nil, a.Token), 204)
	e.expect(e.do("GET", "/conversations/"+sw.Match.ConversationID+"/messages", nil, near.Token), 404)
	e.expect(e.do("DELETE", "/matches/"+sw.Match.ID, nil, a.Token), 404)
	e.expect(e.do("GET", "/discover?limit=20", nil, a.Token), 200).json(t, &feed)
	if contains(ids(feed.Profiles), near.ID) {
		t.Fatal("unmatched profile must not come back")
	}
}

func TestAccountDeletionRemovesData(t *testing.T) {
	e := newEnv(t)
	a := e.onboarded("del-a", alice)
	b := e.onboarded("del-b", bob)
	var sw struct {
		Match *struct{ ID, ConversationID string }
	}
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": b.ID, "action": "like"}, a.Token), 200)
	e.expect(e.do("POST", "/swipes", map[string]string{"userId": a.ID, "action": "like"}, b.Token), 200).json(t, &sw)
	e.expect(e.do("POST", "/conversations/"+sw.Match.ConversationID+"/messages", map[string]string{"body": "hello"}, a.Token), 201)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, b.Token), 204)

	var photoKey string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&photoKey)
	if _, err := e.app.files.Read(photoKey); err != nil {
		t.Fatalf("photo file should exist: %v", err)
	}

	e.expect(e.do("DELETE", "/me", map[string]string{"password": "wrong-pass-1"}, a.Token), 403)
	e.expect(e.do("DELETE", "/me", map[string]string{"password": "correct-horse"}, a.Token), 204)

	e.expect(e.do("GET", "/me", nil, a.Token), 401)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": a.Email, "password": "correct-horse"}, ""), 401)
	if _, err := e.app.files.Read(photoKey); err == nil {
		t.Fatal("photo file should be deleted with the account")
	}
	for table, q := range map[string]string{
		"profiles": `SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
		"photos":   `SELECT COUNT(*) FROM photos WHERE user_id = $1`,
		"likes":    `SELECT COUNT(*) FROM likes WHERE from_user_id = $1 OR to_user_id = $1`,
		"matches":  `SELECT COUNT(*) FROM matches WHERE user_a_id = $1 OR user_b_id = $1`,
		"messages": `SELECT COUNT(*) FROM messages WHERE sender_id = $1`,
	} {
		var n int
		_ = e.pool.QueryRow(context.Background(), q, a.ID).Scan(&n)
		if n != 0 {
			t.Fatalf("%s rows remain after deletion: %d", table, n)
		}
	}
	// reports are kept for moderation but detached from the deleted account
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL`, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("report should survive with reported_id NULL, got %d", n)
	}
	var convs struct{ Conversations []any }
	e.expect(e.do("GET", "/conversations", nil, b.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("conversation must disappear for the other user")
	}
}

func TestRateLimitOnAuth(t *testing.T) {
	e := newEnv(t)
	limited := false
	for i := 0; i < 40; i++ {
		if e.do("POST", "/auth/login", map[string]string{"email": "nobody" + testEmailDomain, "password": "whatever-1"}, "").Status == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected 429 after repeated login attempts")
	}
}

func TestWebSocketRequiresAuth(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": "bad"})
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected the server to close the connection")
	}
}

func dialWS(t *testing.T, serverURL, token string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		t.Fatal(err)
	}
	waitEvent(t, conn, "ready") // registered with the hub: events are delivered from now on
	return conn
}

// waitEvent reads frames until one of the wanted type arrives.
func waitEvent(t *testing.T, conn *websocket.Conn, want string) []byte {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %q event: %v", want, err)
		}
		var ev struct{ Type string }
		_ = json.Unmarshal(raw, &ev)
		if ev.Type == want {
			return raw
		}
	}
	t.Fatalf("timeout waiting for %q event", want)
	return nil
}

func itoa(v int64) string { b, _ := json.Marshal(v); return string(b) }
