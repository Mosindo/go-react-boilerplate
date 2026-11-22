package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type photoJSON struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type cardJSON struct {
	UserID     string      `json:"userId"`
	FirstName  string      `json:"firstName"`
	Age        int         `json:"age"`
	DistanceKm *int        `json:"distanceKm"`
	Photos     []photoJSON `json:"photos"`
}

type ownProfileJSON struct {
	UserID        string      `json:"userId"`
	FirstName     string      `json:"firstName"`
	Age           *int        `json:"age"`
	IsComplete    bool        `json:"isComplete"`
	MissingFields []string    `json:"missingFields"`
	Photos        []photoJSON `json:"photos"`
	HasLocation   bool        `json:"hasLocation"`
}

type discoveryJSON struct {
	Profiles      []cardJSON `json:"profiles"`
	Status        string     `json:"status"`
	MissingFields []string   `json:"missingFields"`
}

type swipeJSON struct {
	Action  string `json:"action"`
	Matched bool   `json:"matched"`
	Match   *struct {
		ID   string   `json:"id"`
		User cardJSON `json:"user"`
	} `json:"match"`
}

type chatJSON struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	LastMessage *struct {
		Content string `json:"content"`
	} `json:"lastMessage"`
	UnreadCount int `json:"unreadCount"`
}

type chatsJSON struct {
	Chats       []chatJSON `json:"chats"`
	TotalUnread int        `json:"totalUnread"`
}

type messageJSON struct {
	ID           string `json:"id"`
	SenderUserID string `json:"senderUserId"`
	Content      string `json:"content"`
}

type messagesJSON struct {
	Messages        []messageJSON `json:"messages"`
	HasMore         bool          `json:"hasMore"`
	OtherLastReadAt *time.Time    `json:"otherLastReadAt"`
}

type member struct {
	auth authResponse
	id   string
	tok  string
}

type profileSpec struct {
	name, birthdate, gender string
	interestedIn            []string
	lat, lng                float64
	minAge, maxAge, maxKm   int
}

func pngImage(t *testing.T, w, h int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func doMultipart(t *testing.T, handler http.Handler, method, path string, data []byte, token string) testResponse {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

func expectStatus(t *testing.T, resp testResponse, want int, what string) {
	t.Helper()
	if resp.Status != want {
		t.Fatalf("%s: expected %d, got %d body=%s", what, want, resp.Status, resp.Body)
	}
}

func newMember(t *testing.T, router *gin.Engine, pool *pgxpool.Pool, prefix string) member {
	auth := registerUser(t, router, pool, prefix)
	return member{auth: auth, id: auth.User.ID, tok: auth.AccessToken}
}

func setupProfile(t *testing.T, router *gin.Engine, m member, spec profileSpec) {
	t.Helper()
	if spec.minAge == 0 {
		spec.minAge, spec.maxAge, spec.maxKm = 18, 99, 50
	}
	expectStatus(t, doRequest(t, router, http.MethodPatch, "/me/profile", map[string]any{
		"firstName": spec.name, "birthdate": spec.birthdate, "gender": spec.gender,
		"bio": "Bonjour, je suis " + spec.name, "interests": []string{"hiking", "coffee", "music"},
	}, m.tok), http.StatusOK, "patch profile "+spec.name)
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": spec.interestedIn, "minAge": spec.minAge, "maxAge": spec.maxAge, "maxDistanceKm": spec.maxKm,
	}, m.tok), http.StatusOK, "preferences "+spec.name)
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/location", map[string]any{
		"latitude": spec.lat, "longitude": spec.lng, "city": "Paris",
	}, m.tok), http.StatusOK, "location "+spec.name)
	expectStatus(t, doMultipart(t, router, http.MethodPost, "/me/photos", pngImage(t, 400, 500, color.RGBA{200, 120, 90, 255}), m.tok),
		http.StatusCreated, "photo "+spec.name)
}

func discover(t *testing.T, router *gin.Engine, m member) discoveryJSON {
	t.Helper()
	resp := doRequest(t, router, http.MethodGet, "/discovery?limit=20", nil, m.tok)
	expectStatus(t, resp, http.StatusOK, "discovery")
	var d discoveryJSON
	mustJSON(t, resp.Body, &d)
	return d
}

func inFeed(d discoveryJSON, userID string) bool {
	for _, p := range d.Profiles {
		if p.UserID == userID {
			return true
		}
	}
	return false
}

func swipe(t *testing.T, router *gin.Engine, m member, target, action string) (testResponse, swipeJSON) {
	t.Helper()
	resp := doRequest(t, router, http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "action": action}, m.tok)
	var s swipeJSON
	if resp.Status == http.StatusOK {
		mustJSON(t, resp.Body, &s)
	}
	return resp, s
}

func chats(t *testing.T, router *gin.Engine, m member) chatsJSON {
	t.Helper()
	resp := doRequest(t, router, http.MethodGet, "/chats", nil, m.tok)
	expectStatus(t, resp, http.StatusOK, "chats")
	var c chatsJSON
	mustJSON(t, resp.Body, &c)
	return c
}

func findChat(c chatsJSON, userID string) *chatJSON {
	for i := range c.Chats {
		if c.Chats[i].User.ID == userID {
			return &c.Chats[i]
		}
	}
	return nil
}

func drainEvents(ch <-chan []byte) []string {
	types := []string{}
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(raw, &evt)
			types = append(types, evt.Type)
		default:
			return types
		}
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// Paris-area members with compatible preferences.
func parisWoman(name string) profileSpec {
	return profileSpec{name: name, birthdate: "1994-04-12", gender: "woman", interestedIn: []string{"man"}, lat: 48.8566, lng: 2.3522}
}

func parisMan(name string) profileSpec {
	return profileSpec{name: name, birthdate: "1991-09-30", gender: "man", interestedIn: []string{"woman"}, lat: 48.8606, lng: 2.3376}
}

func TestDatingCriticalFlow(t *testing.T) {
	router, pool := setupTestRouter(t)
	hub := lastTestEnv.hub

	alice := newMember(t, router, pool, "alice")
	bob := newMember(t, router, pool, "bob")
	carol := newMember(t, router, pool, "carol")
	dave := newMember(t, router, pool, "dave")

	// 1. An incomplete profile can neither browse nor swipe.
	d := discover(t, router, alice)
	if d.Status != "incomplete_profile" || len(d.MissingFields) == 0 {
		t.Fatalf("expected incomplete_profile status, got %+v", d)
	}
	resp, _ := swipe(t, router, alice, bob.id, "like")
	expectStatus(t, resp, http.StatusForbidden, "swipe with incomplete profile")

	// 2. Onboarding.
	setupProfile(t, router, alice, parisWoman("Alice"))
	setupProfile(t, router, bob, parisMan("Bob"))
	carolSpec := parisWoman("Carol")
	carolSpec.interestedIn = []string{"woman"}
	setupProfile(t, router, carol, carolSpec)
	setupProfile(t, router, dave, parisMan("Dave"))

	own := doRequest(t, router, http.MethodGet, "/me/profile", nil, alice.tok)
	expectStatus(t, own, http.StatusOK, "own profile")
	var ownProfile ownProfileJSON
	mustJSON(t, own.Body, &ownProfile)
	if !ownProfile.IsComplete || !ownProfile.HasLocation || len(ownProfile.Photos) != 1 || ownProfile.Age == nil {
		t.Fatalf("expected a complete profile, got %+v", ownProfile)
	}
	if strings.Contains(string(own.Body), "latitude") || strings.Contains(string(own.Body), "48.85") {
		t.Fatalf("coordinates must never be returned, even to the owner: %s", own.Body)
	}

	// 3. Discovery is mutual and privacy-preserving.
	bobFeed := discover(t, router, bob)
	if !inFeed(bobFeed, alice.id) {
		t.Fatalf("bob should discover alice: %+v", bobFeed)
	}
	if inFeed(bobFeed, carol.id) {
		t.Fatalf("carol is only interested in women and must not be shown to bob")
	}
	if inFeed(bobFeed, dave.id) || inFeed(bobFeed, bob.id) {
		t.Fatalf("bob must not see men or himself")
	}
	if inFeed(discover(t, router, carol), bob.id) {
		t.Fatalf("bob must not be shown to carol")
	}
	raw := string(doRequest(t, router, http.MethodGet, "/discovery?limit=20", nil, bob.tok).Body)
	for _, secret := range []string{"@lueur.test", "latitude", "longitude", "birthdate", "1994-04-12"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("discovery leaked %q: %s", secret, raw)
		}
	}
	var aliceCard cardJSON
	for _, p := range bobFeed.Profiles {
		if p.UserID == alice.id {
			aliceCard = p
		}
	}
	if aliceCard.DistanceKm == nil || *aliceCard.DistanceKm < 1 || aliceCard.Age < 30 || len(aliceCard.Photos) != 1 {
		t.Fatalf("unexpected card: %+v", aliceCard)
	}

	// Signed photo URL works, tampered or unsigned ones do not.
	photo := doRequest(t, router, http.MethodGet, aliceCard.Photos[0].URL, nil, "")
	expectStatus(t, photo, http.StatusOK, "signed photo")
	if !bytes.HasPrefix(photo.Body, []byte{0xFF, 0xD8}) {
		t.Fatalf("photos must be served as re-encoded JPEG")
	}
	tampered := strings.Replace(aliceCard.Photos[0].URL, "sig=", "sig=00", 1)
	expectStatus(t, doRequest(t, router, http.MethodGet, tampered, nil, ""), http.StatusNotFound, "tampered photo URL")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/media/photos/"+aliceCard.Photos[0].ID, nil, ""), http.StatusNotFound, "unsigned photo URL")

	aliceEvents := hub.Subscribe(alice.id)
	defer hub.Unsubscribe(aliceEvents)
	bobEvents := hub.Subscribe(bob.id)
	defer hub.Unsubscribe(bobEvents)

	// 4. A like without reciprocity is not a match; repeats are idempotent.
	resp, s := swipe(t, router, bob, alice.id, "like")
	expectStatus(t, resp, http.StatusOK, "bob likes alice")
	if s.Matched {
		t.Fatalf("one-sided like must not match")
	}
	resp, s = swipe(t, router, bob, alice.id, "like")
	expectStatus(t, resp, http.StatusOK, "repeated like")
	if s.Matched {
		t.Fatalf("repeated like must stay unmatched")
	}
	resp, _ = swipe(t, router, bob, alice.id, "pass")
	expectStatus(t, resp, http.StatusConflict, "changing an answer")
	if inFeed(discover(t, router, bob), alice.id) {
		t.Fatalf("already processed profiles must not reappear")
	}
	if !contains(drainEvents(aliceEvents.C), "like.received") {
		t.Fatalf("alice should be notified in real time of a new like")
	}

	likes := doRequest(t, router, http.MethodGet, "/likes/received", nil, alice.tok)
	expectStatus(t, likes, http.StatusOK, "likes received")
	if !strings.Contains(string(likes.Body), bob.id) {
		t.Fatalf("alice should see bob in her likes: %s", likes.Body)
	}

	// 5. No chat before a match.
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+alice.id+"/messages", map[string]string{"content": "hi"}, bob.tok),
		http.StatusNotFound, "message before match")

	// 6. Reciprocal like => exactly one match, both notified.
	resp, s = swipe(t, router, alice, bob.id, "like")
	expectStatus(t, resp, http.StatusOK, "alice likes bob")
	if !s.Matched || s.Match == nil || s.Match.User.UserID != bob.id {
		t.Fatalf("expected a match with bob, got %+v", s)
	}
	var matchCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
	`, alice.id, bob.id).Scan(&matchCount); err != nil || matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d (%v)", matchCount, err)
	}
	if !contains(drainEvents(bobEvents.C), "match.created") {
		t.Fatalf("bob should receive match.created in real time")
	}
	notifs := doRequest(t, router, http.MethodGet, "/notifications", nil, bob.tok)
	if !strings.Contains(string(notifs.Body), `"type":"match.created"`) || !strings.Contains(string(notifs.Body), "Alice") {
		t.Fatalf("bob should have a match notification naming Alice: %s", notifs.Body)
	}
	if c := findChat(chats(t, router, alice), bob.id); c == nil || c.LastMessage != nil {
		t.Fatalf("a new match must appear as an empty conversation, got %+v", c)
	}
	contacts := doRequest(t, router, http.MethodGet, "/users", nil, alice.tok)
	if !strings.Contains(string(contacts.Body), bob.id) || strings.Contains(string(contacts.Body), "@") {
		t.Fatalf("/users should list bob without e-mail: %s", contacts.Body)
	}

	// 7. Chat: send, unread, real time, read receipts.
	drainEvents(aliceEvents.C)
	sent := doRequest(t, router, http.MethodPost, "/chats/"+alice.id+"/messages", map[string]string{"content": "  Salut Alice !  "}, bob.tok)
	expectStatus(t, sent, http.StatusCreated, "send message")
	var sentMsg messageJSON
	mustJSON(t, sent.Body, &sentMsg)
	if sentMsg.Content != "Salut Alice !" || sentMsg.SenderUserID != bob.id {
		t.Fatalf("unexpected message %+v", sentMsg)
	}
	if !contains(drainEvents(aliceEvents.C), "message.created") {
		t.Fatalf("alice should receive the message in real time")
	}
	aliceChats := chats(t, router, alice)
	if c := findChat(aliceChats, bob.id); c == nil || c.UnreadCount != 1 || c.LastMessage == nil || aliceChats.TotalUnread != 1 {
		t.Fatalf("alice should have 1 unread message from bob, got %+v", aliceChats)
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+alice.id+"/messages", map[string]string{"content": "   "}, bob.tok),
		http.StatusBadRequest, "blank message")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+bob.id+"/read", nil, alice.tok), http.StatusOK, "mark read")
	if !contains(drainEvents(bobEvents.C), "conversation.read") {
		t.Fatalf("bob should be told alice read the conversation")
	}
	if c := findChat(chats(t, router, alice), bob.id); c.UnreadCount != 0 {
		t.Fatalf("unread count must reset after reading")
	}
	bobThread := doRequest(t, router, http.MethodGet, "/chats/"+alice.id+"/messages", nil, bob.tok)
	var bobMsgs messagesJSON
	mustJSON(t, bobThread.Body, &bobMsgs)
	if bobMsgs.OtherLastReadAt == nil || len(bobMsgs.Messages) != 1 {
		t.Fatalf("bob should see 1 message with a read receipt, got %+v", bobMsgs)
	}

	// Cursor pagination.
	for i := 0; i < 5; i++ {
		expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+bob.id+"/messages", map[string]string{"content": fmt.Sprintf("m%d", i)}, alice.tok),
			http.StatusCreated, "send batch")
	}
	page := doRequest(t, router, http.MethodGet, "/chats/"+bob.id+"/messages?limit=2", nil, alice.tok)
	var page1 messagesJSON
	mustJSON(t, page.Body, &page1)
	if len(page1.Messages) != 2 || !page1.HasMore || page1.Messages[1].Content != "m4" {
		t.Fatalf("unexpected first page %+v", page1)
	}
	page = doRequest(t, router, http.MethodGet, "/chats/"+bob.id+"/messages?limit=10&before="+page1.Messages[0].ID, nil, alice.tok)
	var page2 messagesJSON
	mustJSON(t, page.Body, &page2)
	if len(page2.Messages) != 4 || page2.HasMore || page2.Messages[0].Content != "Salut Alice !" {
		t.Fatalf("unexpected second page %+v", page2)
	}

	// 8. Outsiders can never read or write a conversation.
	for _, path := range []string{"/chats/" + alice.id + "/messages", "/chats/" + bob.id + "/messages"} {
		expectStatus(t, doRequest(t, router, http.MethodGet, path, nil, dave.tok), http.StatusNotFound, "outsider read "+path)
		expectStatus(t, doRequest(t, router, http.MethodPost, path, map[string]string{"content": "intrusion"}, dave.tok), http.StatusNotFound, "outsider write")
	}
	if findChat(chats(t, router, dave), alice.id) != nil {
		t.Fatalf("dave must not see alice's conversations")
	}

	// 9. Local clear only affects the caller.
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/chats/"+bob.id, nil, alice.tok), http.StatusNoContent, "clear chat")
	var cleared messagesJSON
	mustJSON(t, doRequest(t, router, http.MethodGet, "/chats/"+bob.id+"/messages", nil, alice.tok).Body, &cleared)
	if len(cleared.Messages) != 0 {
		t.Fatalf("cleared history must be hidden for alice")
	}
	var bobStill messagesJSON
	mustJSON(t, doRequest(t, router, http.MethodGet, "/chats/"+alice.id+"/messages?limit=50", nil, bob.tok).Body, &bobStill)
	if len(bobStill.Messages) != 6 {
		t.Fatalf("bob must keep his history, got %d messages", len(bobStill.Messages))
	}

	// 10. Blocking removes the match and every access path.
	expectStatus(t, doRequest(t, router, http.MethodPost, "/blocks", map[string]string{"userId": bob.id}, alice.tok), http.StatusNoContent, "block")
	if findChat(chats(t, router, bob), alice.id) != nil {
		t.Fatalf("blocked match must disappear for bob")
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+alice.id+"/messages", map[string]string{"content": "?"}, bob.tok), http.StatusNotFound, "message after block")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/profiles/"+alice.id, nil, bob.tok), http.StatusNotFound, "profile after block")
	resp, _ = swipe(t, router, bob, alice.id, "like")
	expectStatus(t, resp, http.StatusNotFound, "swipe after block")
	blocks := doRequest(t, router, http.MethodGet, "/blocks", nil, alice.tok)
	if !strings.Contains(string(blocks.Body), bob.id) {
		t.Fatalf("block list should contain bob: %s", blocks.Body)
	}
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/blocks/"+bob.id, nil, alice.tok), http.StatusNoContent, "unblock")
	if strings.Contains(string(doRequest(t, router, http.MethodGet, "/blocks", nil, alice.tok).Body), bob.id) {
		t.Fatalf("unblock should remove bob from the list")
	}
}

func TestUnmatchAndResetPasses(t *testing.T) {
	router, pool := setupTestRouter(t)
	eve := newMember(t, router, pool, "eve")
	frank := newMember(t, router, pool, "frank")
	setupProfile(t, router, eve, parisWoman("Eve"))
	setupProfile(t, router, frank, parisMan("Frank"))

	swipe(t, router, eve, frank.id, "like")
	_, s := swipe(t, router, frank, eve.id, "like")
	if !s.Matched {
		t.Fatalf("expected match")
	}
	matches := doRequest(t, router, http.MethodGet, "/matches", nil, eve.tok)
	if !strings.Contains(string(matches.Body), frank.id) {
		t.Fatalf("matches should list frank: %s", matches.Body)
	}
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/matches/"+frank.id, nil, eve.tok), http.StatusNoContent, "unmatch")
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/matches/"+frank.id, nil, eve.tok), http.StatusNotFound, "unmatch twice")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/chats/"+eve.id+"/messages", nil, frank.tok), http.StatusNotFound, "chat after unmatch")
	if inFeed(discover(t, router, frank), eve.id) {
		t.Fatalf("unmatched profiles must not resurface")
	}

	grace := newMember(t, router, pool, "grace")
	setupProfile(t, router, grace, parisWoman("Grace"))
	swipe(t, router, frank, grace.id, "pass")
	if inFeed(discover(t, router, frank), grace.id) {
		t.Fatalf("passed profile must not reappear")
	}
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/discovery/passes", nil, frank.tok), http.StatusOK, "reset passes")
	if !inFeed(discover(t, router, frank), grace.id) {
		t.Fatalf("passed profiles come back after an explicit reset")
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	router, pool := setupTestRouter(t)
	for round := 0; round < 3; round++ {
		w := newMember(t, router, pool, fmt.Sprintf("race_w%d", round))
		m := newMember(t, router, pool, fmt.Sprintf("race_m%d", round))
		setupProfile(t, router, w, parisWoman("Ines"))
		setupProfile(t, router, m, parisMan("Jules"))

		var wg sync.WaitGroup
		results := make([]swipeJSON, 2)
		for i, pair := range [][2]member{{w, m}, {m, w}} {
			wg.Add(1)
			go func(i int, from, to member) {
				defer wg.Done()
				resp := doRequest(t, router, http.MethodPost, "/swipes", map[string]string{"targetUserId": to.id, "action": "like"}, from.tok)
				if resp.Status == http.StatusOK {
					_ = json.Unmarshal(resp.Body, &results[i])
				}
			}(i, pair[0], pair[1])
		}
		wg.Wait()

		var n int
		if err := pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid, $2::uuid) AND user_b_id = GREATEST($1::uuid, $2::uuid)
		`, w.id, m.id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("round %d: simultaneous likes must create exactly one match, got %d", round, n)
		}
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: one of the two swipes must report the match", round)
		}
	}
}

func TestDiscoveryHonoursDistanceAndAge(t *testing.T) {
	router, pool := setupTestRouter(t)
	viewer := newMember(t, router, pool, "viewer")
	spec := parisMan("Hugo")
	spec.minAge, spec.maxAge, spec.maxKm = 25, 35, 30
	setupProfile(t, router, viewer, spec)

	near := newMember(t, router, pool, "near")
	setupProfile(t, router, near, parisWoman("Lea"))
	lyon := newMember(t, router, pool, "lyon")
	lyonSpec := parisWoman("Manon")
	lyonSpec.lat, lyonSpec.lng = 45.764, 4.8357
	setupProfile(t, router, lyon, lyonSpec)
	older := newMember(t, router, pool, "older")
	olderSpec := parisWoman("Odile")
	olderSpec.birthdate = "1970-01-01"
	setupProfile(t, router, older, olderSpec)
	picky := newMember(t, router, pool, "picky")
	pickySpec := parisWoman("Pia")
	pickySpec.minAge, pickySpec.maxAge, pickySpec.maxKm = 40, 60, 50
	setupProfile(t, router, picky, pickySpec)
	paused := newMember(t, router, pool, "paused")
	setupProfile(t, router, paused, parisWoman("Rose"))
	expectStatus(t, doRequest(t, router, http.MethodPatch, "/me/profile", map[string]any{"discoverable": false}, paused.tok), http.StatusOK, "pause")

	feed := discover(t, router, viewer)
	if !inFeed(feed, near.id) {
		t.Fatalf("nearby compatible member should be shown")
	}
	if inFeed(feed, lyon.id) {
		t.Fatalf("members beyond max distance must be hidden")
	}
	if inFeed(feed, older.id) {
		t.Fatalf("members outside the viewer's age range must be hidden")
	}
	if inFeed(feed, picky.id) {
		t.Fatalf("members whose own age range excludes the viewer must be hidden")
	}
	if inFeed(feed, paused.id) {
		t.Fatalf("paused profiles must be hidden")
	}

	// Hide distance privacy setting.
	expectStatus(t, doRequest(t, router, http.MethodPatch, "/me/profile", map[string]any{"showDistance": false}, near.tok), http.StatusOK, "hide distance")
	for _, p := range discover(t, router, viewer).Profiles {
		if p.UserID == near.id && p.DistanceKm != nil {
			t.Fatalf("distance must be hidden when the member disabled it")
		}
	}
}

func TestProfileValidation(t *testing.T) {
	router, pool := setupTestRouter(t)
	m := newMember(t, router, pool, "validation")
	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")

	cases := []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"birthdate": minor}, "underage"},
		{map[string]any{"birthdate": "31/12/1990"}, "invalid_birthdate"},
		{map[string]any{"firstName": "<b>x</b>"}, "invalid_first_name"},
		{map[string]any{"gender": "robot"}, "invalid_gender"},
		{map[string]any{"bio": strings.Repeat("a", 501)}, "bio_too_long"},
		{map[string]any{"interests": []string{"not-a-real-interest"}}, "unknown_interest"},
	}
	for _, tc := range cases {
		resp := doRequest(t, router, http.MethodPatch, "/me/profile", tc.body, m.tok)
		if resp.Status != http.StatusBadRequest || !strings.Contains(string(resp.Body), tc.code) {
			t.Errorf("%v: expected 400 %s, got %d %s", tc.body, tc.code, resp.Status, resp.Body)
		}
	}
	expectStatus(t, doRequest(t, router, http.MethodPatch, "/me/profile", map[string]any{"birthdate": "1990-05-05"}, m.tok), http.StatusOK, "set birthdate")
	resp := doRequest(t, router, http.MethodPatch, "/me/profile", map[string]any{"birthdate": "1999-05-05"}, m.tok)
	if resp.Status != http.StatusBadRequest || !strings.Contains(string(resp.Body), "birthdate_locked") {
		t.Fatalf("birthdate must be locked once set, got %d %s", resp.Status, resp.Body)
	}
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": []string{"man"}, "minAge": 50, "maxAge": 30, "maxDistanceKm": 10,
	}, m.tok), http.StatusBadRequest, "incoherent age range")
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/location", map[string]any{"latitude": 123, "longitude": 2}, m.tok), http.StatusBadRequest, "invalid latitude")

	interests := doRequest(t, router, http.MethodGet, "/interests", nil, m.tok)
	if !strings.Contains(string(interests.Body), `"slug":"hiking"`) {
		t.Fatalf("interest catalog should be seeded: %s", interests.Body)
	}
}

func TestPhotoManagement(t *testing.T) {
	router, pool := setupTestRouter(t)
	m := newMember(t, router, pool, "photos")

	expectStatus(t, doMultipart(t, router, http.MethodPost, "/me/photos", []byte("#!/bin/sh\nrm -rf /"), m.tok), http.StatusUnsupportedMediaType, "script upload")
	expectStatus(t, doMultipart(t, router, http.MethodPost, "/me/photos", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), m.tok), http.StatusUnsupportedMediaType, "svg upload")
	expectStatus(t, doMultipart(t, router, http.MethodPost, "/me/photos", pngImage(t, 64, 64, color.RGBA{1, 2, 3, 255}), m.tok), http.StatusBadRequest, "tiny image")

	var list struct {
		Photos []photoJSON `json:"photos"`
	}
	for i := 0; i < 6; i++ {
		resp := doMultipart(t, router, http.MethodPost, "/me/photos", pngImage(t, 300, 300, color.RGBA{uint8(i * 40), 100, 100, 255}), m.tok)
		expectStatus(t, resp, http.StatusCreated, "upload")
		mustJSON(t, resp.Body, &list)
	}
	if len(list.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(list.Photos))
	}
	expectStatus(t, doMultipart(t, router, http.MethodPost, "/me/photos", pngImage(t, 300, 300, color.RGBA{9, 9, 9, 255}), m.tok), http.StatusConflict, "7th photo")

	// Reorder: last becomes primary.
	ids := []string{list.Photos[5].ID, list.Photos[0].ID, list.Photos[1].ID, list.Photos[2].ID, list.Photos[3].ID, list.Photos[4].ID}
	resp := doRequest(t, router, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": ids}, m.tok)
	expectStatus(t, resp, http.StatusOK, "reorder")
	mustJSON(t, resp.Body, &list)
	if list.Photos[0].ID != ids[0] || list.Photos[0].Position != 0 {
		t.Fatalf("reorder did not apply: %+v", list.Photos)
	}
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": ids[:3]}, m.tok), http.StatusBadRequest, "partial order")

	// Replace keeps position, old URL stops working.
	oldURL := list.Photos[1].URL
	resp = doMultipart(t, router, http.MethodPut, "/me/photos/"+list.Photos[1].ID, pngImage(t, 500, 400, color.RGBA{0, 200, 0, 255}), m.tok)
	expectStatus(t, resp, http.StatusOK, "replace")
	expectStatus(t, doRequest(t, router, http.MethodGet, oldURL, nil, ""), http.StatusOK, "same id still served after replace")

	// Delete compacts positions.
	resp = doRequest(t, router, http.MethodDelete, "/me/photos/"+list.Photos[0].ID, nil, m.tok)
	expectStatus(t, resp, http.StatusOK, "delete")
	deletedURL := list.Photos[0].URL
	mustJSON(t, resp.Body, &list)
	if len(list.Photos) != 5 {
		t.Fatalf("expected 5 photos after delete")
	}
	for i, p := range list.Photos {
		if p.Position != i {
			t.Fatalf("positions must stay dense, got %+v", list.Photos)
		}
	}
	expectStatus(t, doRequest(t, router, http.MethodGet, deletedURL, nil, ""), http.StatusNotFound, "deleted photo URL")

	// Another member cannot touch these photos.
	other := newMember(t, router, pool, "photos_other")
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/me/photos/"+list.Photos[0].ID, nil, other.tok), http.StatusNotFound, "foreign delete")
}

func TestReportsBlockAndEscalate(t *testing.T) {
	router, pool := setupTestRouter(t)
	target := newMember(t, router, pool, "reported")
	setupProfile(t, router, target, parisWoman("Zoe"))
	watcher := newMember(t, router, pool, "watcher")
	setupProfile(t, router, watcher, parisMan("Victor"))
	if !inFeed(discover(t, router, watcher), target.id) {
		t.Fatalf("target should initially be discoverable")
	}

	bad := doRequest(t, router, http.MethodPost, "/reports", map[string]string{"userId": target.id, "reason": "because"}, watcher.tok)
	expectStatus(t, bad, http.StatusBadRequest, "invalid reason")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/reports", map[string]string{"userId": watcher.id, "reason": "spam"}, watcher.tok), http.StatusBadRequest, "self report")

	for i := 0; i < 3; i++ {
		reporter := newMember(t, router, pool, fmt.Sprintf("reporter%d", i))
		resp := doRequest(t, router, http.MethodPost, "/reports", map[string]string{"userId": target.id, "reason": "fake_profile", "details": "photos volées"}, reporter.tok)
		expectStatus(t, resp, http.StatusCreated, "report")
		// Reporting twice keeps a single open report.
		expectStatus(t, doRequest(t, router, http.MethodPost, "/reports", map[string]string{"userId": target.id, "reason": "spam"}, reporter.tok), http.StatusCreated, "report again")
	}
	var open int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reported_id = $1 AND status = 'open'`, target.id).Scan(&open); err != nil || open != 3 {
		t.Fatalf("expected 3 open reports, got %d (%v)", open, err)
	}
	if inFeed(discover(t, router, watcher), target.id) {
		t.Fatalf("a profile reported by 3 members must be hidden pending review")
	}
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT moderation_status FROM profiles WHERE user_id = $1`, target.id).Scan(&status); err != nil || status != "under_review" {
		t.Fatalf("expected under_review, got %q (%v)", status, err)
	}
}

func TestAccountDeletionErasesData(t *testing.T) {
	router, pool := setupTestRouter(t)
	a := newMember(t, router, pool, "leaver")
	b := newMember(t, router, pool, "stayer")
	setupProfile(t, router, a, parisWoman("Nina"))
	setupProfile(t, router, b, parisMan("Paul"))
	swipe(t, router, a, b.id, "like")
	swipe(t, router, b, a.id, "like")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+a.id+"/messages", map[string]string{"content": "coucou"}, b.tok), http.StatusCreated, "message")

	var own ownProfileJSON
	mustJSON(t, doRequest(t, router, http.MethodGet, "/me/profile", nil, a.tok).Body, &own)
	photoURL := own.Photos[0].URL

	expectStatus(t, doRequest(t, router, http.MethodDelete, "/me", map[string]string{"password": "Wrong1234"}, a.tok), http.StatusForbidden, "delete with wrong password")
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/me", map[string]string{"password": testPassword}, a.tok), http.StatusNoContent, "delete account")

	expectStatus(t, doRequest(t, router, http.MethodGet, "/me", nil, a.tok), http.StatusUnauthorized, "token after deletion")
	expectStatus(t, doRequest(t, router, http.MethodGet, photoURL, nil, ""), http.StatusNotFound, "photo after deletion")
	if findChat(chats(t, router, b), a.id) != nil {
		t.Fatalf("deleted member's conversation must disappear")
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM messages WHERE sender_user_id = $1 OR recipient_user_id = $1`,
		`SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
		`SELECT COUNT(*) FROM photos WHERE user_id = $1`,
		`SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1`,
		`SELECT COUNT(*) FROM matches WHERE user_a_id = $1 OR user_b_id = $1`,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`,
	} {
		var n int
		if err := pool.QueryRow(context.Background(), q, a.id).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: expected 0 rows, got %d (%v)", q, n, err)
		}
	}
	login := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": a.auth.User.Email, "password": testPassword}, "")
	expectStatus(t, login, http.StatusUnauthorized, "login after deletion")
}

func TestPasswordRecoveryOverHTTP(t *testing.T) {
	router, pool := setupTestRouter(t)
	m := newMember(t, router, pool, "recover")
	mail := lastTestEnv.mail

	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/forgot", map[string]string{"email": "ghost@lueur.test"}, ""), http.StatusAccepted, "forgot unknown")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/forgot", map[string]string{"email": m.auth.User.Email}, ""), http.StatusAccepted, "forgot")
	msg, ok := mail.Last()
	if !ok || msg.To != m.auth.User.Email {
		t.Fatalf("expected a recovery mail")
	}
	code := msg.Body[strings.Index(msg.Body, ": ")+2:][:6]
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{
		"email": m.auth.User.Email, "code": code, "password": "Nouveau2026",
	}, ""), http.StatusNoContent, "reset")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/me", nil, m.tok), http.StatusUnauthorized, "old session after reset")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": m.auth.User.Email, "password": "Nouveau2026"}, ""), http.StatusOK, "login with new password")
}

func TestRealtimeStreamDeliversEvents(t *testing.T) {
	router, pool := setupTestRouter(t)
	m := newMember(t, router, pool, "stream")
	srv := httptest.NewServer(router)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer "+m.tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("unexpected stream response %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}

	deadline := time.Now().Add(2 * time.Second)
	for lastTestEnv.hub.Connections(m.id) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/notifications", map[string]string{"type": "system", "title": "Hello", "body": "Stream test"}, m.tok), http.StatusCreated, "create notification")

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"type":"notification.created"`) {
			return
		}
	}
}
