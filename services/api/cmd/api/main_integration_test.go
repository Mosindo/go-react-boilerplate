package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/storage"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testJWTSecret = "integration-suite-secret-0123456789abcdef"

var (
	migrateOnce sync.Once
	migrateErr  error
)

type env struct {
	t     *testing.T
	srv   *httptest.Server
	pool  *pgxpool.Pool
	files *storage.Memory
	mail  *mail.Capture
}

type user struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

// newEnv boots the full HTTP stack against a real Postgres (DATABASE_URL_TEST) with empty tables.
func newEnv(t *testing.T, relaxLimits bool) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST (or DATABASE_URL) must point to a disposable Postgres database")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	migrateOnce.Do(func() {
		if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
			migrateErr = err
			return
		}
		migrateErr = db.RunMigrations(ctx, pool)
	})
	if migrateErr != nil {
		t.Fatalf("migrate: %v", migrateErr)
	}
	if err := db.RunMigrations(ctx, pool); err != nil { // idempotent second run
		t.Fatalf("re-run migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, reports CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	e := &env{t: t, pool: pool, files: storage.NewMemory(), mail: &mail.Capture{}}
	router, err := newRouter(Dependencies{
		Config: config.Config{Env: "test", JWTSecret: testJWTSecret, MaxPhotoBytes: 2 << 20},
		DB:     pool, Files: e.files, Mailer: e.mail, RelaxLimits: relaxLimits,
	})
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	e.srv = httptest.NewServer(router)
	t.Cleanup(e.srv.Close)
	return e
}

type obj = map[string]any

// call performs a JSON request and decodes the JSON response (if any) into out.
func (e *env) call(method, path, token string, body any, out any) int {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, reader)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			e.t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	return res.StatusCode
}

func (e *env) expect(want int, method, path, token string, body any, out any) {
	e.t.Helper()
	if got := e.call(method, path, token, body, out); got != want {
		e.t.Fatalf("%s %s: status %d, want %d", method, path, got, want)
	}
}

func (e *env) upload(token string, data []byte, filename string) (int, obj) {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "image/jpeg") // a lie the server must not trust
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/profile/photos", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	var out obj
	raw, _ := io.ReadAll(res.Body)
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

func testJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func testPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: 30, B: 30, A: 128})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func (e *env) register(email string) user {
	e.t.Helper()
	var res struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e.expect(http.StatusCreated, "POST", "/auth/register", "", obj{"email": email, "password": "correct-horse"}, &res)
	return user{ID: res.User.ID, Email: email, Token: res.AccessToken, Refresh: res.RefreshToken}
}

type profileOpts struct {
	name         string
	gender       string
	interestedIn []string
	birth        string
	interests    []string
	lat, lng     *float64
	maxDistance  int
	noPhoto      bool
	discoverable bool
}

func (e *env) onboard(email string, o profileOpts) user {
	e.t.Helper()
	u := e.register(email)
	if o.birth == "" {
		o.birth = "1992-06-15"
	}
	if o.interests == nil {
		o.interests = []string{"travel", "music"}
	}
	disc := true
	e.expect(200, "PUT", "/profile", u.Token, obj{
		"displayName": o.name, "birthDate": o.birth, "gender": o.gender, "bio": "Hello there",
		"city": "Paris", "interests": o.interests, "showDistance": true, "discoverable": disc,
	}, nil)
	maxDist := o.maxDistance
	if maxDist == 0 {
		maxDist = 100
	}
	e.expect(200, "PUT", "/preferences", u.Token, obj{"interestedIn": o.interestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": maxDist}, nil)
	if o.lat != nil {
		e.expect(204, "PUT", "/profile/location", u.Token, obj{"latitude": *o.lat, "longitude": *o.lng}, nil)
	}
	if !o.noPhoto {
		if status, body := e.upload(u.Token, testJPEG(640, 480), "me.jpg"); status != http.StatusCreated {
			e.t.Fatalf("upload: %d %v", status, body)
		}
	}
	return u
}

func fp(v float64) *float64 { return &v }

func (e *env) discover(u user) []obj {
	e.t.Helper()
	var res struct {
		Profiles []obj `json:"profiles"`
	}
	e.expect(200, "GET", "/discover?limit=20", u.Token, nil, &res)
	return res.Profiles
}

func containsUser(list []obj, id string) bool {
	for _, p := range list {
		if p["userId"] == id {
			return true
		}
	}
	return false
}

func (e *env) swipe(u user, target, action string) obj {
	e.t.Helper()
	var out obj
	e.expect(200, "POST", "/swipes", u.Token, obj{"targetId": target, "action": action}, &out)
	return out
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// pair creates a mutual match between a woman and a man and returns both plus the conversation id.
func (e *env) matchedPair() (a, b user, conv, matchID string) {
	e.t.Helper()
	a = e.onboard("alice@test.invalid", profileOpts{name: "Alice", gender: "woman", interestedIn: []string{"man"}})
	b = e.onboard("bob@test.invalid", profileOpts{name: "Bob", gender: "man", interestedIn: []string{"woman"}})
	e.swipe(a, b.ID, "like")
	res := e.swipe(b, a.ID, "like")
	m, _ := res["match"].(obj)
	if res["matched"] != true || m == nil {
		e.t.Fatalf("expected match, got %v", res)
	}
	return a, b, m["conversationId"].(string), m["id"].(string)
}

func TestHealth(t *testing.T) {
	e := newEnv(t, true)
	var out obj
	e.expect(200, "GET", "/health", "", nil, &out)
	if out["status"] != "ok" {
		t.Fatalf("health: %v", out)
	}
	e.expect(401, "GET", "/me", "", nil, nil)
	e.expect(401, "GET", "/discover", "garbage", nil, nil)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t, true)

	e.expect(400, "POST", "/auth/register", "", obj{"email": "not-an-email", "password": "correct-horse"}, nil)
	e.expect(400, "POST", "/auth/register", "", obj{"email": "x@test.invalid", "password": "short"}, nil)
	e.expect(400, "POST", "/auth/register", "", obj{"email": "x@test.invalid", "password": strings.Repeat("a", 73)}, nil)

	u := e.register("  Mixed.Case@Test.Invalid ")
	if u.Email != "  Mixed.Case@Test.Invalid " {
		t.Fatal("unexpected helper behaviour")
	}
	var me obj
	e.expect(200, "GET", "/me", u.Token, nil, &me)
	if me["email"] != "mixed.case@test.invalid" || me["profileCompleted"] != false {
		t.Fatalf("me: %v", me)
	}
	e.expect(409, "POST", "/auth/register", "", obj{"email": "MIXED.case@test.invalid", "password": "correct-horse"}, nil)

	e.expect(401, "POST", "/auth/login", "", obj{"email": "mixed.case@test.invalid", "password": "wrong-password"}, nil)
	e.expect(401, "POST", "/auth/login", "", obj{"email": "nobody@test.invalid", "password": "wrong-password"}, nil)
	var login struct {
		AccessToken, RefreshToken string
	}
	var loginRaw obj
	e.expect(200, "POST", "/auth/login", "", obj{"email": "Mixed.Case@test.invalid", "password": "correct-horse"}, &loginRaw)
	login.AccessToken, login.RefreshToken = loginRaw["accessToken"].(string), loginRaw["refreshToken"].(string)

	// Refresh rotates: the old refresh token dies, the new one works.
	var rotated obj
	e.expect(200, "POST", "/auth/refresh", "", obj{"refreshToken": login.RefreshToken}, &rotated)
	e.expect(401, "POST", "/auth/refresh", "", obj{"refreshToken": login.RefreshToken}, nil)
	next := rotated["refreshToken"].(string)

	// Logout revokes the session immediately, even though the access token has not expired.
	e.expect(204, "POST", "/auth/logout", "", obj{"refreshToken": next}, nil)
	e.expect(401, "GET", "/me", rotated["accessToken"].(string), nil, nil)
	e.expect(401, "POST", "/auth/refresh", "", obj{"refreshToken": next}, nil)

	// Password recovery: never reveals existence, token is single-use, kills every session.
	e.expect(202, "POST", "/auth/forgot-password", "", obj{"email": "nobody@test.invalid"}, nil)
	if len(e.mail.Sent) != 0 {
		t.Fatal("mail sent for unknown account")
	}
	e.expect(202, "POST", "/auth/forgot-password", "", obj{"email": "mixed.case@test.invalid"}, nil)
	if len(e.mail.Sent) != 1 {
		t.Fatalf("expected 1 reset mail, got %d", len(e.mail.Sent))
	}
	token := regexp.MustCompile(`Reset code[^:]*: (\S+)`).FindStringSubmatch(e.mail.Sent[0].Body)[1]
	e.expect(400, "POST", "/auth/reset-password", "", obj{"token": "bogus", "password": "brand-new-pass"}, nil)
	e.expect(400, "POST", "/auth/reset-password", "", obj{"token": token, "password": "short"}, nil)
	e.expect(204, "POST", "/auth/reset-password", "", obj{"token": token, "password": "brand-new-pass"}, nil)
	e.expect(400, "POST", "/auth/reset-password", "", obj{"token": token, "password": "another-new-pass"}, nil)
	e.expect(401, "GET", "/me", u.Token, nil, nil)
	e.expect(401, "POST", "/auth/login", "", obj{"email": "mixed.case@test.invalid", "password": "correct-horse"}, nil)
	var again obj
	e.expect(200, "POST", "/auth/login", "", obj{"email": "mixed.case@test.invalid", "password": "brand-new-pass"}, &again)

	// Change password keeps the current session, revokes the others.
	other := e.call("POST", "/auth/login", "", obj{"email": "mixed.case@test.invalid", "password": "brand-new-pass"}, &loginRaw)
	if other != 200 {
		t.Fatal("second login failed")
	}
	otherToken := loginRaw["accessToken"].(string)
	cur := again["accessToken"].(string)
	e.expect(400, "POST", "/me/password", cur, obj{"currentPassword": "nope", "newPassword": "yet-another-pass"}, nil)
	e.expect(204, "POST", "/me/password", cur, obj{"currentPassword": "brand-new-pass", "newPassword": "yet-another-pass"}, nil)
	e.expect(200, "GET", "/me", cur, nil, nil)
	e.expect(401, "GET", "/me", otherToken, nil, nil)
}

func TestJWTHardening(t *testing.T) {
	e := newEnv(t, true)
	u := e.register("jwt@test.invalid")
	parts := strings.Split(u.Token, ".")
	// alg=none must be rejected.
	none := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." + parts[1] + "."
	e.expect(401, "GET", "/me", none, nil, nil)
	// Tampered payload must be rejected.
	e.expect(401, "GET", "/me", parts[0]+"."+parts[1]+"x."+parts[2], nil, nil)
}

func TestRateLimitOnAuth(t *testing.T) {
	e := newEnv(t, false)
	limited := false
	for i := 0; i < 40; i++ {
		if e.call("POST", "/auth/login", "", obj{"email": "spam@test.invalid", "password": "whatever-pass"}, nil) == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected /auth/login to be rate limited")
	}
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t, true)
	u := e.register("val@test.invalid")
	e.expect(404, "GET", "/profile", u.Token, nil, nil)

	base := obj{"displayName": "Val", "birthDate": "1990-01-02", "gender": "woman", "bio": "", "city": "Lyon", "interests": []string{"travel"}}
	with := func(k string, v any) obj {
		c := obj{}
		for kk, vv := range base {
			c[kk] = vv
		}
		c[k] = v
		return c
	}
	underage := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(400, "PUT", "/profile", u.Token, with("birthDate", underage), nil)
	e.expect(400, "PUT", "/profile", u.Token, with("birthDate", "31/12/1990"), nil)
	e.expect(400, "PUT", "/profile", u.Token, with("displayName", "V"), nil)
	e.expect(400, "PUT", "/profile", u.Token, with("gender", "robot"), nil)
	e.expect(400, "PUT", "/profile", u.Token, with("bio", strings.Repeat("x", 501)), nil)
	e.expect(400, "PUT", "/profile", u.Token, with("interests", []string{"does-not-exist"}), nil)
	e.expect(400, "PUT", "/profile", u.Token, with("interests", []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10", "a11"}), nil)

	var p obj
	e.expect(200, "PUT", "/profile", u.Token, with("displayName", "  Val   \x00 Ent "), &p)
	if p["displayName"] != "Val Ent" || p["completed"] != false {
		t.Fatalf("profile: %v", p)
	}
	e.expect(400, "PUT", "/profile", u.Token, with("birthDate", "1985-05-05"), nil) // immutable

	e.expect(400, "PUT", "/preferences", u.Token, obj{"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 10}, nil)
	e.expect(400, "PUT", "/preferences", u.Token, obj{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10}, nil)
	e.expect(400, "PUT", "/preferences", u.Token, obj{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 20, "maxDistanceKm": 10}, nil)
	e.expect(400, "PUT", "/preferences", u.Token, obj{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 9000}, nil)
	e.expect(200, "PUT", "/preferences", u.Token, obj{"interestedIn": []string{"man", "man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 25}, nil)

	// Coordinates are rounded to ~1 km before storage; invalid ones are refused.
	e.expect(400, "PUT", "/profile/location", u.Token, obj{"latitude": 123, "longitude": 2}, nil)
	e.expect(400, "PUT", "/profile/location", u.Token, obj{"latitude": 48.8566}, nil)
	e.expect(204, "PUT", "/profile/location", u.Token, obj{"latitude": 48.856614, "longitude": 2.352222}, nil)
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates not rounded: %v %v", lat, lng)
	}
	e.expect(204, "DELETE", "/profile/location", u.Token, nil, nil)
	if e.count(`SELECT COUNT(*) FROM profiles WHERE latitude IS NOT NULL`) != 0 {
		t.Fatal("location not cleared")
	}

	var interests struct {
		Interests []obj `json:"interests"`
	}
	e.expect(200, "GET", "/interests", u.Token, nil, &interests)
	if len(interests.Interests) < 20 {
		t.Fatalf("interest catalog too small: %d", len(interests.Interests))
	}
}

func TestPhotoLifecycleAndSecurity(t *testing.T) {
	e := newEnv(t, true)
	owner := e.onboard("owner@test.invalid", profileOpts{name: "Owner", gender: "woman", interestedIn: []string{"man"}, noPhoto: true})

	// Hostile or invalid uploads.
	if s, _ := e.upload(owner.Token, []byte("<?php echo 1; ?>"), "evil.jpg"); s != 400 {
		t.Fatalf("text disguised as jpg: %d", s)
	}
	if s, _ := e.upload(owner.Token, []byte("GIF89a\x01\x00\x01\x00"), "x.gif"); s != 400 {
		t.Fatalf("gif accepted: %d", s)
	}
	if s, _ := e.upload(owner.Token, testJPEG(50, 50), "tiny.jpg"); s != 400 {
		t.Fatalf("tiny image accepted: %d", s)
	}
	if s, _ := e.upload(owner.Token, append(testJPEG(300, 300), bytes.Repeat([]byte{0}, 3<<20)...), "big.jpg"); s != 413 {
		t.Fatalf("oversized upload: %d", s)
	}
	e.expect(401, "POST", "/profile/photos", "", nil, nil)

	var ids []string
	for i := 0; i < 6; i++ {
		data := testJPEG(2000, 1500)
		if i == 1 {
			data = testPNG(400, 400)
		}
		s, body := e.upload(owner.Token, data, "p.jpg")
		if s != 201 {
			t.Fatalf("upload %d: %d %v", i, s, body)
		}
		ids = append(ids, body["id"].(string))
		if int(body["position"].(float64)) != i {
			t.Fatalf("position %v != %d", body["position"], i)
		}
	}
	if s, _ := e.upload(owner.Token, testJPEG(400, 400), "7.jpg"); s != 409 {
		t.Fatalf("7th photo: %d", s)
	}

	// Stored renditions are real, bounded JPEGs.
	var stored int
	for key, data := range e.files.Files {
		stored++
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || format != "jpeg" {
			t.Fatalf("%s is not a jpeg: %v %s", key, err, format)
		}
		limit := 1440
		if strings.HasSuffix(key, ".thumb.jpg") {
			limit = 360
		}
		if cfg.Width > limit || cfg.Height > limit {
			t.Fatalf("%s too large: %dx%d", key, cfg.Width, cfg.Height)
		}
	}
	if stored != 12 {
		t.Fatalf("expected 12 stored files, got %d", stored)
	}

	// Reorder.
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	e.expect(400, "PUT", "/profile/photos/order", owner.Token, obj{"ids": ids[:5]}, nil)
	e.expect(400, "PUT", "/profile/photos/order", owner.Token, obj{"ids": []string{ids[0], ids[0], ids[1], ids[2], ids[3], ids[4]}}, nil)
	var ordered struct {
		Photos []obj `json:"photos"`
	}
	e.expect(200, "PUT", "/profile/photos/order", owner.Token, obj{"ids": rev}, &ordered)
	if ordered.Photos[0]["id"] != ids[5] {
		t.Fatalf("main photo not updated: %v", ordered.Photos)
	}

	// Owner can fetch both renditions with proper headers.
	req, _ := http.NewRequest("GET", e.srv.URL+"/photos/"+ids[0]+"/file?size=thumb", nil)
	req.Header.Set("Authorization", "Bearer "+owner.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" || res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("file response: %d %v", res.StatusCode, res.Header)
	}
	e.expect(401, "GET", "/photos/"+ids[0]+"/file", "", nil, nil)
	e.expect(404, "GET", "/photos/not-a-uuid/file", owner.Token, nil, nil)
	e.expect(400, "GET", "/photos/"+ids[0]+"/file?size=huge", owner.Token, nil, nil)

	// A stranger who has no right to see the profile (hidden profile) gets 404, not the bytes.
	stranger := e.onboard("stranger@test.invalid", profileOpts{name: "Stranger", gender: "man", interestedIn: []string{"woman"}})
	e.expect(200, "GET", "/photos/"+ids[0]+"/file", stranger.Token, nil, nil) // discoverable -> visible
	e.expect(200, "PUT", "/profile", owner.Token, obj{"displayName": "Owner", "birthDate": "1992-06-15", "gender": "woman", "discoverable": false, "interests": []string{}}, nil)
	e.expect(404, "GET", "/photos/"+ids[0]+"/file", stranger.Token, nil, nil)
	e.expect(404, "GET", "/profiles/"+owner.ID, stranger.Token, nil, nil)
	e.expect(200, "GET", "/photos/"+ids[0]+"/file", owner.Token, nil, nil)
	e.expect(404, "DELETE", "/profile/photos/"+ids[0], stranger.Token, nil, nil) // not yours

	// Delete compacts positions and removes the files.
	e.expect(204, "DELETE", "/profile/photos/"+ids[5], owner.Token, nil, nil)
	if len(e.files.Files) != 10+2 { // owner: 5 photos*2 files, stranger: 1 photo*2 files
		t.Fatalf("files after delete: %d", len(e.files.Files))
	}
	var prof obj
	e.expect(200, "GET", "/profile", owner.Token, nil, &prof)
	photos := prof["photos"].([]any)
	for i, p := range photos {
		if int(p.(obj)["position"].(float64)) != i {
			t.Fatalf("positions not compact: %v", photos)
		}
	}
}

func TestDiscoveryFiltersAndRanking(t *testing.T) {
	e := newEnv(t, true)
	paris, marseille := fp(48.86), fp(43.30)
	parisLng, marseilleLng := fp(2.35), fp(5.37)

	me := e.onboard("me@test.invalid", profileOpts{name: "Me", gender: "man", interestedIn: []string{"woman"}, lat: paris, lng: parisLng, maxDistance: 100, interests: []string{"travel", "music", "yoga"}})
	match := e.onboard("match@test.invalid", profileOpts{name: "Match", gender: "woman", interestedIn: []string{"man"}, lat: paris, lng: parisLng, interests: []string{"travel", "music"}})
	far := e.onboard("far@test.invalid", profileOpts{name: "Far", gender: "woman", interestedIn: []string{"man"}, lat: marseille, lng: marseilleLng})
	notInto := e.onboard("notinto@test.invalid", profileOpts{name: "NotInto", gender: "woman", interestedIn: []string{"woman"}, lat: paris, lng: parisLng})
	wrongSex := e.onboard("guy@test.invalid", profileOpts{name: "Guy", gender: "man", interestedIn: []string{"man"}, lat: paris, lng: parisLng})
	old := e.onboard("old@test.invalid", profileOpts{name: "Old", gender: "woman", interestedIn: []string{"man"}, birth: "1950-01-01", lat: paris, lng: parisLng})
	noPhoto := e.onboard("nophoto@test.invalid", profileOpts{name: "NoPhoto", gender: "woman", interestedIn: []string{"man"}, noPhoto: true})
	hidden := e.onboard("hidden@test.invalid", profileOpts{name: "Hidden", gender: "woman", interestedIn: []string{"man"}, lat: paris, lng: parisLng})
	e.expect(200, "PUT", "/profile", hidden.Token, obj{"displayName": "Hidden", "birthDate": "1992-06-15", "gender": "woman", "discoverable": false, "interests": []string{}}, nil)
	incomplete := e.register("incomplete@test.invalid")

	list := e.discover(me)
	if !containsUser(list, match.ID) {
		t.Fatalf("expected the compatible profile, got %v", list)
	}
	for name, u := range map[string]user{"far": far, "notInto": notInto, "wrongSex": wrongSex, "old": old, "noPhoto": noPhoto, "hidden": hidden, "self": me, "incomplete": incomplete} {
		if containsUser(list, u.ID) {
			t.Fatalf("%s must not be discoverable", name)
		}
	}

	// Privacy of the payload: no coordinates, birth date or email; distance is approximate.
	p := list[0]
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "lat", "lng"} {
		if _, ok := p[forbidden]; ok {
			t.Fatalf("payload leaks %q: %v", forbidden, p)
		}
	}
	if d, ok := p["distanceKm"].(float64); !ok || d != 5 {
		t.Fatalf("expected approximate distance of 5, got %v", p["distanceKm"])
	}
	if p["commonInterests"].(float64) != 2 {
		t.Fatalf("common interests: %v", p["commonInterests"])
	}

	// Incomplete profiles are told why.
	var errBody obj
	e.expect(409, "GET", "/discover", incomplete.Token, nil, &errBody)
	if errBody["code"] != "profile_incomplete" {
		t.Fatalf("code: %v", errBody)
	}
	e.expect(409, "POST", "/swipes", incomplete.Token, obj{"targetId": me.ID, "action": "like"}, nil)
	e.expect(400, "GET", "/discover?limit=500", me.Token, nil, nil)

	// A pass removes the profile forever.
	e.swipe(me, match.ID, "pass")
	if containsUser(e.discover(me), match.ID) {
		t.Fatal("passed profile came back")
	}
}

func TestSwipeAndMatchRules(t *testing.T) {
	e := newEnv(t, true)
	a := e.onboard("a@test.invalid", profileOpts{name: "Ann", gender: "woman", interestedIn: []string{"man"}})
	b := e.onboard("b@test.invalid", profileOpts{name: "Ben", gender: "man", interestedIn: []string{"woman"}})

	e.expect(400, "POST", "/swipes", a.Token, obj{"targetId": a.ID, "action": "like"}, nil)
	e.expect(400, "POST", "/swipes", a.Token, obj{"targetId": b.ID, "action": "superlike"}, nil)
	e.expect(400, "POST", "/swipes", a.Token, obj{"targetId": "nope", "action": "like"}, nil)
	e.expect(404, "POST", "/swipes", a.Token, obj{"targetId": "00000000-0000-0000-0000-000000000000", "action": "like"}, nil)

	// The server refuses likes for people the viewer could never have been shown.
	incompatible := e.onboard("c@test.invalid", profileOpts{name: "Cy", gender: "man", interestedIn: []string{"man"}})
	e.expect(404, "POST", "/swipes", a.Token, obj{"targetId": incompatible.ID, "action": "like"}, nil)

	first := e.swipe(a, b.ID, "like")
	if first["matched"] != false {
		t.Fatalf("unexpected match: %v", first)
	}
	// Repeating the like (or trying to flip it) is idempotent.
	e.swipe(a, b.ID, "like")
	flipped := e.swipe(a, b.ID, "pass")
	if flipped["action"] != "like" {
		t.Fatalf("decision changed: %v", flipped)
	}
	if e.count(`SELECT COUNT(*) FROM swipes`) != 1 || e.count(`SELECT COUNT(*) FROM matches`) != 0 {
		t.Fatal("unexpected rows after repeated swipes")
	}

	// B sees A first (they already liked B), likes back and matches.
	list := e.discover(b)
	if len(list) == 0 || list[0]["userId"] != a.ID {
		t.Fatalf("A should lead B's deck: %v", list)
	}
	second := e.swipe(b, a.ID, "like")
	if second["matched"] != true || second["match"] == nil {
		t.Fatalf("expected match: %v", second)
	}
	again := e.swipe(b, a.ID, "like")
	if again["matched"] != true {
		t.Fatalf("repeat should still report the match: %v", again)
	}
	if e.count(`SELECT COUNT(*) FROM matches`) != 1 || e.count(`SELECT COUNT(*) FROM conversations`) != 1 ||
		e.count(`SELECT COUNT(*) FROM conversation_participants`) != 2 || e.count(`SELECT COUNT(*) FROM notifications WHERE type='match'`) != 2 {
		t.Fatal("match side effects are inconsistent")
	}

	// Both sides see the match and a notification.
	for _, u := range []user{a, b} {
		var ms struct {
			Matches []obj `json:"matches"`
		}
		e.expect(200, "GET", "/matches", u.Token, nil, &ms)
		if len(ms.Matches) != 1 {
			t.Fatalf("matches for %s: %v", u.Email, ms.Matches)
		}
		var ns struct {
			Notifications []obj `json:"notifications"`
		}
		e.expect(200, "GET", "/notifications", u.Token, nil, &ns)
		if len(ns.Notifications) != 1 || ns.Notifications[0]["type"] != "match" || ns.Notifications[0]["readAt"] != nil {
			t.Fatalf("notifications: %v", ns.Notifications)
		}
		var cnt obj
		e.expect(200, "GET", "/notifications/unread-count", u.Token, nil, &cnt)
		if cnt["unreadCount"].(float64) != 1 {
			t.Fatalf("unread count: %v", cnt)
		}
		e.expect(204, "POST", "/notifications/"+ns.Notifications[0]["id"].(string)+"/read", u.Token, nil, nil)
		e.expect(404, "POST", "/notifications/00000000-0000-0000-0000-000000000000/read", u.Token, nil, nil)
		e.expect(204, "POST", "/notifications/read-all", u.Token, nil, nil)
		e.expect(200, "GET", "/notifications/unread-count", u.Token, nil, &cnt)
		if cnt["unreadCount"].(float64) != 0 {
			t.Fatalf("unread after read: %v", cnt)
		}
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t, true)
	for i := 0; i < 5; i++ {
		a := e.onboard("ra"+string(rune('a'+i))+"@test.invalid", profileOpts{name: "Alpha", gender: "woman", interestedIn: []string{"man"}})
		b := e.onboard("rb"+string(rune('a'+i))+"@test.invalid", profileOpts{name: "Bravo", gender: "man", interestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]obj, 2)
		for idx, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(idx int, from, to user) {
				defer wg.Done()
				e.call("POST", "/swipes", from.Token, obj{"targetId": to.ID, "action": "like"}, &results[idx])
			}(idx, pair[0], pair[1])
		}
		wg.Wait()
		if results[0]["matched"] != true && results[1]["matched"] != true {
			t.Fatalf("round %d: nobody saw the match: %v", i, results)
		}
		if n := e.count(`SELECT COUNT(*) FROM matches WHERE user_a IN ($1, $2)`, a.ID, b.ID); n != 1 {
			t.Fatalf("round %d: %d matches", i, n)
		}
	}
}

func TestChatPermissionsAndReadState(t *testing.T) {
	e := newEnv(t, true)
	a, b, conv, matchID := e.matchedPair()
	eve := e.onboard("eve@test.invalid", profileOpts{name: "Eve", gender: "woman", interestedIn: []string{"man"}})

	// Strangers are locked out of every conversation endpoint (404, never 403: no existence leak).
	e.expect(404, "GET", "/conversations/"+conv+"/messages", eve.Token, nil, nil)
	e.expect(404, "POST", "/conversations/"+conv+"/messages", eve.Token, obj{"body": "hi"}, nil)
	e.expect(404, "POST", "/conversations/"+conv+"/read", eve.Token, nil, nil)
	e.expect(404, "DELETE", "/conversations/"+conv, eve.Token, nil, nil)
	e.expect(404, "DELETE", "/matches/"+matchID, eve.Token, nil, nil)
	e.expect(404, "GET", "/conversations/not-a-uuid/messages", a.Token, nil, nil)

	e.expect(400, "POST", "/conversations/"+conv+"/messages", a.Token, obj{"body": "   \n "}, nil)
	e.expect(400, "POST", "/conversations/"+conv+"/messages", a.Token, obj{"body": strings.Repeat("é", 2001)}, nil)

	var m1 obj
	e.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, obj{"body": "  Hello Bob \x00"}, &m1)
	if m1["body"] != "Hello Bob" || m1["senderId"] != a.ID || m1["readAt"] != nil {
		t.Fatalf("message: %v", m1)
	}
	e.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, obj{"body": "<script>alert(1)</script>"}, nil)

	// Unread counters and conversation list for the recipient.
	var cnt obj
	e.expect(200, "GET", "/conversations/unread-count", b.Token, nil, &cnt)
	if cnt["unreadCount"].(float64) != 2 {
		t.Fatalf("unread: %v", cnt)
	}
	var convs struct {
		Conversations []obj `json:"conversations"`
	}
	e.expect(200, "GET", "/conversations", b.Token, nil, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0]["unreadCount"].(float64) != 2 {
		t.Fatalf("conversations: %v", convs)
	}
	e.expect(200, "GET", "/conversations", eve.Token, nil, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("eve sees a conversation")
	}

	// Reading flips unread to 0 and exposes readAt to the sender.
	e.expect(204, "POST", "/conversations/"+conv+"/read", b.Token, nil, nil)
	e.expect(200, "GET", "/conversations/unread-count", b.Token, nil, &cnt)
	if cnt["unreadCount"].(float64) != 0 {
		t.Fatalf("unread after read: %v", cnt)
	}
	var msgs struct {
		Messages   []obj   `json:"messages"`
		NextCursor *string `json:"nextCursor"`
	}
	e.expect(200, "GET", "/conversations/"+conv+"/messages", a.Token, nil, &msgs)
	if len(msgs.Messages) != 2 || msgs.Messages[0]["readAt"] == nil {
		t.Fatalf("read receipt missing: %v", msgs.Messages)
	}

	// Pagination, newest first.
	for i := 0; i < 5; i++ {
		e.expect(201, "POST", "/conversations/"+conv+"/messages", b.Token, obj{"body": "reply"}, nil)
	}
	e.expect(200, "GET", "/conversations/"+conv+"/messages?limit=3", a.Token, nil, &msgs)
	if len(msgs.Messages) != 3 || msgs.NextCursor == nil {
		t.Fatalf("page 1: %v", msgs)
	}
	seen := map[string]bool{}
	total := 0
	cursor := ""
	for {
		path := "/conversations/" + conv + "/messages?limit=3"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		e.expect(200, "GET", path, a.Token, nil, &msgs)
		for _, m := range msgs.Messages {
			if seen[m["id"].(string)] {
				t.Fatal("duplicate across pages")
			}
			seen[m["id"].(string)] = true
			total++
		}
		if msgs.NextCursor == nil {
			break
		}
		cursor = *msgs.NextCursor
	}
	if total != 7 {
		t.Fatalf("paged %d messages, want 7", total)
	}
	e.expect(400, "GET", "/conversations/"+conv+"/messages?cursor=garbage", a.Token, nil, nil)

	// Local deletion clears history for me only; new activity brings the thread back.
	e.expect(204, "DELETE", "/conversations/"+conv, a.Token, nil, nil)
	e.expect(200, "GET", "/conversations", a.Token, nil, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("hidden conversation still listed")
	}
	e.expect(200, "GET", "/conversations/"+conv+"/messages", a.Token, nil, &msgs)
	if len(msgs.Messages) != 0 {
		t.Fatalf("history not cleared for me: %d", len(msgs.Messages))
	}
	e.expect(200, "GET", "/conversations/"+conv+"/messages", b.Token, nil, &msgs)
	if len(msgs.Messages) != 7 {
		t.Fatalf("other side lost history: %d", len(msgs.Messages))
	}
	e.expect(201, "POST", "/conversations/"+conv+"/messages", b.Token, obj{"body": "are you there?"}, nil)
	e.expect(200, "GET", "/conversations", a.Token, nil, &convs)
	if len(convs.Conversations) != 1 {
		t.Fatal("conversation did not resurface")
	}

	// Unmatch closes the conversation for both.
	e.expect(204, "DELETE", "/matches/"+matchID, a.Token, nil, nil)
	e.expect(403, "POST", "/conversations/"+conv+"/messages", b.Token, obj{"body": "hello?"}, nil)
	e.expect(404, "DELETE", "/matches/"+matchID, a.Token, nil, nil)
	var ms struct {
		Matches []obj `json:"matches"`
	}
	e.expect(200, "GET", "/matches", b.Token, nil, &ms)
	if len(ms.Matches) != 0 {
		t.Fatal("unmatched pair still listed")
	}
	// And they never reappear in discovery (the like is remembered).
	if containsUser(e.discover(a), b.ID) || containsUser(e.discover(b), a.ID) {
		t.Fatal("unmatched user resurfaced")
	}
}

func TestBlockingAndReporting(t *testing.T) {
	e := newEnv(t, true)
	a, b, conv, _ := e.matchedPair()
	c := e.onboard("carl@test.invalid", profileOpts{name: "Carl", gender: "man", interestedIn: []string{"woman"}})

	e.expect(400, "POST", "/blocks", a.Token, obj{"userId": a.ID}, nil)
	e.expect(400, "POST", "/blocks", a.Token, obj{"userId": "x"}, nil)
	e.expect(404, "POST", "/blocks", a.Token, obj{"userId": "00000000-0000-0000-0000-000000000000"}, nil)
	e.expect(204, "POST", "/blocks", a.Token, obj{"userId": b.ID}, nil)
	e.expect(204, "POST", "/blocks", a.Token, obj{"userId": b.ID}, nil) // idempotent

	// The block removes the match and hides both people from each other, in both directions.
	e.expect(403, "POST", "/conversations/"+conv+"/messages", b.Token, obj{"body": "please"}, nil)
	e.expect(404, "GET", "/profiles/"+b.ID, a.Token, nil, nil)
	e.expect(404, "GET", "/profiles/"+a.ID, b.Token, nil, nil)
	if containsUser(e.discover(a), b.ID) || containsUser(e.discover(b), a.ID) {
		t.Fatal("blocked users still discoverable")
	}
	var bl struct {
		Blocks []obj `json:"blocks"`
	}
	e.expect(200, "GET", "/blocks", a.Token, nil, &bl)
	if len(bl.Blocks) != 1 || bl.Blocks[0]["userId"] != b.ID || bl.Blocks[0]["displayName"] != "Bob" {
		t.Fatalf("blocks: %v", bl)
	}
	e.expect(200, "GET", "/blocks", b.Token, nil, &bl)
	if len(bl.Blocks) != 0 {
		t.Fatal("blocked user can see who blocked them")
	}

	// Unblocking restores visibility but never the old match.
	e.expect(204, "DELETE", "/blocks/"+b.ID, a.Token, nil, nil)
	e.expect(403, "POST", "/conversations/"+conv+"/messages", b.Token, obj{"body": "hi again"}, nil)
	var ms struct {
		Matches []obj `json:"matches"`
	}
	e.expect(200, "GET", "/matches", a.Token, nil, &ms)
	if len(ms.Matches) != 0 {
		t.Fatal("match resurrected by unblock")
	}

	// Reporting.
	e.expect(400, "POST", "/reports", a.Token, obj{"userId": c.ID, "reason": "meh"}, nil)
	e.expect(400, "POST", "/reports", a.Token, obj{"userId": a.ID, "reason": "spam"}, nil)
	e.expect(404, "POST", "/reports", a.Token, obj{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}, nil)
	e.expect(400, "POST", "/reports", a.Token, obj{"userId": c.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}, nil)
	var r1, r2 obj
	e.expect(201, "POST", "/reports", a.Token, obj{"userId": c.ID, "reason": "harassment", "details": "rude"}, &r1)
	e.expect(201, "POST", "/reports", a.Token, obj{"userId": c.ID, "reason": "harassment", "details": "rude"}, &r2)
	if r1["id"] != r2["id"] || e.count(`SELECT COUNT(*) FROM reports`) != 1 {
		t.Fatal("duplicate report not collapsed")
	}
	e.expect(201, "POST", "/reports", a.Token, obj{"userId": c.ID, "reason": "fake", "block": true}, nil)
	e.expect(404, "GET", "/profiles/"+c.ID, a.Token, nil, nil)
	e.expect(404, "POST", "/swipes", c.Token, obj{"targetId": a.ID, "action": "like"}, nil)
	if e.count(`SELECT COUNT(*) FROM reports`) != 2 {
		t.Fatal("reports missing")
	}
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	e := newEnv(t, true)
	a, b, conv, _ := e.matchedPair()
	e.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, obj{"body": "bye"}, nil)
	e.expect(201, "POST", "/reports", b.Token, obj{"userId": a.ID, "reason": "spam"}, nil)
	e.expect(204, "POST", "/blocks", a.Token, obj{"userId": b.ID}, nil)
	filesBefore := len(e.files.Files)
	if filesBefore != 4 {
		t.Fatalf("setup files: %d", filesBefore)
	}

	e.expect(400, "DELETE", "/me", a.Token, obj{"password": "wrong-password"}, nil)
	e.expect(204, "DELETE", "/me", a.Token, obj{"password": "correct-horse"}, nil)

	e.expect(401, "GET", "/me", a.Token, nil, nil)
	e.expect(401, "POST", "/auth/login", "", obj{"email": a.Email, "password": "correct-horse"}, nil)
	for _, q := range []string{
		`SELECT COUNT(*) FROM users WHERE id = '` + a.ID + `'`,
		`SELECT COUNT(*) FROM profiles WHERE user_id = '` + a.ID + `'`,
		`SELECT COUNT(*) FROM photos WHERE user_id = '` + a.ID + `'`,
		`SELECT COUNT(*) FROM swipes WHERE actor_id = '` + a.ID + `' OR target_id = '` + a.ID + `'`,
		`SELECT COUNT(*) FROM matches`,
		`SELECT COUNT(*) FROM messages`,
		`SELECT COUNT(*) FROM blocks`,
		`SELECT COUNT(*) FROM notifications WHERE user_id = '` + a.ID + `'`,
		`SELECT COUNT(*) FROM sessions WHERE user_id = '` + a.ID + `'`,
	} {
		if n := e.count(q); n != 0 {
			t.Fatalf("%s -> %d rows left", q, n)
		}
	}
	if len(e.files.Files) != 2 {
		t.Fatalf("photo files not removed: %d left", len(e.files.Files))
	}
	// The report is kept for moderation, detached from the deleted account.
	if e.count(`SELECT COUNT(*) FROM reports WHERE reported_id IS NULL`) != 1 {
		t.Fatal("report should survive anonymised")
	}
	// The other person is unaffected.
	e.expect(200, "GET", "/me", b.Token, nil, nil)
}

func TestWebSocketRealtime(t *testing.T) {
	e := newEnv(t, true)
	a := e.onboard("wa@test.invalid", profileOpts{name: "Wa", gender: "woman", interestedIn: []string{"man"}})
	b := e.onboard("wb@test.invalid", profileOpts{name: "Wb", gender: "man", interestedIn: []string{"woman"}})

	dial := func(token string) *websocket.Conn {
		url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close() })
		_ = c.WriteJSON(obj{"type": "auth", "token": token})
		return c
	}
	read := func(c *websocket.Conn) obj {
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev obj
		if err := c.ReadJSON(&ev); err != nil {
			t.Fatalf("read event: %v", err)
		}
		return ev
	}

	// Bad token -> closed with 4401.
	bad := dial("garbage")
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("expected close")
	} else if ce, ok := err.(*websocket.CloseError); !ok || ce.Code != 4401 {
		t.Fatalf("expected close 4401, got %v", err)
	}

	wa, wb := dial(a.Token), dial(b.Token)
	if read(wa)["type"] != "ready" || read(wb)["type"] != "ready" {
		t.Fatal("no ready frame")
	}

	e.swipe(a, b.ID, "like")
	res := e.swipe(b, a.ID, "like")
	conv := res["match"].(obj)["conversationId"].(string)
	gotMatch := map[string]bool{}
	for _, c := range []*websocket.Conn{wa, wb} {
		for i := 0; i < 2; i++ { // match.new + notification.new, any order
			gotMatch[read(c)["type"].(string)] = true
		}
	}
	if !gotMatch["match.new"] || !gotMatch["notification.new"] {
		t.Fatalf("events: %v", gotMatch)
	}

	e.expect(201, "POST", "/conversations/"+conv+"/messages", a.Token, obj{"body": "live!"}, nil)
	ev := read(wb)
	if ev["type"] != "message.new" || ev["data"].(obj)["body"] != "live!" {
		t.Fatalf("event: %v", ev)
	}
	read(wa) // echo to sender's other devices

	e.expect(204, "POST", "/conversations/"+conv+"/read", b.Token, nil, nil)
	if read(wa)["type"] != "message.read" {
		t.Fatal("expected message.read")
	}

	// Logging out kills the socket on the next connection attempt.
	e.expect(204, "POST", "/auth/logout", "", obj{"refreshToken": a.Refresh}, nil)
	late := dial(a.Token)
	_ = late.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := late.ReadMessage(); err == nil {
		t.Fatal("revoked session accepted on websocket")
	}
}
