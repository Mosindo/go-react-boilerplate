package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("DATABASE_URL_TEST or DATABASE_URL must point to a PostgreSQL database")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	r, err := newRouter(pool, config.Config{Env: "test", JWTSecret: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	return &env{t: t, pool: pool, router: r}
}

type resp struct {
	Status int
	Body   []byte
}

func (r resp) JSON(v any) {
	if err := json.Unmarshal(r.Body, v); err != nil {
		panic(fmt.Sprintf("decode %q: %v", r.Body, err))
	}
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
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", rand.Intn(250), rand.Intn(250), rand.Intn(250)) // isolate rate limits
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes()}
}

func testJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, nil)
	return b.Bytes()
}

func (e *env) upload(method, path, token string, content []byte) resp {
	e.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "x.jpg")
	_, _ = fw.Write(content)
	_ = mw.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:1234", rand.Intn(250), rand.Intn(250), rand.Intn(250))
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{Status: w.Code, Body: w.Body.Bytes()}
}

type user struct {
	ID, Email, Token, Refresh string
}

func (e *env) register(prefix string) user {
	e.t.Helper()
	email := fmt.Sprintf("%s_%d_%d@tandem.test", prefix, time.Now().UnixNano(), rand.Intn(1e6))
	r := e.do("POST", "/auth/register", "", map[string]string{"email": email, "password": "Password123"})
	if r.Status != http.StatusCreated {
		e.t.Fatalf("register: %d %s", r.Status, r.Body)
	}
	var out struct {
		AccessToken, RefreshToken string
		User                      struct{ ID, Email string }
	}
	r.JSON(&out)
	e.t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email) })
	return user{ID: out.User.ID, Email: email, Token: out.AccessToken, Refresh: out.RefreshToken}
}

type spec struct {
	Name, Gender string
	Birth        string
	Interested   []string
	Lat, Lng     float64
}

// onboard completes profile + preferences + one photo.
func (e *env) onboard(u user, s spec) {
	e.t.Helper()
	r := e.do("PUT", "/me/profile", u.Token, map[string]any{
		"firstName": s.Name, "birthDate": s.Birth, "gender": s.Gender, "bio": "bio " + s.Name,
		"city": "Testville", "latitude": s.Lat, "longitude": s.Lng, "interestIds": []int{1, 2},
	})
	if r.Status != 200 {
		e.t.Fatalf("profile: %d %s", r.Status, r.Body)
	}
	r = e.do("PUT", "/me/preferences", u.Token, map[string]any{
		"interestedIn": s.Interested, "minAge": 18, "maxAge": 99, "maxDistanceKm": 25,
	})
	if r.Status != 200 {
		e.t.Fatalf("prefs: %d %s", r.Status, r.Body)
	}
	r = e.upload("POST", "/me/photos", u.Token, testJPEG(400, 500))
	if r.Status != 201 {
		e.t.Fatalf("photo: %d %s", r.Status, r.Body)
	}
}

// region returns an isolated spot on Earth so tests do not see each other's users.
func region() (float64, float64) {
	return -60 + rand.Float64()*100, -170 + rand.Float64()*340
}

func birth(age int) string {
	return time.Now().AddDate(-age, -1, 0).Format("2006-01-02")
}

func (e *env) pair() (a, b user) {
	lat, lng := region()
	a, b = e.register("alice"), e.register("bob")
	e.onboard(a, spec{"Alice", "woman", birth(28), []string{"man"}, lat, lng})
	e.onboard(b, spec{"Bob", "man", birth(30), []string{"woman"}, lat + 0.02, lng})
	return
}
