package app

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
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/mailer"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "integration-k8s2Jd9fLq3Zp0Xv7Nw4Rt6Yb1Hc5Gm8"

type captureMailer struct {
	app      *App
	mu       sync.Mutex
	messages []mailer.Message
}

func (m *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

var resetCodePattern = regexp.MustCompile(`: ([A-Z2-9]{10})`)

func (m *captureMailer) lastCodeFor(email string) string {
	m.app.Shutdown() // e-mails are sent asynchronously
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].To == email {
			if match := resetCodePattern.FindStringSubmatch(m.messages[i].Body); match != nil {
				return match[1]
			}
		}
	}
	return ""
}

type env struct {
	t         *testing.T
	pool      *pgxpool.Pool
	app       *App
	server    *httptest.Server
	mail      *captureMailer
	uploadDir string
}

func relaxedLimits() Limits {
	big := rate{Count: 100000, Window: time.Minute, Burst: 100000}
	return Limits{GlobalPerIP: big, Auth: big, Swipes: big, Messages: big, Uploads: big, Reports: big}
}

func newEnv(t *testing.T, limits Limits) *env {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL_TEST")
	if databaseURL == "" {
		t.Skip("set DATABASE_URL_TEST to run integration tests")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	// Integration tests own this database: start from a clean slate.
	if _, err := pool.Exec(ctx, `DELETE FROM users`); err != nil {
		t.Fatalf("reset: %v", err)
	}
	mail := &captureMailer{}
	uploadDir := t.TempDir()
	cfg := config.Config{AppEnv: config.EnvTest, JWTSecret: testSecret, UploadDir: uploadDir}
	application, err := New(cfg, pool, Options{Mailer: mail, Limits: limits})
	if err != nil {
		t.Fatalf("app: %v", err)
	}
	mail.app = application
	e := &env{t: t, pool: pool, app: application, server: httptest.NewServer(application.Engine), mail: mail, uploadDir: uploadDir}
	t.Cleanup(func() {
		e.server.Close()
		_, _ = pool.Exec(context.Background(), `DELETE FROM users`)
		pool.Close()
	})
	return e
}

type response struct {
	Status int
	Body   []byte
}

func (r response) decode(t *testing.T, dst any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
}

func (r response) code(t *testing.T) string {
	t.Helper()
	var payload struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(r.Body, &payload)
	return payload.Code
}

func (e *env) do(method, path, token string, body any) response {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		e.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.send(req)
}

func (e *env) send(req *http.Request) response {
	e.t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return response{Status: res.StatusCode, Body: raw}
}

func (e *env) expect(r response, status int, context string) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: expected HTTP %d, got %d: %s", context, status, r.Status, r.Body)
	}
}

type session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID              string `json:"id"`
		Email           string `json:"email"`
		ProfileComplete bool   `json:"profileComplete"`
	} `json:"user"`
}

var emailCounter atomic.Int64

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s.%d.%d@lueur.test", prefix, time.Now().UnixNano(), emailCounter.Add(1))
}

const testPassword = "Password123"

func (e *env) register(prefix string) session {
	e.t.Helper()
	res := e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": uniqueEmail(prefix), "password": testPassword})
	e.expect(res, http.StatusCreated, "register")
	var s session
	res.decode(e.t, &s)
	return s
}

func pngBytes(t *testing.T, w, h int, tint uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{tint, uint8(x), uint8(y), 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func (e *env) upload(method, path, token string, data []byte) response {
	e.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("photo", "photo.png")
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = part.Write(data)
	_ = writer.Close()
	req, err := http.NewRequest(method, e.server.URL+path, &body)
	if err != nil {
		e.t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.send(req)
}

type member struct {
	session
	Name string
}

type profileSpec struct {
	Name         string
	Birthdate    string
	Gender       string
	InterestedIn []string
	Lat, Lng     *float64
	Interests    []int
	MaxKm        int
}

func coords(lat, lng float64) (*float64, *float64) { return &lat, &lng }

// onboard registers a member and completes the profile like the app does.
func (e *env) onboard(spec profileSpec) member {
	e.t.Helper()
	s := e.register(strings.ToLower(spec.Name))
	if spec.Birthdate == "" {
		spec.Birthdate = "1994-05-10"
	}
	if spec.MaxKm == 0 {
		spec.MaxKm = 50
	}
	res := e.do(http.MethodPut, "/profile", s.AccessToken, map[string]any{
		"firstName": spec.Name, "birthdate": spec.Birthdate, "gender": spec.Gender,
		"bio": "Bonjour, je suis " + spec.Name, "interestIds": spec.Interests,
	})
	e.expect(res, http.StatusOK, "create profile "+spec.Name)
	res = e.do(http.MethodPut, "/profile/preferences", s.AccessToken, map[string]any{
		"interestedIn": spec.InterestedIn, "ageMin": 18, "ageMax": 99, "maxDistanceKm": spec.MaxKm,
	})
	e.expect(res, http.StatusOK, "preferences "+spec.Name)
	if spec.Lat != nil {
		res = e.do(http.MethodPut, "/profile/location", s.AccessToken, map[string]any{"latitude": *spec.Lat, "longitude": *spec.Lng})
		e.expect(res, http.StatusNoContent, "location "+spec.Name)
	}
	res = e.upload(http.MethodPost, "/profile/photos", s.AccessToken, pngBytes(e.t, 320, 400, 10))
	e.expect(res, http.StatusCreated, "photo "+spec.Name)
	return member{session: s, Name: spec.Name}
}

type card struct {
	UserID     string `json:"userId"`
	FirstName  string `json:"firstName"`
	Age        int    `json:"age"`
	DistanceKm *int   `json:"distanceKm"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		ID     int  `json:"id"`
		Shared bool `json:"shared"`
	} `json:"interests"`
}

func (e *env) discover(token string) []card {
	e.t.Helper()
	res := e.do(http.MethodGet, "/discovery?limit=25", token, nil)
	e.expect(res, http.StatusOK, "discovery")
	var payload struct {
		Profiles []card `json:"profiles"`
	}
	res.decode(e.t, &payload)
	return payload.Profiles
}

func containsCard(cards []card, userID string) bool {
	for _, c := range cards {
		if c.UserID == userID {
			return true
		}
	}
	return false
}

type swipeResult struct {
	Matched        bool    `json:"matched"`
	MatchID        *string `json:"matchId"`
	ConversationID *string `json:"conversationId"`
}

func (e *env) swipe(token, target, action string) (response, swipeResult) {
	e.t.Helper()
	res := e.do(http.MethodPost, "/swipes", token, map[string]string{"targetUserId": target, "action": action})
	var out swipeResult
	if res.Status == http.StatusOK {
		res.decode(e.t, &out)
	}
	return res, out
}

// match makes a and b like each other and returns the conversation id.
func (e *env) match(a, b member) (matchID, conversationID string) {
	e.t.Helper()
	res, first := e.swipe(a.AccessToken, b.User.ID, "like")
	e.expect(res, http.StatusOK, "first like")
	if first.Matched {
		e.t.Fatal("a single like must not create a match")
	}
	res, second := e.swipe(b.AccessToken, a.User.ID, "like")
	e.expect(res, http.StatusOK, "second like")
	if !second.Matched || second.ConversationID == nil || second.MatchID == nil {
		e.t.Fatalf("mutual like must create a match: %s", res.Body)
	}
	return *second.MatchID, *second.ConversationID
}
