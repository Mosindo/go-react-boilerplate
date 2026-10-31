package app_test

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
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const password = "Password123"

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mails  *captureMailer
	emails []string
}

type captureMailer struct {
	mu   sync.Mutex
	last string
}

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last = body
	return nil
}

func (m *captureMailer) body() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last
}

func setup(t *testing.T) *env {
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
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("migrations: %v", err)
	}
	gin.SetMode(gin.TestMode)
	mails := &captureMailer{}
	e := &env{t: t, pool: pool, mails: mails}
	e.router = app.NewRouter(app.Deps{
		DB:          pool,
		JWTSecret:   []byte("0123456789abcdef0123456789abcdef"),
		Mailer:      mails,
		AuthLimiter: middleware.NewRateLimiter(100000, 100000),
	})
	t.Cleanup(func() {
		for _, email := range e.emails {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email)
		}
		pool.Close()
	})
	return e
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (e *env) do(method, path string, body any, token string) resp {
	e.t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{rec.Code, rec.Body.Bytes()}
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func testPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), 90, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func (e *env) upload(path, token, filename string, data []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("photo", filename)
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return resp{rec.Code, rec.Body.Bytes()}
}

type user struct {
	ID    string
	Email string
	Token string
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s-%d@test.invalid", prefix, time.Now().UnixNano())
	e.emails = append(e.emails, email)
	r := e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": password, "birthDate": "1992-04-12"}, ""), 201)
	var out struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	r.json(e.t, &out)
	return user{ID: out.User.ID, Email: email, Token: out.AccessToken}
}

// onboard creates a complete, discoverable profile.
func (e *env) onboard(u user, name, gender string, interestedIn []string, lat, lng float64) {
	e.t.Helper()
	e.expect(e.do("PUT", "/me/profile", map[string]any{
		"firstName": name, "gender": gender, "bio": "Hello", "city": "Lyon",
		"location": map[string]float64{"latitude": lat, "longitude": lng}, "interests": []string{"travel", "music"},
	}, u.Token), 200)
	e.expect(e.upload("/me/photos", u.Token, "me.png", testPNG()), 201)
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": interestedIn, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 100}, u.Token), 200)
}

type card struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	DistanceKm *int   `json:"distanceKm"`
}

func (e *env) feed(u user) []card {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", nil, u.Token), 200)
	var out struct {
		Profiles []card `json:"profiles"`
	}
	r.json(e.t, &out)
	return out.Profiles
}

func contains(cards []card, id string) bool {
	for _, c := range cards {
		if c.ID == id {
			return true
		}
	}
	return false
}

type swipeResp struct {
	Matched        bool `json:"matched"`
	AlreadyHandled bool `json:"alreadyHandled"`
	Match          *struct {
		ID             string `json:"id"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (e *env) swipe(u user, targetID, kind string) swipeResp {
	e.t.Helper()
	r := e.expect(e.do("POST", "/discover/"+targetID+"/"+kind, nil, u.Token), 200)
	var out swipeResp
	r.json(e.t, &out)
	return out
}

func TestHealthAndAuthBasics(t *testing.T) {
	e := setup(t)
	e.expect(e.do("GET", "/health", nil, ""), 200)

	u := e.register("auth")
	e.expect(e.do("GET", "/me", nil, ""), 401)
	e.expect(e.do("GET", "/me", nil, "garbage"), 401)
	me := e.expect(e.do("GET", "/me", nil, u.Token), 200)
	if strings.Contains(string(me.Body), "password") {
		t.Fatal("/me must never expose password data")
	}

	e.expect(e.do("POST", "/auth/register", map[string]string{"email": strings.ToUpper(u.Email), "password": password, "birthDate": "1992-04-12"}, ""), 409)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "wrong-password"}, ""), 401)
	login := e.expect(e.do("POST", "/auth/login", map[string]string{"email": strings.ToUpper(u.Email), "password": password}, ""), 200)

	var tokens struct {
		RefreshToken string `json:"refreshToken"`
	}
	login.json(t, &tokens)
	refreshed := e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": tokens.RefreshToken}, ""), 200)
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": tokens.RefreshToken}, ""), 401) // rotation: old token is dead
	var next struct {
		RefreshToken string `json:"refreshToken"`
	}
	refreshed.json(t, &next)
	e.expect(e.do("POST", "/auth/logout", map[string]string{"refreshToken": next.RefreshToken}, ""), 204)
	e.expect(e.do("POST", "/auth/refresh", map[string]string{"refreshToken": next.RefreshToken}, ""), 401)
}

func TestRegisterRejectsUnderageAndWeakInput(t *testing.T) {
	e := setup(t)
	email := fmt.Sprintf("minor-%d@test.invalid", time.Now().UnixNano())
	e.emails = append(e.emails, email)
	young := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": password, "birthDate": young}, ""), 403)
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": password, "birthDate": "garbage"}, ""), 400)
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": email, "password": "short", "birthDate": "1990-01-01"}, ""), 400)
	e.expect(e.do("POST", "/auth/register", map[string]string{"email": "not-an-email", "password": password, "birthDate": "1990-01-01"}, ""), 400)
}

func TestPasswordResetFlow(t *testing.T) {
	e := setup(t)
	u := e.register("reset")
	e.expect(e.do("POST", "/auth/password/forgot", map[string]string{"email": "nobody@test.invalid"}, ""), 202) // no account enumeration
	e.expect(e.do("POST", "/auth/password/forgot", map[string]string{"email": u.Email}, ""), 202)
	body := e.mails.body()
	idx := strings.Index(body, ": ")
	if idx < 0 {
		t.Fatalf("mail body missing code: %q", body)
	}
	code := body[idx+2 : idx+10]

	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": "WRONGCODE", "newPassword": "NewPassword456"}, ""), 400)
	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword456"}, ""), 204)
	e.expect(e.do("POST", "/auth/password/reset", map[string]string{"email": u.Email, "code": code, "newPassword": "Another789"}, ""), 400) // single use
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": password}, ""), 401)
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": u.Email, "password": "NewPassword456"}, ""), 200)
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := setup(t)
	u := e.register("prof")
	other := e.register("prof2")

	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "", "gender": "man"}, u.Token), 400)
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Léo", "gender": "alien"}, u.Token), 400)
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Léo", "gender": "man", "interests": []string{"does-not-exist"}}, u.Token), 400)
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Léo", "gender": "man", "bio": strings.Repeat("x", 501)}, u.Token), 400)
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"woman"}, "ageMin": 18, "ageMax": 40, "maxDistanceKm": 20}, u.Token), 409) // profile first

	e.onboard(u, "Léo", "man", []string{"woman"}, 45.7640, 4.8357)
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"woman"}, "ageMin": 17, "ageMax": 40, "maxDistanceKm": 20}, u.Token), 400)
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"woman"}, "ageMin": 30, "ageMax": 20, "maxDistanceKm": 20}, u.Token), 400)

	own := e.expect(e.do("GET", "/me/profile", nil, u.Token), 200)
	var p struct {
		Complete bool `json:"complete"`
		Location struct {
			Latitude float64 `json:"latitude"`
		} `json:"location"`
	}
	own.json(t, &p)
	if !p.Complete {
		t.Fatal("profile should be complete")
	}
	if p.Location.Latitude != 45.76 {
		t.Fatalf("location must be rounded to 2 decimals, got %v", p.Location.Latitude)
	}

	// Another user not related to u cannot open a profile that is hidden from discovery.
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Léo", "gender": "man", "isDiscoverable": false}, u.Token), 200)
	e.expect(e.do("GET", "/profiles/"+u.ID, nil, other.Token), 404)
	e.expect(e.do("GET", "/profiles/not-a-uuid", nil, other.Token), 404)
}

func TestFullDatingFlow(t *testing.T) {
	e := setup(t)
	alice := e.register("alice")
	bob := e.register("bob")
	carol := e.register("carol") // man interested in men: must not match alice's feed
	e.onboard(alice, "Alice", "woman", []string{"man"}, 45.76, 4.84)
	e.onboard(bob, "Bob", "man", []string{"woman"}, 45.77, 4.85)
	e.onboard(carol, "Carl", "man", []string{"man"}, 45.78, 4.86)

	// Discovery respects mutual preferences and hides exact location.
	feedA := e.feed(alice)
	if !contains(feedA, bob.ID) {
		t.Fatal("alice should see bob")
	}
	if contains(feedA, carol.ID) || contains(feedA, alice.ID) {
		t.Fatal("alice must not see carol (preferences) nor herself")
	}
	for _, c := range feedA {
		if c.ID == bob.ID && (c.DistanceKm == nil || *c.DistanceKm < 1) {
			t.Fatalf("distance should be an approximate number of km, got %v", c.DistanceKm)
		}
	}
	prof := e.expect(e.do("GET", "/profiles/"+bob.ID, nil, alice.Token), 200)
	for _, forbidden := range []string{"latitude", "longitude", "email", "birthDate", "45.77"} {
		if strings.Contains(string(prof.Body), forbidden) {
			t.Fatalf("public profile leaks %q: %s", forbidden, prof.Body)
		}
	}

	// A likes B: no match yet; repeated like is idempotent.
	first := e.swipe(alice, bob.ID, "like")
	if first.Matched {
		t.Fatal("one-sided like must not match")
	}
	if again := e.swipe(alice, bob.ID, "like"); !again.AlreadyHandled || again.Matched {
		t.Fatalf("repeat like should be a no-op, got %+v", again)
	}
	if contains(e.feed(alice), bob.ID) {
		t.Fatal("already swiped profiles must not reappear")
	}
	// Changing your mind via the API does not flip the first decision.
	if flip := e.swipe(alice, bob.ID, "pass"); !flip.AlreadyHandled {
		t.Fatal("pass after like must be ignored")
	}

	// B sees A first, and in "likes received".
	feedB := e.feed(bob)
	if len(feedB) == 0 || feedB[0].ID != alice.ID {
		t.Fatalf("bob's feed should start with alice, got %+v", feedB)
	}
	var received struct {
		Profiles []card `json:"profiles"`
	}
	e.expect(e.do("GET", "/likes/received", nil, bob.Token), 200).json(t, &received)
	if !contains(received.Profiles, alice.ID) {
		t.Fatal("alice should appear in bob's received likes")
	}

	// B likes A: match, for both.
	second := e.swipe(bob, alice.ID, "like")
	if !second.Matched || second.Match == nil || second.Match.ConversationID == "" {
		t.Fatalf("expected match, got %+v", second)
	}
	convID := second.Match.ConversationID
	var matches struct {
		Matches []struct {
			ConversationID string `json:"conversationId"`
			User           card   `json:"user"`
		} `json:"matches"`
	}
	e.expect(e.do("GET", "/matches", nil, alice.Token), 200).json(t, &matches)
	if len(matches.Matches) != 1 || matches.Matches[0].User.ID != bob.ID || matches.Matches[0].ConversationID != convID {
		t.Fatalf("unexpected alice matches: %+v", matches)
	}

	// Notifications: both got a match notification.
	for _, u := range []user{alice, bob} {
		var n struct {
			Notifications []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		e.expect(e.do("GET", "/notifications", nil, u.Token), 200).json(t, &n)
		found := false
		for _, item := range n.Notifications {
			if item.Type == "match" {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s missing match notification: %+v", u.ID, n)
		}
	}

	// Chat: send, unread, read.
	msgPath := "/conversations/" + convID + "/messages"
	e.expect(e.do("POST", msgPath, map[string]string{"body": "   "}, alice.Token), 400)
	e.expect(e.do("POST", msgPath, map[string]string{"body": strings.Repeat("a", 2001)}, alice.Token), 400)
	e.expect(e.do("POST", msgPath, map[string]string{"body": "Salut Bob !"}, alice.Token), 201)
	e.expect(e.do("POST", msgPath, map[string]string{"body": "Salut Alice"}, bob.Token), 201)
	e.expect(e.do("POST", msgPath, map[string]string{"body": "Ça va ?"}, bob.Token), 201)

	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.expect(e.do("GET", "/conversations", nil, alice.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 2 || convs.Conversations[0].LastMessage.Body != "Ça va ?" {
		t.Fatalf("unexpected conversations for alice: %+v", convs)
	}
	e.expect(e.do("POST", "/conversations/"+convID+"/read", nil, alice.Token), 204)
	convs.Conversations = nil
	e.expect(e.do("GET", "/conversations", nil, alice.Token), 200).json(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread should be 0 after read")
	}

	var msgs struct {
		Messages []struct {
			Body string `json:"body"`
		} `json:"messages"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	e.expect(e.do("GET", msgPath+"?limit=2", nil, bob.Token), 200).json(t, &msgs)
	if len(msgs.Messages) != 2 || msgs.Messages[0].Body != "Ça va ?" || msgs.OtherLastReadAt == nil {
		t.Fatalf("unexpected messages: %+v", msgs)
	}

	// Permissions: an outsider cannot read, write, read-mark or hide the conversation.
	e.expect(e.do("GET", msgPath, nil, carol.Token), 404)
	e.expect(e.do("POST", msgPath, map[string]string{"body": "intrus"}, carol.Token), 404)
	e.expect(e.do("POST", "/conversations/"+convID+"/read", nil, carol.Token), 404)
	e.expect(e.do("DELETE", "/conversations/"+convID, nil, carol.Token), 404)
	e.expect(e.do("GET", "/conversations/not-a-uuid/messages", nil, alice.Token), 404)

	// Local delete hides it only for the caller, and a new message revives it.
	e.expect(e.do("DELETE", "/conversations/"+convID, nil, alice.Token), 204)
	convs.Conversations = nil
	e.expect(e.do("GET", "/conversations", nil, alice.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("hidden conversation must not be listed")
	}
	e.expect(e.do("GET", "/conversations", nil, bob.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("other participant keeps the conversation")
	}
	e.expect(e.do("POST", msgPath, map[string]string{"body": "Tu es là ?"}, bob.Token), 201)
	convs.Conversations = nil
	e.expect(e.do("GET", "/conversations", nil, alice.Token), 200).json(t, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("new message must revive a hidden conversation")
	}

	// Photos are only fetchable by people who may see the owner.
	var own struct {
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
	}
	e.expect(e.do("GET", "/me/profile", nil, bob.Token), 200).json(t, &own)
	e.expect(e.do("GET", own.Photos[0].URL, nil, alice.Token), 200)
	e.expect(e.do("GET", own.Photos[0].URL, nil, ""), 401)

	// Block: conversation closes for both, profiles vanish, feed excludes.
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": alice.ID}, bob.Token), 204)
	e.expect(e.do("GET", msgPath, nil, alice.Token), 404)
	e.expect(e.do("POST", msgPath, map[string]string{"body": "?"}, bob.Token), 404)
	e.expect(e.do("GET", "/profiles/"+bob.ID, nil, alice.Token), 404)
	e.expect(e.do("GET", own.Photos[0].URL, nil, alice.Token), 404)
	if contains(e.feed(alice), bob.ID) || contains(e.feed(bob), alice.ID) {
		t.Fatal("blocked users must disappear from feeds")
	}
	e.expect(e.do("POST", "/discover/"+bob.ID+"/like", nil, alice.Token), 404)
	matches.Matches = nil
	e.expect(e.do("GET", "/matches", nil, alice.Token), 200).json(t, &matches)
	if len(matches.Matches) != 0 {
		t.Fatal("block must end the match")
	}
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": bob.ID}, bob.Token), 400) // self
	e.expect(e.do("DELETE", "/blocks/"+alice.ID, nil, bob.Token), 204)
	e.expect(e.do("GET", msgPath, nil, alice.Token), 404) // unblocking does not restore the match

	// Report (with block) and validation.
	e.expect(e.do("POST", "/reports", map[string]any{"userId": carol.ID, "reason": "nonsense"}, alice.Token), 400)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": carol.ID, "reason": "harassment", "details": "rude", "alsoBlock": true}, alice.Token), 201)
	var blocked struct {
		Blocked []struct {
			ID string `json:"id"`
		} `json:"blocked"`
	}
	e.expect(e.do("GET", "/blocks", nil, alice.Token), 200).json(t, &blocked)
	if len(blocked.Blocked) != 1 || blocked.Blocked[0].ID != carol.ID {
		t.Fatalf("unexpected blocked list %+v", blocked)
	}

	// Account deletion needs the password and removes every dependent row.
	e.expect(e.do("DELETE", "/me", map[string]string{"password": "wrong"}, bob.Token), 403)
	e.expect(e.do("DELETE", "/me", map[string]string{"password": password}, bob.Token), 204)
	for _, table := range []string{"profiles", "photos", "preferences", "likes", "sessions"} {
		col := "user_id"
		if table == "likes" {
			col = "swiper_id"
		}
		var n int
		if err := e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = $1`, table, col), bob.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s rows left after deletion: %d (%v)", table, n, err)
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE sender_id = $1`, bob.ID).Scan(&n)
	if n != 0 {
		t.Fatal("messages must be deleted with the account")
	}
	e.expect(e.do("POST", "/auth/login", map[string]string{"email": bob.Email, "password": password}, ""), 401)
}

func TestConcurrentReciprocalLikesCreateOneMatch(t *testing.T) {
	e := setup(t)
	a := e.register("race-a")
	b := e.register("race-b")
	e.onboard(a, "A", "woman", []string{"man"}, 45.76, 4.84)
	e.onboard(b, "B", "man", []string{"woman"}, 45.76, 4.84)

	var wg sync.WaitGroup
	results := make([]swipeResp, 2)
	for i, pair := range [][2]user{{a, b}, {b, a}} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/discover/"+pair[1].ID+"/like", nil)
			req.Header.Set("Authorization", "Bearer "+pair[0].Token)
			rec := httptest.NewRecorder()
			e.router.ServeHTTP(rec, req)
			_ = json.Unmarshal(rec.Body.Bytes(), &results[i])
		}()
	}
	wg.Wait()

	matched := 0
	for _, r := range results {
		if r.Matched {
			matched++
		}
	}
	if matched != 1 {
		t.Fatalf("exactly one of the two simultaneous likes must observe the match, got %d (%+v)", matched, results)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low IN ($1,$2) AND user_high IN ($1,$2)`, a.ID, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("expected 1 match row, got %d", n)
	}
}

func TestFeedFiltersAgeDistanceAndDiscoverability(t *testing.T) {
	e := setup(t)
	me := e.register("filter-me")
	near := e.register("filter-near")
	far := e.register("filter-far")
	hidden := e.register("filter-hidden")
	e.onboard(me, "Me", "woman", []string{"man"}, 45.76, 4.84)
	e.onboard(near, "Near", "man", []string{"woman"}, 45.77, 4.85)
	e.onboard(far, "Far", "man", []string{"woman"}, 48.85, 2.35) // Paris, ~390 km
	e.onboard(hidden, "Hidden", "man", []string{"woman"}, 45.77, 4.85)
	e.expect(e.do("PUT", "/me/profile", map[string]any{"firstName": "Hidden", "gender": "man", "isDiscoverable": false,
		"location": map[string]float64{"latitude": 45.77, "longitude": 4.85}}, hidden.Token), 200)

	feed := e.feed(me)
	if !contains(feed, near.ID) {
		t.Fatal("near user should be visible")
	}
	if contains(feed, far.ID) {
		t.Fatal("user 390 km away must be excluded by the 100 km preference")
	}
	if contains(feed, hidden.ID) {
		t.Fatal("non-discoverable users must be excluded")
	}

	// Age preference: restrict to 18-25; everybody was born in 1992 so nobody matches.
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 25, "maxDistanceKm": 100}, me.Token), 200)
	if len(e.feed(me)) != 0 {
		t.Fatal("age filter should exclude everyone")
	}
}

func TestSwipeRequiresCompleteProfileAndValidTarget(t *testing.T) {
	e := setup(t)
	newbie := e.register("newbie")
	target := e.register("target")
	e.onboard(target, "T", "man", []string{"woman"}, 45.76, 4.84)
	e.expect(e.do("POST", "/discover/"+target.ID+"/like", nil, newbie.Token), 409)
	e.expect(e.do("GET", "/discover", nil, newbie.Token), 200) // empty feed, not an error
	e.expect(e.do("POST", "/discover/not-a-uuid/like", nil, target.Token), 400)
	e.expect(e.do("POST", "/discover/"+target.ID+"/like", nil, target.Token), 400)
	e.expect(e.do("POST", "/discover/00000000-0000-0000-0000-000000000000/like", nil, target.Token), 404)
}

func TestPhotoUploadValidationAndOrdering(t *testing.T) {
	e := setup(t)
	u := e.register("photos")
	e.expect(e.upload("/me/photos", u.Token, "evil.png", []byte("<script>alert(1)</script>")), 415)
	e.expect(e.upload("/me/photos", u.Token, "huge.png", bytes.Repeat([]byte{1}, 6<<20)), 413)

	var ids []string
	for i := 0; i < 3; i++ {
		var out struct {
			Photos []struct {
				ID string `json:"id"`
			} `json:"photos"`
		}
		e.expect(e.upload("/me/photos", u.Token, "p.png", testPNG()), 201).json(t, &out)
		ids = nil
		for _, p := range out.Photos {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) != 3 {
		t.Fatalf("expected 3 photos, got %d", len(ids))
	}
	e.expect(e.do("PUT", "/me/photos/order", map[string]any{"ids": []string{ids[2], ids[0]}}, u.Token), 400) // must list every photo
	var out struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	e.expect(e.do("PUT", "/me/photos/order", map[string]any{"ids": []string{ids[2], ids[0], ids[1]}}, u.Token), 200).json(t, &out)
	if out.Photos[0].ID != ids[2] {
		t.Fatal("first photo should be the main photo after reorder")
	}
	e.expect(e.do("DELETE", "/me/photos/"+ids[2], nil, u.Token), 200).json(t, &out)
	if len(out.Photos) != 2 || out.Photos[0].ID != ids[0] {
		t.Fatalf("delete should compact order, got %+v", out)
	}
	for i := 0; i < 4; i++ {
		e.expect(e.upload("/me/photos", u.Token, "p.png", testPNG()), 201)
	}
	e.expect(e.upload("/me/photos", u.Token, "p.png", testPNG()), 409) // max 6

	// Another user cannot delete someone else's photo.
	other := e.register("photos-other")
	e.expect(e.do("DELETE", "/me/photos/"+ids[0], nil, other.Token), 404)
}

func TestWebSocketReceivesMessagesAndMatches(t *testing.T) {
	e := setup(t)
	a := e.register("ws-a")
	b := e.register("ws-b")
	e.onboard(a, "A", "woman", []string{"man"}, 45.76, 4.84)
	e.onboard(b, "B", "man", []string{"woman"}, 45.76, 4.84)

	srv := httptest.NewServer(e.router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// Unauthenticated sockets are closed.
	bad, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "nope"})
	_ = bad.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("expected close for bad token")
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": a.Token})
	read := func() map[string]any {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev map[string]any
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read ws: %v", err)
		}
		return ev
	}
	if ev := read(); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}

	e.swipe(a, b.ID, "like")
	m := e.swipe(b, a.ID, "like")
	if ev := read(); ev["type"] != "match" {
		t.Fatalf("expected match event, got %v", ev)
	}
	e.expect(e.do("POST", "/conversations/"+m.Match.ConversationID+"/messages", map[string]string{"body": "coucou"}, b.Token), 201)
	ev := read()
	if ev["type"] != "message" || !strings.Contains(fmt.Sprint(ev["data"]), "coucou") {
		t.Fatalf("expected message event, got %v", ev)
	}
}
