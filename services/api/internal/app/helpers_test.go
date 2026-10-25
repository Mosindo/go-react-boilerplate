package app_test

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
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type captureMailer struct {
	mu   sync.Mutex
	sent map[string]string
}

func (m *captureMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sent == nil {
		m.sent = map[string]string{}
	}
	m.sent[to] = body
	return nil
}

func (m *captureMailer) last(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent[to]
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	srv    *httptest.Server
	mail   *captureMailer
	secret string
}

func newEnv(t *testing.T) *env {
	return newEnvWithLimits(t, app.Limits{Auth: 10000, Swipe: 10000, Message: 10000, Report: 10000, Upload: 10000, API: 100000})
}

func newEnvWithLimits(t *testing.T, limits app.Limits) *env {
	t.Helper()
	pool := testutil.NewPool(t)
	mail := &captureMailer{}
	secret := "integration-test-secret-0123456789abcdef"
	engine := app.New(app.Options{Config: config.Config{JWTSecret: secret}, Pool: pool, Mailer: mail, Limits: limits, BcryptCost: bcrypt.MinCost})
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return &env{t: t, pool: pool, srv: srv, mail: mail, secret: secret}
}

type resp struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r resp) decode(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r resp) errCode() string {
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(r.Body, &e)
	return e.Code
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, reader)
	if err != nil {
		e.t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(req)
}

func (e *env) send(req *http.Request) resp {
	e.t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Body: data, Header: res.Header}
}

func (e *env) upload(method, path, token string, content []byte, filename string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		e.t.Fatalf("form file: %v", err)
	}
	_, _ = part.Write(content)
	_ = w.Close()
	req, _ := http.NewRequest(method, e.srv.URL+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(req)
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Body)
	}
	return r
}

func pngBytes(w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type testUser struct {
	env     *env
	ID      string
	Email   string
	Token   string
	Refresh string
}

type authBody struct {
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

func uniqueEmail(prefix string) string {
	emailMu.Lock()
	defer emailMu.Unlock()
	emailSeq++
	return fmt.Sprintf("%s%d@test.invalid", prefix, emailSeq)
}

func (e *env) register(prefix string) *testUser {
	e.t.Helper()
	email := uniqueEmail(prefix)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "correct-horse-1"}), 201)
	var a authBody
	r.decode(e.t, &a)
	return &testUser{env: e, ID: a.User.ID, Email: email, Token: a.AccessToken, Refresh: a.RefreshToken}
}

func birthDateForAge(age int) string {
	return time.Now().AddDate(-age, 0, -10).Format("2006-01-02")
}

type signupOpts struct {
	Gender       string
	InterestedIn []string
	Age          int
	MinAge       int
	MaxAge       int
	MaxDistance  *int
	Interests    []string
	NoPhoto      bool
}

// signup registers a user and completes the whole onboarding (profile, preferences, photo).
func (e *env) signup(name string, o signupOpts) *testUser {
	e.t.Helper()
	if o.Age == 0 {
		o.Age = 30
	}
	if o.MinAge == 0 {
		o.MinAge = 18
	}
	if o.MaxAge == 0 {
		o.MaxAge = 99
	}
	u := e.register(strings.ToLower(name))
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{
		"firstName": name, "birthDate": birthDateForAge(o.Age), "gender": o.Gender, "bio": "Bio de " + name, "interests": o.Interests,
	}), 200)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{
		"interestedIn": o.InterestedIn, "minAge": o.MinAge, "maxAge": o.MaxAge, "maxDistanceKm": o.MaxDistance,
	}), 200)
	if !o.NoPhoto {
		e.expect(e.upload("POST", "/photos", u.Token, pngBytes(64, 48, color.RGBA{200, 80, 60, 255}), "a.png"), 201)
	}
	return u
}

func (u *testUser) do(method, path string, body any) resp {
	return u.env.do(method, path, u.Token, body)
}

func (u *testUser) feedIDs() []string {
	u.env.t.Helper()
	r := u.env.expect(u.do("GET", "/discovery?limit=20", nil), 200)
	var out struct {
		Items []struct {
			UserID string `json:"userId"`
		} `json:"items"`
	}
	r.decode(u.env.t, &out)
	ids := make([]string, len(out.Items))
	for i, it := range out.Items {
		ids[i] = it.UserID
	}
	return ids
}

type swipeResult struct {
	Result string `json:"result"`
	Match  *struct {
		MatchID        string `json:"matchId"`
		ConversationID string `json:"conversationId"`
	} `json:"match"`
}

func (u *testUser) swipe(target, action string) (int, swipeResult) {
	u.env.t.Helper()
	r := u.do("POST", "/discovery/swipes", map[string]string{"userId": target, "action": action})
	var out swipeResult
	if r.Status == 200 {
		r.decode(u.env.t, &out)
	}
	return r.Status, out
}

// matchWith makes both users like each other and returns the conversation id.
func matchWith(t *testing.T, a, b *testUser) string {
	t.Helper()
	if st, res := a.swipe(b.ID, "like"); st != 200 || res.Result != "liked" {
		t.Fatalf("first like: %d %+v", st, res)
	}
	st, res := b.swipe(a.ID, "like")
	if st != 200 || res.Result != "matched" || res.Match == nil {
		t.Fatalf("second like should match: %d %+v", st, res)
	}
	return res.Match.ConversationID
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

var codeRe = regexp.MustCompile(`: ([A-Z0-9]{10})\n`)

func extractCode(t *testing.T, body string) string {
	t.Helper()
	m := codeRe.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no reset code in %q", body)
	}
	return m[1]
}
