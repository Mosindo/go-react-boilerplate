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
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	authfeature "example.com/api/internal/features/auth"
	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "k3v9Xq2LmZp8Rt5Wy7Ub4NcAe1HdGfJs"

type captureMailer struct {
	mu   sync.Mutex
	body string
}

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.body = body
	return nil
}

func (m *captureMailer) token() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	i := strings.Index(m.body, "code dans l'application : ")
	if i < 0 {
		return ""
	}
	rest := m.body[i+len("code dans l'application : "):]
	return strings.Fields(rest)[0]
}

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mailer *captureMailer
}

var (
	setupOnce sync.Once
	sharedEnv *env
)

// newEnv connects to DATABASE_URL_TEST (schema wiped once per test run) or skips.
func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST (or DATABASE_URL) not set; skipping integration tests")
	}
	setupOnce.Do(func() {
		ctx := context.Background()
		pool, err := db.Connect(ctx, url)
		if err != nil {
			t.Skipf("postgres unavailable: %v", err)
			return
		}
		if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
			t.Fatalf("reset schema: %v", err)
		}
		if err := db.RunMigrations(ctx, pool); err != nil {
			t.Fatalf("migrations: %v", err)
		}
		mailer := &captureMailer{}
		gin.SetMode(gin.TestMode)
		router, err := setupRouter(&app{
			dbPool: pool, jwtSecret: []byte(testSecret), appLinkBase: "ember://", mailer: mailer,
			limits: limits{auth: authLimitsHigh(), swipesPerMinute: 10000, messagesPerMinute: 10000, uploadsPerHour: 10000, reportsPerHour: 10000},
		})
		if err != nil {
			t.Fatal(err)
		}
		sharedEnv = &env{router: router, pool: pool, mailer: mailer}
	})
	if sharedEnv == nil {
		t.Skip("postgres unavailable")
	}
	e := *sharedEnv
	e.t = t
	return &e
}

func (e *env) do(method, path, token string, body any) (int, []byte) {
	e.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

func (e *env) expect(want int, method, path, token string, body any) []byte {
	e.t.Helper()
	code, out := e.do(method, path, token, body)
	if code != want {
		e.t.Fatalf("%s %s: got %d want %d: %s", method, path, code, want, out)
	}
	return out
}

func parse[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return v
}

type authResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID              string `json:"id"`
		Email           string `json:"email"`
		ProfileComplete bool   `json:"profileComplete"`
	} `json:"user"`
}

var emailSeq int
var emailMu sync.Mutex

func uniqueEmail() string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("user%d-%d@test.invalid", emailSeq, time.Now().UnixNano())
}

func (e *env) register() authResp {
	return parse[authResp](e.t, e.expect(http.StatusCreated, "POST", "/auth/register", "", map[string]string{"email": uniqueEmail(), "password": "correct horse"}))
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func (e *env) upload(method, path, token string, data []byte, filename string) (int, []byte) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", filename)
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes()
}

type person struct {
	authResp
	token string
}

type profileSpec struct {
	name, gender string
	interestedIn []string
	birth        string
	lat, lng     *float64
	maxDistance  *int
}

func f(v float64) *float64 { return &v }
func i(v int) *int         { return &v }

// newPerson registers a user with a complete profile, preferences and one photo.
func (e *env) newPerson(spec profileSpec) person {
	e.t.Helper()
	a := e.register()
	if spec.birth == "" {
		spec.birth = "1992-04-10"
	}
	e.expect(http.StatusOK, "PUT", "/me/profile", a.AccessToken, map[string]any{
		"firstName": spec.name, "birthDate": spec.birth, "gender": spec.gender, "bio": "Bonjour", "city": "Paris", "interestIds": []int{1, 2},
	})
	e.expect(http.StatusOK, "PUT", "/me/preferences", a.AccessToken, map[string]any{
		"interestedIn": spec.interestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": spec.maxDistance,
	})
	if spec.lat != nil {
		e.expect(http.StatusNoContent, "PUT", "/me/location", a.AccessToken, map[string]float64{"latitude": *spec.lat, "longitude": *spec.lng})
	}
	if code, out := e.upload("POST", "/me/photos", a.AccessToken, pngBytes(400, 500), "me.png"); code != http.StatusCreated {
		e.t.Fatalf("photo upload: %d %s", code, out)
	}
	return person{authResp: a, token: a.AccessToken}
}

type profilesResp struct {
	Profiles []struct {
		UserID     string `json:"userId"`
		FirstName  string `json:"firstName"`
		Age        int    `json:"age"`
		DistanceKm *int   `json:"distanceKm"`
		LikedYou   bool   `json:"likedYou"`
		Photos     []struct {
			URL string `json:"url"`
		} `json:"photos"`
	} `json:"profiles"`
}

func (e *env) discover(p person) profilesResp {
	return parse[profilesResp](e.t, e.expect(http.StatusOK, "GET", "/discover?limit=20", p.token, nil))
}

func (e *env) discoverIDs(p person) map[string]bool {
	ids := map[string]bool{}
	for _, pr := range e.discover(p).Profiles {
		ids[pr.UserID] = true
	}
	return ids
}

type swipeResp struct {
	Action       string `json:"action"`
	Matched      bool   `json:"matched"`
	Conversation *struct {
		ID string `json:"id"`
	} `json:"conversation"`
}

func (e *env) swipe(from person, to person, action string) swipeResp {
	return parse[swipeResp](e.t, e.expect(http.StatusOK, "POST", "/swipes", from.token, map[string]string{"userId": to.User.ID, "action": action}))
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(http.StatusOK, "GET", "/health", "", nil)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail()
	creds := map[string]string{"email": strings.ToUpper(" " + email + " "), "password": "correct horse"}
	reg := parse[authResp](t, e.expect(http.StatusCreated, "POST", "/auth/register", "", creds))
	if reg.User.Email != email || reg.User.ProfileComplete {
		t.Fatalf("email must be normalised and profile incomplete: %+v", reg.User)
	}
	e.expect(http.StatusConflict, "POST", "/auth/register", "", creds)
	e.expect(http.StatusBadRequest, "POST", "/auth/register", "", map[string]string{"email": uniqueEmail(), "password": "short"})
	e.expect(http.StatusBadRequest, "POST", "/auth/register", "", map[string]string{"email": "nope", "password": "correct horse"})

	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": email, "password": "wrong password"})
	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": "ghost@test.invalid", "password": "wrong password"})
	login := parse[authResp](t, e.expect(http.StatusOK, "POST", "/auth/login", "", creds))

	e.expect(http.StatusUnauthorized, "GET", "/me", "", nil)
	me := parse[struct{ ID, Email string }](t, e.expect(http.StatusOK, "GET", "/me", login.AccessToken, nil))
	if me.ID != reg.User.ID {
		t.Fatal("me returned another user")
	}

	// Refresh rotates: the old token is dead, the new one works.
	ref := parse[authResp](t, e.expect(http.StatusOK, "POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}))
	e.expect(http.StatusUnauthorized, "POST", "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken})
	e.expect(http.StatusNoContent, "POST", "/auth/logout", "", map[string]string{"refreshToken": ref.RefreshToken})
	e.expect(http.StatusUnauthorized, "POST", "/auth/refresh", "", map[string]string{"refreshToken": ref.RefreshToken})
}

func TestPasswordRecoveryAndChange(t *testing.T) {
	e := newEnv(t)
	email := uniqueEmail()
	reg := parse[authResp](t, e.expect(http.StatusCreated, "POST", "/auth/register", "", map[string]string{"email": email, "password": "correct horse"}))

	e.expect(http.StatusAccepted, "POST", "/auth/forgot-password", "", map[string]string{"email": "nobody@test.invalid"})
	e.expect(http.StatusAccepted, "POST", "/auth/forgot-password", "", map[string]string{"email": email})
	token := e.mailer.token()
	if token == "" {
		t.Fatal("no reset token e-mailed")
	}
	e.expect(http.StatusBadRequest, "POST", "/auth/reset-password", "", map[string]string{"token": "bogus", "password": "brand new pass"})
	e.expect(http.StatusNoContent, "POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "brand new pass"})
	e.expect(http.StatusBadRequest, "POST", "/auth/reset-password", "", map[string]string{"token": token, "password": "another pass 1"}) // single use
	e.expect(http.StatusUnauthorized, "POST", "/auth/refresh", "", map[string]string{"refreshToken": reg.RefreshToken})                  // sessions revoked
	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": email, "password": "correct horse"})
	login := parse[authResp](t, e.expect(http.StatusOK, "POST", "/auth/login", "", map[string]string{"email": email, "password": "brand new pass"}))

	e.expect(http.StatusUnauthorized, "POST", "/me/password", login.AccessToken, map[string]string{"currentPassword": "nope nope nope", "newPassword": "third password"})
	e.expect(http.StatusNoContent, "POST", "/me/password", login.AccessToken, map[string]string{"currentPassword": "brand new pass", "newPassword": "third password"})
	e.expect(http.StatusOK, "POST", "/auth/login", "", map[string]string{"email": email, "password": "third password"})
}

func TestProfileValidationAndOwnership(t *testing.T) {
	e := newEnv(t)
	a := e.register()
	base := map[string]any{"firstName": "Sam", "birthDate": "1990-01-02", "gender": "man", "bio": "", "city": "", "interestIds": []int{}}
	with := func(k string, v any) map[string]any {
		m := map[string]any{}
		for kk, vv := range base {
			m[kk] = vv
		}
		m[k] = v
		return m
	}
	e.expect(http.StatusNotFound, "GET", "/me/profile", a.AccessToken, nil)
	e.expect(http.StatusNotFound, "PUT", "/me/preferences", a.AccessToken, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 40})
	under := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	if body := e.expect(http.StatusBadRequest, "PUT", "/me/profile", a.AccessToken, with("birthDate", under)); !strings.Contains(string(body), "underage") {
		t.Fatalf("expected underage code: %s", body)
	}
	e.expect(http.StatusBadRequest, "PUT", "/me/profile", a.AccessToken, with("gender", "alien"))
	e.expect(http.StatusBadRequest, "PUT", "/me/profile", a.AccessToken, with("interestIds", []int{99999}))
	e.expect(http.StatusOK, "PUT", "/me/profile", a.AccessToken, base)
	e.expect(http.StatusBadRequest, "PUT", "/me/profile", a.AccessToken, with("birthDate", "1985-01-02")) // immutable
	e.expect(http.StatusOK, "PUT", "/me/profile", a.AccessToken, with("bio", "updated"))
	e.expect(http.StatusBadRequest, "PUT", "/me/preferences", a.AccessToken, map[string]any{"interestedIn": []string{"woman"}, "minAge": 17, "maxAge": 40})
	e.expect(http.StatusOK, "PUT", "/me/privacy", a.AccessToken, map[string]bool{"discoverable": false, "showDistance": false})
	e.expect(http.StatusBadRequest, "PUT", "/me/location", a.AccessToken, map[string]float64{"latitude": 120, "longitude": 3})
	e.expect(http.StatusNoContent, "PUT", "/me/location", a.AccessToken, map[string]float64{"latitude": 48.85661, "longitude": 2.35222})
	prof := parse[struct {
		Bio         string `json:"bio"`
		HasLocation bool   `json:"hasLocation"`
	}](t, e.expect(http.StatusOK, "GET", "/me/profile", a.AccessToken, nil))
	if prof.Bio != "updated" || !prof.HasLocation {
		t.Fatalf("profile not persisted: %+v", prof)
	}
	var lat float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, a.User.ID).Scan(&lat); err != nil || lat != 48.86 {
		t.Fatalf("latitude must be stored rounded to 0.01, got %v (%v)", lat, err)
	}
	e.expect(http.StatusNoContent, "DELETE", "/me/location", a.AccessToken, nil)
	if body := e.expect(http.StatusOK, "GET", "/interests", a.AccessToken, nil); !strings.Contains(string(body), "Voyage") {
		t.Fatal("interest catalogue missing")
	}
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	a := e.register()
	e.expect(http.StatusOK, "PUT", "/me/profile", a.AccessToken, map[string]any{"firstName": "Ana", "birthDate": "1990-01-02", "gender": "woman", "bio": "", "city": "", "interestIds": []int{}})

	if code, _ := e.upload("POST", "/me/photos", a.AccessToken, []byte("<html>not an image</html>"), "x.png"); code != http.StatusBadRequest {
		t.Fatalf("non-image accepted: %d", code)
	}
	type photo struct {
		ID, URL   string
		Position  int
		IsPrimary bool
	}
	var ids []string
	for n := 0; n < 6; n++ {
		code, out := e.upload("POST", "/me/photos", a.AccessToken, pngBytes(300+n, 400), "p.png")
		if code != http.StatusCreated {
			t.Fatalf("upload %d: %d %s", n, code, out)
		}
		p := parse[photo](t, out)
		if p.Position != n || p.IsPrimary != (n == 0) {
			t.Fatalf("bad position: %+v", p)
		}
		ids = append(ids, p.ID)
	}
	if code, _ := e.upload("POST", "/me/photos", a.AccessToken, pngBytes(300, 300), "p.png"); code != http.StatusConflict {
		t.Fatalf("7th photo must be refused, got %d", code)
	}

	// Reorder: reverse. Duplicates / unknown ids are refused.
	rev := make([]string, len(ids))
	for k := range ids {
		rev[k] = ids[len(ids)-1-k]
	}
	e.expect(http.StatusBadRequest, "PUT", "/me/photos/order", a.AccessToken, map[string]any{"photoIds": ids[:5]})
	e.expect(http.StatusBadRequest, "PUT", "/me/photos/order", a.AccessToken, map[string]any{"photoIds": []string{ids[0], ids[0], ids[1], ids[2], ids[3], ids[4]}})
	ord := parse[struct{ Photos []photo }](t, e.expect(http.StatusOK, "PUT", "/me/photos/order", a.AccessToken, map[string]any{"photoIds": rev}))
	if ord.Photos[0].ID != ids[5] || !ord.Photos[0].IsPrimary {
		t.Fatalf("primary should be the first listed: %+v", ord.Photos)
	}

	// Signed content URL works without auth, a tampered one doesn't.
	url := ord.Photos[0].URL
	code, body := e.do("GET", url, "", nil)
	if code != http.StatusOK || len(body) < 100 || body[0] != 0xFF {
		t.Fatalf("photo content: %d (%d bytes)", code, len(body))
	}
	if code, _ := e.do("GET", url[:len(url)-2]+"00", "", nil); code != http.StatusNotFound {
		t.Fatalf("tampered signature must 404, got %d", code)
	}

	// Another user can't touch my photos.
	b := e.register()
	e.expect(http.StatusNotFound, "DELETE", "/me/photos/"+ids[0], b.AccessToken, nil)
	if code, _ := e.upload("PUT", "/me/photos/"+ids[0], b.AccessToken, pngBytes(300, 300), "p.png"); code != http.StatusNotFound {
		t.Fatalf("replace by non-owner: %d", code)
	}

	// Replace keeps the position, gives a new id; delete compacts positions.
	code, out := e.upload("PUT", "/me/photos/"+ids[5], a.AccessToken, pngBytes(500, 500), "n.png")
	if code != http.StatusOK {
		t.Fatalf("replace: %d %s", code, out)
	}
	rep := parse[photo](t, out)
	if rep.ID == ids[5] || rep.Position != 0 {
		t.Fatalf("replace result: %+v", rep)
	}
	e.expect(http.StatusNoContent, "DELETE", "/me/photos/"+rep.ID, a.AccessToken, nil)
	own := parse[struct{ Photos []photo }](t, e.expect(http.StatusOK, "GET", "/me/profile", a.AccessToken, nil))
	if len(own.Photos) != 5 || own.Photos[0].Position != 0 || own.Photos[4].Position != 4 {
		t.Fatalf("positions must be compacted: %+v", own.Photos)
	}
}

func TestDiscoveryRulesAndMatching(t *testing.T) {
	e := newEnv(t)
	paris := f(48.8566)
	parisLng := f(2.3522)
	versaillesLat, versaillesLng := f(48.8049), f(2.1204)
	lyonLat, lyonLng := f(45.76), f(4.84)

	me := e.newPerson(profileSpec{name: "Moi", gender: "man", interestedIn: []string{"woman"}, lat: paris, lng: parisLng, maxDistance: i(50)})
	near := e.newPerson(profileSpec{name: "Proche", gender: "woman", interestedIn: []string{"man"}, lat: versaillesLat, lng: versaillesLng, maxDistance: i(100)})
	far := e.newPerson(profileSpec{name: "Loin", gender: "woman", interestedIn: []string{"man"}, lat: lyonLat, lng: lyonLng})
	wrongInterest := e.newPerson(profileSpec{name: "Hetero", gender: "woman", interestedIn: []string{"woman"}, lat: paris, lng: parisLng})
	wrongGender := e.newPerson(profileSpec{name: "Homme", gender: "man", interestedIn: []string{"woman"}, lat: paris, lng: parisLng})
	hidden := e.newPerson(profileSpec{name: "Cachee", gender: "woman", interestedIn: []string{"man"}, lat: paris, lng: parisLng})
	e.expect(http.StatusOK, "PUT", "/me/privacy", hidden.token, map[string]bool{"discoverable": false, "showDistance": true})
	tooOld := e.newPerson(profileSpec{name: "Ancienne", gender: "woman", interestedIn: []string{"man"}, birth: "1950-01-01", lat: paris, lng: parisLng})
	e.expect(http.StatusOK, "PUT", "/me/preferences", tooOld.token, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 25})
	blockedMe := e.newPerson(profileSpec{name: "Bloquee", gender: "woman", interestedIn: []string{"man"}, lat: paris, lng: parisLng})
	e.expect(http.StatusNoContent, "POST", "/blocks", blockedMe.token, map[string]string{"userId": me.User.ID})
	// Registered but without a photo/profile: never shown.
	e.register()

	ids := e.discoverIDs(me)
	if !ids[near.User.ID] {
		t.Fatal("compatible nearby profile missing from discovery")
	}
	for name, p := range map[string]person{"far": far, "wrong interest": wrongInterest, "wrong gender": wrongGender, "hidden": hidden, "age mismatch": tooOld, "blocked": blockedMe, "self": me} {
		if ids[p.User.ID] {
			t.Errorf("%s must not be discoverable", name)
		}
	}
	prof := e.discover(me).Profiles
	for _, p := range prof {
		if p.UserID == near.User.ID {
			if p.DistanceKm == nil || *p.DistanceKm < 10 || *p.DistanceKm > 25 || p.Age < 30 {
				t.Errorf("approximate distance/age wrong: %+v", p)
			}
			if len(p.Photos) != 1 || !strings.HasPrefix(p.Photos[0].URL, "/photos/") {
				t.Errorf("photo url missing: %+v", p.Photos)
			}
		}
	}

	// Profile access: hidden users can't be fetched, blocked users can't either.
	e.expect(http.StatusNotFound, "GET", "/profiles/"+hidden.User.ID, me.token, nil)
	e.expect(http.StatusNotFound, "GET", "/profiles/"+blockedMe.User.ID, me.token, nil)
	e.expect(http.StatusOK, "GET", "/profiles/"+near.User.ID, me.token, nil)
	e.expect(http.StatusNotFound, "GET", "/profiles/not-a-uuid", me.token, nil)

	// Swiping rules.
	e.expect(http.StatusBadRequest, "POST", "/swipes", me.token, map[string]string{"userId": me.User.ID, "action": "like"})
	e.expect(http.StatusBadRequest, "POST", "/swipes", me.token, map[string]string{"userId": near.User.ID, "action": "super"})
	e.expect(http.StatusNotFound, "POST", "/swipes", me.token, map[string]string{"userId": hidden.User.ID, "action": "like"})
	e.expect(http.StatusNotFound, "POST", "/swipes", me.token, map[string]string{"userId": blockedMe.User.ID, "action": "like"})
	e.expect(http.StatusForbidden, "POST", "/swipes", e.register().AccessToken, map[string]string{"userId": near.User.ID, "action": "like"})

	// First like: no match yet, the other side sees it in "likes received" and gets a notification.
	first := e.swipe(me, near, "like")
	if first.Matched || first.Conversation != nil {
		t.Fatal("one-sided like must not match")
	}
	if again := e.swipe(me, near, "like"); again.Matched {
		t.Fatal("repeating a like must stay idempotent")
	}
	if e.discoverIDs(me)[near.User.ID] {
		t.Fatal("swiped profile must not reappear")
	}
	received := parse[profilesResp](t, e.expect(http.StatusOK, "GET", "/likes/received", near.token, nil))
	if len(received.Profiles) != 1 || received.Profiles[0].UserID != me.User.ID {
		t.Fatalf("likes received: %+v", received.Profiles)
	}
	nearDiscover := e.discover(near)
	if len(nearDiscover.Profiles) == 0 || nearDiscover.Profiles[0].UserID != me.User.ID || !nearDiscover.Profiles[0].LikedYou {
		t.Fatalf("who liked you should rank first in discovery: %+v", nearDiscover.Profiles)
	}
	notifs := parse[struct {
		Notifications []struct{ Type string } `json:"notifications"`
		UnreadCount   int                     `json:"unreadCount"`
	}](t, e.expect(http.StatusOK, "GET", "/notifications", near.token, nil))
	if notifs.UnreadCount != 1 || notifs.Notifications[0].Type != "new_like" {
		t.Fatalf("expected a like notification: %+v", notifs)
	}

	// Reciprocal like => match + conversation, for both.
	m := e.swipe(near, me, "like")
	if !m.Matched || m.Conversation == nil {
		t.Fatal("reciprocal like must create a match")
	}
	if again := e.swipe(near, me, "like"); !again.Matched || again.Conversation.ID != m.Conversation.ID {
		t.Fatal("repeated like on a match must return the same conversation")
	}
	var matches, convs int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, me.User.ID, near.User.ID).Scan(&matches)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a IN ($1,$2) AND m.user_b IN ($1,$2)`, me.User.ID, near.User.ID).Scan(&convs)
	if matches != 1 || convs != 1 {
		t.Fatalf("expected exactly one match/conversation, got %d/%d", matches, convs)
	}
	for _, p := range []person{me, near} {
		n := parse[struct {
			Notifications []struct{ Type string } `json:"notifications"`
		}](t, e.expect(http.StatusOK, "GET", "/notifications", p.token, nil))
		found := false
		for _, x := range n.Notifications {
			found = found || x.Type == "new_match"
		}
		if !found {
			t.Fatal("both users must receive a match notification")
		}
	}

	// A pass can later be upgraded to a like; a like can't be downgraded.
	// Preferences are enforced server-side on swipes too, not only when building the deck.
	for _, p := range []person{wrongGender, wrongInterest, far, tooOld} {
		e.expect(http.StatusNotFound, "POST", "/swipes", me.token, map[string]string{"userId": p.User.ID, "action": "like"})
	}
	if down := e.swipe(me, near, "pass"); down.Action != "like" {
		t.Fatalf("a like must not be downgraded, got %s", down.Action)
	}
}

func TestPassUpgradeAndNoReappear(t *testing.T) {
	e := newEnv(t)
	a := e.newPerson(profileSpec{name: "Ada", gender: "woman", interestedIn: []string{"man"}})
	b := e.newPerson(profileSpec{name: "Ben", gender: "man", interestedIn: []string{"woman"}})
	e.swipe(a, b, "pass")
	if e.discoverIDs(a)[b.User.ID] {
		t.Fatal("passed profile reappeared")
	}
	if up := e.swipe(a, b, "like"); up.Action != "like" {
		t.Fatal("pass should be upgradable to like")
	}
	if !e.swipe(b, a, "like").Matched {
		t.Fatal("expected match after upgrade")
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.newPerson(profileSpec{name: "Alix", gender: "woman", interestedIn: []string{"man"}})
		b := e.newPerson(profileSpec{name: "Bruno", gender: "man", interestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]swipeResp, 2)
		for k, pair := range [][2]person{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[k] = e.swipe(pair[0], pair[1], "like")
			}()
		}
		wg.Wait()
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("round %d: nobody saw the match", round)
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.User.ID, b.User.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: %d matches", round, n)
		}
	}
}

type convList struct {
	Conversations []struct {
		ID          string `json:"id"`
		UnreadCount int    `json:"unreadCount"`
		User        struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"user"`
		LastMessage *struct {
			Body string `json:"body"`
		} `json:"lastMessage"`
	} `json:"conversations"`
}

func (e *env) matchedPair() (person, person, string) {
	a := e.newPerson(profileSpec{name: "Alice", gender: "woman", interestedIn: []string{"man"}})
	b := e.newPerson(profileSpec{name: "Bob", gender: "man", interestedIn: []string{"woman"}})
	e.swipe(a, b, "like")
	m := e.swipe(b, a, "like")
	if !m.Matched {
		e.t.Fatal("pair did not match")
	}
	return a, b, m.Conversation.ID
}

func TestChatFlowAndPermissions(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchedPair()
	outsider := e.newPerson(profileSpec{name: "Eve", gender: "woman", interestedIn: []string{"man"}})

	// Permissions: non-participants can't read, write, mark read or hide.
	e.expect(http.StatusNotFound, "GET", "/conversations/"+conv+"/messages", outsider.token, nil)
	e.expect(http.StatusNotFound, "POST", "/conversations/"+conv+"/messages", outsider.token, map[string]string{"body": "hi"})
	e.expect(http.StatusNotFound, "POST", "/conversations/"+conv+"/read", outsider.token, nil)
	e.expect(http.StatusNotFound, "DELETE", "/conversations/"+conv, outsider.token, nil)
	e.expect(http.StatusUnauthorized, "GET", "/conversations/"+conv+"/messages", "", nil)

	// Validation.
	e.expect(http.StatusBadRequest, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": "   "})
	e.expect(http.StatusBadRequest, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": strings.Repeat("x", 2001)})

	// New match is listed even without messages.
	list := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", a.token, nil))
	if len(list.Conversations) != 1 || list.Conversations[0].LastMessage != nil || list.Conversations[0].User.FirstName != "Bob" {
		t.Fatalf("fresh match list: %+v", list)
	}

	for _, body := range []string{"Salut", "Ça va ?", "Dernier"} {
		e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": body})
	}
	bl := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", b.token, nil))
	if bl.Conversations[0].UnreadCount != 3 || bl.Conversations[0].LastMessage.Body != "Dernier" {
		t.Fatalf("recipient unread/lastMessage: %+v", bl.Conversations[0])
	}
	if al := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", a.token, nil)); al.Conversations[0].UnreadCount != 0 {
		t.Fatal("sender has no unread")
	}

	type msgs struct {
		Messages []struct {
			ID, Body string
			SenderID string `json:"senderId"`
		} `json:"messages"`
		NextCursor      *string    `json:"nextCursor"`
		OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	}
	page1 := parse[msgs](t, e.expect(http.StatusOK, "GET", "/conversations/"+conv+"/messages?limit=2", b.token, nil))
	if len(page1.Messages) != 2 || page1.Messages[0].Body != "Dernier" || page1.NextCursor == nil {
		t.Fatalf("page 1: %+v", page1)
	}
	page2 := parse[msgs](t, e.expect(http.StatusOK, "GET", "/conversations/"+conv+"/messages?limit=2&before="+*page1.NextCursor, b.token, nil))
	if len(page2.Messages) != 1 || page2.Messages[0].Body != "Salut" || page2.NextCursor != nil {
		t.Fatalf("page 2: %+v", page2)
	}

	// Read receipts.
	before := parse[msgs](t, e.expect(http.StatusOK, "GET", "/conversations/"+conv+"/messages", a.token, nil))
	e.expect(http.StatusNoContent, "POST", "/conversations/"+conv+"/read", b.token, nil)
	after := parse[msgs](t, e.expect(http.StatusOK, "GET", "/conversations/"+conv+"/messages", a.token, nil))
	if after.OtherLastReadAt == nil || (before.OtherLastReadAt != nil && !after.OtherLastReadAt.After(*before.OtherLastReadAt)) {
		t.Fatal("read receipt not updated")
	}
	if bl := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", b.token, nil)); bl.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread must reset after read")
	}

	// Local deletion hides the thread and its history for that user only, until a new message arrives.
	e.expect(http.StatusNoContent, "DELETE", "/conversations/"+conv, b.token, nil)
	if l := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", b.token, nil)); len(l.Conversations) != 0 {
		t.Fatal("hidden conversation still listed")
	}
	if l := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", a.token, nil)); len(l.Conversations) != 1 {
		t.Fatal("hiding must not affect the other user")
	}
	e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": "Re-coucou"})
	l := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", b.token, nil))
	if len(l.Conversations) != 1 || l.Conversations[0].LastMessage.Body != "Re-coucou" {
		t.Fatalf("conversation should reappear on new message: %+v", l)
	}
	if h := parse[msgs](t, e.expect(http.StatusOK, "GET", "/conversations/"+conv+"/messages", b.token, nil)); len(h.Messages) != 1 {
		t.Fatalf("history before deletion must stay hidden, got %d messages", len(h.Messages))
	}
}

func TestBlockUnmatchReportAndNotifications(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchedPair()
	e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": "hello"})

	// Reports.
	e.expect(http.StatusBadRequest, "POST", "/reports", a.token, map[string]string{"userId": b.User.ID, "reason": "boredom"})
	e.expect(http.StatusBadRequest, "POST", "/reports", a.token, map[string]string{"userId": a.User.ID, "reason": "spam"})
	e.expect(http.StatusNotFound, "POST", "/reports", a.token, map[string]string{"userId": "00000000-0000-4000-8000-000000000000", "reason": "spam"})
	e.expect(http.StatusCreated, "POST", "/reports", a.token, map[string]string{"userId": b.User.ID, "reason": "harassment", "details": "rude"})
	e.expect(http.StatusCreated, "POST", "/reports", a.token, map[string]string{"userId": b.User.ID, "reason": "harassment", "details": "again"})
	var reports int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1`, a.User.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("duplicate open report must be merged, got %d", reports)
	}

	// Notifications can be marked read.
	n := parse[struct {
		Notifications []struct{ ID string } `json:"notifications"`
		UnreadCount   int                   `json:"unreadCount"`
	}](t, e.expect(http.StatusOK, "GET", "/notifications", a.token, nil))
	if n.UnreadCount < 1 {
		t.Fatal("expected unread notification")
	}
	e.expect(http.StatusNoContent, "POST", "/notifications/"+n.Notifications[0].ID+"/read", a.token, nil)
	e.expect(http.StatusNotFound, "POST", "/notifications/"+n.Notifications[0].ID+"/read", b.token, nil) // not theirs
	e.expect(http.StatusNoContent, "POST", "/notifications/read-all", a.token, nil)
	if c := parse[struct {
		UnreadCount int `json:"unreadCount"`
	}](t, e.expect(http.StatusOK, "GET", "/notifications", a.token, nil)); c.UnreadCount != 0 {
		t.Fatal("read-all failed")
	}

	// Blocking dissolves the match and the conversation for both sides, and hides each user from the other.
	e.expect(http.StatusBadRequest, "POST", "/blocks", a.token, map[string]string{"userId": a.User.ID})
	e.expect(http.StatusNotFound, "POST", "/blocks", a.token, map[string]string{"userId": "00000000-0000-4000-8000-000000000000"})
	e.expect(http.StatusNoContent, "POST", "/blocks", a.token, map[string]string{"userId": b.User.ID})
	e.expect(http.StatusNoContent, "POST", "/blocks", a.token, map[string]string{"userId": b.User.ID}) // idempotent
	for _, p := range []person{a, b} {
		if l := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", p.token, nil)); len(l.Conversations) != 0 {
			t.Fatal("conversation must disappear after a block")
		}
	}
	e.expect(http.StatusNotFound, "GET", "/conversations/"+conv+"/messages", a.token, nil)
	e.expect(http.StatusNotFound, "GET", "/profiles/"+a.User.ID, b.token, nil)
	e.expect(http.StatusNotFound, "GET", "/profiles/"+b.User.ID, a.token, nil)
	bl := parse[struct {
		Blocks []struct{ UserID, FirstName string } `json:"blocks"`
	}](t, e.expect(http.StatusOK, "GET", "/blocks", a.token, nil))
	if len(bl.Blocks) != 1 || bl.Blocks[0].FirstName != "Bob" {
		t.Fatalf("block list: %+v", bl)
	}
	e.expect(http.StatusNoContent, "DELETE", "/blocks/"+b.User.ID, a.token, nil)
	e.expect(http.StatusOK, "GET", "/profiles/"+b.User.ID, a.token, nil)

	// Unmatch: a fresh pair dissolves the conversation through DELETE /matches/:id.
	c, d, conv2 := e.matchedPair()
	var matchID string
	_ = e.pool.QueryRow(context.Background(), `SELECT match_id FROM conversations WHERE id = $1`, conv2).Scan(&matchID)
	e.expect(http.StatusNotFound, "DELETE", "/matches/"+matchID, e.newPerson(profileSpec{name: "Zed", gender: "man", interestedIn: []string{"woman"}}).token, nil)
	e.expect(http.StatusNoContent, "DELETE", "/matches/"+matchID, c.token, nil)
	e.expect(http.StatusNotFound, "GET", "/conversations/"+conv2+"/messages", d.token, nil)
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchedPair()
	e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": "bye"})
	e.expect(http.StatusCreated, "POST", "/reports", b.token, map[string]string{"userId": a.User.ID, "reason": "spam"})

	e.expect(http.StatusUnauthorized, "DELETE", "/me", a.token, map[string]string{"password": "wrong password"})
	e.expect(http.StatusNoContent, "DELETE", "/me", a.token, map[string]string{"password": "correct horse"})
	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": a.User.Email, "password": "correct horse"})
	e.expect(http.StatusUnauthorized, "POST", "/auth/refresh", "", map[string]string{"refreshToken": a.RefreshToken})

	ctx := context.Background()
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "preferences": "user_id", "user_interests": "user_id",
		"swipes": "from_user_id", "messages": "sender_id", "notifications": "user_id", "sessions": "user_id"} {
		var n int
		if err := e.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = $1`, table, col), a.User.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s still holds %d rows of the deleted user (%v)", table, n, err)
		}
	}
	var matches int
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM matches WHERE user_a = $1 OR user_b = $1`, a.User.ID).Scan(&matches)
	if matches != 0 {
		t.Error("matches of the deleted user must be removed")
	}
	if l := parse[convList](t, e.expect(http.StatusOK, "GET", "/conversations", b.token, nil)); len(l.Conversations) != 0 {
		t.Fatal("the other user must no longer see the conversation")
	}
	e.expect(http.StatusNotFound, "GET", "/profiles/"+a.User.ID, b.token, nil)
	// Reports survive (anonymised) so moderation history is not erased by deleting an account.
	var kept int
	_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL`, b.User.ID).Scan(&kept)
	if kept != 1 {
		t.Fatalf("report should be kept with a null target, got %d", kept)
	}
}

func TestRateLimitedLogin(t *testing.T) {
	e := newEnv(t)
	gin.SetMode(gin.TestMode)
	router, err := setupRouter(&app{
		dbPool: e.pool, jwtSecret: []byte(testSecret), mailer: e.mailer,
		limits: limits{auth: authLimits(3), swipesPerMinute: 10, messagesPerMinute: 10, uploadsPerHour: 10, reportsPerHour: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	last := 0
	for n := 0; n < 5; n++ {
		raw, _ := json.Marshal(map[string]string{"email": "x@test.invalid", "password": "whatever pass"})
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		last = w.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after the budget is spent, got %d", last)
	}
}

func TestRealtimeDelivery(t *testing.T) {
	e := newEnv(t)
	a, b, conv := e.matchedPair()
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	// Tickets are required and single use.
	if _, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket=nope", nil); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatal("connection without a valid ticket must be refused")
	}
	ticket := parse[struct{ Ticket string }](t, e.expect(http.StatusOK, "POST", "/ws/ticket", b.token, nil)).Ticket
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?ticket=" + url.QueryEscape(ticket)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL, nil); err == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatal("ticket must be single use")
	}

	e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": "temps réel"})
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var frame struct {
		Type string `json:"type"`
		Data struct {
			Body           string `json:"body"`
			ConversationID string `json:"conversationId"`
		} `json:"data"`
	}
	if err := conn.ReadJSON(&frame); err != nil {
		t.Fatal(err)
	}
	if frame.Type != "message.new" || frame.Data.Body != "temps réel" || frame.Data.ConversationID != conv {
		t.Fatalf("unexpected frame: %+v", frame)
	}

	// An unrelated user never receives it.
	other := e.newPerson(profileSpec{name: "Oscar", gender: "man", interestedIn: []string{"woman"}})
	t2 := parse[struct{ Ticket string }](t, e.expect(http.StatusOK, "POST", "/ws/ticket", other.token, nil)).Ticket
	oc, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket="+url.QueryEscape(t2), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer oc.Close()
	e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.token, map[string]string{"body": "second"})
	_ = oc.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if err := oc.ReadJSON(&frame); err == nil {
		t.Fatal("outsider received a conversation event")
	}

	// Ping/pong.
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_ = conn.ReadJSON(&frame) // drain "second"
	_ = conn.WriteJSON(map[string]string{"type": "ping"})
	if err := conn.ReadJSON(&frame); err != nil || frame.Type != "pong" {
		t.Fatalf("expected pong, got %+v (%v)", frame, err)
	}
}

func TestSecurityHeadersAndUnknownRoutes(t *testing.T) {
	e := newEnv(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers missing")
	}
	e.expect(http.StatusNotFound, "GET", "/does-not-exist", "", nil)
}

func authLimitsHigh() authfeature.Limits {
	return authfeature.Limits{AuthPerMinute: 100000, ResetPerHour: 100000, SensitivePerHr: 100000}
}

func authLimits(perMinute int) authfeature.Limits {
	return authfeature.Limits{AuthPerMinute: perMinute, ResetPerHour: 100, SensitivePerHr: 100}
}
