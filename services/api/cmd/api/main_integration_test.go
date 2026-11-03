package main

import (
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
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/gorilla/websocket"
)

type captureMailer struct{ tokens map[string]string }

func (m *captureMailer) SendPasswordReset(_ context.Context, email, token string) error {
	m.tokens[email] = token
	return nil
}

type env struct {
	t      *testing.T
	server *httptest.Server
	mailer *captureMailer
	upload string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must be set for integration tests")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	mailer := &captureMailer{tokens: map[string]string{}}
	dir := t.TempDir()
	router, err := setupRouter(&app{
		dbPool: pool, jwtSecret: []byte("integration-secret-0123456789abcdef-xyz"),
		uploadDir: dir, mailer: mailer,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return &env{t: t, server: srv, mailer: mailer, upload: dir}
}

func (e *env) do(method, path, token string, body any) (int, []byte) {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, e.server.URL+path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.Bytes()
}

func (e *env) expect(want int, method, path, token string, body any) []byte {
	e.t.Helper()
	status, raw := e.do(method, path, token, body)
	if status != want {
		e.t.Fatalf("%s %s: want %d got %d: %s", method, path, want, status, raw)
	}
	return raw
}

func decode[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return v
}

func (e *env) upload_(token, filename string, data []byte) (int, []byte) {
	e.t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", "image/jpeg")
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, e.server.URL+"/me/photos", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.Bytes()
}

func testPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 320, 400))
	for x := 0; x < 320; x++ {
		for y := 0; y < 400; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type user struct {
	ID, Token, Email string
	Photo            string
}

type authResp struct {
	AccessToken string `json:"accessToken"`
	User        struct {
		ID string `json:"id"`
	} `json:"user"`
}

func (e *env) newUser(name, gender, interestedIn string, complete bool) user {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d@demo.invalid", name, time.Now().UnixNano())
	ar := decode[authResp](e.t, e.expect(201, "POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": "Password123"}))
	u := user{ID: ar.User.ID, Token: ar.AccessToken, Email: email}
	e.expect(200, "PUT", "/me/profile", u.Token, map[string]any{
		"firstName": name, "birthDate": "1995-06-15", "gender": gender, "bio": "Hello", "city": "Paris",
		"interests": []string{"travel", "music"},
	})
	if !complete {
		return u
	}
	e.expect(204, "PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{interestedIn}, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 500})
	status, raw := e.upload_(u.Token, "me.jpg", testPNG())
	if status != 201 {
		e.t.Fatalf("upload: %d %s", status, raw)
	}
	u.Photo = decode[struct {
		ID string `json:"id"`
	}](e.t, raw).ID
	return u
}

type card struct {
	ID         string `json:"id"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID string `json:"id"`
	} `json:"photos"`
}

func (e *env) discoverIDs(u user) map[string]card {
	e.t.Helper()
	res := decode[struct {
		Profiles []card `json:"profiles"`
	}](e.t, e.expect(200, "GET", "/discover?limit=20", u.Token, nil))
	out := map[string]card{}
	for _, c := range res.Profiles {
		out[c.ID] = c
	}
	return out
}

func TestHealthAndAuthGuards(t *testing.T) {
	e := newEnv(t)
	e.expect(200, "GET", "/health", "", nil)
	for _, p := range []string{"/me", "/discover", "/matches", "/conversations", "/notifications", "/blocks", "/me/profile"} {
		e.expect(401, "GET", p, "", nil)
	}
	e.expect(401, "GET", "/me", "not-a-token", nil)
	e.expect(400, "POST", "/auth/register", "", map[string]string{"email": "bad", "password": "Password123"})
	e.expect(400, "POST", "/auth/register", "", map[string]string{"email": "a@b.co", "password": "short"})
}

func TestDatingFlow(t *testing.T) {
	e := newEnv(t)
	alice := e.newUser("alice", "woman", "man", true)
	bob := e.newUser("bob", "man", "woman", true)
	carol := e.newUser("carol", "woman", "man", true) // same side as alice: never matched with her
	incomplete := e.newUser("ivan", "man", "woman", false)
	t.Cleanup(func() {
		for _, u := range []user{alice, bob, carol, incomplete} {
			e.do("DELETE", "/me", u.Token, map[string]string{"password": "Password123"})
		}
	})

	// Login works with the normalised e-mail; me profile is complete.
	e.expect(200, "POST", "/auth/login", "", map[string]string{"email": bob.Email, "password": "Password123"})
	me := decode[struct {
		Complete bool     `json:"complete"`
		Missing  []string `json:"missing"`
	}](t, e.expect(200, "GET", "/me/profile", alice.Token, nil))
	if !me.Complete {
		t.Fatalf("profile should be complete: %+v", me)
	}

	// Under 18 is rejected server-side.
	e.expect(403, "PUT", "/me/profile", incomplete.Token, map[string]any{
		"firstName": "Kid", "birthDate": time.Now().AddDate(-17, 0, 0).Format("2006-01-02"), "gender": "man",
	})

	// Incomplete profile can neither swipe nor be discovered.
	e.expect(409, "POST", "/swipes", incomplete.Token, map[string]string{"targetId": alice.ID, "action": "like"})
	if _, ok := e.discoverIDs(alice)[incomplete.ID]; ok {
		t.Fatal("profile without photo must not be discoverable")
	}

	// Discovery honours mutual preferences and hides self.
	found := e.discoverIDs(alice)
	if _, ok := found[bob.ID]; !ok {
		t.Fatal("alice should discover bob")
	}
	if _, ok := found[alice.ID]; ok {
		t.Fatal("must not discover self")
	}
	if _, ok := found[carol.ID]; ok {
		t.Fatal("carol is not interested in women; mutual preference must exclude her")
	}
	if found[bob.ID].DistanceKm != nil {
		t.Fatal("no location set: distance must be null")
	}

	// Approximate distance only, never coordinates.
	e.expect(204, "PUT", "/me/location", alice.Token, map[string]float64{"latitude": 48.85661, "longitude": 2.35222})
	e.expect(204, "PUT", "/me/location", bob.Token, map[string]float64{"latitude": 48.9, "longitude": 2.4})
	raw := e.expect(200, "GET", "/discover?limit=20", alice.Token, nil)
	if bytes.Contains(raw, []byte("latitude")) || bytes.Contains(raw, []byte("lat\"")) {
		t.Fatal("coordinates must never be exposed")
	}
	if d := e.discoverIDs(alice)[bob.ID].DistanceKm; d == nil || *d%5 != 0 || *d < 5 {
		t.Fatalf("distance should be a bucket of 5 km, got %v", d)
	}
	e.expect(400, "PUT", "/me/location", alice.Token, map[string]float64{"latitude": 120, "longitude": 2})

	// Like -> no match yet, repeated like -> conflict, hidden from discovery.
	res := decode[struct {
		Matched bool `json:"matched"`
	}](t, e.expect(200, "POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}))
	if res.Matched {
		t.Fatal("no match on a one-sided like")
	}
	e.expect(409, "POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"})
	e.expect(409, "POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "pass"})
	e.expect(400, "POST", "/swipes", alice.Token, map[string]string{"targetId": alice.ID, "action": "like"})
	e.expect(400, "POST", "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "superlike"})
	if _, ok := e.discoverIDs(alice)[bob.ID]; ok {
		t.Fatal("swiped profile must not reappear")
	}
	likes := decode[struct {
		Likes []struct {
			User card `json:"user"`
		} `json:"likes"`
	}](t, e.expect(200, "GET", "/likes/received", bob.Token, nil))
	if len(likes.Likes) == 0 || likes.Likes[0].User.ID != alice.ID {
		t.Fatalf("bob should see alice's like: %+v", likes)
	}

	// Reciprocal like -> match + conversation + notifications.
	type swipeRes struct {
		Matched        bool   `json:"matched"`
		MatchID        string `json:"matchId"`
		ConversationID string `json:"conversationId"`
	}
	m := decode[swipeRes](t, e.expect(200, "POST", "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}))
	if !m.Matched || m.MatchID == "" || m.ConversationID == "" {
		t.Fatalf("expected match: %+v", m)
	}
	for _, u := range []user{alice, bob} {
		n := decode[struct {
			Notifications []struct {
				ID, Type string
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}](t, e.expect(200, "GET", "/notifications", u.Token, nil))
		if n.UnreadCount != 1 || n.Notifications[0].Type != "match" {
			t.Fatalf("expected one match notification: %+v", n)
		}
		e.expect(204, "POST", "/notifications/"+n.Notifications[0].ID+"/read", u.Token, nil)
		e.expect(404, "POST", "/notifications/"+m.MatchID+"/read", u.Token, nil)
	}
	matches := decode[struct {
		Matches []struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
		} `json:"matches"`
	}](t, e.expect(200, "GET", "/matches", alice.Token, nil))
	if len(matches.Matches) != 1 || matches.Matches[0].ConversationID != m.ConversationID {
		t.Fatalf("unexpected matches: %+v", matches)
	}

	// Realtime: bob listens on the WebSocket and receives alice's message.
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": bob.Token})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var ready map[string]any
	if err := conn.ReadJSON(&ready); err != nil || ready["type"] != "ready" {
		t.Fatalf("ws ready: %v %v", ready, err)
	}

	msgPath := "/conversations/" + m.ConversationID + "/messages"
	e.expect(400, "POST", msgPath, alice.Token, map[string]string{"body": "   "})
	e.expect(400, "POST", msgPath, alice.Token, map[string]string{"body": strings.Repeat("x", 2001)})
	e.expect(201, "POST", msgPath, alice.Token, map[string]string{"body": "  Salut Bob  "})

	var ev struct {
		Type string `json:"type"`
		Data struct {
			Body string `json:"body"`
		} `json:"data"`
	}
	if err := conn.ReadJSON(&ev); err != nil || ev.Type != "message" || ev.Data.Body != "Salut Bob" {
		t.Fatalf("ws message: %+v %v", ev, err)
	}

	// Unread / read state.
	convs := func(u user) []struct {
		ID          string `json:"id"`
		UnreadCount int    `json:"unreadCount"`
	} {
		return decode[struct {
			Conversations []struct {
				ID          string `json:"id"`
				UnreadCount int    `json:"unreadCount"`
			} `json:"conversations"`
		}](t, e.expect(200, "GET", "/conversations", u.Token, nil)).Conversations
	}
	if c := convs(bob); len(c) != 1 || c[0].UnreadCount != 1 {
		t.Fatalf("bob should have 1 unread: %+v", c)
	}
	if c := convs(alice); c[0].UnreadCount != 0 {
		t.Fatalf("sender has no unread: %+v", c)
	}
	e.expect(204, "POST", "/conversations/"+m.ConversationID+"/read", bob.Token, nil)
	if c := convs(bob); c[0].UnreadCount != 0 {
		t.Fatalf("read must clear unread: %+v", c)
	}
	page := decode[struct {
		Messages               []struct{ ID int64 } `json:"messages"`
		OtherLastReadMessageID int64                `json:"otherLastReadMessageId"`
	}](t, e.expect(200, "GET", msgPath, alice.Token, nil))
	if len(page.Messages) != 1 || page.OtherLastReadMessageID != page.Messages[0].ID {
		t.Fatalf("alice should see her message as read: %+v", page)
	}

	// Permissions: an outsider can neither read, write, read-mark nor hide.
	e.expect(404, "GET", msgPath, carol.Token, nil)
	e.expect(404, "POST", msgPath, carol.Token, map[string]string{"body": "hi"})
	e.expect(404, "POST", "/conversations/"+m.ConversationID+"/read", carol.Token, nil)
	e.expect(404, "DELETE", "/conversations/"+m.ConversationID, carol.Token, nil)
	e.expect(404, "DELETE", "/matches/"+m.MatchID, carol.Token, nil)
	e.expect(400, "GET", "/conversations/not-a-uuid/messages", alice.Token, nil)

	// Local deletion hides it for bob only, a new message re-opens it.
	e.expect(204, "DELETE", "/conversations/"+m.ConversationID, bob.Token, nil)
	if len(convs(bob)) != 0 || len(convs(alice)) != 1 {
		t.Fatal("hide must be local to the deleting user")
	}
	if p := decode[struct {
		Messages []struct{} `json:"messages"`
	}](t, e.expect(200, "GET", msgPath, bob.Token, nil)); len(p.Messages) != 0 {
		t.Fatal("cleared history must stay hidden for bob")
	}
	e.expect(201, "POST", msgPath, alice.Token, map[string]string{"body": "Tu es là ?"})
	if len(convs(bob)) != 1 {
		t.Fatal("new message must re-open the conversation")
	}

	// Photos: private to authenticated, non-blocked viewers; JPEG only.
	status, _ := e.do("GET", "/photos/"+alice.Photo, "", nil)
	if status != 401 {
		t.Fatalf("anonymous photo access must be 401, got %d", status)
	}
	e.expect(200, "GET", "/photos/"+alice.Photo, bob.Token, nil)
	if status, _ := e.upload_(alice.Token, "evil.jpg", []byte("<?php system($_GET['c']); ?>")); status != 415 {
		t.Fatalf("non-image must be 415, got %d", status)
	}
	if status, _ := e.upload_(alice.Token, "evil.png", []byte("<svg xmlns='http://www.w3.org/2000/svg' onload='alert(1)'/>")); status != 415 {
		t.Fatalf("svg must be 415, got %d", status)
	}
	second := decode[struct {
		ID string `json:"id"`
	}](t, func() []byte { s, r := e.upload_(alice.Token, "two.png", testPNG()); _ = s; return r }())
	e.expect(204, "PUT", "/me/photos/order", alice.Token, map[string]any{"ids": []string{second.ID, alice.Photo}})
	e.expect(400, "PUT", "/me/photos/order", alice.Token, map[string]any{"ids": []string{second.ID, second.ID}})
	e.expect(400, "PUT", "/me/photos/order", alice.Token, map[string]any{"ids": []string{second.ID}})
	e.expect(404, "DELETE", "/me/photos/"+second.ID, bob.Token, nil)
	e.expect(204, "DELETE", "/me/photos/"+second.ID, alice.Token, nil)
	mine := decode[struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}](t, e.expect(200, "GET", "/me/profile", alice.Token, nil))
	if len(mine.Photos) != 1 || mine.Photos[0].Position != 0 || mine.Photos[0].ID != alice.Photo {
		t.Fatalf("photos should be repacked: %+v", mine.Photos)
	}

	// Block: dissolves the match, hides profiles and photos both ways.
	e.expect(204, "POST", "/blocks", bob.Token, map[string]string{"userId": alice.ID})
	e.expect(404, "GET", msgPath, alice.Token, nil)
	e.expect(404, "POST", msgPath, alice.Token, map[string]string{"body": "hello?"})
	e.expect(404, "GET", "/profiles/"+bob.ID, alice.Token, nil)
	e.expect(404, "GET", "/photos/"+bob.Photo, alice.Token, nil)
	e.expect(404, "GET", "/photos/"+alice.Photo, bob.Token, nil)
	if _, ok := e.discoverIDs(alice)[bob.ID]; ok {
		t.Fatal("blocked pair must not be discoverable")
	}
	if got := decode[struct {
		Matches []struct{} `json:"matches"`
	}](t, e.expect(200, "GET", "/matches", alice.Token, nil)); len(got.Matches) != 0 {
		t.Fatal("block must dissolve the match")
	}
	blocked := decode[struct {
		Blocks []struct{ ID string } `json:"blocks"`
	}](t, e.expect(200, "GET", "/blocks", bob.Token, nil))
	if len(blocked.Blocks) != 1 {
		t.Fatalf("expected one block: %+v", blocked)
	}
	e.expect(400, "POST", "/blocks", bob.Token, map[string]string{"userId": bob.ID})

	// Reports: validated, deduplicated.
	e.expect(202, "POST", "/reports", bob.Token, map[string]string{"userId": alice.ID, "reason": "spam", "details": "test"})
	e.expect(202, "POST", "/reports", bob.Token, map[string]string{"userId": alice.ID, "reason": "spam"})
	e.expect(400, "POST", "/reports", bob.Token, map[string]string{"userId": alice.ID, "reason": "nope"})
	e.expect(400, "POST", "/reports", bob.Token, map[string]string{"userId": bob.ID, "reason": "spam"})

	// Unblock restores nothing automatically (swipes stay, match stays gone).
	e.expect(204, "DELETE", "/blocks/"+alice.ID, bob.Token, nil)
	if _, ok := e.discoverIDs(alice)[bob.ID]; ok {
		t.Fatal("already-swiped pair must not reappear after unblock")
	}
}

func TestSimultaneousLikesProduceExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	a := e.newUser("racea", "woman", "man", true)
	b := e.newUser("raceb", "man", "woman", true)
	t.Cleanup(func() {
		e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"})
		e.do("DELETE", "/me", b.Token, map[string]string{"password": "Password123"})
	})
	type res struct {
		Matched bool `json:"matched"`
	}
	results := make(chan bool, 2)
	for _, pair := range [][2]user{{a, b}, {b, a}} {
		go func(from, to user) {
			_, raw := e.do("POST", "/swipes", from.Token, map[string]string{"targetId": to.ID, "action": "like"})
			var r res
			_ = json.Unmarshal(raw, &r)
			results <- r.Matched
		}(pair[0], pair[1])
	}
	matched := 0
	for i := 0; i < 2; i++ {
		if <-results {
			matched++
		}
	}
	if matched != 1 {
		t.Fatalf("exactly one request must observe the match, got %d", matched)
	}
	list := decode[struct {
		Matches []struct{} `json:"matches"`
	}](t, e.expect(200, "GET", "/matches", a.Token, nil))
	if len(list.Matches) != 1 {
		t.Fatalf("expected a single match row, got %d", len(list.Matches))
	}
}

func TestUnmatch(t *testing.T) {
	e := newEnv(t)
	a := e.newUser("una", "woman", "man", true)
	b := e.newUser("unb", "man", "woman", true)
	t.Cleanup(func() {
		e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"})
		e.do("DELETE", "/me", b.Token, map[string]string{"password": "Password123"})
	})
	e.expect(200, "POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"})
	m := decode[struct {
		MatchID        string `json:"matchId"`
		ConversationID string `json:"conversationId"`
	}](t, e.expect(200, "POST", "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}))
	e.expect(204, "DELETE", "/matches/"+m.MatchID, a.Token, nil)
	e.expect(404, "GET", "/conversations/"+m.ConversationID+"/messages", b.Token, nil)
	e.expect(409, "POST", "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"})
}

func TestPasswordResetAndAccountDeletion(t *testing.T) {
	e := newEnv(t)
	u := e.newUser("reset", "woman", "man", true)
	e.expect(202, "POST", "/auth/forgot-password", "", map[string]string{"email": "nobody@demo.invalid"})
	e.expect(202, "POST", "/auth/forgot-password", "", map[string]string{"email": u.Email})
	token := e.mailer.tokens[u.Email]
	if token == "" {
		t.Fatal("reset token should have been mailed")
	}
	if _, ok := e.mailer.tokens["nobody@demo.invalid"]; ok {
		t.Fatal("unknown accounts get no mail")
	}
	e.expect(400, "POST", "/auth/reset-password", "", map[string]string{"token": "bogus", "password": "NewPassword123"})
	e.expect(204, "POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "NewPassword123"})
	e.expect(400, "POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "Another12345"})
	e.expect(401, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"})
	login := decode[authResp](t, e.expect(200, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword123"}))

	// Account deletion needs the password and erases files + data.
	e.expect(403, "DELETE", "/me", login.AccessToken, map[string]string{"password": "wrong-password"})
	files, _ := filepath.Glob(filepath.Join(e.upload, u.ID, "*"))
	if len(files) != 1 {
		t.Fatalf("expected one stored file, got %v", files)
	}
	e.expect(204, "DELETE", "/me", login.AccessToken, map[string]string{"password": "NewPassword123"})
	if files, _ := filepath.Glob(filepath.Join(e.upload, u.ID, "*")); len(files) != 0 {
		t.Fatalf("files must be erased: %v", files)
	}
	e.expect(401, "POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword123"})
	e.expect(404, "GET", "/me/profile", login.AccessToken, nil)
}

func TestRefreshRotation(t *testing.T) {
	e := newEnv(t)
	email := fmt.Sprintf("refresh_%d@demo.invalid", time.Now().UnixNano())
	reg := decode[struct {
		RefreshToken string `json:"refreshToken"`
		AccessToken  string `json:"accessToken"`
	}](t, e.expect(201, "POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"}))
	next := decode[struct {
		RefreshToken string `json:"refreshToken"`
		AccessToken  string `json:"accessToken"`
	}](t, e.expect(200, "POST", "/auth/refresh", "", map[string]string{"refreshToken": reg.RefreshToken}))
	if next.RefreshToken == reg.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(401, "POST", "/auth/refresh", "", map[string]string{"refreshToken": reg.RefreshToken})
	e.expect(204, "POST", "/auth/logout", "", map[string]string{"refreshToken": next.RefreshToken})
	e.expect(401, "POST", "/auth/refresh", "", map[string]string{"refreshToken": next.RefreshToken})
	e.expect(204, "DELETE", "/me", next.AccessToken, map[string]string{"password": "Password123"})
}
