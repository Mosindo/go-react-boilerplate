package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/db/dbtest"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

const testSecret = "0123456789abcdef0123456789abcdef0123"

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *mailer.Memory
	dir    string
}

type user struct {
	ID    string
	Token string
	Email string
}

func newEnv(t *testing.T, rateLimit bool) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pool := dbtest.New(t)
	dir := t.TempDir()
	files, err := storage.NewLocal(dir)
	if err != nil {
		t.Fatal(err)
	}
	mail := &mailer.Memory{}
	router := app.NewRouter(app.Options{
		Pool: pool, JWTSecret: []byte(testSecret), Storage: files, Mailer: mail, RateLimit: rateLimit,
	})
	return &env{t: t, router: router, pool: pool, mail: mail, dir: dir}
}

func (e *env) raw(method, path, token string, body []byte, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// call sends a JSON request and decodes the JSON response (if any).
func (e *env) call(method, path, token string, body any) (int, map[string]any) {
	e.t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	rec := e.raw(method, path, token, payload, "application/json")
	out := map[string]any{}
	if rec.Body.Len() > 0 && strings.Contains(rec.Header().Get("Content-Type"), "json") {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			e.t.Fatalf("%s %s: bad json %q: %v", method, path, rec.Body.String(), err)
		}
	}
	return rec.Code, out
}

func (e *env) expect(want int, method, path, token string, body any) map[string]any {
	e.t.Helper()
	status, out := e.call(method, path, token, body)
	if status != want {
		e.t.Fatalf("%s %s: want %d got %d (%v)", method, path, want, status, out)
	}
	return out
}

func (e *env) upload(token, path, method string, data []byte) (int, map[string]any) {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="file"; filename="photo.jpg"`)
	h.Set("Content-Type", "image/jpeg") // deliberately lies about the type for non-images
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	rec := e.raw(method, path, token, buf.Bytes(), w.FormDataContentType())
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func testImage(w, h int, asPNG bool) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if asPNG {
		_ = png.Encode(&buf, img)
	} else {
		_ = jpeg.Encode(&buf, img, nil)
	}
	return buf.Bytes()
}

func (e *env) register(email string) user {
	e.t.Helper()
	out := e.expect(http.StatusCreated, "POST", "/auth/register", "", map[string]string{"email": email, "password": "correct-horse-1"})
	u := out["user"].(map[string]any)
	return user{ID: u["id"].(string), Token: out["accessToken"].(string), Email: email}
}

type profileSpec struct {
	name    string
	gender  string
	seeking []string
	born    string
	lat     float64
	lon     float64
	ageMin  int
	ageMax  int
	maxKm   *int
}

func (e *env) profileBody(p profileSpec) map[string]any {
	if p.ageMin == 0 {
		p.ageMin = 18
	}
	if p.ageMax == 0 {
		p.ageMax = 99
	}
	return map[string]any{
		"firstName": p.name, "birthDate": p.born, "gender": p.gender, "bio": "Hello, I like long walks.",
		"city": "Lyon", "latitude": p.lat, "longitude": p.lon, "interests": []string{"travel", "music"},
		"preferences": map[string]any{"interestedIn": p.seeking, "ageMin": p.ageMin, "ageMax": p.ageMax, "maxDistanceKm": p.maxKm},
	}
}

// onboard registers a user, completes the profile and uploads one photo.
func (e *env) onboard(email string, p profileSpec) user {
	e.t.Helper()
	u := e.register(email)
	e.expect(http.StatusOK, "PUT", "/me/profile", u.Token, e.profileBody(p))
	status, out := e.upload(u.Token, "/me/photos", "POST", testImage(600, 800, false))
	if status != http.StatusCreated {
		e.t.Fatalf("upload photo: %d %v", status, out)
	}
	return u
}

func ptr(i int) *int { return &i }

func TestHealth(t *testing.T) {
	e := newEnv(t, false)
	e.expect(http.StatusOK, "GET", "/health", "", nil)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t, false)

	e.expect(http.StatusBadRequest, "POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "correct-horse-1"})
	e.expect(http.StatusBadRequest, "POST", "/auth/register", "", map[string]string{"email": "a@b.io", "password": "short"})
	e.expect(http.StatusBadRequest, "POST", "/auth/register", "", map[string]string{"email": "a@b.io", "password": strings.Repeat("x", 100)})

	reg := e.expect(http.StatusCreated, "POST", "/auth/register", "", map[string]string{"email": "  Alice@Example.IO ", "password": "correct-horse-1"})
	if reg["user"].(map[string]any)["email"] != "alice@example.io" {
		t.Fatalf("email must be trimmed and lower-cased: %v", reg["user"])
	}
	e.expect(http.StatusConflict, "POST", "/auth/register", "", map[string]string{"email": "alice@example.io", "password": "correct-horse-1"})

	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": "alice@example.io", "password": "wrong-password"})
	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": "nobody@example.io", "password": "correct-horse-1"})
	login := e.expect(http.StatusOK, "POST", "/auth/login", "", map[string]string{"email": "ALICE@example.io", "password": "correct-horse-1"})

	token := login["accessToken"].(string)
	me := e.expect(http.StatusOK, "GET", "/me", token, nil)
	if me["email"] != "alice@example.io" {
		t.Fatalf("unexpected me: %v", me)
	}
	if _, leaked := me["passwordHash"]; leaked {
		t.Fatal("password hash must never be serialised")
	}
	e.expect(http.StatusUnauthorized, "GET", "/me", "", nil)
	e.expect(http.StatusUnauthorized, "GET", "/me", "garbage", nil)

	// Refresh rotates the token: the old one stops working (replay protection).
	oldRefresh := login["refreshToken"].(string)
	next := e.expect(http.StatusOK, "POST", "/auth/refresh", "", map[string]string{"refreshToken": oldRefresh})
	if next["refreshToken"] == oldRefresh {
		t.Fatal("refresh token must rotate")
	}
	e.expect(http.StatusUnauthorized, "POST", "/auth/refresh", "", map[string]string{"refreshToken": oldRefresh})

	e.expect(http.StatusOK, "GET", "/me", next["accessToken"].(string), nil)
	e.expect(http.StatusNoContent, "POST", "/auth/logout", "", map[string]string{"refreshToken": next["refreshToken"].(string)})
	e.expect(http.StatusUnauthorized, "POST", "/auth/refresh", "", map[string]string{"refreshToken": next["refreshToken"].(string)})
	// Logging out revokes the session: its access token stops working at once.
	e.expect(http.StatusUnauthorized, "GET", "/me", next["accessToken"].(string), nil)
}

func TestPasswordRecoveryAndChange(t *testing.T) {
	e := newEnv(t, false)
	u := e.register("carol@example.io")

	// Unknown addresses get the same answer and no mail.
	e.expect(http.StatusAccepted, "POST", "/auth/password/forgot", "", map[string]string{"email": "ghost@example.io"})
	e.expect(http.StatusAccepted, "POST", "/auth/password/forgot", "", map[string]string{"email": "carol@example.io"})

	var code string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && code == "" {
		for _, m := range e.mail.Messages() {
			if m.To == "carol@example.io" {
				i := strings.Index(m.Body, "code is ")
				code = m.Body[i+len("code is ") : i+len("code is ")+8]
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if code == "" {
		t.Fatal("reset mail was not sent")
	}
	if n := len(e.mail.Messages()); n != 1 {
		t.Fatalf("exactly one mail expected, got %d", n)
	}

	e.expect(http.StatusBadRequest, "POST", "/auth/password/reset", "", map[string]string{"email": "carol@example.io", "code": "AAAAAAAA", "newPassword": "brand-new-pass-1"})
	e.expect(http.StatusBadRequest, "POST", "/auth/password/reset", "", map[string]string{"email": "carol@example.io", "code": code, "newPassword": "short"})
	e.expect(http.StatusNoContent, "POST", "/auth/password/reset", "", map[string]string{"email": "carol@example.io", "code": strings.ToLower(code), "newPassword": "brand-new-pass-1"})
	// A code works once.
	e.expect(http.StatusBadRequest, "POST", "/auth/password/reset", "", map[string]string{"email": "carol@example.io", "code": code, "newPassword": "another-pass-12"})

	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": "carol@example.io", "password": "correct-horse-1"})
	login := e.expect(http.StatusOK, "POST", "/auth/login", "", map[string]string{"email": "carol@example.io", "password": "brand-new-pass-1"})
	_ = u

	token := login["accessToken"].(string)
	e.expect(http.StatusForbidden, "POST", "/me/password", token, map[string]string{"currentPassword": "nope-nope-nope", "newPassword": "yet-another-1"})
	e.expect(http.StatusNoContent, "POST", "/me/password", token, map[string]string{"currentPassword": "brand-new-pass-1", "newPassword": "yet-another-1"})
	e.expect(http.StatusOK, "POST", "/auth/login", "", map[string]string{"email": "carol@example.io", "password": "yet-another-1"})
}

func TestResetCodeLockoutAfterTooManyAttempts(t *testing.T) {
	e := newEnv(t, false)
	e.register("dan@example.io")
	e.expect(http.StatusAccepted, "POST", "/auth/password/forgot", "", map[string]string{"email": "dan@example.io"})
	for i := 0; i < 6; i++ {
		e.expect(http.StatusBadRequest, "POST", "/auth/password/reset", "", map[string]string{"email": "dan@example.io", "code": "ZZZZZZZZ", "newPassword": "brand-new-pass-1"})
	}
	var attempts int
	if err := e.pool.QueryRow(t.Context(), `SELECT max(attempts) FROM password_resets`).Scan(&attempts); err != nil || attempts != 5 {
		t.Fatalf("attempts must stop at 5, got %d (%v)", attempts, err)
	}
}

func TestRateLimitOnAuthEndpoints(t *testing.T) {
	e := newEnv(t, true)
	limited := false
	for i := 0; i < 30; i++ {
		status, _ := e.call("POST", "/auth/login", "", map[string]string{"email": "x@example.io", "password": "whatever-pass"})
		if status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("login must be rate limited")
	}
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t, false)
	u := e.register("eve@example.io")

	e.expect(http.StatusNotFound, "GET", "/me/profile", u.Token, nil)

	base := profileSpec{name: "Eve", gender: "woman", seeking: []string{"man"}, born: "1995-04-12", lat: 45.7640, lon: 4.8357}
	underage := base
	underage.born = time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	e.expect(http.StatusForbidden, "PUT", "/me/profile", u.Token, e.profileBody(underage))

	badName := base
	badName.name = "<script>alert(1)</script>"
	e.expect(http.StatusUnprocessableEntity, "PUT", "/me/profile", u.Token, e.profileBody(badName))

	badDate := base
	badDate.born = "12/04/1995"
	e.expect(http.StatusUnprocessableEntity, "PUT", "/me/profile", u.Token, e.profileBody(badDate))

	body := e.profileBody(base)
	body["interests"] = []string{"travel", "not-an-interest"}
	e.expect(http.StatusUnprocessableEntity, "PUT", "/me/profile", u.Token, body)

	body = e.profileBody(base)
	body["preferences"] = map[string]any{"interestedIn": []string{"man"}, "ageMin": 40, "ageMax": 30}
	e.expect(http.StatusUnprocessableEntity, "PUT", "/me/profile", u.Token, body)

	body = e.profileBody(base)
	body["latitude"] = 123.0
	e.expect(http.StatusBadRequest, "PUT", "/me/profile", u.Token, body)

	got := e.expect(http.StatusOK, "PUT", "/me/profile", u.Token, e.profileBody(base))
	if got["latitude"].(float64) != 45.76 || got["longitude"].(float64) != 4.84 {
		t.Fatalf("coordinates must be rounded to 2 decimals, got %v,%v", got["latitude"], got["longitude"])
	}
	if got["age"].(float64) < 18 || len(got["interests"].([]any)) != 2 {
		t.Fatalf("unexpected profile: %v", got)
	}

	e.expect(http.StatusOK, "PATCH", "/me/settings", u.Token, map[string]any{"isVisible": false})
	e.expect(http.StatusBadRequest, "PATCH", "/me/settings", u.Token, map[string]any{})
	if got := e.expect(http.StatusOK, "GET", "/me/profile", u.Token, nil); got["isVisible"] != false {
		t.Fatalf("visibility not updated: %v", got)
	}
	cat := e.expect(http.StatusOK, "GET", "/interests", u.Token, nil)
	if len(cat["interests"].([]any)) < 10 {
		t.Fatal("interest catalog is empty")
	}
}

func TestPhotoLifecycleAndSafety(t *testing.T) {
	e := newEnv(t, false)
	u := e.register("fay@example.io")
	other := e.register("gus@example.io")
	e.expect(http.StatusOK, "PUT", "/me/profile", u.Token, e.profileBody(profileSpec{name: "Fay", gender: "woman", seeking: []string{"man"}, born: "1994-01-01", lat: 45.76, lon: 4.84}))

	// Dangerous or invalid uploads are refused even when the client lies about the type.
	if status, _ := e.upload(u.Token, "/me/photos", "POST", []byte("<?php system($_GET['c']); ?>")); status != http.StatusUnprocessableEntity {
		t.Fatalf("script upload must be rejected, got %d", status)
	}
	if status, _ := e.upload(u.Token, "/me/photos", "POST", append([]byte("GIF89a"), make([]byte, 600)...)); status != http.StatusUnprocessableEntity {
		t.Fatalf("gif must be rejected, got %d", status)
	}
	if status, _ := e.upload(u.Token, "/me/photos", "POST", testImage(50, 50, true)); status != http.StatusUnprocessableEntity {
		t.Fatalf("tiny image must be rejected, got %d", status)
	}
	if status, _ := e.upload(u.Token, "/me/photos", "POST", bytes.Repeat([]byte{0xff}, 9<<20)); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized upload must be rejected, got %d", status)
	}
	if status, _ := e.upload("", "/me/photos", "POST", testImage(400, 400, false)); status != http.StatusUnauthorized {
		t.Fatalf("anonymous upload must be rejected, got %d", status)
	}

	var ids []string
	for i := 0; i < 6; i++ {
		status, out := e.upload(u.Token, "/me/photos", "POST", testImage(900, 1200, i%2 == 0))
		if status != http.StatusCreated {
			t.Fatalf("upload %d: %d %v", i, status, out)
		}
		if int(out["position"].(float64)) != i {
			t.Fatalf("position %d expected, got %v", i, out["position"])
		}
		ids = append(ids, out["id"].(string))
	}
	if status, _ := e.upload(u.Token, "/me/photos", "POST", testImage(400, 400, false)); status != http.StatusConflict {
		t.Fatalf("7th photo must be refused, got %d", status)
	}

	// Stored files are re-encoded JPEGs bounded in size, served through signed URLs.
	list := e.expect(http.StatusOK, "GET", "/me/photos", u.Token, nil)["photos"].([]any)
	first := list[0].(map[string]any)
	if int(first["width"].(float64)) > 1280 || int(first["height"].(float64)) > 1280 {
		t.Fatalf("photo must be downscaled: %v", first)
	}
	url := first["url"].(string)
	rec := e.raw("GET", url, "", nil, "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("signed url must serve a jpeg: %d %v", rec.Code, rec.Header())
	}
	if _, err := jpeg.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Fatalf("served file is not a jpeg: %v", err)
	}
	if rec := e.raw("GET", strings.Replace(url, "sig=", "sig=x", 1), "", nil, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("tampered signature must 404, got %d", rec.Code)
	}
	if rec := e.raw("GET", strings.Split(url, "?")[0], "", nil, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unsigned url must 404, got %d", rec.Code)
	}
	if rec := e.raw("GET", strings.Replace(url, "/full", "/original", 1), "", nil, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown variant must 404, got %d", rec.Code)
	}

	// Ownership: another user can neither delete, replace nor reorder these photos.
	e.expect(http.StatusNotFound, "DELETE", "/me/photos/"+ids[0], other.Token, nil)
	if status, _ := e.upload(other.Token, "/me/photos/"+ids[0], "PUT", testImage(400, 400, false)); status != http.StatusNotFound {
		t.Fatalf("foreign replace must 404, got %d", status)
	}
	e.expect(http.StatusBadRequest, "PUT", "/me/photos/order", other.Token, map[string]any{"photoIds": ids})

	// Reorder: last becomes primary.
	order := append([]string{ids[5]}, ids[:5]...)
	re := e.expect(http.StatusOK, "PUT", "/me/photos/order", u.Token, map[string]any{"photoIds": order})["photos"].([]any)
	if re[0].(map[string]any)["id"] != ids[5] {
		t.Fatalf("primary photo not updated: %v", re[0])
	}
	e.expect(http.StatusBadRequest, "PUT", "/me/photos/order", u.Token, map[string]any{"photoIds": order[:3]})
	dup := append([]string{ids[0]}, order[1:]...)
	dup[1] = ids[0]
	e.expect(http.StatusBadRequest, "PUT", "/me/photos/order", u.Token, map[string]any{"photoIds": dup})

	// Replace keeps position and id; delete compacts positions and removes files.
	status, replaced := e.upload(u.Token, "/me/photos/"+ids[5], "PUT", testImage(500, 500, true))
	if status != http.StatusOK || replaced["id"] != ids[5] {
		t.Fatalf("replace failed: %d %v", status, replaced)
	}
	if rec := e.raw("GET", url, "", nil, ""); rec.Code != http.StatusOK {
		t.Fatalf("an untouched photo keeps serving: %d", rec.Code)
	}
	filesBefore := countFiles(t, e.dir)
	e.expect(http.StatusNoContent, "DELETE", "/me/photos/"+ids[5], u.Token, nil)
	after := e.expect(http.StatusOK, "GET", "/me/photos", u.Token, nil)["photos"].([]any)
	if len(after) != 5 {
		t.Fatalf("5 photos expected, got %d", len(after))
	}
	for i, p := range after {
		if int(p.(map[string]any)["position"].(float64)) != i {
			t.Fatalf("positions must be compact: %v", after)
		}
	}
	if got := countFiles(t, e.dir); got != filesBefore-2 {
		t.Fatalf("deleting a photo must remove its two files: %d -> %d", filesBefore, got)
	}
	if rec := e.raw("GET", replaced["url"].(string), "", nil, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("deleted photo url should 404, got %d", rec.Code)
	}
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func discoverIDs(t *testing.T, e *env, u user) (ids []string, raw []any) {
	t.Helper()
	out := e.expect(http.StatusOK, "GET", "/discover?limit=20", u.Token, nil)
	for _, p := range out["profiles"].([]any) {
		raw = append(raw, p)
		ids = append(ids, p.(map[string]any)["userId"].(string))
	}
	return
}

func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func TestDiscoveryRulesAndMatching(t *testing.T) {
	e := newEnv(t, false)
	lyon := profileSpec{lat: 45.76, lon: 4.84}

	alice := e.onboard("alice@example.io", profileSpec{name: "Alice", gender: "woman", seeking: []string{"man"}, born: "1994-05-05", lat: lyon.lat, lon: lyon.lon, maxKm: ptr(50)})
	bob := e.onboard("bob@example.io", profileSpec{name: "Bob", gender: "man", seeking: []string{"woman"}, born: "1992-03-03", lat: 45.80, lon: 4.90})
	carl := e.onboard("carl@example.io", profileSpec{name: "Carl", gender: "man", seeking: []string{"man"}, born: "1992-03-03", lat: lyon.lat, lon: lyon.lon})   // not interested in women
	dave := e.onboard("dave@example.io", profileSpec{name: "Dave", gender: "man", seeking: []string{"woman"}, born: "1992-03-03", lat: 48.85, lon: 2.35})        // Paris: beyond Alice's 50 km
	erik := e.onboard("erik@example.io", profileSpec{name: "Erik", gender: "man", seeking: []string{"woman"}, born: "1970-03-03", lat: lyon.lat, lon: lyon.lon}) // 54: Alice's range is 18-99 but Erik wants 18-40
	_ = erik
	e.expect(http.StatusOK, "PUT", "/me/profile", e.register("tmp@example.io").Token, e.profileBody(profileSpec{name: "Tmp", gender: "man", seeking: []string{"woman"}, born: "1992-03-03", lat: lyon.lat, lon: lyon.lon})) // no photo
	hidden := e.onboard("hidden@example.io", profileSpec{name: "Hid", gender: "man", seeking: []string{"woman"}, born: "1992-03-03", lat: lyon.lat, lon: lyon.lon})
	e.expect(http.StatusOK, "PATCH", "/me/settings", hidden.Token, map[string]any{"isVisible": false})

	// Erik only wants people up to 40 years old: re-save his preferences.
	body := e.profileBody(profileSpec{name: "Erik", gender: "man", seeking: []string{"woman"}, born: "1970-03-03", lat: lyon.lat, lon: lyon.lon, ageMin: 18, ageMax: 40})
	e.expect(http.StatusOK, "PUT", "/me/profile", erik.Token, body)

	ids, raw := discoverIDs(t, e, alice)
	if len(ids) != 2 || !contains(ids, bob.ID) || !contains(ids, erik.ID) {
		// Erik is 56 and Alice's range is 18-99 but Erik wants <=40 while Alice is 31: he IS eligible.
		t.Fatalf("alice should see bob and erik only, got %v", ids)
	}
	for _, id := range []string{carl.ID, dave.ID, hidden.ID, alice.ID} {
		if contains(ids, id) {
			t.Fatalf("user %s must not be discoverable by alice", id)
		}
	}
	// No exact location or personal identifiers leak.
	for _, p := range raw {
		blob, _ := json.Marshal(p)
		for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "isVisible"} {
			if strings.Contains(string(blob), forbidden) {
				t.Fatalf("public profile leaks %q: %s", forbidden, blob)
			}
		}
		if d := p.(map[string]any)["distanceKm"]; d == nil {
			t.Fatal("distance should be shown")
		}
	}

	// A profile that is not discoverable cannot be fetched or swiped on.
	e.expect(http.StatusNotFound, "GET", "/profiles/"+dave.ID, alice.Token, nil)
	e.expect(http.StatusNotFound, "POST", "/swipes", alice.Token, map[string]any{"userId": dave.ID, "action": "like"})
	e.expect(http.StatusNotFound, "POST", "/swipes", alice.Token, map[string]any{"userId": alice.ID, "action": "like"})
	e.expect(http.StatusBadRequest, "POST", "/swipes", alice.Token, map[string]any{"userId": bob.ID, "action": "superlike"})
	e.expect(http.StatusBadRequest, "POST", "/swipes", alice.Token, map[string]any{"userId": "nope", "action": "like"})

	// Erik has not liked Alice yet: a like is not a match, and bob/erik reciprocity differs.
	e.expect(http.StatusOK, "GET", "/profiles/"+bob.ID, alice.Token, nil)
	res := e.expect(http.StatusOK, "POST", "/swipes", alice.Token, map[string]any{"userId": bob.ID, "action": "like"})
	if res["matched"] != false {
		t.Fatal("a one-sided like must not match")
	}
	e.expect(http.StatusConflict, "POST", "/swipes", alice.Token, map[string]any{"userId": bob.ID, "action": "like"})
	e.expect(http.StatusConflict, "POST", "/swipes", alice.Token, map[string]any{"userId": bob.ID, "action": "pass"})
	e.expect(http.StatusOK, "POST", "/swipes", alice.Token, map[string]any{"userId": erik.ID, "action": "pass"})
	if ids, _ := discoverIDs(t, e, alice); len(ids) != 0 {
		t.Fatalf("handled profiles must not reappear, got %v", ids)
	}

	// Bob sees Alice, likes her back: match with a conversation, notifications for both.
	if ids, _ := discoverIDs(t, e, bob); !contains(ids, alice.ID) {
		t.Fatal("bob should see alice")
	}
	res = e.expect(http.StatusOK, "POST", "/swipes", bob.Token, map[string]any{"userId": alice.ID, "action": "like"})
	if res["matched"] != true {
		t.Fatalf("reciprocal like must match: %v", res)
	}
	match := res["match"].(map[string]any)
	if match["conversationId"] == "" || match["user"].(map[string]any)["firstName"] != "Alice" {
		t.Fatalf("unexpected match payload: %v", match)
	}
	var matches int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&matches)
	if matches != 1 {
		t.Fatalf("exactly one match expected, got %d", matches)
	}
	for _, u := range []user{alice, bob} {
		n := e.expect(http.StatusOK, "GET", "/notifications", u.Token, nil)
		if n["unreadCount"].(float64) != 1 || n["notifications"].([]any)[0].(map[string]any)["type"] != "match" {
			t.Fatalf("match notification expected for %s: %v", u.Email, n)
		}
	}
	// Matched people stay viewable even though they left discovery.
	e.expect(http.StatusOK, "GET", "/profiles/"+alice.ID, bob.Token, nil)

	// Passes can be revisited on request.
	reset := e.expect(http.StatusOK, "DELETE", "/swipes/passes", alice.Token, nil)
	if reset["reset"].(float64) != 1 {
		t.Fatalf("one pass should reset: %v", reset)
	}
	if ids, _ := discoverIDs(t, e, alice); len(ids) != 1 || ids[0] != erik.ID {
		t.Fatalf("erik should be back, got %v", ids)
	}

	// A user without photos cannot use discovery.
	noPhoto := e.register("nophoto@example.io")
	e.expect(http.StatusOK, "PUT", "/me/profile", noPhoto.Token, e.profileBody(profileSpec{name: "No", gender: "man", seeking: []string{"woman"}, born: "1990-01-01", lat: lyon.lat, lon: lyon.lon}))
	if out := e.expect(http.StatusConflict, "GET", "/discover", noPhoto.Token, nil); out["code"] != "profile_incomplete" {
		t.Fatalf("expected profile_incomplete: %v", out)
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t, false)
	const pairs = 8
	type pair struct{ a, b user }
	var ps []pair
	for i := 0; i < pairs; i++ {
		a := e.onboard(fmt.Sprintf("a%d@example.io", i), profileSpec{name: "Ana", gender: "woman", seeking: []string{"man"}, born: "1994-05-05", lat: 45.76, lon: 4.84})
		b := e.onboard(fmt.Sprintf("b%d@example.io", i), profileSpec{name: "Ben", gender: "man", seeking: []string{"woman"}, born: "1994-05-05", lat: 45.76, lon: 4.84})
		ps = append(ps, pair{a, b})
	}
	var wg sync.WaitGroup
	statuses := make(chan int, pairs*2)
	for _, p := range ps {
		for _, side := range [][2]user{{p.a, p.b}, {p.b, p.a}} {
			wg.Add(1)
			go func(from, to user) {
				defer wg.Done()
				code, _ := e.call("POST", "/swipes", from.Token, map[string]any{"userId": to.ID, "action": "like"})
				statuses <- code
			}(side[0], side[1])
		}
	}
	wg.Wait()
	close(statuses)
	for code := range statuses {
		if code != http.StatusOK {
			t.Fatalf("concurrent like failed with %d", code)
		}
	}
	var matches, conversations int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&matches)
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM conversations`).Scan(&conversations)
	if matches != pairs || conversations != pairs {
		t.Fatalf("want %d matches/conversations, got %d/%d", pairs, matches, conversations)
	}
}

// matchedPair returns two users who matched, plus their conversation id and an outsider.
func matchedPair(t *testing.T, e *env) (a, b, outsider user, conversationID string) {
	t.Helper()
	a = e.onboard("ann@example.io", profileSpec{name: "Ann", gender: "woman", seeking: []string{"man"}, born: "1994-05-05", lat: 45.76, lon: 4.84})
	b = e.onboard("ben@example.io", profileSpec{name: "Ben", gender: "man", seeking: []string{"woman"}, born: "1993-05-05", lat: 45.77, lon: 4.85})
	outsider = e.onboard("mallory@example.io", profileSpec{name: "Mallory", gender: "man", seeking: []string{"woman"}, born: "1993-05-05", lat: 45.77, lon: 4.85})
	e.expect(http.StatusOK, "POST", "/swipes", a.Token, map[string]any{"userId": b.ID, "action": "like"})
	res := e.expect(http.StatusOK, "POST", "/swipes", b.Token, map[string]any{"userId": a.ID, "action": "like"})
	conversationID = res["match"].(map[string]any)["conversationId"].(string)
	return
}

func TestChatPermissionsReadStateAndHiding(t *testing.T) {
	e := newEnv(t, false)
	a, b, mallory, conv := matchedPair(t, e)
	msgs := "/conversations/" + conv + "/messages"

	// Outsiders learn nothing: no read, no write, no read receipt, no hide.
	e.expect(http.StatusNotFound, "GET", msgs, mallory.Token, nil)
	e.expect(http.StatusNotFound, "POST", msgs, mallory.Token, map[string]string{"content": "hi"})
	e.expect(http.StatusNotFound, "POST", "/conversations/"+conv+"/read", mallory.Token, nil)
	e.expect(http.StatusNotFound, "DELETE", "/conversations/"+conv, mallory.Token, nil)
	e.expect(http.StatusUnauthorized, "GET", msgs, "", nil)
	e.expect(http.StatusNotFound, "GET", "/conversations/not-a-uuid/messages", a.Token, nil)

	e.expect(http.StatusUnprocessableEntity, "POST", msgs, a.Token, map[string]string{"content": "   "})
	e.expect(http.StatusUnprocessableEntity, "POST", msgs, a.Token, map[string]string{"content": strings.Repeat("é", 2001)})
	e.expect(http.StatusBadRequest, "POST", msgs, a.Token, map[string]string{})

	sent := e.expect(http.StatusCreated, "POST", msgs, a.Token, map[string]string{"content": "  Salut Ben !  "})
	if sent["content"] != "Salut Ben !" || sent["senderId"] != a.ID {
		t.Fatalf("unexpected message: %v", sent)
	}
	e.expect(http.StatusCreated, "POST", msgs, a.Token, map[string]string{"content": "Tu es libre ce soir ?"})

	list := e.expect(http.StatusOK, "GET", "/conversations", b.Token, nil)
	row := list["conversations"].([]any)[0].(map[string]any)
	if row["unreadCount"].(float64) != 2 || list["unreadTotal"].(float64) != 2 || row["lastMessage"].(map[string]any)["content"] != "Tu es libre ce soir ?" {
		t.Fatalf("unexpected conversation row: %v", list)
	}
	if row["user"].(map[string]any)["thumbUrl"] == nil || row["user"].(map[string]any)["firstName"] != "Ann" {
		t.Fatalf("conversation should show the other person: %v", row["user"])
	}
	// Sender has nothing unread; a single coalesced message notification for the recipient.
	if e.expect(http.StatusOK, "GET", "/conversations", a.Token, nil)["unreadTotal"].(float64) != 0 {
		t.Fatal("sender must not have unread messages")
	}
	notifs := e.expect(http.StatusOK, "GET", "/notifications", b.Token, nil)
	var messageNotifs int
	for _, n := range notifs["notifications"].([]any) {
		if n.(map[string]any)["type"] == "message" {
			messageNotifs++
		}
	}
	if messageNotifs != 1 {
		t.Fatalf("message notifications must be coalesced, got %d", messageNotifs)
	}

	// Newest first; read receipts visible to the sender.
	got := e.expect(http.StatusOK, "GET", msgs+"?limit=1", b.Token, nil)
	if len(got["messages"].([]any)) != 1 || got["nextCursor"] == nil {
		t.Fatalf("pagination expected: %v", got)
	}
	older := e.expect(http.StatusOK, "GET", msgs+"?limit=1&cursor="+got["nextCursor"].(string), b.Token, nil)
	if older["messages"].([]any)[0].(map[string]any)["content"] != "Salut Ben !" {
		t.Fatalf("older page wrong: %v", older)
	}
	e.expect(http.StatusBadRequest, "GET", msgs+"?cursor=garbage", b.Token, nil)

	before := e.expect(http.StatusOK, "GET", msgs, a.Token, nil)
	e.expect(http.StatusOK, "POST", "/conversations/"+conv+"/read", b.Token, nil)
	after := e.expect(http.StatusOK, "GET", msgs, a.Token, nil)
	if before["otherLastReadAt"] == after["otherLastReadAt"] {
		t.Fatal("read receipt should advance")
	}
	if e.expect(http.StatusOK, "GET", "/conversations", b.Token, nil)["unreadTotal"].(float64) != 0 {
		t.Fatal("unread should be cleared")
	}
	if e.expect(http.StatusOK, "GET", "/notifications", b.Token, nil)["unreadCount"].(float64) != 1 { // only the match one remains
		t.Fatal("message notification should be read with the conversation")
	}

	// Hiding is local: gone for b, intact for a, and reappears (without history) on a new message.
	e.expect(http.StatusNoContent, "DELETE", "/conversations/"+conv, b.Token, nil)
	if n := len(e.expect(http.StatusOK, "GET", "/conversations", b.Token, nil)["conversations"].([]any)); n != 0 {
		t.Fatalf("hidden conversation must disappear, got %d", n)
	}
	if n := len(e.expect(http.StatusOK, "GET", "/conversations", a.Token, nil)["conversations"].([]any)); n != 1 {
		t.Fatalf("other side must be unaffected, got %d", n)
	}
	time.Sleep(10 * time.Millisecond)
	e.expect(http.StatusCreated, "POST", msgs, a.Token, map[string]string{"content": "Coucou ?"})
	back := e.expect(http.StatusOK, "GET", msgs, b.Token, nil)["messages"].([]any)
	if len(back) != 1 || back[0].(map[string]any)["content"] != "Coucou ?" {
		t.Fatalf("only post-hide messages should show: %v", back)
	}

	// Unmatching removes the conversation for both.
	var matchID string
	_ = e.pool.QueryRow(t.Context(), `SELECT match_id FROM conversations WHERE id = $1`, conv).Scan(&matchID)
	e.expect(http.StatusNotFound, "DELETE", "/matches/"+matchID, mallory.Token, nil)
	e.expect(http.StatusNoContent, "DELETE", "/matches/"+matchID, a.Token, nil)
	e.expect(http.StatusNotFound, "GET", msgs, b.Token, nil)
}

func TestBlockReportAndAccountDeletion(t *testing.T) {
	e := newEnv(t, false)
	a, b, mallory, conv := matchedPair(t, e)
	msgs := "/conversations/" + conv + "/messages"
	e.expect(http.StatusCreated, "POST", msgs, a.Token, map[string]string{"content": "hello"})

	// Reports are validated; reporting yourself is refused; duplicates collapse.
	e.expect(http.StatusBadRequest, "POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "because"})
	e.expect(http.StatusBadRequest, "POST", "/reports", a.Token, map[string]any{"userId": a.ID, "reason": "spam"})
	e.expect(http.StatusNotFound, "POST", "/reports", a.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"})
	r1 := e.expect(http.StatusCreated, "POST", "/reports", a.Token, map[string]any{"userId": mallory.ID, "reason": "harassment", "details": "rude"})
	r2 := e.expect(http.StatusCreated, "POST", "/reports", a.Token, map[string]any{"userId": mallory.ID, "reason": "harassment", "details": "rude again"})
	if r1["id"] != r2["id"] {
		t.Fatal("duplicate open reports from one reporter must collapse")
	}

	// Blocking: conversation vanishes for both, discovery and profile access are cut both ways.
	e.expect(http.StatusBadRequest, "POST", "/blocks", b.Token, map[string]any{"userId": b.ID})
	e.expect(http.StatusNotFound, "POST", "/blocks", b.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000"})
	e.expect(http.StatusNoContent, "POST", "/blocks", b.Token, map[string]any{"userId": a.ID})
	e.expect(http.StatusNotFound, "GET", msgs, a.Token, nil)
	e.expect(http.StatusNotFound, "GET", msgs, b.Token, nil)
	e.expect(http.StatusNotFound, "POST", msgs, a.Token, map[string]string{"content": "still there?"})
	for _, u := range []user{a, b} {
		if n := len(e.expect(http.StatusOK, "GET", "/conversations", u.Token, nil)["conversations"].([]any)); n != 0 {
			t.Fatalf("conversation must disappear for %s, got %d", u.Email, n)
		}
	}
	e.expect(http.StatusNotFound, "GET", "/profiles/"+a.ID, b.Token, nil)
	e.expect(http.StatusNotFound, "GET", "/profiles/"+b.ID, a.Token, nil)
	if ids, _ := discoverIDs(t, e, a); contains(ids, b.ID) {
		t.Fatal("blocked user must not be discoverable")
	}
	if ids, _ := discoverIDs(t, e, b); contains(ids, a.ID) {
		t.Fatal("blocker must not be discoverable by the blocked user")
	}
	e.expect(http.StatusNotFound, "POST", "/swipes", a.Token, map[string]any{"userId": b.ID, "action": "like"})
	var matches int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM matches`).Scan(&matches)
	if matches != 0 {
		t.Fatal("block must end the match")
	}

	blocks := e.expect(http.StatusOK, "GET", "/blocks", b.Token, nil)["blocks"].([]any)
	if len(blocks) != 1 || blocks[0].(map[string]any)["userId"] != a.ID {
		t.Fatalf("unexpected blocks: %v", blocks)
	}
	e.expect(http.StatusNoContent, "DELETE", "/blocks/"+a.ID, b.Token, nil)
	e.expect(http.StatusNoContent, "DELETE", "/blocks/"+a.ID, b.Token, nil) // idempotent
	e.expect(http.StatusOK, "GET", "/profiles/"+a.ID, b.Token, nil)

	// Report + block in one call.
	e.expect(http.StatusCreated, "POST", "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "fake", "block": true})
	e.expect(http.StatusNotFound, "GET", "/profiles/"+b.ID, a.Token, nil)

	// Account deletion needs the password and removes every trace, files included.
	e.expect(http.StatusForbidden, "DELETE", "/me", a.Token, map[string]string{"password": "wrong-password"})
	e.expect(http.StatusBadRequest, "DELETE", "/me", a.Token, nil)
	if countFiles(t, e.dir) == 0 {
		t.Fatal("fixture should have photo files")
	}
	filesBefore := countFiles(t, e.dir)
	e.expect(http.StatusNoContent, "DELETE", "/me", a.Token, map[string]string{"password": "correct-horse-1"})
	if got := countFiles(t, e.dir); got != filesBefore-2 {
		t.Fatalf("photo files of the deleted account must be removed: %d -> %d", filesBefore, got)
	}
	e.expect(http.StatusUnauthorized, "POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "correct-horse-1"})
	e.expect(http.StatusUnauthorized, "GET", "/me", a.Token, nil)
	for _, table := range []string{"profiles", "preferences", "photos", "user_interests", "swipes", "blocks", "reports", "notifications", "sessions", "conversation_participants"} {
		col := "user_id"
		switch table {
		case "swipes":
			col = "swiper_id"
		case "blocks":
			col = "blocker_id"
		case "reports":
			col = "reporter_id"
		}
		var n int
		if err := e.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s still holds %d rows for the deleted user (%v)", table, n, err)
		}
	}
	var sent int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM messages WHERE sender_id = $1`, a.ID).Scan(&sent)
	if sent != 0 {
		t.Fatal("messages of the deleted user must be removed")
	}
	var ops int
	_ = e.pool.QueryRow(t.Context(), `SELECT count(*) FROM users WHERE id = $1`, b.ID).Scan(&ops)
	if ops != 1 {
		t.Fatal("other users must be untouched")
	}
}

func TestRealtimeDeliversMessagesAndMatches(t *testing.T) {
	e := newEnv(t, false)
	srv := httptest.NewServer(e.router)
	defer srv.Close()

	a := e.onboard("wa@example.io", profileSpec{name: "Wa", gender: "woman", seeking: []string{"man"}, born: "1994-05-05", lat: 45.76, lon: 4.84})
	b := e.onboard("wb@example.io", profileSpec{name: "Wb", gender: "man", seeking: []string{"woman"}, born: "1993-05-05", lat: 45.77, lon: 4.85})

	connect := func(u user) *websocket.Conn {
		t.Helper()
		ticket := e.expect(http.StatusOK, "POST", "/realtime/ticket", u.Token, nil)["ticket"].(string)
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/realtime/ws?ticket="+ticket, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		var hello map[string]any
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err := conn.ReadJSON(&hello); err != nil || hello["type"] != "ready" {
			t.Fatalf("expected ready, got %v (%v)", hello, err)
		}
		return conn
	}
	next := func(conn *websocket.Conn, typ string) map[string]any {
		t.Helper()
		for {
			var ev map[string]any
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			if err := conn.ReadJSON(&ev); err != nil {
				t.Fatalf("waiting for %s: %v", typ, err)
			}
			if ev["type"] == typ {
				return ev
			}
		}
	}

	// Access tokens are not valid tickets, and tickets are not valid access tokens.
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/realtime/ws?ticket="+a.Token, nil); err == nil {
		t.Fatal("an access token must not open a socket")
	}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/realtime/ws", nil); err == nil {
		t.Fatal("a socket without ticket must be refused")
	}
	ticket := e.expect(http.StatusOK, "POST", "/realtime/ticket", a.Token, nil)["ticket"].(string)
	e.expect(http.StatusUnauthorized, "GET", "/me", ticket, nil)

	ca, cb := connect(a), connect(b)
	defer ca.Close()
	defer cb.Close()

	e.expect(http.StatusOK, "POST", "/swipes", a.Token, map[string]any{"userId": b.ID, "action": "like"})
	res := e.expect(http.StatusOK, "POST", "/swipes", b.Token, map[string]any{"userId": a.ID, "action": "like"})
	conv := res["match"].(map[string]any)["conversationId"].(string)
	// b swiped last and knows from the HTTP response; a, who liked first, is told live.
	if ev := next(ca, "match.new"); ev["data"].(map[string]any)["firstName"] != "Wb" {
		t.Fatalf("a should learn about the match: %v", ev)
	}

	e.expect(http.StatusCreated, "POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"content": "live!"})
	ev := next(cb, "message.new")
	if ev["data"].(map[string]any)["content"] != "live!" {
		t.Fatalf("b should receive the message in real time: %v", ev)
	}
	next(cb, "notification.new")
	next(ca, "message.new") // the sender's other devices stay in sync

	e.expect(http.StatusOK, "POST", "/conversations/"+conv+"/read", b.Token, nil)
	next(ca, "message.read")
}
