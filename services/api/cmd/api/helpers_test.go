package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var emailCounter atomic.Int64

type env struct {
	t      *testing.T
	router *gin.Engine
	pool   *pgxpool.Pool
	mail   *mailer.Memory
	store  *storage.Local
	emails []string
}

type testUser struct {
	ID      string
	Email   string
	Token   string
	Refresh string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skip integration tests: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		t.Skipf("skip integration tests: postgres unavailable (%v)", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mail := &mailer.Memory{}
	limits := Limits{Public: [2]int{6000, 6000}, Swipes: [2]int{6000, 6000}, Messages: [2]int{6000, 6000}, Uploads: [2]int{6000, 6000}, Reports: [2]int{6000, 6000}}
	router, err := NewRouter(Deps{DB: pool, JWTSecret: []byte("integration-test-secret-0123456789"), Storage: store, Mailer: mail, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, router: router, pool: pool, mail: mail, store: store}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email = ANY($1)`, e.emails)
		pool.Close()
	})
	return e
}

func (e *env) do(method, path string, payload any, token string) (int, []byte) {
	e.t.Helper()
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			e.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

// call performs the request, asserts the status and decodes the JSON response into out (if non-nil).
func (e *env) call(method, path string, payload any, token string, wantStatus int, out any) {
	e.t.Helper()
	status, body := e.do(method, path, payload, token)
	if status != wantStatus {
		e.t.Fatalf("%s %s: want %d, got %d body=%s", method, path, wantStatus, status, body)
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			e.t.Fatalf("%s %s: decode %q: %v", method, path, body, err)
		}
	}
}

func (e *env) upload(method, path, token string, data []byte) (int, []byte) {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("photo", "x.jpg")
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
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

func uniqueEmail(prefix string) string {
	return prefix + "-" + time.Now().UTC().Format("150405.000000") + "-" + strings.TrimSpace(string(rune('a'+emailCounter.Add(1)%26))) + "@test.invalid"
}

type spec struct {
	name         string
	gender       string
	interestedIn []string
	lat, lng     float64
	birth        string
	noPhoto      bool
}

// newUser registers an account and completes its profile (unless spec says otherwise).
func (e *env) newUser(s spec) testUser {
	e.t.Helper()
	if s.birth == "" {
		s.birth = "1994-05-12"
	}
	email := uniqueEmail(s.name)
	e.emails = append(e.emails, email)
	var res struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	e.call("POST", "/auth/register", map[string]any{"email": email, "password": "correct-horse", "firstName": s.name, "birthDate": s.birth}, "", 201, &res)
	u := testUser{ID: res.User.ID, Email: email, Token: res.AccessToken, Refresh: res.RefreshToken}
	e.call("PUT", "/profile", map[string]any{"gender": s.gender, "bio": "Hello from " + s.name, "interests": []string{"music", "hiking"}}, u.Token, 200, nil)
	e.call("PUT", "/profile/location", map[string]any{"latitude": s.lat, "longitude": s.lng, "city": "Testville"}, u.Token, 200, nil)
	e.call("PUT", "/preferences", map[string]any{"interestedIn": s.interestedIn, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 100}, u.Token, 200, nil)
	if !s.noPhoto {
		status, body := e.upload("POST", "/photos", u.Token, pngBytes(500, 600))
		if status != 201 {
			e.t.Fatalf("upload photo: %d %s", status, body)
		}
	}
	return u
}

func (e *env) swipe(u testUser, target testUser, action string) (matched bool, matchID, conversationID string) {
	e.t.Helper()
	var res struct {
		Matched bool `json:"matched"`
		Match   *struct {
			ID             string `json:"id"`
			ConversationID string `json:"conversationId"`
		} `json:"match"`
	}
	e.call("POST", "/swipes", map[string]any{"userId": target.ID, "action": action}, u.Token, 200, &res)
	if res.Match != nil {
		return res.Matched, res.Match.ID, res.Match.ConversationID
	}
	return res.Matched, "", ""
}
