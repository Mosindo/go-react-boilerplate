package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/testutil"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-secret-0123456789-abcdefghij"

type env struct {
	t      *testing.T
	router http.Handler
	pool   *pgxpool.Pool
	store  *storage.Local
	mailer *captureMailer
}

type captureMailer struct{ last chan string }

func (m *captureMailer) Send(to, subject, body string) error {
	select {
	case m.last <- body:
	default:
	}
	return nil
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testutil.Pool(t)
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mailer := &captureMailer{last: make(chan string, 4)}
	var _ mail.Mailer = mailer
	router, err := app.NewRouter(app.Deps{DB: pool, JWTSecret: []byte(testSecret), Storage: store, Mailer: mailer})
	if err != nil {
		t.Fatal(err)
	}
	return &env{t: t, router: router, pool: pool, store: store, mailer: mailer}
}

type resp struct {
	Status int
	Body   map[string]any
	Raw    []byte
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.serve(req)
}

func (e *env) serve(req *http.Request) resp {
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	out := resp{Status: rec.Code, Raw: rec.Body.Bytes()}
	_ = json.Unmarshal(out.Raw, &out.Body)
	return out
}

func (e *env) upload(token, field string, data []byte, method, path string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile(field, "photo.bin")
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.serve(req)
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type member struct {
	ID, Token, Email string
}

func str(m map[string]any, keys ...string) string {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = mm[k]
	}
	s, _ := cur.(string)
	return s
}

func (e *env) register(prefix, birth string) (member, resp) {
	email := testutil.Email(prefix)
	r := e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": "Sup3rSecret!", "birthDate": birth})
	return member{ID: str(r.Body, "user", "id"), Token: str(r.Body, "accessToken"), Email: email}, r
}

// onboard creates a complete member: account, profile, one photo, preferences.
func (e *env) onboard(prefix, gender string, interestedIn []string, lat, lng float64) member {
	e.t.Helper()
	m, r := e.register(prefix, "1994-06-15")
	if r.Status != 201 {
		e.t.Fatalf("register %s: %d %s", prefix, r.Status, r.Raw)
	}
	if r := e.do("PUT", "/profile", m.Token, map[string]any{
		"firstName": strings.ToUpper(prefix[:1]) + prefix[1:], "gender": gender, "bio": "Hello", "city": "Paris",
		"latitude": lat, "longitude": lng, "interests": []string{"travel", "music"},
	}); r.Status != 200 {
		e.t.Fatalf("profile %s: %d %s", prefix, r.Status, r.Raw)
	}
	if r := e.do("PUT", "/preferences", m.Token, map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 100, "discoverable": true, "showDistance": true,
	}); r.Status != 200 {
		e.t.Fatalf("preferences %s: %d %s", prefix, r.Status, r.Raw)
	}
	if r := e.upload(m.Token, "photo", pngBytes(500, 600), "POST", "/photos"); r.Status != 201 {
		e.t.Fatalf("photo %s: %d %s", prefix, r.Status, r.Raw)
	}
	return m
}

func (e *env) discoverIDs(m member) []string {
	r := e.do("GET", "/discover?limit=20", m.Token, nil)
	if r.Status != 200 {
		e.t.Fatalf("discover: %d %s", r.Status, r.Raw)
	}
	var ids []string
	for _, it := range r.Body["items"].([]any) {
		ids = append(ids, it.(map[string]any)["userId"].(string))
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

func TestHealth(t *testing.T) {
	e := newEnv(t)
	if r := e.do("GET", "/health", "", nil); r.Status != 200 {
		t.Fatalf("health: %d", r.Status)
	}
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)

	if _, r := e.register("minor", time.Now().AddDate(-17, 0, 0).Format("2006-01-02")); r.Status != 403 || str(r.Body, "code") != "underage" {
		t.Fatalf("minor must be rejected, got %d %s", r.Status, r.Raw)
	}
	if _, r := e.register("badbirth", "not-a-date"); r.Status != 400 {
		t.Fatalf("bad birth date: %d", r.Status)
	}

	m, r := e.register("auth", "1990-01-01")
	if r.Status != 201 || m.Token == "" {
		t.Fatalf("register: %d %s", r.Status, r.Raw)
	}
	if _, dup := e.register("auth", "1990-01-01"); dup.Status != 201 { // different unique email
		t.Fatalf("second user: %d", dup.Status)
	}
	if r := e.do("POST", "/auth/register", "", map[string]any{"email": "  " + strings.ToUpper(m.Email) + " ", "password": "Sup3rSecret!", "birthDate": "1990-01-01"}); r.Status != 409 {
		t.Fatalf("duplicate email (case/space-insensitive) must conflict, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/register", "", map[string]any{"email": testutil.Email("weak"), "password": "short", "birthDate": "1990-01-01"}); r.Status != 400 {
		t.Fatalf("weak password: %d", r.Status)
	}

	if r := e.do("POST", "/auth/login", "", map[string]any{"email": m.Email, "password": "wrong-password"}); r.Status != 401 {
		t.Fatalf("wrong password: %d", r.Status)
	}
	login := e.do("POST", "/auth/login", "", map[string]any{"email": m.Email, "password": "Sup3rSecret!"})
	if login.Status != 200 {
		t.Fatalf("login: %d %s", login.Status, login.Raw)
	}

	if r := e.do("GET", "/me", "", nil); r.Status != 401 {
		t.Fatalf("/me without token: %d", r.Status)
	}
	me := e.do("GET", "/me", str(login.Body, "accessToken"), nil)
	if me.Status != 200 || str(me.Body, "email") != m.Email || me.Body["hasProfile"] != false {
		t.Fatalf("/me: %d %s", me.Status, me.Raw)
	}
	if me.Body["age"].(float64) < 30 {
		t.Fatalf("age not computed: %v", me.Body["age"])
	}

	// Refresh rotates the token: the old one stops working.
	oldRefresh := str(login.Body, "refreshToken")
	ref := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": oldRefresh})
	if ref.Status != 200 || str(ref.Body, "refreshToken") == oldRefresh {
		t.Fatalf("refresh: %d %s", ref.Status, ref.Raw)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": oldRefresh}); r.Status != 401 {
		t.Fatalf("rotated refresh token must be rejected, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": str(ref.Body, "refreshToken")}); r.Status != 204 {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": str(ref.Body, "refreshToken")}); r.Status != 401 {
		t.Fatalf("logged out refresh token must be rejected, got %d", r.Status)
	}
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	m, _ := e.register("recover", "1990-01-01")

	// Unknown addresses get the same answer (no enumeration) and no mail.
	if r := e.do("POST", "/auth/forgot-password", "", map[string]any{"email": testutil.Email("ghost")}); r.Status != 202 {
		t.Fatalf("forgot unknown: %d", r.Status)
	}
	if r := e.do("POST", "/auth/forgot-password", "", map[string]any{"email": m.Email}); r.Status != 202 {
		t.Fatalf("forgot: %d", r.Status)
	}
	var body string
	select {
	case body = <-e.mailer.last:
	case <-time.After(3 * time.Second):
		t.Fatal("no recovery mail sent")
	}
	idx := strings.Index(body, ": ")
	code := strings.Fields(body[idx+2:])[0]

	if r := e.do("POST", "/auth/reset-password", "", map[string]any{"code": "AAAA-BBBB-CCCC-DDDD", "password": "N3wPassword!"}); r.Status != 400 {
		t.Fatalf("bad code: %d", r.Status)
	}
	if r := e.do("POST", "/auth/reset-password", "", map[string]any{"code": code, "password": "N3wPassword!"}); r.Status != 204 {
		t.Fatalf("reset: %d", r.Status)
	}
	if r := e.do("POST", "/auth/reset-password", "", map[string]any{"code": code, "password": "An0therOne!!"}); r.Status != 400 {
		t.Fatalf("code reuse must fail, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": m.Email, "password": "Sup3rSecret!"}); r.Status != 401 {
		t.Fatalf("old password must stop working: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": m.Email, "password": "N3wPassword!"}); r.Status != 200 {
		t.Fatalf("new password login: %d", r.Status)
	}
}

func TestProfilePhotosAndValidation(t *testing.T) {
	e := newEnv(t)
	m, _ := e.register("prof", "1992-03-03")

	if r := e.do("GET", "/profile", m.Token, nil); r.Status != 404 {
		t.Fatalf("profile before creation: %d", r.Status)
	}
	if r := e.do("GET", "/discover", m.Token, nil); r.Status != 409 {
		t.Fatalf("discover without profile must be 409, got %d", r.Status)
	}
	bad := []map[string]any{
		{"firstName": "", "gender": "man"},
		{"firstName": "Léo", "gender": "alien"},
		{"firstName": "Léo", "gender": "man", "interests": []string{"does-not-exist"}},
		{"firstName": "Léo", "gender": "man", "heightCm": 20},
		{"firstName": "Léo", "gender": "man", "latitude": 95.0, "longitude": 2.0},
		{"firstName": "Léo", "gender": "man", "bio": strings.Repeat("x", 501)},
	}
	for i, p := range bad {
		if r := e.do("PUT", "/profile", m.Token, p); r.Status != 400 {
			t.Fatalf("invalid profile #%d accepted: %d", i, r.Status)
		}
	}

	r := e.do("PUT", "/profile", m.Token, map[string]any{
		"firstName": "  Léo  ", "gender": "man", "city": "Lyon", "latitude": 45.764043, "longitude": 4.835659, "interests": []string{"music", "music", "travel"},
	})
	if r.Status != 200 || str(r.Body, "firstName") != "Léo" {
		t.Fatalf("save profile: %d %s", r.Status, r.Raw)
	}
	if lat := r.Body["latitude"].(float64); lat != 45.76 {
		t.Fatalf("coordinates must be rounded to 2 decimals, got %v", lat)
	}
	if n := len(r.Body["interests"].([]any)); n != 2 {
		t.Fatalf("interests must be deduplicated, got %d", n)
	}

	// Photos: validation, limit, reorder, replace, delete.
	if r := e.upload(m.Token, "photo", []byte("<?php echo 1; ?>"), "POST", "/photos"); r.Status != 400 {
		t.Fatalf("non-image upload must be rejected, got %d", r.Status)
	}
	if r := e.upload(m.Token, "photo", pngBytes(50, 50), "POST", "/photos"); r.Status != 400 {
		t.Fatalf("tiny image must be rejected, got %d", r.Status)
	}
	var ids []string
	for i := 0; i < 6; i++ {
		up := e.upload(m.Token, "photo", pngBytes(300+i, 300), "POST", "/photos")
		if up.Status != 201 {
			t.Fatalf("upload #%d: %d %s", i, up.Status, up.Raw)
		}
		ids = append(ids, str(up.Body, "id"))
	}
	if r := e.upload(m.Token, "photo", pngBytes(300, 300), "POST", "/photos"); r.Status != 409 {
		t.Fatalf("7th photo must hit the limit, got %d", r.Status)
	}
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	ord := e.do("PUT", "/photos/order", m.Token, map[string]any{"photoIds": rev})
	if ord.Status != 200 || str(ord.Body["items"].([]any)[0].(map[string]any), "id") != ids[5] {
		t.Fatalf("reorder: %d %s", ord.Status, ord.Raw)
	}
	if r := e.do("PUT", "/photos/order", m.Token, map[string]any{"photoIds": rev[:3]}); r.Status != 400 {
		t.Fatalf("partial reorder must fail, got %d", r.Status)
	}
	if r := e.upload(m.Token, "photo", pngBytes(400, 400), "PUT", "/photos/"+ids[0]); r.Status != 200 {
		t.Fatalf("replace: %d %s", r.Status, r.Raw)
	}
	if r := e.do("DELETE", "/photos/"+ids[5], m.Token, nil); r.Status != 204 {
		t.Fatalf("delete: %d", r.Status)
	}
	list := e.do("GET", "/photos", m.Token, nil)
	items := list.Body["items"].([]any)
	if len(items) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(items))
	}
	for i, it := range items {
		if int(it.(map[string]any)["position"].(float64)) != i {
			t.Fatalf("positions must stay contiguous: %v", items)
		}
	}

	// Another member cannot touch my photos.
	other, _ := e.register("intruder", "1990-01-01")
	if r := e.do("DELETE", "/photos/"+ids[1], other.Token, nil); r.Status != 404 {
		t.Fatalf("deleting someone else's photo must be 404, got %d", r.Status)
	}

	// Signed media URLs: valid works, tampering and unsigned fail; the file is a re-encoded JPEG.
	url := str(items[0].(map[string]any), "url")
	get := e.do("GET", url, "", nil)
	if get.Status != 200 || len(get.Raw) < 2 || get.Raw[0] != 0xFF || get.Raw[1] != 0xD8 {
		t.Fatalf("signed media: %d", get.Status)
	}
	if r := e.do("GET", strings.Replace(url, "sig=", "sig=00", 1), "", nil); r.Status != 404 {
		t.Fatalf("tampered signature must fail, got %d", r.Status)
	}
	if r := e.do("GET", "/media/photos/"+ids[1], "", nil); r.Status != 404 {
		t.Fatalf("unsigned media must fail, got %d", r.Status)
	}
}

func TestDiscoveryMatchAndChat(t *testing.T) {
	e := newEnv(t)
	// Alice (woman → men) and Bob (man → women) are compatible; Carol (woman → women) is not compatible with Bob.
	alice := e.onboard("alice", "woman", []string{"man"}, 48.8566, 2.3522)
	bob := e.onboard("bob", "man", []string{"woman"}, 48.8600, 2.3400)
	carol := e.onboard("carol", "woman", []string{"woman"}, 48.8700, 2.3300)
	faraway := e.onboard("faraway", "man", []string{"woman"}, 43.2965, 5.3698) // Marseille, > 100 km

	aIDs := e.discoverIDs(alice)
	if !contains(aIDs, bob.ID) {
		t.Fatalf("Alice should discover Bob, got %v", aIDs)
	}
	if contains(aIDs, carol.ID) || contains(aIDs, alice.ID) {
		t.Fatalf("incompatible/self profiles must not appear: %v", aIDs)
	}
	if contains(aIDs, faraway.ID) {
		t.Fatalf("profiles beyond max distance must not appear: %v", aIDs)
	}
	// Public profiles never leak coordinates or exact location.
	disc := e.do("GET", "/discover?limit=20", alice.Token, nil)
	for _, it := range disc.Body["items"].([]any) {
		m := it.(map[string]any)
		if _, ok := m["latitude"]; ok {
			t.Fatal("public profile must not expose latitude")
		}
		if _, ok := m["email"]; ok {
			t.Fatal("public profile must not expose email")
		}
		if m["userId"] == bob.ID {
			if d, ok := m["distanceKm"].(float64); !ok || d < 1 || d > 5 {
				t.Fatalf("expected coarse distance, got %v", m["distanceKm"])
			}
		}
	}

	// Alice likes Bob: no match yet. A repeated like is idempotent.
	r := e.do("POST", "/swipes", alice.Token, map[string]any{"targetId": bob.ID, "action": "like"})
	if r.Status != 200 || r.Body["matched"] != false {
		t.Fatalf("first like: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/swipes", alice.Token, map[string]any{"targetId": bob.ID, "action": "like"}); r.Status != 200 || r.Body["matched"] != false {
		t.Fatalf("repeated like: %d %s", r.Status, r.Raw)
	}
	if contains(e.discoverIDs(alice), bob.ID) {
		t.Fatal("a processed profile must not come back")
	}
	// Bob sees Alice first (she already liked him) and likes back → match.
	bIDs := e.discoverIDs(bob)
	if len(bIDs) == 0 || bIDs[0] != alice.ID {
		t.Fatalf("Bob should see Alice first, got %v", bIDs)
	}
	r = e.do("POST", "/swipes", bob.Token, map[string]any{"targetId": alice.ID, "action": "like"})
	if r.Status != 200 || r.Body["matched"] != true {
		t.Fatalf("expected match: %d %s", r.Status, r.Raw)
	}
	convID := str(r.Body, "match", "conversationId")
	if convID == "" {
		t.Fatal("match must carry a conversation id")
	}
	// Changing one's mind afterwards must not undo or duplicate the match.
	if r := e.do("POST", "/swipes", bob.Token, map[string]any{"targetId": alice.ID, "action": "pass"}); r.Body["matched"] != true || r.Body["action"] != "like" {
		t.Fatalf("swipes are final and idempotent: %s", r.Raw)
	}
	var matches int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, alice.ID, bob.ID).Scan(&matches)
	if matches != 1 {
		t.Fatalf("exactly one match expected, got %d", matches)
	}
	if r := e.do("POST", "/swipes", alice.Token, map[string]any{"targetId": alice.ID, "action": "like"}); r.Status != 400 {
		t.Fatalf("self swipe: %d", r.Status)
	}
	if r := e.do("POST", "/swipes", alice.Token, map[string]any{"targetId": bob.ID, "action": "super"}); r.Status != 400 {
		t.Fatalf("invalid action: %d", r.Status)
	}

	// Notifications for both sides.
	for _, m := range []member{alice, bob} {
		n := e.do("GET", "/notifications", m.Token, nil)
		if n.Status != 200 || n.Body["unreadCount"].(float64) != 1 {
			t.Fatalf("match notification expected: %d %s", n.Status, n.Raw)
		}
	}
	if r := e.do("POST", "/notifications/read", alice.Token, nil); r.Status != 204 {
		t.Fatalf("mark all read: %d", r.Status)
	}
	if n := e.do("GET", "/notifications", alice.Token, nil); n.Body["unreadCount"].(float64) != 0 {
		t.Fatalf("notifications should be read: %s", n.Raw)
	}

	// Chat.
	if r := e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "   "}); r.Status != 400 {
		t.Fatalf("empty message: %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": strings.Repeat("a", 2001)}); r.Status != 400 {
		t.Fatalf("long message: %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "Salut Bob !"}); r.Status != 201 {
		t.Fatalf("send: %d %s", r.Status, r.Raw)
	}
	e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "Tu es là ?"})

	convs := e.do("GET", "/conversations", bob.Token, nil)
	first := convs.Body["items"].([]any)[0].(map[string]any)
	if first["unreadCount"].(float64) != 2 || str(first, "lastMessage", "body") != "Tu es là ?" || str(first, "user", "id") != alice.ID {
		t.Fatalf("Bob's conversation list: %s", convs.Raw)
	}
	if n := e.do("GET", "/notifications", bob.Token, nil); n.Body["unreadCount"].(float64) != 2 { // match + one coalesced message notification
		t.Fatalf("Bob notifications: %s", n.Raw)
	}

	// A stranger can neither read, write, mark read nor hide the conversation.
	if r := e.do("GET", "/conversations/"+convID+"/messages", carol.Token, nil); r.Status != 404 {
		t.Fatalf("stranger reading messages must get 404, got %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+convID+"/messages", carol.Token, map[string]any{"body": "intrusion"}); r.Status != 404 {
		t.Fatalf("stranger posting must get 404, got %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+convID+"/read", carol.Token, nil); r.Status != 404 {
		t.Fatalf("stranger mark-read must get 404, got %d", r.Status)
	}
	if r := e.do("DELETE", "/conversations/"+convID, carol.Token, nil); r.Status != 404 {
		t.Fatalf("stranger hide must get 404, got %d", r.Status)
	}

	msgs := e.do("GET", "/conversations/"+convID+"/messages?limit=1", bob.Token, nil)
	if len(msgs.Body["items"].([]any)) != 1 || msgs.Body["nextBefore"] == nil {
		t.Fatalf("message pagination: %s", msgs.Raw)
	}
	if r := e.do("POST", "/conversations/"+convID+"/read", bob.Token, nil); r.Status != 204 {
		t.Fatalf("mark read: %d", r.Status)
	}
	convs = e.do("GET", "/conversations", bob.Token, nil)
	if convs.Body["items"].([]any)[0].(map[string]any)["unreadCount"].(float64) != 0 {
		t.Fatalf("unread should be cleared: %s", convs.Raw)
	}
	aliceView := e.do("GET", "/conversations/"+convID+"/messages", alice.Token, nil)
	if aliceView.Body["otherLastReadAt"] == nil {
		t.Fatal("read receipts missing")
	}

	// Local deletion hides the conversation for Bob only, and a new message brings it back.
	if r := e.do("DELETE", "/conversations/"+convID, bob.Token, nil); r.Status != 204 {
		t.Fatalf("hide: %d", r.Status)
	}
	if n := len(e.do("GET", "/conversations", bob.Token, nil).Body["items"].([]any)); n != 0 {
		t.Fatalf("hidden conversation still listed (%d)", n)
	}
	if n := len(e.do("GET", "/conversations", alice.Token, nil).Body["items"].([]any)); n != 1 {
		t.Fatalf("Alice must keep her conversation, got %d", n)
	}
	e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "Coucou revenu"})
	bobMsgs := e.do("GET", "/conversations/"+convID+"/messages", bob.Token, nil)
	if n := len(bobMsgs.Body["items"].([]any)); n != 1 {
		t.Fatalf("after hiding Bob must only see newer messages, got %d", n)
	}

	// Blocking dissolves the match and conversation and hides both profiles from each other.
	if r := e.do("PUT", "/blocks/"+alice.ID, bob.Token, nil); r.Status != 204 {
		t.Fatalf("block: %d", r.Status)
	}
	if r := e.do("GET", "/conversations/"+convID+"/messages", alice.Token, nil); r.Status != 404 {
		t.Fatalf("conversation must be gone after block, got %d", r.Status)
	}
	if r := e.do("GET", "/users/"+bob.ID, alice.Token, nil); r.Status != 404 {
		t.Fatalf("blocked profile must be invisible, got %d", r.Status)
	}
	if r := e.do("POST", "/swipes", alice.Token, map[string]any{"targetId": bob.ID, "action": "like"}); r.Status != 404 {
		t.Fatalf("swiping a blocked member must fail, got %d", r.Status)
	}
	bl := e.do("GET", "/blocks", bob.Token, nil)
	if len(bl.Body["items"].([]any)) != 1 {
		t.Fatalf("block list: %s", bl.Raw)
	}
	if r := e.do("DELETE", "/blocks/"+alice.ID, bob.Token, nil); r.Status != 204 {
		t.Fatalf("unblock: %d", r.Status)
	}

	// Reports.
	if r := e.do("POST", "/reports", carol.Token, map[string]any{"userId": faraway.ID, "reason": "spam", "details": "annoying", "alsoBlock": true}); r.Status != 201 {
		t.Fatalf("report: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/reports", carol.Token, map[string]any{"userId": faraway.ID, "reason": "bogus"}); r.Status != 400 {
		t.Fatalf("invalid report reason: %d", r.Status)
	}
	if r := e.do("POST", "/reports", carol.Token, map[string]any{"userId": carol.ID, "reason": "spam"}); r.Status != 400 {
		t.Fatalf("self report: %d", r.Status)
	}

	// Account deletion erases every trace, files included.
	var keyBefore string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, bob.ID).Scan(&keyBefore)
	if r := e.do("DELETE", "/me", bob.Token, map[string]any{"password": "bad"}); r.Status != 403 {
		t.Fatalf("delete with wrong password: %d", r.Status)
	}
	if r := e.do("DELETE", "/me", bob.Token, map[string]any{"password": "Sup3rSecret!"}); r.Status != 204 {
		t.Fatalf("delete account: %d", r.Status)
	}
	for _, q := range []string{
		`SELECT COUNT(*) FROM users WHERE id = $1`,
		`SELECT COUNT(*) FROM profiles WHERE user_id = $1`,
		`SELECT COUNT(*) FROM photos WHERE user_id = $1`,
		`SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1`,
		`SELECT COUNT(*) FROM messages WHERE sender_id = $1`,
		`SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 OR blocked_id = $1`,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 OR actor_id = $1`,
	} {
		var n int
		if err := e.pool.QueryRow(context.Background(), q, bob.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("data left after account deletion (%s): %d %v", q, n, err)
		}
	}
	if rc, err := e.store.Open(keyBefore); err == nil {
		rc.Close()
		t.Fatal("photo file must be removed with the account")
	}
	if r := e.do("GET", "/users/"+bob.ID, alice.Token, nil); r.Status != 404 {
		t.Fatalf("deleted member must be gone: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": bob.Email, "password": "Sup3rSecret!"}); r.Status != 401 {
		t.Fatalf("deleted account login: %d", r.Status)
	}
}

func TestPassAndPrivacy(t *testing.T) {
	e := newEnv(t)
	dana := e.onboard("dana", "woman", []string{"man"}, 48.85, 2.35)
	eric := e.onboard("eric", "man", []string{"woman"}, 48.86, 2.34)

	if !contains(e.discoverIDs(dana), eric.ID) {
		t.Fatal("Dana should see Eric")
	}
	if r := e.do("POST", "/swipes", dana.Token, map[string]any{"targetId": eric.ID, "action": "pass"}); r.Status != 200 || r.Body["matched"] != false {
		t.Fatalf("pass: %d %s", r.Status, r.Raw)
	}
	if contains(e.discoverIDs(dana), eric.ID) {
		t.Fatal("passed profile must not reappear")
	}
	if r := e.do("DELETE", "/swipes/passes", dana.Token, nil); r.Status != 204 {
		t.Fatalf("reset passes: %d", r.Status)
	}
	if !contains(e.discoverIDs(dana), eric.ID) {
		t.Fatal("after resetting passes Eric is back")
	}

	// Pausing the profile removes it from discovery and makes it unviewable by strangers.
	prefs := map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 60, "maxDistanceKm": 100, "discoverable": false, "showDistance": false}
	if r := e.do("PUT", "/preferences", eric.Token, prefs); r.Status != 200 {
		t.Fatalf("preferences: %d %s", r.Status, r.Raw)
	}
	if contains(e.discoverIDs(dana), eric.ID) {
		t.Fatal("paused profile must not be discoverable")
	}
	if r := e.do("GET", "/users/"+eric.ID, dana.Token, nil); r.Status != 404 {
		t.Fatalf("paused profile must be hidden, got %d", r.Status)
	}
	if r := e.do("GET", "/users/"+eric.ID, eric.Token, nil); r.Status != 200 {
		t.Fatalf("own profile via /users/:id: %d", r.Status)
	}
	if r := e.do("PUT", "/preferences", eric.Token, map[string]any{"interestedIn": []string{}, "minAge": 18, "maxAge": 60, "discoverable": true, "showDistance": true}); r.Status != 400 {
		t.Fatalf("empty interestedIn: %d", r.Status)
	}
	if r := e.do("PUT", "/preferences", eric.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 60, "discoverable": true, "showDistance": true}); r.Status != 400 {
		t.Fatalf("min age 17: %d", r.Status)
	}
}

func TestWebSocketDeliversMessages(t *testing.T) {
	e := newEnv(t)
	fay := e.onboard("fay", "woman", []string{"man"}, 48.85, 2.35)
	gus := e.onboard("gus", "man", []string{"woman"}, 48.86, 2.34)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	dial := func(token string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.WriteJSON(map[string]string{"type": "auth", "token": token})
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		var ev map[string]any
		if err := conn.ReadJSON(&ev); err != nil || ev["type"] != "ready" {
			t.Fatalf("expected ready, got %v %v", ev, err)
		}
		return conn
	}
	next := func(conn *websocket.Conn, typ string) map[string]any {
		for {
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			var ev map[string]any
			if err := conn.ReadJSON(&ev); err != nil {
				t.Fatalf("waiting for %s: %v", typ, err)
			}
			if ev["type"] == typ {
				return ev
			}
		}
	}

	// Bad token is refused.
	bad, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "garbage"})
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	var ev map[string]any
	_ = bad.ReadJSON(&ev)
	if ev["type"] == "ready" {
		t.Fatal("invalid token must not be accepted")
	}
	bad.Close()

	fayConn, gusConn := dial(fay.Token), dial(gus.Token)
	defer fayConn.Close()
	defer gusConn.Close()

	e.do("POST", "/swipes", fay.Token, map[string]any{"targetId": gus.ID, "action": "like"})
	r := e.do("POST", "/swipes", gus.Token, map[string]any{"targetId": fay.ID, "action": "like"})
	if r.Body["matched"] != true {
		t.Fatalf("expected match: %s", r.Raw)
	}
	next(fayConn, "match.new")
	conv := str(r.Body, "match", "conversationId")

	e.do("POST", "/conversations/"+conv+"/messages", gus.Token, map[string]any{"body": "temps réel"})
	msg := next(fayConn, "message.new")
	if data := msg["data"].(map[string]any); data["body"] != "temps réel" || fmt.Sprint(data["senderId"]) != gus.ID {
		t.Fatalf("unexpected realtime payload: %v", msg)
	}
	e.do("POST", "/conversations/"+conv+"/read", fay.Token, nil)
	next(gusConn, "conversation.read")
}

// Two members liking each other at the very same instant must produce exactly one match.
func TestSimultaneousMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.onboard(fmt.Sprintf("racea%d", round), "woman", []string{"man"}, 48.85, 2.35)
		b := e.onboard(fmt.Sprintf("raceb%d", round), "man", []string{"woman"}, 48.86, 2.34)

		results := make(chan resp, 2)
		start := make(chan struct{})
		for _, pair := range [][2]member{{a, b}, {b, a}} {
			go func(from, to member) {
				<-start
				results <- e.do("POST", "/swipes", from.Token, map[string]any{"targetId": to.ID, "action": "like"})
			}(pair[0], pair[1])
		}
		close(start)
		matched := 0
		for i := 0; i < 2; i++ {
			r := <-results
			if r.Status != 200 {
				t.Fatalf("swipe failed: %d %s", r.Status, r.Raw)
			}
			if r.Body["matched"] == true {
				matched++
			}
		}
		if matched < 1 {
			t.Fatalf("round %d: nobody saw the match", round)
		}
		var n, convs int
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a IN ($1,$2) AND m.user_b IN ($1,$2)`, a.ID, b.ID).Scan(&convs)
		if n != 1 || convs != 1 {
			t.Fatalf("round %d: %d matches, %d conversations", round, n, convs)
		}
	}
}

// Logging out, changing the password elsewhere or deleting the account must kill access tokens at once
// (the test router checks the session on every request, i.e. a zero cache TTL).
func TestSessionRevocationInvalidatesAccessTokens(t *testing.T) {
	e := newEnv(t)
	m, _ := e.register("revoke", "1990-01-01")
	login := e.do("POST", "/auth/login", "", map[string]any{"email": m.Email, "password": "Sup3rSecret!"})
	second := str(login.Body, "accessToken")

	if r := e.do("GET", "/me", m.Token, nil); r.Status != 200 {
		t.Fatalf("fresh token: %d", r.Status)
	}
	// Password change keeps the calling session and revokes the other one.
	if r := e.do("POST", "/me/password", m.Token, map[string]any{"currentPassword": "Sup3rSecret!", "newPassword": "An0therSecret!"}); r.Status != 204 {
		t.Fatalf("change password: %d", r.Status)
	}
	if r := e.do("GET", "/me", second, nil); r.Status != 401 {
		t.Fatalf("other session must be revoked after a password change, got %d", r.Status)
	}
	if r := e.do("GET", "/me", m.Token, nil); r.Status != 200 {
		t.Fatalf("calling session must survive, got %d", r.Status)
	}
	// Logout.
	lg := e.do("POST", "/auth/login", "", map[string]any{"email": m.Email, "password": "An0therSecret!"})
	if r := e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": str(lg.Body, "refreshToken")}); r.Status != 204 {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := e.do("GET", "/me", str(lg.Body, "accessToken"), nil); r.Status != 401 {
		t.Fatalf("access token must die with its session, got %d", r.Status)
	}
	// Account deletion.
	if r := e.do("DELETE", "/me", m.Token, map[string]any{"password": "An0therSecret!"}); r.Status != 204 {
		t.Fatalf("delete: %d", r.Status)
	}
	if r := e.do("PUT", "/profile", m.Token, map[string]any{"firstName": "Ghost", "gender": "man"}); r.Status != 401 {
		t.Fatalf("deleted account's token must be rejected, got %d", r.Status)
	}
}
