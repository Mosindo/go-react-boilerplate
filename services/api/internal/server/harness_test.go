package server_test

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
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/server"
	"example.com/api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type captureMailer struct{ last string }

func (m *captureMailer) Send(_ context.Context, _, _, body string) error {
	m.last = body
	return nil
}

type harness struct {
	t         *testing.T
	app       *server.App
	pool      *pgxpool.Pool
	mail      *captureMailer
	uploadDir string
}

var counter atomic.Int64

func newHarness(t *testing.T) *harness {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)
	log.SetOutput(io.Discard)
	pool := testutil.NewPool(t)
	mail := &captureMailer{}
	dir := t.TempDir()
	app, err := server.New(server.Options{
		DB: pool, JWTSecret: []byte("0123456789abcdef0123456789abcdef"), UploadDir: dir, Mailer: mail,
		AllowedOrigins: []string{"https://app.test"},
	})
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	return &harness{t: t, app: app, pool: pool, mail: mail, uploadDir: dir}
}

func (h *harness) raw(req *http.Request) (int, []byte, http.Header) {
	rec := httptest.NewRecorder()
	h.app.Engine.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes(), rec.Header()
}

func (h *harness) do(method, path, token string, body any) (int, []byte) {
	h.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	code, out, _ := h.raw(req)
	return code, out
}

// expect performs a request, asserts the status and decodes the body into out (if non-nil).
func (h *harness) expect(status int, method, path, token string, body, out any) {
	h.t.Helper()
	code, raw := h.do(method, path, token, body)
	if code != status {
		h.t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, code, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			h.t.Fatalf("%s %s: decode %q: %v", method, path, raw, err)
		}
	}
}

func (h *harness) upload(method, path, token string, data []byte, filename, contentType string) (int, []byte) {
	h.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	hdr.Set("Content-Type", contentType)
	part, _ := mw.CreatePart(hdr)
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	code, out, _ := h.raw(req)
	return code, out
}

func jpegBytes(w, hgt int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, hgt))
	for y := 0; y < hgt; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / hgt), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

type user struct {
	ID, Token, Email, Refresh string
}

func (h *harness) signup(prefix string) user {
	h.t.Helper()
	email := fmt.Sprintf("%s%d@example.com", prefix, counter.Add(1))
	var res struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		User         struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	h.expect(201, "POST", "/auth/register", "", map[string]string{"email": email, "password": "Passw0rd!x"}, &res)
	return user{ID: res.User.ID, Token: res.AccessToken, Email: email, Refresh: res.RefreshToken}
}

type spec struct {
	Name         string
	Gender       string
	AgeYears     int
	InterestedIn []string
	MinAge       int
	MaxAge       int
	Lat, Lng     *float64
	MaxDistance  *int
	Interests    []string
	NoPhoto      bool
}

func f(v float64) *float64 { return &v }

// onboard registers a user and completes profile, preferences, optional location and a photo.
func (h *harness) onboard(s spec) user {
	h.t.Helper()
	u := h.signup(strings.ToLower(s.Name))
	birth := time.Now().AddDate(-s.AgeYears, 0, -3).Format("2006-01-02")
	h.expect(200, "PUT", "/me/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthdate": birth, "gender": s.Gender, "bio": "Hello", "interests": s.Interests,
	}, nil)
	if s.InterestedIn == nil {
		s.InterestedIn = []string{"man", "woman", "non_binary"}
	}
	if s.MinAge == 0 {
		s.MinAge = 18
	}
	if s.MaxAge == 0 {
		s.MaxAge = 99
	}
	h.expect(200, "PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": s.InterestedIn, "minAge": s.MinAge, "maxAge": s.MaxAge, "maxDistanceKm": s.MaxDistance,
		"isVisible": true, "showDistance": true,
	}, nil)
	if s.Lat != nil {
		h.expect(200, "PUT", "/me/location", u.Token, map[string]any{"latitude": *s.Lat, "longitude": *s.Lng, "city": "Ville"}, nil)
	}
	if !s.NoPhoto {
		code, body := h.upload("POST", "/me/photos", u.Token, jpegBytes(600, 800), "p.jpg", "image/jpeg")
		if code != 201 {
			h.t.Fatalf("upload photo: %d %s", code, body)
		}
	}
	return u
}

type card struct {
	ID         string `json:"id"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
}

func (h *harness) discover(u user) []card {
	h.t.Helper()
	var res struct {
		Profiles []card `json:"profiles"`
	}
	h.expect(200, "GET", "/discover?limit=20", u.Token, nil, &res)
	return res.Profiles
}

type swipeResult struct {
	Matched        bool   `json:"matched"`
	AlreadyDone    bool   `json:"alreadyProcessed"`
	MatchID        string `json:"matchId"`
	ConversationID string `json:"conversationId"`
}

func (h *harness) swipe(u user, target user, action string) swipeResult {
	h.t.Helper()
	var r swipeResult
	h.expect(200, "POST", "/swipes", u.Token, map[string]string{"userId": target.ID, "action": action}, &r)
	return r
}

func ids(cards []card) []string {
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.ID
	}
	return out
}

func contains(cards []card, id string) bool {
	for _, c := range cards {
		if c.ID == id {
			return true
		}
	}
	return false
}
