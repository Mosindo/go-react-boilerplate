package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/testutil"
	"github.com/gorilla/websocket"
)

// End-to-end API test of the critical journey:
// register -> profile -> photos -> discovery -> like -> match -> chat,
// plus the permission rules around it. Requires DATABASE_URL_TEST.

type client struct {
	t      *testing.T
	base   string
	token  string
	userID string
	email  string
}

type response struct {
	Status int
	Body   []byte
}

func (r response) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", string(r.Body), err)
	}
}

func (c *client) do(method, path string, body any) response {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, c.base+path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return send(c.t, req)
}

func (c *client) upload(path string, data []byte) response {
	c.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", "photo.png")
	_, _ = part.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, c.base+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	return send(c.t, req)
}

func send(t *testing.T, req *http.Request) response {
	t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return response{Status: res.StatusCode, Body: body}
}

func expect(t *testing.T, r response, status int, context string) {
	t.Helper()
	if r.Status != status {
		t.Fatalf("%s: expected %d, got %d: %s", context, status, r.Status, string(r.Body))
	}
}

func testPNG(t *testing.T, shade uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 480, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 480; x++ {
			img.Set(x, y, color.RGBA{shade, uint8(x), uint8(y), 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func setupServer(t *testing.T) (*httptest.Server, func(prefix string) *client) {
	pool := testutil.DB(t)
	cfg := config.Config{
		AppEnv:    config.EnvTest,
		JWTSecret: "integration-secret-0123456789abcdef",
		UploadDir: t.TempDir(),
	}
	application, err := app.NewWithOptions(cfg, pool, app.Options{Limits: app.DisabledLimits(), Mailer: &testutil.CaptureMailer{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go application.Hub.Run(ctx)
	server := httptest.NewServer(application.Router)
	t.Cleanup(func() {
		server.Close()
		cancel()
	})

	register := func(prefix string) *client {
		c := &client{t: t, base: server.URL}
		c.email = testutil.Email(t, pool, prefix)
		r := c.do(http.MethodPost, "/auth/register", map[string]string{"email": c.email, "password": "Password123"})
		expect(t, r, http.StatusCreated, "register "+prefix)
		var auth struct {
			AccessToken string `json:"accessToken"`
			User        struct {
				ID string `json:"id"`
			} `json:"user"`
		}
		r.json(t, &auth)
		c.token, c.userID = auth.AccessToken, auth.User.ID
		return c
	}
	return server, register
}

type profileCard struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	SharedInterests int `json:"sharedInterests"`
}

func completeProfile(t *testing.T, c *client, name, birthdate, gender string, interestedIn []string, lat, lon float64, interests []int) {
	t.Helper()
	expect(t, c.do(http.MethodPatch, "/profile", map[string]any{
		"firstName": name, "birthdate": birthdate, "gender": gender, "bio": "Bonjour, je suis " + name,
		"relationshipGoal": "long_term",
	}), http.StatusOK, "update profile")
	expect(t, c.do(http.MethodPut, "/profile/preferences", map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 50,
	}), http.StatusOK, "preferences")
	expect(t, c.do(http.MethodPut, "/profile/location", map[string]any{"latitude": lat, "longitude": lon, "city": "Paris"}), http.StatusOK, "location")
	expect(t, c.do(http.MethodPut, "/profile/interests", map[string]any{"interestIds": interests}), http.StatusOK, "interests")
	expect(t, c.upload("/profile/photos", testPNG(t, 10)), http.StatusCreated, "upload photo")
}

func discover(t *testing.T, c *client) []profileCard {
	t.Helper()
	r := c.do(http.MethodGet, "/discovery?limit=20", nil)
	expect(t, r, http.StatusOK, "discovery")
	var payload struct {
		Profiles []profileCard `json:"profiles"`
	}
	r.json(t, &payload)
	return payload.Profiles
}

func findCard(cards []profileCard, userID string) *profileCard {
	for i := range cards {
		if cards[i].UserID == userID {
			return &cards[i]
		}
	}
	return nil
}

func TestHealth(t *testing.T) {
	server, _ := setupServer(t)
	res, err := http.Get(server.URL + "/health")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("health: %v %v", err, res)
	}
}

func TestDatingCriticalFlow(t *testing.T) {
	server, register := setupServer(t)
	alex := register("alex")
	sam := register("sam")
	outsider := register("outsider")

	// Unauthenticated and malformed requests.
	anon := &client{t: t, base: server.URL}
	expect(t, anon.do(http.MethodGet, "/discovery", nil), http.StatusUnauthorized, "anonymous discovery")
	expect(t, alex.do(http.MethodGet, "/conversations/not-a-uuid/messages", nil), http.StatusBadRequest, "invalid uuid")
	expect(t, alex.do(http.MethodGet, "/me", nil), http.StatusOK, "me")

	// Discovery requires a complete profile.
	expect(t, alex.do(http.MethodGet, "/discovery", nil), http.StatusConflict, "incomplete discovery")

	var interests struct {
		Interests []struct {
			ID int `json:"id"`
		} `json:"interests"`
	}
	r := alex.do(http.MethodGet, "/interests", nil)
	expect(t, r, http.StatusOK, "interests")
	r.json(t, &interests)
	if len(interests.Interests) < 10 {
		t.Fatalf("expected seeded interests, got %d", len(interests.Interests))
	}
	i1, i2, i3 := interests.Interests[0].ID, interests.Interests[1].ID, interests.Interests[2].ID

	completeProfile(t, alex, "Alex", "1994-05-10", "man", []string{"woman"}, 48.8566, 2.3522, []int{i1, i2})
	completeProfile(t, sam, "Sam", "1996-02-20", "woman", []string{"man"}, 48.8700, 2.3300, []int{i1, i3})
	completeProfile(t, outsider, "Noa", "1990-01-01", "man", []string{"man"}, 48.85, 2.35, []int{i3})

	// Profile completeness and privacy of the own profile.
	var own struct {
		Completeness struct {
			Complete bool `json:"complete"`
		} `json:"completeness"`
		Photos []struct{ ID string } `json:"photos"`
	}
	alex.do(http.MethodGet, "/profile", nil).json(t, &own)
	if !own.Completeness.Complete || len(own.Photos) != 1 {
		t.Fatalf("alex profile should be complete with one photo: %+v", own)
	}

	// Birthdate is immutable once set; minors are rejected.
	expect(t, alex.do(http.MethodPatch, "/profile", map[string]any{"birthdate": "1990-01-01"}), http.StatusConflict, "birthdate locked")

	// Discovery: mutual preferences and privacy of the payload.
	r = alex.do(http.MethodGet, "/discovery?limit=20", nil)
	expect(t, r, http.StatusOK, "alex discovery")
	for _, forbidden := range []string{sam.email, "latitude", "longitude", "birthdate", "48.87"} {
		if strings.Contains(string(r.Body), forbidden) {
			t.Fatalf("discovery payload leaks %q: %s", forbidden, r.Body)
		}
	}
	alexCards := discover(t, alex)
	samCard := findCard(alexCards, sam.userID)
	if samCard == nil {
		t.Fatal("alex should discover sam")
	}
	if findCard(alexCards, outsider.userID) != nil {
		t.Fatal("gender preferences must be mutual: outsider must not appear")
	}
	if samCard.DistanceKm == nil || *samCard.DistanceKm < 2 || samCard.SharedInterests != 1 || samCard.Age < 18 {
		t.Fatalf("unexpected card: %+v", samCard)
	}

	// Signed photo URL works; tampered URL does not.
	photoURL := samCard.Photos[0].URL
	res, err := http.Get(server.URL + photoURL)
	if err != nil || res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("signed photo url: %v %+v", err, res)
	}
	res.Body.Close()
	res, _ = http.Get(server.URL + strings.Replace(photoURL, "sig=", "sig=x", 1))
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("tampered photo url must be forbidden, got %d", res.StatusCode)
	}
	res.Body.Close()

	// Like without reciprocity, duplicates refused.
	var swipe struct {
		Matched bool `json:"matched"`
		Match   *struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
		} `json:"match"`
	}
	r = alex.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": sam.userID, "action": "like"})
	expect(t, r, http.StatusOK, "alex likes sam")
	r.json(t, &swipe)
	if swipe.Matched {
		t.Fatal("no match expected before reciprocity")
	}
	expect(t, alex.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": sam.userID, "action": "like"}), http.StatusConflict, "duplicate like")
	expect(t, alex.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": alex.userID, "action": "like"}), http.StatusBadRequest, "self like")
	expect(t, alex.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": sam.userID, "action": "superlike"}), http.StatusBadRequest, "invalid action")
	if findCard(discover(t, alex), sam.userID) != nil {
		t.Fatal("a swiped profile must never reappear")
	}

	// Sam gets Alex ranked first (Alex already liked Sam).
	samCards := discover(t, sam)
	if len(samCards) == 0 || samCards[0].UserID != alex.userID {
		t.Fatalf("alex should be ranked first for sam: %+v", samCards)
	}

	// Realtime: Alex listens before the match happens.
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	alex.do(http.MethodPost, "/realtime/ticket", nil).json(t, &ticket)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime?ticket=" + ticket.Ticket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	defer ws.Close()
	if _, _, err := websocket.DefaultDialer.Dial(strings.Replace(wsURL, "ticket=", "ticket=bad", 1), nil); err == nil {
		t.Fatal("websocket must refuse an invalid ticket")
	}
	events := make(chan map[string]any, 16)
	go func() {
		for {
			var evt map[string]any
			if err := ws.ReadJSON(&evt); err != nil {
				close(events)
				return
			}
			events <- evt
		}
	}()
	waitEvent := func(eventType string) map[string]any {
		t.Helper()
		timeout := time.After(5 * time.Second)
		for {
			select {
			case evt, ok := <-events:
				if !ok {
					t.Fatalf("websocket closed while waiting for %s", eventType)
				}
				if evt["type"] == eventType {
					return evt
				}
			case <-timeout:
				t.Fatalf("timeout waiting for realtime event %s", eventType)
			}
		}
	}
	waitEvent("connected")

	// Reciprocal like creates exactly one match.
	r = sam.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": alex.userID, "action": "like"})
	expect(t, r, http.StatusOK, "sam likes alex")
	r.json(t, &swipe)
	if !swipe.Matched || swipe.Match == nil || swipe.Match.ConversationID == "" {
		t.Fatalf("expected match: %s", r.Body)
	}
	conversationID := swipe.Match.ConversationID
	waitEvent("match.created")

	// Conversations list.
	var convs struct {
		Conversations []struct {
			ID   string `json:"id"`
			User struct {
				UserID    string `json:"userId"`
				FirstName string `json:"firstName"`
			} `json:"user"`
			UnreadCount int `json:"unreadCount"`
		} `json:"conversations"`
	}
	alex.do(http.MethodGet, "/conversations", nil).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].ID != conversationID || convs.Conversations[0].User.FirstName != "Sam" {
		t.Fatalf("unexpected conversations: %+v", convs)
	}

	// Messaging, realtime delivery and permissions.
	msgPath := "/conversations/" + conversationID + "/messages"
	expect(t, alex.do(http.MethodPost, msgPath, map[string]string{"body": "   "}), http.StatusBadRequest, "empty message")
	expect(t, alex.do(http.MethodPost, msgPath, map[string]string{"body": strings.Repeat("a", 2001)}), http.StatusBadRequest, "long message")
	expect(t, sam.do(http.MethodPost, msgPath, map[string]string{"body": "Salut Alex !"}), http.StatusCreated, "sam sends")
	evt := waitEvent("message.created")
	if data, _ := evt["data"].(map[string]any); data["body"] != "Salut Alex !" {
		t.Fatalf("unexpected realtime payload: %+v", evt)
	}
	expect(t, outsider.do(http.MethodGet, msgPath, nil), http.StatusNotFound, "outsider reads")
	expect(t, outsider.do(http.MethodPost, msgPath, map[string]string{"body": "intrusion"}), http.StatusNotFound, "outsider writes")
	expect(t, outsider.do(http.MethodPost, "/conversations/"+conversationID+"/read", nil), http.StatusNotFound, "outsider marks read")

	alex.do(http.MethodGet, "/conversations", nil).json(t, &convs)
	if convs.Conversations[0].UnreadCount != 1 {
		t.Fatalf("alex should have 1 unread message, got %d", convs.Conversations[0].UnreadCount)
	}
	expect(t, alex.do(http.MethodPost, "/conversations/"+conversationID+"/read", nil), http.StatusNoContent, "alex reads")
	var msgs struct {
		Messages []struct {
			Body     string `json:"body"`
			SenderID string `json:"senderId"`
		} `json:"messages"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	sam.do(http.MethodGet, msgPath, nil).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.OtherLastReadAt == nil {
		t.Fatalf("sam should see the read receipt: %+v", msgs)
	}

	// Pagination with cursor.
	for i := 0; i < 5; i++ {
		expect(t, alex.do(http.MethodPost, msgPath, map[string]string{"body": "message " + string(rune('A'+i))}), http.StatusCreated, "alex sends")
	}
	var page struct {
		Messages   []struct{ Body string } `json:"messages"`
		NextCursor *string                 `json:"nextCursor"`
	}
	sam.do(http.MethodGet, msgPath+"?limit=4", nil).json(t, &page)
	if len(page.Messages) != 4 || page.NextCursor == nil || page.Messages[0].Body != "message E" {
		t.Fatalf("unexpected first page: %+v", page)
	}
	sam.do(http.MethodGet, msgPath+"?limit=4&cursor="+*page.NextCursor, nil).json(t, &page)
	if len(page.Messages) != 2 || page.NextCursor != nil {
		t.Fatalf("unexpected second page: %+v", page)
	}

	// Notifications: sam got a match notification, alex a deduplicated message one.
	var notifs struct {
		Notifications []struct {
			Type string `json:"type"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	sam.do(http.MethodGet, "/notifications", nil).json(t, &notifs)
	if len(notifs.Notifications) == 0 || notifs.UnreadCount == 0 {
		t.Fatalf("sam should have notifications: %+v", notifs)
	}
	messageNotifs := 0
	for _, n := range notifs.Notifications {
		if n.Type == "message" {
			messageNotifs++
		}
	}
	if messageNotifs != 1 {
		t.Fatalf("message notifications must be deduplicated, got %d", messageNotifs)
	}
	expect(t, sam.do(http.MethodPost, "/notifications/read-all", nil), http.StatusNoContent, "read all")

	// Public profile visibility.
	expect(t, alex.do(http.MethodGet, "/profiles/"+sam.userID, nil), http.StatusOK, "view match profile")

	// Local deletion hides the conversation only for the caller.
	expect(t, alex.do(http.MethodDelete, "/conversations/"+conversationID, nil), http.StatusNoContent, "hide conversation")
	alex.do(http.MethodGet, "/conversations", nil).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("hidden conversation must disappear for alex")
	}
	sam.do(http.MethodGet, "/conversations", nil).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("hiding must not affect sam")
	}
	expect(t, sam.do(http.MethodPost, msgPath, map[string]string{"body": "Tu es là ?"}), http.StatusCreated, "sam sends again")
	alex.do(http.MethodGet, "/conversations", nil).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("a new message must bring the conversation back")
	}
	alex.do(http.MethodGet, msgPath, nil).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].Body != "Tu es là ?" {
		t.Fatalf("messages before local deletion must stay hidden: %+v", msgs.Messages)
	}

	// Block: match and conversation disappear, profiles become invisible.
	expect(t, sam.do(http.MethodPost, "/blocks", map[string]string{"userId": alex.userID}), http.StatusNoContent, "sam blocks alex")
	expect(t, alex.do(http.MethodGet, msgPath, nil), http.StatusNotFound, "blocked conversation")
	expect(t, alex.do(http.MethodGet, "/profiles/"+sam.userID, nil), http.StatusNotFound, "blocked profile")
	var blocks struct {
		Blocks []struct {
			User struct {
				UserID string `json:"userId"`
			} `json:"user"`
		} `json:"blocks"`
	}
	sam.do(http.MethodGet, "/blocks", nil).json(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].User.UserID != alex.userID {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}

	// Report (auto-blocks by default) and validation.
	expect(t, outsider.do(http.MethodPost, "/reports", map[string]any{"userId": sam.userID, "reason": "not-a-reason"}), http.StatusBadRequest, "invalid report")
	expect(t, outsider.do(http.MethodPost, "/reports", map[string]any{"userId": sam.userID, "reason": "spam", "details": "liens suspects"}), http.StatusCreated, "report")
	expect(t, outsider.do(http.MethodPost, "/reports", map[string]any{"userId": sam.userID, "reason": "spam"}), http.StatusCreated, "duplicate report is idempotent")

	// Photos management.
	expect(t, alex.upload("/profile/photos", testPNG(t, 200)), http.StatusCreated, "second photo")
	var photos struct {
		Photos []struct{ ID string } `json:"photos"`
	}
	alex.do(http.MethodGet, "/profile/photos", nil).json(t, &photos)
	if len(photos.Photos) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(photos.Photos))
	}
	first, second := photos.Photos[0].ID, photos.Photos[1].ID
	alex.do(http.MethodPut, "/profile/photos/order", map[string]any{"photoIds": []string{second, first}}).json(t, &photos)
	if photos.Photos[0].ID != second {
		t.Fatal("reorder must change the main photo")
	}
	expect(t, alex.do(http.MethodPut, "/profile/photos/order", map[string]any{"photoIds": []string{second}}), http.StatusBadRequest, "partial reorder")
	expect(t, sam.do(http.MethodDelete, "/profile/photos/"+first, nil), http.StatusNotFound, "delete someone else's photo")
	expect(t, alex.upload("/profile/photos", []byte("<svg onload=alert(1)>")), http.StatusUnsupportedMediaType, "reject svg")
	alex.do(http.MethodDelete, "/profile/photos/"+first, nil).json(t, &photos)
	if len(photos.Photos) != 1 {
		t.Fatal("photo should be deleted")
	}

	// Account deletion revokes access immediately.
	expect(t, alex.do(http.MethodDelete, "/me", map[string]string{"password": "WrongPass1"}), http.StatusForbidden, "delete with wrong password")
	expect(t, alex.do(http.MethodDelete, "/me", map[string]string{"password": "Password123"}), http.StatusNoContent, "delete account")
	expect(t, alex.do(http.MethodGet, "/me", nil), http.StatusUnauthorized, "deleted account token")
}

func TestUnmatchRemovesConversation(t *testing.T) {
	_, register := setupServer(t)
	a := register("unmatch_a")
	b := register("unmatch_b")
	completeProfile(t, a, "Lina", "1992-03-03", "woman", []string{"woman"}, 45.76, 4.83, []int{1})
	completeProfile(t, b, "Maya", "1993-04-04", "woman", []string{"woman"}, 45.75, 4.84, []int{1})

	expect(t, a.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": b.userID, "action": "like"}), http.StatusOK, "a likes b")
	var swipe struct {
		Match struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
		} `json:"match"`
	}
	b.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": a.userID, "action": "like"}).json(t, &swipe)
	if swipe.Match.ID == "" {
		t.Fatal("expected match")
	}
	expect(t, a.do(http.MethodDelete, "/matches/"+swipe.Match.ID, nil), http.StatusNoContent, "unmatch")
	expect(t, b.do(http.MethodGet, "/conversations/"+swipe.Match.ConversationID+"/messages", nil), http.StatusNotFound, "conversation gone")
	expect(t, a.do(http.MethodDelete, "/matches/"+swipe.Match.ID, nil), http.StatusNotFound, "double unmatch")
	if findCard(discover(t, b), a.userID) != nil || findCard(discover(t, a), b.userID) != nil {
		t.Fatal("unmatched profiles must not come back in discovery")
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	_, register := setupServer(t)
	for i := 0; i < 5; i++ {
		a := register("race_a")
		b := register("race_b")
		completeProfile(t, a, "Hugo", "1991-01-01", "man", []string{"man"}, 43.6, 1.44, []int{1})
		completeProfile(t, b, "Tom", "1992-02-02", "man", []string{"man"}, 43.61, 1.45, []int{1})

		results := make(chan bool, 2)
		like := func(from, to *client) {
			var swipe struct {
				Matched bool `json:"matched"`
			}
			r := from.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": to.userID, "action": "like"})
			if r.Status != http.StatusOK {
				t.Errorf("like failed: %d %s", r.Status, r.Body)
			}
			_ = json.Unmarshal(r.Body, &swipe)
			results <- swipe.Matched
		}
		go like(a, b)
		go like(b, a)
		matched := 0
		for j := 0; j < 2; j++ {
			if <-results {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("iteration %d: expected exactly one match notification, got %d", i, matched)
		}
		var convs struct {
			Conversations []struct{ ID string } `json:"conversations"`
		}
		a.do(http.MethodGet, "/conversations", nil).json(t, &convs)
		if len(convs.Conversations) != 1 {
			t.Fatalf("iteration %d: expected one conversation, got %d", i, len(convs.Conversations))
		}
	}
}

func TestModerationFlow(t *testing.T) {
	_, register := setupServer(t)
	pool := testutil.DB(t)
	mod := register("moderator")
	reporter := register("reporter")
	target := register("target")
	completeProfile(t, reporter, "Rita", "1990-05-05", "woman", []string{"man"}, 48.85, 2.35, []int{1})
	completeProfile(t, target, "Tony", "1989-06-06", "man", []string{"woman"}, 48.86, 2.34, []int{1})

	// Not a moderator yet.
	expect(t, mod.do(http.MethodGet, "/moderation/reports", nil), http.StatusForbidden, "member cannot moderate")
	if _, err := pool.Exec(context.Background(), `UPDATE users SET role = 'moderator' WHERE id = $1`, mod.userID); err != nil {
		t.Fatal(err)
	}

	expect(t, reporter.do(http.MethodPost, "/reports", map[string]any{"userId": target.userID, "reason": "harassment", "details": "messages insistants", "block": false}), http.StatusCreated, "report")
	var list struct {
		Reports []struct {
			ID           string `json:"id"`
			Reason       string `json:"reason"`
			ReportedUser struct {
				UserID string `json:"userId"`
			} `json:"reportedUser"`
		} `json:"reports"`
	}
	r := mod.do(http.MethodGet, "/moderation/reports", nil)
	expect(t, r, http.StatusOK, "list reports")
	if strings.Contains(string(r.Body), reporter.userID) || strings.Contains(string(r.Body), reporter.email) {
		t.Fatal("reporter identity must not be exposed to moderators")
	}
	r.json(t, &list)
	var reportID string
	for _, rep := range list.Reports {
		if rep.ReportedUser.UserID == target.userID {
			reportID = rep.ID
		}
	}
	if reportID == "" {
		t.Fatalf("report not listed: %s", r.Body)
	}

	// Suspension: sessions revoked, login refused, hidden from discovery.
	expect(t, mod.do(http.MethodPost, "/moderation/users/"+mod.userID+"/suspend", nil), http.StatusForbidden, "self suspension")
	expect(t, mod.do(http.MethodPost, "/moderation/users/"+target.userID+"/suspend", nil), http.StatusNoContent, "suspend")
	expect(t, target.do(http.MethodGet, "/me", nil), http.StatusUnauthorized, "suspended session revoked")
	login := (&client{t: t, base: target.base}).do(http.MethodPost, "/auth/login", map[string]string{"email": target.email, "password": "Password123"})
	expect(t, login, http.StatusForbidden, "suspended login")
	if findCard(discover(t, reporter), target.userID) != nil {
		t.Fatal("suspended member must not be discoverable")
	}
	expect(t, mod.do(http.MethodPost, "/moderation/reports/"+reportID+"/resolve", map[string]string{"status": "dismissed"}), http.StatusNotFound, "report already closed by suspension")

	expect(t, mod.do(http.MethodPost, "/moderation/users/"+target.userID+"/unsuspend", nil), http.StatusNoContent, "unsuspend")
	expect(t, (&client{t: t, base: target.base}).do(http.MethodPost, "/auth/login", map[string]string{"email": target.email, "password": "Password123"}), http.StatusOK, "login after unsuspend")
}

func TestPushTokenRegistration(t *testing.T) {
	_, register := setupServer(t)
	u := register("pushtoken")
	expect(t, u.do(http.MethodPut, "/push-tokens", map[string]string{"token": "<script>", "platform": "ios"}), http.StatusBadRequest, "invalid token")
	expect(t, u.do(http.MethodPut, "/push-tokens", map[string]string{"token": "ExponentPushToken[abcdefghijklmnop]", "platform": "web"}), http.StatusBadRequest, "invalid platform")
	expect(t, u.do(http.MethodPut, "/push-tokens", map[string]string{"token": "ExponentPushToken[abcdefghijklmnop]", "platform": "ios"}), http.StatusNoContent, "register token")
	expect(t, u.do(http.MethodPut, "/push-tokens", map[string]string{"token": "ExponentPushToken[abcdefghijklmnop]", "platform": "ios"}), http.StatusNoContent, "idempotent")
	expect(t, u.do(http.MethodDelete, "/push-tokens", map[string]string{"token": "ExponentPushToken[abcdefghijklmnop]"}), http.StatusNoContent, "unregister")
}
