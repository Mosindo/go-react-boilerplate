// Package testutil holds helpers for integration tests that run against a real PostgreSQL.
// Tests skip themselves when DATABASE_URL_TEST / DATABASE_URL is not set.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/urlsign"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const Secret = "integration-test-secret-0123456789abcdef"

var counter atomic.Int64

// Pool connects, migrates and returns a pool; it skips the test when no database is configured.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL_TEST")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("skip integration test: DATABASE_URL_TEST or DATABASE_URL must be set")
	}
	pool, err := db.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("skip integration test: postgres unavailable (%v)", err)
	}
	if err := db.RunMigrations(context.Background(), pool); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Deps builds shared platform dependencies with a temp upload dir and generous rate limits.
// notifier and publisher may be nil (no-ops are used).
func Deps(t *testing.T, notifier notify.Notifier, publisher realtime.Publisher) deps.Common {
	t.Helper()
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if notifier == nil {
		notifier = notify.Nop{}
	}
	if publisher == nil {
		publisher = realtime.NopPublisher{}
	}
	return deps.Common{
		Config:        config.Config{Env: "test", JWTSecret: Secret},
		JWTSecret:     []byte(Secret),
		RequireUser:   middleware.RequireUser([]byte(Secret)),
		Storage:       store,
		Signer:        urlsign.New([]byte(Secret)),
		Mailer:        mailer.LogMailer{},
		Publisher:     publisher,
		Notifier:      notifier,
		AuthLimiter:   ratelimit.New(1000, time.Minute),
		ActionLimiter: ratelimit.New(100000, time.Minute),
	}
}

// Router builds a Gin engine and lets the caller register the feature(s) under test.
func Router(register func(r gin.IRouter)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	register(r)
	return r
}

// User inserts a user directly (bypassing the auth feature) and returns its id and a valid access token.
func User(t *testing.T, pool *pgxpool.Pool, prefix string, birthDate string) (id, token string) {
	t.Helper()
	ctx := context.Background()
	email := prefix + "_" + time.Now().UTC().Format("150405.000000000") + "_" + itoa(counter.Add(1)) + "@test.local"
	var sessionID string
	err := pool.QueryRow(ctx, `
		WITH u AS (
			INSERT INTO users (email, password_hash, birth_date) VALUES ($1, 'x', $2) RETURNING id
		)
		INSERT INTO sessions (user_id, token_hash, expires_at)
		SELECT id, md5(random()::text || $1), NOW() + INTERVAL '1 day' FROM u
		RETURNING user_id, id`, email, birthDate).Scan(&id, &sessionID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id) })
	return id, Token(id, sessionID)
}

// Token signs an access token exactly like the auth feature does.
func Token(userID, sessionID string) string {
	claims := jwt.MapClaims{
		"uid": userID,
		"sid": sessionID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(Secret))
	if err != nil {
		panic(err)
	}
	return s
}

type Response struct {
	Status int
	Body   []byte
}

// JSON decodes the body into v, failing the test on error.
func (r Response) JSON(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %q: %v", string(r.Body), err)
	}
}

// Do performs a JSON request against the handler.
func Do(t *testing.T, h http.Handler, method, path string, payload any, token string) Response {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			t.Fatal(err)
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
	h.ServeHTTP(rec, req)
	return Response{Status: rec.Code, Body: rec.Body.Bytes()}
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
