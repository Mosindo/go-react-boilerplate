package main

import (
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.expect(e.do("GET", "/health", "", nil), 200)
}

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	creds := map[string]string{"email": "  Alice@Example.TEST ", "password": "correct-horse-1"}
	// Email is trimmed and normalised.
	r := e.expect(e.do("POST", "/auth/register", "", creds), 201)
	body := r.json(t)
	if body["user"].(map[string]any)["email"] != "alice@example.test" {
		t.Fatalf("email not normalised: %v", body["user"])
	}
	access := body["accessToken"].(string)
	refresh := body["refreshToken"].(string)

	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "alice@example.test", "password": "another-pass-1"}), 409)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "bob@example.test", "password": "short"}), 400)
	e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "not-an-email", "password": "correct-horse-1"}), 400)

	e.expect(e.do("GET", "/me", "", nil), 401)
	e.expect(e.do("GET", "/me", "garbage", nil), 401)
	e.expect(e.do("GET", "/me", access, nil), 200)

	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "alice@example.test", "password": "wrong-password"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "ghost@example.test", "password": "wrong-password"}), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": "ALICE@example.test", "password": "correct-horse-1"}), 200)

	// Refresh rotates: the old refresh token is dead.
	r = e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), 200)
	next := r.json(t)["refreshToken"].(string)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": refresh}), 401)

	// Logout revokes the session immediately, including the access token.
	nextAccess := r.json(t)["accessToken"].(string)
	e.expect(e.do("GET", "/me", nextAccess, nil), 200)
	e.expect(e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": next}), 204)
	e.expect(e.do("GET", "/me", nextAccess, nil), 401)
	e.expect(e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": next}), 401)
}

func TestJWTMustBeHS256AndSigned(t *testing.T) {
	e := newEnv(t)
	// alg=none token
	none := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOiJ4Iiwic2lkIjoieSIsImV4cCI6OTk5OTk5OTk5OX0."
	e.expect(e.do("GET", "/me", none, nil), 401)
}

func TestPasswordRecoveryAndChange(t *testing.T) {
	e := newEnv(t)
	u := e.onboard(userSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 30})

	// Unknown account answers identically and sends nothing.
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "nobody@example.test"}), 202)
	if e.mail.body("nobody@example.test") != "" {
		t.Fatal("mail sent to unknown account")
	}
	e.expect(e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	mail := e.mail.body(u.Email)
	idx := strings.Index(mail, "code de réinitialisation : ")
	if idx < 0 {
		t.Fatalf("no reset token in mail: %q", mail)
	}
	token := strings.Fields(mail[idx+len("code de réinitialisation : "):])[0]

	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": "bogus", "newPassword": "brand-new-pass-1"}), 400)
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": token, "newPassword": "brand-new-pass-1"}), 204)
	// Single use + old sessions revoked.
	e.expect(e.do("POST", "/auth/password/reset", "", map[string]string{"token": token, "newPassword": "brand-new-pass-2"}), 400)
	e.expect(e.do("GET", "/me", u.Token, nil), 401)
	e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "correct-horse-1"}), 401)
	r := e.expect(e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "brand-new-pass-1"}), 200)
	tok := r.json(t)["accessToken"].(string)

	e.expect(e.do("POST", "/me/password", tok, map[string]string{"currentPassword": "nope-nope-1", "newPassword": "third-password-1"}), 403)
	e.expect(e.do("POST", "/me/password", tok, map[string]string{"currentPassword": "brand-new-pass-1", "newPassword": "third-password-1"}), 204)
	e.expect(e.do("GET", "/me", tok, nil), 401)
}

func TestDeleteAccountRemovesEverything(t *testing.T) {
	e := newEnv(t)
	a := e.onboard(userSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 30, Interests: []string{"travel"}})
	b := e.onboard(userSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Age: 32})
	conv := e.match(a, b)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "salut"}), 201)

	e.expect(e.do("POST", "/me/delete", a.Token, map[string]string{"password": "wrong-password"}), 403)
	e.expect(e.do("POST", "/me/delete", a.Token, map[string]string{"password": "correct-horse-1"}), 204)
	e.expect(e.do("GET", "/me", a.Token, nil), 401)

	for _, table := range []string{"users", "profiles", "photos", "preferences", "user_interests", "swipes", "matches", "conversations", "conversation_participants", "messages", "notifications", "sessions"} {
		var n int
		q := "SELECT COUNT(*) FROM " + table
		switch table {
		case "users":
			q += " WHERE id = '" + a.ID + "'"
		case "swipes":
			q += " WHERE swiper_id = '" + a.ID + "' OR target_id = '" + a.ID + "'"
		case "matches", "conversations", "conversation_participants", "messages":
			// Everything tied to the deleted user's match must be gone.
		case "notifications", "sessions", "profiles", "photos", "preferences", "user_interests":
			q += " WHERE user_id = '" + a.ID + "'"
		}
		if err := e.pool.QueryRow(t.Context(), q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d rows left after account deletion", table, n)
		}
	}
	// The other party is untouched and no longer sees the match.
	r := e.expect(e.do("GET", "/conversations", b.Token, nil), 200)
	if strings.Contains(string(r.Body), a.ID) {
		t.Fatal("deleted user still visible")
	}
}

func TestAuthRateLimit(t *testing.T) {
	e := newEnvWith(t, 1, 3)
	bad := map[string]string{"email": "x@example.test", "password": "whatever-123"}
	for i := 0; i < 3; i++ {
		e.expect(e.do("POST", "/auth/login", "", bad), 401)
	}
	e.expect(e.do("POST", "/auth/login", "", bad), 429)
}
