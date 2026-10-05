package main

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestAuthRegisterLoginMe(t *testing.T) {
	env := newEnv(t)
	s := env.register("alice")

	// Duplicate emails are rejected, whatever the casing or whitespace.
	env.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(s.Email) + " ", "password": testPassword}).
		expect(t, http.StatusConflict)

	var login authBody
	env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": s.Email, "password": testPassword}).
		expect(t, http.StatusOK).json(t, &login)
	if login.User.ID != s.ID {
		t.Fatalf("login returned another user")
	}
	env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": s.Email, "password": "wrong-password"}).expect(t, http.StatusUnauthorized)
	env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "nobody@test.local", "password": testPassword}).expect(t, http.StatusUnauthorized)

	var me struct {
		ID, Email string
	}
	env.do(http.MethodGet, "/me", login.AccessToken, nil).expect(t, http.StatusOK).json(t, &me)
	if me.ID != s.ID || me.Email != s.Email {
		t.Fatalf("unexpected /me: %+v", me)
	}
	env.do(http.MethodGet, "/me", "", nil).expect(t, http.StatusUnauthorized)
	env.do(http.MethodGet, "/me", "not-a-jwt", nil).expect(t, http.StatusUnauthorized)
}

func TestAuthPasswordValidation(t *testing.T) {
	env := newEnv(t)
	for name, password := range map[string]string{
		"too short":  "short",
		"too long":   strings.Repeat("a", 73), // bcrypt only reads 72 bytes
		"multi-byte": strings.Repeat("é", 40), // 80 bytes
	} {
		r := env.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "pw@test.local", "password": password})
		if r.Status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", name, r.Status)
		}
	}
	env.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "not-an-email", "password": testPassword}).expect(t, http.StatusBadRequest)
}

func TestAuthRefreshRotationAndLogout(t *testing.T) {
	env := newEnv(t)
	s := env.register("rotate")

	var next authBody
	env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": s.Refresh}).expect(t, http.StatusOK).json(t, &next)
	if next.RefreshToken == s.Refresh || next.AccessToken == "" {
		t.Fatal("refresh token must rotate")
	}
	// A refresh token is single-use.
	env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": s.Refresh}).expect(t, http.StatusUnauthorized)

	env.do(http.MethodGet, "/me", next.AccessToken, nil).expect(t, http.StatusOK)
	env.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": next.RefreshToken}).expect(t, http.StatusNoContent)
	// Logout takes effect immediately, even for access tokens that have not expired yet.
	env.do(http.MethodGet, "/me", next.AccessToken, nil).expect(t, http.StatusUnauthorized)
	env.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": next.RefreshToken}).expect(t, http.StatusUnauthorized)
}

var resetTokenPattern = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func TestAuthPasswordRecovery(t *testing.T) {
	env := newEnv(t)
	s := env.register("recover")

	// Unknown address: same answer, no email.
	env.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": "ghost@test.local"}).expect(t, http.StatusAccepted)
	if n := len(env.mail.Messages()); n != 0 {
		t.Fatalf("no email expected for unknown address, got %d", n)
	}

	env.do(http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": s.Email}).expect(t, http.StatusAccepted)
	msgs := env.mail.Messages()
	if len(msgs) != 1 || msgs[0].To != s.Email {
		t.Fatalf("expected one recovery email, got %+v", msgs)
	}
	m := resetTokenPattern.FindStringSubmatch(msgs[0].Body)
	if m == nil {
		t.Fatalf("no token in email body: %s", msgs[0].Body)
	}
	token := m[1]

	env.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"token": "bogus", "password": "brand-new-password"}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"token": token, "password": "short"}).expect(t, http.StatusBadRequest)
	env.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"token": token, "password": "brand-new-password"}).expect(t, http.StatusNoContent)
	// Single use.
	env.do(http.MethodPost, "/auth/password/reset", "", map[string]string{"token": token, "password": "another-password"}).expect(t, http.StatusBadRequest)

	// Old sessions are revoked, old password is dead, new password works.
	env.do(http.MethodGet, "/me", s.Token, nil).expect(t, http.StatusUnauthorized)
	env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": s.Email, "password": testPassword}).expect(t, http.StatusUnauthorized)
	env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": s.Email, "password": "brand-new-password"}).expect(t, http.StatusOK)
}

func TestAuthChangePassword(t *testing.T) {
	env := newEnv(t)
	s := env.register("change")
	other := env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": s.Email, "password": testPassword})
	var second authBody
	other.json(t, &second)

	env.do(http.MethodPost, "/auth/password/change", s.Token, map[string]string{"currentPassword": "nope-nope-nope", "newPassword": "brand-new-password"}).expect(t, http.StatusForbidden)
	env.do(http.MethodPost, "/auth/password/change", s.Token, map[string]string{"currentPassword": testPassword, "newPassword": "brand-new-password"}).expect(t, http.StatusNoContent)

	env.do(http.MethodGet, "/me", s.Token, nil).expect(t, http.StatusOK)                      // current session survives
	env.do(http.MethodGet, "/me", second.AccessToken, nil).expect(t, http.StatusUnauthorized) // other sessions are revoked
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	env := newEnv(t)
	a := env.register("del-a")
	b := env.register("del-b")
	env.onboard(a, profileSpec{Name: "Ada", Gender: "woman", Interested: []string{"man"}, Lat: parisLat, Lng: parisLng})
	env.onboard(b, profileSpec{Name: "Ben", Gender: "man", Interested: []string{"woman"}, Lat: parisLat, Lng: parisLng})
	env.swipe(a, b.ID, "like").expect(t, http.StatusOK)
	env.swipe(b, a.ID, "like").expect(t, http.StatusOK)

	files, _ := os.ReadDir(env.upload)
	if len(files) != 2 {
		t.Fatalf("expected 2 stored photos, got %d", len(files))
	}

	// Wrong password: nothing happens.
	env.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": "wrong-password"}).expect(t, http.StatusForbidden)
	env.do(http.MethodGet, "/me", a.Token, nil).expect(t, http.StatusOK)

	env.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": testPassword}).expect(t, http.StatusNoContent)

	env.do(http.MethodGet, "/me", a.Token, nil).expect(t, http.StatusUnauthorized)
	env.do(http.MethodPost, "/auth/login", "", map[string]string{"email": a.Email, "password": testPassword}).expect(t, http.StatusUnauthorized)

	for table, column := range map[string]string{
		"profiles": "user_id", "photos": "user_id", "preferences": "user_id", "user_interests": "user_id",
		"swipes": "swiper_id", "notifications": "user_id", "sessions": "user_id",
	} {
		var n int
		if err := env.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE "+column+" = $1", a.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s still holds %d rows of the deleted user", table, n)
		}
	}
	var matches int
	_ = env.pool.QueryRow(t.Context(), "SELECT count(*) FROM matches").Scan(&matches)
	if matches != 0 {
		t.Errorf("matches of the deleted user must be removed, got %d", matches)
	}
	files, _ = os.ReadDir(env.upload)
	if len(files) != 1 {
		t.Errorf("deleted user's photo file must be removed from storage, %d files left", len(files))
	}
	// The other user is unaffected.
	env.do(http.MethodGet, "/me", b.Token, nil).expect(t, http.StatusOK)
}
