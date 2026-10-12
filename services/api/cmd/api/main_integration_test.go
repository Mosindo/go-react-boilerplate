package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-secret-0123456789abcdef0123456789"

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	server *httptest.Server
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must be set for integration tests")
	}
	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, dsn, 10*time.Second, time.Second)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	gin.SetMode(gin.TestMode)
	log.SetOutput(io.Discard)
	r, err := setupRouter(&app{dbPool: pool, jwtSecret: []byte(testSecret), uploadDir: t.TempDir()})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &env{t: t, pool: pool, router: r, server: srv}
}

type user struct {
	id, email, token, refresh string
}

type resp struct {
	status int
	body   []byte
}

func (r resp) json(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.body, dst); err != nil {
		t.Fatalf("decode %q: %v", r.body, err)
	}
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.RemoteAddr = fmt.Sprintf("10.1.%d.%d:1000", time.Now().UnixNano()%250, time.Now().UnixNano()/7%250)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{status: w.Code, body: w.Body.Bytes()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.status != status {
		e.t.Fatalf("expected HTTP %d, got %d: %s", status, r.status, r.body)
	}
	return r
}

func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *env) upload(method, path, token, filename string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", filename)
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = fmt.Sprintf("10.2.%d.%d:1000", time.Now().UnixNano()%250, time.Now().UnixNano()/7%250)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{status: w.Code, body: w.Body.Bytes()}
}

var emailCounter int

func (e *env) register(prefix string) user {
	e.t.Helper()
	emailCounter++
	email := fmt.Sprintf("%s-%d-%d@itest.invalid", prefix, time.Now().UnixNano(), emailCounter)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": "Password123"}), http.StatusCreated)
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	r.json(e.t, &out)
	if out.User.Email != email {
		e.t.Fatalf("email should be normalised, got %q", out.User.Email)
	}
	e.t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, out.User.ID)
	})
	return user{id: out.User.ID, email: email, token: out.AccessToken, refresh: out.RefreshToken}
}

// onboard completes profile, preferences, a photo and a location near Paris.
func (e *env) onboard(u user, name, gender string, interestedIn []string, birth string, lat, lon float64) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", u.token, map[string]any{
		"firstName": name, "birthDate": birth, "gender": gender, "bio": "Hello", "city": "Paris", "interests": []string{"music", "travel"},
	}), http.StatusOK)
	e.expect(e.do("PUT", "/me/preferences", u.token, map[string]any{
		"interestedIn": interestedIn, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 100,
	}), http.StatusOK)
	e.expect(e.do("PUT", "/me/location", u.token, map[string]any{"latitude": lat, "longitude": lon}), http.StatusOK)
	e.expect(e.upload("POST", "/me/photos", u.token, "a.jpg", testJPEG(e.t, 900, 600)), http.StatusCreated)
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), http.StatusOK)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	u := e.register("auth")

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": u.email, "password": "Password123"}), http.StatusConflict)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "bad", "password": "Password123"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.email, "password": "wrong-password"}), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", "", nil), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", "garbage", nil), http.StatusUnauthorized)
	e.expect(e.do("GET", "/me", u.token, nil), http.StatusOK)

	login := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.email, "password": "Password123"}), http.StatusOK)
	var tokens struct {
		AccessToken, RefreshToken string
	}
	login.json(t, &tokens)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": tokens.RefreshToken}), http.StatusOK)
	// Refresh tokens rotate: the old one must be dead.
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": tokens.RefreshToken}), http.StatusUnauthorized)

	// Logout revokes the session immediately, even though the JWT has not expired.
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": u.refresh}), http.StatusNoContent)
	e.expect(e.do("GET", "/me", u.token, nil), http.StatusUnauthorized)

	// Forgot password never reveals whether the account exists.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "nobody-here@itest.invalid"}), http.StatusAccepted)
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.email}), http.StatusAccepted)
}

func TestPasswordResetFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("reset")
	// Plant a known reset token (the real one only ever travels by e-mail).
	const token = "known-reset-token-for-test"
	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, encode(sha256($2::bytea), 'hex'), NOW() + INTERVAL '1 hour')`, u.id, token)
	if err != nil {
		t.Fatal(err)
	}
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": "wrong", "newPassword": "NewPassword456"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": token, "newPassword": "short"}), http.StatusBadRequest)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": token, "newPassword": "NewPassword456"}), http.StatusNoContent)
	// single use, sessions revoked, new password works
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": token, "newPassword": "Another789pass"}), http.StatusBadRequest)
	e.expect(e.do("GET", "/me", u.token, nil), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.email, "password": "Password123"}), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.email, "password": "NewPassword456"}), http.StatusOK)
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("prof")
	put := func(mod func(m map[string]any)) resp {
		m := map[string]any{"firstName": "Alex", "birthDate": "1995-05-05", "gender": "woman", "bio": "", "city": "", "interests": []string{}}
		mod(m)
		return e.do("PUT", "/me/profile", u.token, m)
	}
	underage := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(put(func(m map[string]any) { m["birthDate"] = underage }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) { m["birthDate"] = "05/05/1995" }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) { m["firstName"] = "" }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) { m["firstName"] = "<script>" }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) { m["gender"] = "robot" }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) { m["bio"] = strings.Repeat("x", 501) }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) { m["interests"] = []string{"does-not-exist"} }), http.StatusUnprocessableEntity)
	e.expect(put(func(m map[string]any) {}), http.StatusOK)
	// Age gate cannot be bypassed by editing the birth date afterwards.
	e.expect(put(func(m map[string]any) { m["birthDate"] = "1990-01-01" }), http.StatusConflict)

	e.expect(e.do("PUT", "/me/preferences", u.token, map[string]any{"interestedIn": []string{}, "ageMin": 18, "ageMax": 40, "maxDistanceKm": 10}), http.StatusUnprocessableEntity)
	e.expect(e.do("PUT", "/me/preferences", u.token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 17, "ageMax": 40, "maxDistanceKm": 10}), http.StatusUnprocessableEntity)
	e.expect(e.do("PUT", "/me/preferences", u.token, map[string]any{"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 20, "maxDistanceKm": 10}), http.StatusUnprocessableEntity)
	e.expect(e.do("PUT", "/me/location", u.token, map[string]any{"latitude": 91, "longitude": 0}), http.StatusUnprocessableEntity)

	// Discovery and swipes are refused until onboarding is complete.
	e.expect(e.do("GET", "/discover", u.token, nil), http.StatusForbidden)
	e.expect(e.do("POST", "/swipes", u.token, map[string]any{"targetId": u.id, "action": "like"}), http.StatusUnprocessableEntity)
}

func TestLocationIsRoundedAndNotLeaked(t *testing.T) {
	e := newEnv(t)
	a := e.register("loc-a")
	b := e.register("loc-b")
	e.onboard(a, "Ana", "woman", []string{"man"}, "1994-03-03", 48.856614, 2.352222)
	e.onboard(b, "Ben", "man", []string{"woman"}, "1992-04-04", 48.9011, 2.4011)

	var lat, lon float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.id).Scan(&lat, &lon); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lon != 2.35 {
		t.Fatalf("coordinates must be rounded to 0.01 degree, stored %v,%v", lat, lon)
	}
	r := e.expect(e.do("GET", "/profiles/"+b.id, a.token, nil), http.StatusOK)
	body := string(r.body)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "48.9"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public profile must not expose %q: %s", forbidden, body)
		}
	}
	var card struct {
		DistanceKm *int `json:"distanceKm"`
		Age        int  `json:"age"`
	}
	r.json(t, &card)
	if card.DistanceKm == nil || *card.DistanceKm < 1 {
		t.Fatalf("expected an approximate distance bucket, got %v", card.DistanceKm)
	}
	// Hiding distance removes it.
	e.expect(e.do("PUT", "/me/privacy", b.token, map[string]any{"showDistance": false}), http.StatusOK)
	r = e.expect(e.do("GET", "/profiles/"+b.id, a.token, nil), http.StatusOK)
	r.json(t, &card)
	if card.DistanceKm != nil {
		t.Fatal("distance must be hidden when the user opted out")
	}
}

func discoverIDs(t *testing.T, e *env, u user) []string {
	t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", u.token, nil), http.StatusOK)
	var out struct {
		Items []struct {
			UserID string `json:"userId"`
		} `json:"items"`
	}
	r.json(t, &out)
	ids := []string{}
	for _, it := range out.Items {
		ids = append(ids, it.UserID)
	}
	return ids
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// TestDatingJourney is the critical path:
// register -> profile -> discover -> like -> match -> conversation -> safety -> deletion.
func TestDatingJourney(t *testing.T) {
	e := newEnv(t)
	ana := e.register("ana")
	ben := e.register("ben")
	cleo := e.register("cleo") // woman interested in women: must never be shown to Ben
	far := e.register("far")   // too far away

	e.onboard(ana, "Ana", "woman", []string{"man"}, "1994-03-03", 48.8566, 2.3522)
	e.onboard(ben, "Ben", "man", []string{"woman"}, "1992-04-04", 48.87, 2.36)
	e.onboard(cleo, "Cleo", "woman", []string{"woman"}, "1993-01-01", 48.86, 2.35)
	e.onboard(far, "Fay", "woman", []string{"man"}, "1993-01-01", 40.7128, -74.006)

	// --- discovery honours mutual preferences and distance
	benFeed := discoverIDs(t, e, ben)
	if !contains(benFeed, ana.id) {
		t.Fatalf("Ben should discover Ana, got %v", benFeed)
	}
	if contains(benFeed, cleo.id) {
		t.Fatal("Cleo is not interested in men and must not be shown")
	}
	if contains(benFeed, far.id) {
		t.Fatal("profiles beyond the max distance must not be shown")
	}
	if contains(benFeed, ben.id) {
		t.Fatal("own profile must not be shown")
	}

	// --- like / duplicate / no premature match
	r := e.expect(e.do("POST", "/swipes", ben.token, map[string]any{"targetId": ana.id, "action": "like"}), http.StatusOK)
	var sw struct {
		Matched          bool    `json:"matched"`
		MatchID          *string `json:"matchId"`
		AlreadyProcessed bool    `json:"alreadyProcessed"`
	}
	r.json(t, &sw)
	if sw.Matched || sw.AlreadyProcessed {
		t.Fatalf("a one-sided like must not match: %+v", sw)
	}
	r = e.expect(e.do("POST", "/swipes", ben.token, map[string]any{"targetId": ana.id, "action": "like"}), http.StatusOK)
	r.json(t, &sw)
	if !sw.AlreadyProcessed {
		t.Fatal("repeated like must be reported as already processed")
	}
	if contains(discoverIDs(t, e, ben), ana.id) {
		t.Fatal("a processed profile must not reappear")
	}
	var swipeRows int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM swipes WHERE swiper_id=$1 AND target_id=$2`, ben.id, ana.id).Scan(&swipeRows)
	if swipeRows != 1 {
		t.Fatalf("expected exactly one swipe row, got %d", swipeRows)
	}
	// Ana sees Ben first (he already liked her).
	if feed := discoverIDs(t, e, ana); len(feed) == 0 || feed[0] != ben.id {
		t.Fatalf("a profile that liked you should be ranked first, got %v", feed)
	}

	// --- invalid swipes
	e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": ana.id, "action": "like"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": ben.id, "action": "superlike"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": "not-a-uuid", "action": "like"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": "00000000-0000-4000-8000-000000000000", "action": "like"}), http.StatusNotFound)

	// --- realtime: both connect before the match
	listen := func(u user) func(typ string) string {
		wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("ws dial: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		_ = conn.WriteJSON(map[string]string{"type": "auth", "token": u.token})
		events := make(chan string, 32)
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					close(events)
					return
				}
				events <- string(msg)
			}
		}()
		wait := func(typ string) string {
			t.Helper()
			deadline := time.After(3 * time.Second)
			for {
				select {
				case m, ok := <-events:
					if !ok {
						t.Fatalf("socket closed while waiting for %s", typ)
					}
					if strings.Contains(m, `"type":"`+typ+`"`) {
						return m
					}
				case <-deadline:
					t.Fatalf("timed out waiting for %s", typ)
				}
			}
		}
		wait("ready")
		return wait
	}
	anaEvents := listen(ana)
	benEvents := listen(ben)
	waitEvent := anaEvents

	// --- mutual like creates the match (and only one)
	r = e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": ben.id, "action": "like"}), http.StatusOK)
	r.json(t, &sw)
	if !sw.Matched || sw.MatchID == nil {
		t.Fatalf("mutual like must create a match: %+v", sw)
	}
	matchID := *sw.MatchID
	anaEvents("match.new")
	benEvents("match.new")
	var matchRows int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE id=$1`, matchID).Scan(&matchRows)
	if matchRows != 1 {
		t.Fatal("expected a single match row")
	}

	// --- notifications for both
	for _, u := range []user{ana, ben} {
		r = e.expect(e.do("GET", "/notifications", u.token, nil), http.StatusOK)
		var n struct {
			UnreadCount int `json:"unreadCount"`
			Items       []struct {
				ID, Kind string
			} `json:"items"`
		}
		r.json(t, &n)
		if n.UnreadCount != 1 || len(n.Items) != 1 || n.Items[0].Kind != "match" {
			t.Fatalf("expected one unread match notification, got %+v", n)
		}
	}

	// --- chat permissions: a stranger can neither read nor write
	e.expect(e.do("GET", "/matches/"+matchID+"/messages", cleo.token, nil), http.StatusNotFound)
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", cleo.token, map[string]any{"body": "hi"}), http.StatusNotFound)
	e.expect(e.do("POST", "/matches/"+matchID+"/read", cleo.token, nil), http.StatusNotFound)
	e.expect(e.do("DELETE", "/matches/"+matchID+"/messages", cleo.token, nil), http.StatusNotFound)
	e.expect(e.do("DELETE", "/matches/"+matchID, cleo.token, nil), http.StatusNotFound)

	// --- messaging
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", ben.token, map[string]any{"body": "   "}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", ben.token, map[string]any{"body": strings.Repeat("a", 2001)}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", ben.token, map[string]any{"body": "Salut Ana !"}), http.StatusCreated)
	if m := waitEvent("message.new"); !strings.Contains(m, "Salut Ana") {
		t.Fatalf("unexpected event %s", m)
	}
	e.expect(e.do("POST", "/matches/"+matchID+"/messages", ben.token, map[string]any{"body": "Ça va ?"}), http.StatusCreated)

	r = e.expect(e.do("GET", "/matches", ana.token, nil), http.StatusOK)
	var list struct {
		Items []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
			Profile struct {
				UserID string `json:"userId"`
			} `json:"profile"`
		} `json:"items"`
	}
	r.json(t, &list)
	if len(list.Items) != 1 || list.Items[0].UnreadCount != 2 || list.Items[0].LastMessage == nil || list.Items[0].LastMessage.Body != "Ça va ?" || list.Items[0].Profile.UserID != ben.id {
		t.Fatalf("unexpected matches list: %s", r.body)
	}

	e.expect(e.do("POST", "/matches/"+matchID+"/read", ana.token, nil), http.StatusNoContent)
	benEvents("message.read")
	r = e.expect(e.do("GET", "/matches", ana.token, nil), http.StatusOK)
	r.json(t, &list)
	if list.Items[0].UnreadCount != 0 {
		t.Fatal("read must reset the unread counter")
	}

	r = e.expect(e.do("GET", "/matches/"+matchID+"/messages?limit=1", ben.token, nil), http.StatusOK)
	var page struct {
		Items []struct {
			ID, Body string
		} `json:"items"`
		NextBefore      string  `json:"nextBefore"`
		OtherLastReadAt *string `json:"otherLastReadAt"`
	}
	r.json(t, &page)
	if len(page.Items) != 1 || page.Items[0].Body != "Ça va ?" || page.NextBefore == "" || page.OtherLastReadAt == nil {
		t.Fatalf("unexpected page: %s", r.body)
	}
	r = e.expect(e.do("GET", "/matches/"+matchID+"/messages?limit=1&before="+page.NextBefore, ben.token, nil), http.StatusOK)
	r.json(t, &page)
	if len(page.Items) != 1 || page.Items[0].Body != "Salut Ana !" || page.NextBefore != "" {
		t.Fatalf("unexpected older page: %s", r.body)
	}

	// --- local deletion only affects the caller
	e.expect(e.do("DELETE", "/matches/"+matchID+"/messages", ana.token, nil), http.StatusNoContent)
	r = e.expect(e.do("GET", "/matches/"+matchID+"/messages", ana.token, nil), http.StatusOK)
	r.json(t, &page)
	if len(page.Items) != 0 {
		t.Fatal("history must be empty for the user who cleared it")
	}
	r = e.expect(e.do("GET", "/matches/"+matchID+"/messages", ben.token, nil), http.StatusOK)
	r.json(t, &page)
	if len(page.Items) != 2 {
		t.Fatal("the other participant keeps the history")
	}

	// --- media links are signed
	r = e.expect(e.do("GET", "/profiles/"+ben.id, ana.token, nil), http.StatusOK)
	var card struct {
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
	}
	r.json(t, &card)
	if len(card.Photos) != 1 {
		t.Fatalf("expected one photo, got %s", r.body)
	}
	e.expect(e.do("GET", card.Photos[0].URL, "", nil), http.StatusOK)
	e.expect(e.do("GET", strings.Split(card.Photos[0].URL, "?")[0], "", nil), http.StatusForbidden)
	e.expect(e.do("GET", strings.Replace(card.Photos[0].URL, "sig=", "sig=00", 1), "", nil), http.StatusForbidden)

	// --- report snapshot + block removes the match and hides both users from each other
	e.expect(e.do("POST", "/reports", ana.token, map[string]any{"userId": ben.id, "reason": "nonsense"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/reports", ana.token, map[string]any{"userId": ana.id, "reason": "spam"}), http.StatusUnprocessableEntity)
	e.expect(e.do("POST", "/reports", ana.token, map[string]any{"userId": ben.id, "reason": "harassment", "details": "rude", "alsoBlock": true}), http.StatusNoContent)
	var reports int
	var ctxJSON string
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*), COALESCE(MAX(context::text), '') FROM reports WHERE reporter_id=$1`, ana.id).Scan(&reports, &ctxJSON)
	if reports != 1 || !strings.Contains(ctxJSON, "Salut Ana") {
		t.Fatalf("report should store a message snapshot, got %d %s", reports, ctxJSON)
	}
	benEvents("match.removed")
}

func TestBlockHidesAndRemovesMatch(t *testing.T) {
	e := newEnv(t)
	ana := e.register("blk-ana")
	ben := e.register("blk-ben")
	e.onboard(ana, "Ana", "woman", []string{"man"}, "1994-03-03", 48.8566, 2.3522)
	e.onboard(ben, "Ben", "man", []string{"woman"}, "1992-04-04", 48.87, 2.36)

	e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": ben.id, "action": "like"}), http.StatusOK)
	r := e.expect(e.do("POST", "/swipes", ben.token, map[string]any{"targetId": ana.id, "action": "like"}), http.StatusOK)
	var sw struct {
		Matched bool    `json:"matched"`
		MatchID *string `json:"matchId"`
	}
	r.json(t, &sw)
	if !sw.Matched {
		t.Fatal("expected match")
	}
	e.expect(e.do("POST", "/blocks", ana.token, map[string]any{"userId": ben.id}), http.StatusNoContent)
	e.expect(e.do("POST", "/blocks", ana.token, map[string]any{"userId": ben.id}), http.StatusNoContent) // idempotent
	e.expect(e.do("POST", "/blocks", ana.token, map[string]any{"userId": ana.id}), http.StatusUnprocessableEntity)

	e.expect(e.do("GET", "/matches/"+*sw.MatchID+"/messages", ben.token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+ana.id, ben.token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+ben.id, ana.token, nil), http.StatusNotFound)
	e.expect(e.do("POST", "/swipes", ben.token, map[string]any{"targetId": ana.id, "action": "like"}), http.StatusNotFound)

	r = e.expect(e.do("GET", "/blocks", ana.token, nil), http.StatusOK)
	if !strings.Contains(string(r.body), ben.id) {
		t.Fatalf("blocked list should contain Ben: %s", r.body)
	}
	e.expect(e.do("DELETE", "/blocks/"+ben.id, ana.token, nil), http.StatusNoContent)
	e.expect(e.do("GET", "/profiles/"+ben.id, ana.token, nil), http.StatusOK)
}

func TestPhotoManagement(t *testing.T) {
	e := newEnv(t)
	u := e.register("photos")
	e.expect(e.do("PUT", "/me/profile", u.token, map[string]any{"firstName": "Pia", "birthDate": "1990-01-01", "gender": "woman", "interests": []string{}}), http.StatusOK)

	e.expect(e.upload("POST", "/me/photos", u.token, "evil.jpg", []byte("<?php echo 1; ?>")), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/me/photos", u.token, "x.jpg", append([]byte("GIF89a"), make([]byte, 64)...)), http.StatusUnsupportedMediaType)

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.expect(e.upload("POST", "/me/photos", u.token, "p.jpg", testJPEG(t, 2400+i, 1800)), http.StatusCreated)
		var out struct{ ID string }
		r.json(t, &out)
		ids = append(ids, out.ID)
	}
	e.expect(e.upload("POST", "/me/photos", u.token, "p.jpg", testJPEG(t, 100, 100)), http.StatusConflict)

	// reorder: make the last photo the primary one
	order := []string{ids[5], ids[0], ids[1], ids[2], ids[3], ids[4]}
	e.expect(e.do("PUT", "/me/photos/order", u.token, map[string]any{"ids": order}), http.StatusNoContent)
	e.expect(e.do("PUT", "/me/photos/order", u.token, map[string]any{"ids": order[:3]}), http.StatusUnprocessableEntity)
	e.expect(e.do("PUT", "/me/photos/order", u.token, map[string]any{"ids": []string{order[0], order[0], order[1], order[2], order[3], order[4]}}), http.StatusUnprocessableEntity)
	me := func() []string {
		r := e.expect(e.do("GET", "/me/profile", u.token, nil), http.StatusOK)
		var out struct{ Photos []struct{ ID string } }
		r.json(t, &out)
		got := []string{}
		for _, p := range out.Photos {
			got = append(got, p.ID)
		}
		return got
	}
	if got := me(); strings.Join(got, ",") != strings.Join(order, ",") {
		t.Fatalf("order not applied: %v vs %v", got, order)
	}

	// another user cannot touch these photos
	other := e.register("photos-other")
	e.expect(e.do("DELETE", "/me/photos/"+ids[0], other.token, nil), http.StatusNotFound)
	e.expect(e.upload("PUT", "/me/photos/"+ids[0], other.token, "r.jpg", testJPEG(t, 50, 50)), http.StatusNotFound)

	e.expect(e.upload("PUT", "/me/photos/"+ids[1], u.token, "r.jpg", testJPEG(t, 300, 300)), http.StatusNoContent)
	e.expect(e.do("DELETE", "/me/photos/"+order[0], u.token, nil), http.StatusNoContent)
	if got := me(); len(got) != 5 || got[0] != order[1] {
		t.Fatalf("positions must compact after delete: %v", got)
	}

	// stored renditions are downsized JPEGs
	r := e.expect(e.do("GET", "/me/profile", u.token, nil), http.StatusOK)
	var prof struct {
		Photos []struct{ URL, ThumbURL string }
	}
	r.json(t, &prof)
	full := e.expect(e.do("GET", prof.Photos[1].URL, "", nil), http.StatusOK)
	cfg, format, err := image.DecodeConfig(bytes.NewReader(full.body))
	if err != nil || format != "jpeg" || cfg.Width > 1600 {
		t.Fatalf("expected a jpeg <=1600px, got %v %v %v", format, cfg, err)
	}
	thumb := e.expect(e.do("GET", prof.Photos[1].ThumbURL, "", nil), http.StatusOK)
	if cfg, _, _ := image.DecodeConfig(bytes.NewReader(thumb.body)); cfg.Width > 400 {
		t.Fatalf("thumb too large: %v", cfg)
	}
}

func TestAccountDeletionErasesData(t *testing.T) {
	e := newEnv(t)
	ana := e.register("del-ana")
	ben := e.register("del-ben")
	e.onboard(ana, "Ana", "woman", []string{"man"}, "1994-03-03", 48.8566, 2.3522)
	e.onboard(ben, "Ben", "man", []string{"woman"}, "1992-04-04", 48.87, 2.36)
	e.expect(e.do("POST", "/swipes", ana.token, map[string]any{"targetId": ben.id, "action": "like"}), http.StatusOK)
	r := e.expect(e.do("POST", "/swipes", ben.token, map[string]any{"targetId": ana.id, "action": "like"}), http.StatusOK)
	var sw struct{ MatchID *string }
	r.json(t, &sw)
	e.expect(e.do("POST", "/matches/"+*sw.MatchID+"/messages", ana.token, map[string]any{"body": "bonjour"}), http.StatusCreated)

	e.expect(e.do("DELETE", "/me", ana.token, map[string]any{"password": "wrong-pass"}), http.StatusForbidden)
	e.expect(e.do("DELETE", "/me", ana.token, map[string]any{"password": "Password123"}), http.StatusNoContent)

	e.expect(e.do("GET", "/me", ana.token, nil), http.StatusUnauthorized)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": ana.email, "password": "Password123"}), http.StatusUnauthorized)
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "swiper_id", "chat_messages": "sender_id", "user_notifications": "user_id"} {
		var n int
		if err := e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = $1`, table, col), ana.id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s still holds %d rows for the deleted user", table, n)
		}
	}
	var matches int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE id = $1`, *sw.MatchID).Scan(&matches)
	if matches != 0 {
		t.Fatal("matches of a deleted user must disappear")
	}
	// Ben's feed no longer contains Ana and his match list is empty.
	if contains(discoverIDs(t, e, ben), ana.id) {
		t.Fatal("deleted user still discoverable")
	}
	r = e.expect(e.do("GET", "/matches", ben.token, nil), http.StatusOK)
	if !strings.Contains(string(r.body), `"items":[]`) {
		t.Fatalf("expected empty matches, got %s", r.body)
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	ana := e.register("race-ana")
	ben := e.register("race-ben")
	e.onboard(ana, "Ana", "woman", []string{"man"}, "1994-03-03", 48.8566, 2.3522)
	e.onboard(ben, "Ben", "man", []string{"woman"}, "1992-04-04", 48.87, 2.36)

	done := make(chan int, 2)
	for _, p := range [][2]user{{ana, ben}, {ben, ana}} {
		go func(from, to user) {
			done <- e.do("POST", "/swipes", from.token, map[string]any{"targetId": to.id, "action": "like"}).status
		}(p[0], p[1])
	}
	for i := 0; i < 2; i++ {
		if s := <-done; s != http.StatusOK {
			t.Fatalf("swipe failed with %d", s)
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, ana.id, ben.id).Scan(&n)
	if n != 1 {
		t.Fatalf("simultaneous mutual likes must yield exactly one match, got %d", n)
	}
}

func TestWebSocketRejectsBadToken(t *testing.T) {
	e := newEnv(t)
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": "nope"})
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("socket with an invalid token must be closed")
	}
}
