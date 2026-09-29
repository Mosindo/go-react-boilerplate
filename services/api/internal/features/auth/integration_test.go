package auth

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type captureMailer struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (m *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *captureMailer) last() (mailer.Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.msgs) == 0 {
		return mailer.Message{}, false
	}
	return m.msgs[len(m.msgs)-1], true
}

type env struct {
	pool *pgxpool.Pool
	r    *gin.Engine
	mail *captureMailer
}

func setup(t *testing.T) env {
	t.Helper()
	pool := testutil.Pool(t)
	d := testutil.Deps(t, nil, nil)
	cm := &captureMailer{}
	d.Mailer = cm
	d.Config.AppBaseURL = "https://app.example.test"
	r := testutil.Router(func(r gin.IRouter) { registerRoutes(r, pool, d, func(f func()) { f() }) })
	return env{pool: pool, r: r, mail: cm}
}

var emailCounter int

func newEmail() string {
	emailCounter++
	return fmt.Sprintf("Auth_%d_%d@Test.Local", time.Now().UnixNano(), emailCounter)
}

func (e env) cleanup(t *testing.T, emails ...string) {
	t.Cleanup(func() {
		for _, em := range emails {
			_, _ = e.pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, strings.ToLower(em))
		}
	})
}

type authResp struct {
	Token        string `json:"token"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         struct {
		ID              string `json:"id"`
		Email           string `json:"email"`
		BirthDate       string `json:"birthDate"`
		Age             int    `json:"age"`
		ProfileComplete bool   `json:"profileComplete"`
	} `json:"user"`
}

func birthYearsAgo(years int) string {
	return time.Now().UTC().AddDate(-years, 0, -1).Format("2006-01-02")
}

func (e env) register(t *testing.T, email, pw string) authResp {
	t.Helper()
	res := testutil.Do(t, e.r, "POST", "/auth/register", map[string]string{"email": email, "password": pw, "birthDate": birthYearsAgo(30)}, "")
	if res.Status != http.StatusCreated {
		t.Fatalf("register: %d %s", res.Status, res.Body)
	}
	var a authResp
	res.JSON(t, &a)
	return a
}

func TestRegisterLoginRefreshLogout(t *testing.T) {
	e := setup(t)
	email := newEmail()
	e.cleanup(t, email)
	pw := "CorrectHorse!1"

	a := e.register(t, "  "+email+"  ", pw)
	if a.User.Email != strings.ToLower(email) || a.User.Age != 30 || a.User.ProfileComplete {
		t.Fatalf("unexpected user %+v", a.User)
	}
	if a.Token != a.AccessToken || a.AccessToken == "" || a.RefreshToken == "" {
		t.Fatalf("tokens missing")
	}
	// Claims are only uid, sid, exp, iat.
	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(a.AccessToken, claims, func(*jwt.Token) (any, error) { return []byte(testutil.Secret), nil }); err != nil {
		t.Fatal(err)
	}
	if len(claims) != 4 || claims["uid"] != a.User.ID || claims["sid"] == nil || claims["exp"] == nil || claims["iat"] == nil {
		t.Fatalf("unexpected claims %v", claims)
	}

	// Duplicate (case-insensitive)
	res := testutil.Do(t, e.r, "POST", "/auth/register", map[string]string{"email": strings.ToUpper(email), "password": pw, "birthDate": birthYearsAgo(30)}, "")
	if res.Status != http.StatusConflict {
		t.Fatalf("dup want 409 got %d", res.Status)
	}

	// Login: wrong password, unknown email -> same 401 body.
	bad := testutil.Do(t, e.r, "POST", "/auth/login", map[string]string{"email": email, "password": "nope-nope-nope"}, "")
	unk := testutil.Do(t, e.r, "POST", "/auth/login", map[string]string{"email": "ghost" + email, "password": "nope-nope-nope"}, "")
	if bad.Status != 401 || unk.Status != 401 || string(bad.Body) != string(unk.Body) {
		t.Fatalf("login failures differ: %d %s / %d %s", bad.Status, bad.Body, unk.Status, unk.Body)
	}

	res = testutil.Do(t, e.r, "POST", "/auth/login", map[string]string{"email": email, "password": pw}, "")
	if res.Status != 200 {
		t.Fatalf("login %d %s", res.Status, res.Body)
	}
	var l authResp
	res.JSON(t, &l)

	// Refresh rotates; old token is dead.
	res = testutil.Do(t, e.r, "POST", "/auth/refresh", map[string]string{"refreshToken": l.RefreshToken}, "")
	if res.Status != 200 {
		t.Fatalf("refresh %d %s", res.Status, res.Body)
	}
	var rf authResp
	res.JSON(t, &rf)
	if rf.RefreshToken == l.RefreshToken {
		t.Fatal("refresh token did not rotate")
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/refresh", map[string]string{"refreshToken": l.RefreshToken}, ""); res.Status != 401 {
		t.Fatalf("reused refresh want 401 got %d", res.Status)
	}

	// Logout revokes.
	if res = testutil.Do(t, e.r, "POST", "/auth/logout", map[string]string{"refreshToken": rf.RefreshToken}, ""); res.Status != 204 {
		t.Fatalf("logout %d", res.Status)
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/refresh", map[string]string{"refreshToken": rf.RefreshToken}, ""); res.Status != 401 {
		t.Fatalf("refresh after logout want 401 got %d", res.Status)
	}

	var stored string
	if err := e.pool.QueryRow(context.Background(), `SELECT password_hash FROM users WHERE id=$1`, a.User.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$2") || strings.Contains(stored, pw) {
		t.Fatal("password not bcrypt hashed")
	}
}

func TestRegisterValidation(t *testing.T) {
	e := setup(t)
	email := newEmail()
	e.cleanup(t, email)
	cases := []struct {
		name string
		body map[string]string
		want int
	}{
		{"under 18", map[string]string{"email": email, "password": "CorrectHorse!1", "birthDate": time.Now().UTC().AddDate(-17, 0, 0).Format("2006-01-02")}, 422},
		{"future", map[string]string{"email": email, "password": "CorrectHorse!1", "birthDate": time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02")}, 422},
		{"absurd", map[string]string{"email": email, "password": "CorrectHorse!1", "birthDate": "1800-01-01"}, 422},
		{"bad format", map[string]string{"email": email, "password": "CorrectHorse!1", "birthDate": "01/02/1990"}, 400},
		{"missing birth", map[string]string{"email": email, "password": "CorrectHorse!1"}, 400},
		{"short password", map[string]string{"email": email, "password": "short", "birthDate": birthYearsAgo(30)}, 400},
		{"bad email", map[string]string{"email": "not-an-email", "password": "CorrectHorse!1", "birthDate": birthYearsAgo(30)}, 400},
	}
	for _, c := range cases {
		res := testutil.Do(t, e.r, "POST", "/auth/register", c.body, "")
		if res.Status != c.want {
			t.Errorf("%s: want %d got %d %s", c.name, c.want, res.Status, res.Body)
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email=$1`, strings.ToLower(email)).Scan(&n)
	if n != 0 {
		t.Fatal("rejected registrations created users")
	}
	// A password longer than bcrypt's 72 bytes must still be fully significant.
	long := strings.Repeat("a", 100)
	longEmail := newEmail()
	e.cleanup(t, longEmail)
	e.register(t, longEmail, long)
	res := testutil.Do(t, e.r, "POST", "/auth/login", map[string]string{"email": longEmail, "password": long[:99] + "b"}, "")
	if res.Status != 401 {
		t.Fatalf("truncated password accepted: %d", res.Status)
	}
}

var tokenRe = regexp.MustCompile(`token=([A-Za-z0-9_\-]+)`)

func TestForgotAndReset(t *testing.T) {
	e := setup(t)
	email := newEmail()
	e.cleanup(t, email)
	a := e.register(t, email, "OldPassword!1")
	b := e.register(t, newEmail(), "OtherPassword!1")
	e.cleanup(t, b.User.Email)

	// Unknown email: 202, no mail, no token in response.
	res := testutil.Do(t, e.r, "POST", "/auth/forgot", map[string]string{"email": "nobody@test.local"}, "")
	if res.Status != 202 {
		t.Fatalf("forgot unknown %d", res.Status)
	}
	if _, sent := e.mail.last(); sent {
		t.Fatal("mail sent for unknown account")
	}

	res = testutil.Do(t, e.r, "POST", "/auth/forgot", map[string]string{"email": "  " + strings.ToUpper(email)}, "")
	if res.Status != 202 {
		t.Fatalf("forgot %d", res.Status)
	}
	msg, ok := e.mail.last()
	if !ok {
		t.Fatal("no mail sent")
	}
	m := tokenRe.FindStringSubmatch(msg.Body)
	if m == nil || !strings.HasPrefix(msg.Body[strings.Index(msg.Body, "https"):], "https://app.example.test/reset-password?token=") {
		t.Fatalf("mail body lacks link: %q", msg.Body)
	}
	token := m[1]
	if strings.Contains(string(res.Body), token) {
		t.Fatal("token leaked in response")
	}
	var hashed string
	_ = e.pool.QueryRow(context.Background(), `SELECT token_hash FROM password_resets WHERE user_id=$1`, a.User.ID).Scan(&hashed)
	if hashed == "" || hashed == token {
		t.Fatal("reset token must be stored hashed")
	}

	// Weak password / bad token.
	if res = testutil.Do(t, e.r, "POST", "/auth/reset", map[string]string{"token": token, "password": "short"}, ""); res.Status != 400 {
		t.Fatalf("weak %d", res.Status)
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/reset", map[string]string{"token": "bogus", "password": "NewPassword!2"}, ""); res.Status != 400 {
		t.Fatalf("bogus %d", res.Status)
	}

	if res = testutil.Do(t, e.r, "POST", "/auth/reset", map[string]string{"token": token, "password": "NewPassword!2"}, ""); res.Status != 204 {
		t.Fatalf("reset %d %s", res.Status, res.Body)
	}
	// Single use.
	if res = testutil.Do(t, e.r, "POST", "/auth/reset", map[string]string{"token": token, "password": "Another!Pass3"}, ""); res.Status != 400 {
		t.Fatalf("reuse want 400 got %d", res.Status)
	}
	// Sessions revoked; old password dead; new works; other user untouched.
	if res = testutil.Do(t, e.r, "POST", "/auth/refresh", map[string]string{"refreshToken": a.RefreshToken}, ""); res.Status != 401 {
		t.Fatalf("session survived reset: %d", res.Status)
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/login", map[string]string{"email": email, "password": "OldPassword!1"}, ""); res.Status != 401 {
		t.Fatalf("old password works: %d", res.Status)
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/login", map[string]string{"email": email, "password": "NewPassword!2"}, ""); res.Status != 200 {
		t.Fatalf("new password fails: %d", res.Status)
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/refresh", map[string]string{"refreshToken": b.RefreshToken}, ""); res.Status != 200 {
		t.Fatalf("other user's session affected: %d", res.Status)
	}

	// Expired token.
	tok2 := "expired-token-value"
	_, err := e.pool.Exec(context.Background(), `INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1,$2, NOW() - INTERVAL '1 minute')`, a.User.ID, hashToken(tok2))
	if err != nil {
		t.Fatal(err)
	}
	if res = testutil.Do(t, e.r, "POST", "/auth/reset", map[string]string{"token": tok2, "password": "NewPassword!3"}, ""); res.Status != 400 {
		t.Fatalf("expired want 400 got %d", res.Status)
	}
}

func TestForgotThrottlePerUser(t *testing.T) {
	e := setup(t)
	email := newEmail()
	e.cleanup(t, email)
	e.register(t, email, "OldPassword!1")
	testutil.Do(t, e.r, "POST", "/auth/forgot", map[string]string{"email": email}, "")
	testutil.Do(t, e.r, "POST", "/auth/forgot", map[string]string{"email": email}, "")
	e.mail.mu.Lock()
	n := len(e.mail.msgs)
	e.mail.mu.Unlock()
	if n != 1 {
		t.Fatalf("want 1 mail, got %d", n)
	}
}
