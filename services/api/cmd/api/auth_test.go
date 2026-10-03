package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	e := newEnv(t, false)
	e.expect(e.do(http.MethodGet, "/health", "", nil), http.StatusOK)
}

func TestAuthRegisterLoginRefreshMe(t *testing.T) {
	e := newEnv(t, false)
	u := e.register("auth")

	// Duplicate registration, with a differently-cased email, is rejected.
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": "Password123"}), http.StatusConflict)
	// Password policy.
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "short@example.test", "password": "short"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "long@example.test", "password": strings.Repeat("a", 73)}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/register", "", map[string]string{"email": "not-an-email", "password": "Password123"}), http.StatusBadRequest)

	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "wrong-password"}), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "nobody@example.test", "password": "Password123"}), http.StatusUnauthorized)

	var login authBody
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"}), http.StatusOK).json(t, &login)

	var me struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	e.expect(e.do(http.MethodGet, "/me", login.AccessToken, nil), http.StatusOK).json(t, &me)
	if me.ID != u.ID || me.Email != u.Email {
		t.Fatalf("unexpected /me: %+v", me)
	}
	e.expect(e.do(http.MethodGet, "/me", "", nil), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, "/me", "garbage.token.value", nil), http.StatusUnauthorized)

	var refreshed authBody
	e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), http.StatusOK).json(t, &refreshed)
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": login.RefreshToken}), http.StatusUnauthorized)

	e.expect(e.do(http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": refreshed.RefreshToken}), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": refreshed.RefreshToken}), http.StatusUnauthorized)
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t, false)
	u := e.register("reset")

	// Unknown emails get the same answer and no mail.
	e.expect(e.do(http.MethodPost, "/auth/password-reset/request", "", map[string]string{"email": "ghost@example.test"}), http.StatusNoContent)
	if e.mailer.body("ghost@example.test") != "" {
		t.Fatal("no email must be sent for unknown accounts")
	}

	e.expect(e.do(http.MethodPost, "/auth/password-reset/request", "", map[string]string{"email": u.Email}), http.StatusNoContent)
	body := e.mailer.body(u.Email)
	if body == "" {
		t.Fatal("expected a recovery email")
	}
	var token string
	for _, line := range strings.Split(body, "\n") {
		if t := strings.TrimSpace(line); len(t) == 43 && !strings.Contains(t, " ") {
			token = t
		}
	}
	if token == "" {
		t.Fatalf("token not found in email: %q", body)
	}

	e.expect(e.do(http.MethodPost, "/auth/password-reset/confirm", "", map[string]string{"token": "bogus", "newPassword": "NewPassword456"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/password-reset/confirm", "", map[string]string{"token": token, "newPassword": "short"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/auth/password-reset/confirm", "", map[string]string{"token": token, "newPassword": "NewPassword456"}), http.StatusNoContent)
	// Single use.
	e.expect(e.do(http.MethodPost, "/auth/password-reset/confirm", "", map[string]string{"token": token, "newPassword": "Another789"}), http.StatusBadRequest)

	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "Password123"}), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword456"}), http.StatusOK)
	// Existing sessions are revoked.
	e.expect(e.do(http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}), http.StatusUnauthorized)
}

func TestAccountDeletionRemovesEverything(t *testing.T) {
	e := newEnv(t, false)
	lat, lng := scenarioOrigin()
	a := e.completeUser("del-a", profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	b := e.completeUser("del-b", profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	e.expect(e.do(http.MethodPost, "/swipes", a.Token, map[string]string{"targetId": b.ID, "action": "like"}), http.StatusOK)
	e.expect(e.do(http.MethodPost, "/swipes", b.Token, map[string]string{"targetId": a.ID, "action": "like"}), http.StatusOK)
	e.expect(e.do(http.MethodPost, "/reports", b.Token, map[string]string{"userId": a.ID, "reason": "spam"}), http.StatusCreated)

	e.expect(e.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": "wrong-password"}), http.StatusForbidden)
	e.expect(e.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": "Password123"}), http.StatusNoContent)

	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil), http.StatusNotFound)

	ctx := context.Background()
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "swiper_id", "sessions": "user_id", "preferences": "user_id", "user_interests": "user_id"} {
		var n int
		if err := e.pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s still has %d rows for the deleted user", table, n)
		}
	}
	var matches int
	_ = e.pool.QueryRow(ctx, "SELECT count(*) FROM matches WHERE user_a = $1 OR user_b = $1", a.ID).Scan(&matches)
	if matches != 0 {
		t.Fatal("matches must be removed with the account")
	}
	// The report survives, anonymised.
	var reports int
	_ = e.pool.QueryRow(ctx, "SELECT count(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL", b.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected the report to be kept with a NULL reported_id, got %d", reports)
	}
	// Stored photo files are gone: only the other user's file remains.
	var files int
	_ = e.pool.QueryRow(ctx, "SELECT count(*) FROM photos WHERE user_id = $1", b.ID).Scan(&files)
	if files != 1 {
		t.Fatal("the other user's photo must be untouched")
	}
}

func TestRateLimitOnAuthEndpoints(t *testing.T) {
	e := newEnv(t, true)
	limited := false
	for i := 0; i < 30; i++ {
		r := e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": "nobody@example.test", "password": "Password123"})
		if r.Status == http.StatusTooManyRequests {
			limited = true
			if r.Header.Get("Retry-After") == "" {
				t.Fatal("missing Retry-After header")
			}
			break
		}
	}
	if !limited {
		t.Fatal("expected brute-force protection to kick in")
	}
}
