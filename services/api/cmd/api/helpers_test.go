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
	"sync/atomic"
	"testing"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var emailSeq atomic.Int64

type testEnv struct {
	t         *testing.T
	pool      *pgxpool.Pool
	router    *gin.Engine
	uploadDir string
}

// newTestEnv ouvre la base de test (TEST_DATABASE_URL, dont le nom doit finir par
// "_test": la base est videe a chaque test), applique les migrations et monte le routeur.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB integration test")
	}
	u, err := url.Parse(dsn)
	if err != nil || !strings.HasSuffix(strings.TrimPrefix(u.Path, "/"), "_test") {
		t.Fatalf("refusing to run: TEST_DATABASE_URL database name must end with _test")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	gin.SetMode(gin.TestMode)
	uploadDir := t.TempDir()
	cfg := config.Config{JWTSecret: "integration-test-jwt-secret-0123456789", UploadDir: uploadDir, AppEnv: "test"}
	router, err := setupRouter(cfg, pool)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	return &testEnv{t: t, pool: pool, router: router, uploadDir: uploadDir}
}

type resp struct {
	Status int
	Body   map[string]any
	Raw    []byte
}

func (r resp) str(key string) string {
	v, _ := r.Body[key].(string)
	return v
}

func (e *testEnv) do(method, path, token string, body any) resp {
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
	return e.serve(req)
}

func (e *testEnv) serve(req *http.Request) resp {
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	r := resp{Status: w.Code, Raw: w.Body.Bytes()}
	_ = json.Unmarshal(w.Body.Bytes(), &r.Body)
	return r
}

func (e *testEnv) upload(token string, data []byte, query string) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "photo.png")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/me/photos"+query, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.serve(req)
}

func testPNG(w, h int, c color.RGBA) []byte {
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

type user struct {
	ID, Token, Refresh, Email string
}

func (e *testEnv) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s%d@test.example", prefix, emailSeq.Add(1))
	r := e.do("POST", "/auth/register", "", map[string]any{"email": email, "password": "Passw0rd!x"})
	if r.Status != http.StatusCreated {
		e.t.Fatalf("register: %d %s", r.Status, r.Raw)
	}
	u, _ := r.Body["user"].(map[string]any)
	id, _ := u["id"].(string)
	return user{ID: id, Token: r.str("accessToken"), Refresh: r.str("refreshToken"), Email: email}
}

// profile cree un profil + une photo. lat/lng en degres.
func (e *testEnv) profile(u user, name, gender string, interestedIn []string, birth string, lat, lng float64, interests ...string) {
	e.t.Helper()
	r := e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": name, "birthDate": birth, "gender": gender, "bio": "Bio de " + name, "city": "Ville",
		"latitude": lat, "longitude": lng, "interests": interests,
	})
	if r.Status != http.StatusOK {
		e.t.Fatalf("profile %s: %d %s", name, r.Status, r.Raw)
	}
	s := e.do("PUT", "/me/settings", u.Token, map[string]any{
		"interestedIn": interestedIn, "minAge": 18, "maxAge": 99, "maxDistanceKm": 100, "discoverable": true, "showDistance": true,
	})
	if s.Status != http.StatusOK {
		e.t.Fatalf("settings %s: %d %s", name, s.Status, s.Raw)
	}
	up := e.upload(u.Token, testPNG(400, 400, color.RGBA{200, 80, 80, 255}), "")
	if up.Status != http.StatusCreated {
		e.t.Fatalf("upload %s: %d %s", name, up.Status, up.Raw)
	}
}
