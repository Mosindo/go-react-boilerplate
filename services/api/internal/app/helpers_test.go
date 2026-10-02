package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "0123456789abcdef0123456789abcdef-integration"

type capturingMailer struct {
	mu   sync.Mutex
	last map[string]string
}

func (m *capturingMailer) Send(_ context.Context, to, _, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.last == nil {
		m.last = map[string]string{}
	}
	m.last[to] = body
	return nil
}

func (m *capturingMailer) body(to string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last[to]
}

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	svc    app.Services
	mail   *capturingMailer
	server *httptest.Server
	ids    []string
	// every test works in its own geographic cell so parallel/previous runs never interfere
	lat, lng float64
	run      string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST (or DATABASE_URL) is required for integration tests")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mail := &capturingMailer{}
	gin.SetMode(gin.TestMode)
	router, svc := app.New(app.Deps{DB: pool, JWTSecret: []byte(testSecret), Store: store, Mailer: mail, BcryptCost: 4})
	e := &env{t: t, pool: pool, router: router, svc: svc, mail: mail,
		lat: -60 + rand.Float64()*100, lng: -170 + rand.Float64()*340, run: fmt.Sprint(time.Now().UnixNano())}
	e.server = httptest.NewServer(router)
	t.Cleanup(func() {
		e.server.Close()
		if len(e.ids) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = ANY($1::uuid[])`, e.ids)
		}
		pool.Close()
	})
	return e
}

type resp struct {
	Status int
	Raw    []byte
	JSON   map[string]any
}

func (r resp) str(path ...string) string {
	var cur any = r.JSON
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[p]
	}
	s, _ := cur.(string)
	return s
}

func (e *env) do(method, path, token string, body any) resp {
	e.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.RemoteAddr = "10.1.1.1:1234"
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	out := resp{Status: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(out.Raw, &out.JSON)
	return out
}

func (e *env) expect(r resp, status int) resp {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, r.Raw)
	}
	return r
}

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 255), uint8(y % 255), 120, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func (e *env) upload(method, path, token string, data []byte, filename string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", filename)
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = "10.1.1.1:1234"
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	out := resp{Status: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(out.Raw, &out.JSON)
	return out
}

type user struct {
	ID, Email, Token, Refresh string
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s-%s@integration.test", prefix, e.run)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(email) + " ", "password": "Password123"}), 201)
	u := user{ID: r.str("user", "id"), Email: email, Token: r.str("accessToken"), Refresh: r.str("refreshToken")}
	e.ids = append(e.ids, u.ID)
	return u
}

type profileOpts struct {
	name, gender, birth string
	interestedIn        []string
	minAge, maxAge      int
	maxDistance         int
	dLat                float64 // offset in degrees from the test cell
	interests           []string
	noPhoto             bool
}

// onboard creates profile + preferences + one photo, like the mobile onboarding does.
func (e *env) onboard(u user, o profileOpts) {
	e.t.Helper()
	if o.birth == "" {
		o.birth = time.Now().AddDate(-28, 0, -10).Format("2006-01-02")
	}
	if o.minAge == 0 {
		o.minAge, o.maxAge = 18, 99
	}
	if o.maxDistance == 0 {
		o.maxDistance = 20
	}
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{
		"firstName": o.name, "birthDate": o.birth, "gender": o.gender, "bio": "bio " + o.name,
		"city": "Testville", "latitude": e.lat + o.dLat, "longitude": e.lng, "interests": o.interests,
	}), 200)
	e.expect(e.do("PUT", "/preferences", u.Token, map[string]any{
		"interestedIn": o.interestedIn, "minAge": o.minAge, "maxAge": o.maxAge, "maxDistanceKm": o.maxDistance,
	}), 200)
	if !o.noPhoto {
		e.expect(e.upload("POST", "/photos", u.Token, pngBytes(400, 500), "me.png"), 201)
	}
}

func (e *env) discoverIDs(u user) []string {
	e.t.Helper()
	r := e.expect(e.do("GET", "/discover?limit=20", u.Token, nil), 200)
	var ids []string
	list, _ := r.JSON["profiles"].([]any)
	for _, p := range list {
		ids = append(ids, p.(map[string]any)["id"].(string))
	}
	return ids
}

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func httptestGet(e *env, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}
