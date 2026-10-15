package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlockRemovesMatchAndHidesBothWays(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Bea", Gender: "woman", Seeks: []string{"man"}, Lat: 38.0, Lng: 8.0})
	b := e.signup(profileSpec{Name: "Bram", Gender: "man", Seeks: []string{"woman"}, Lat: 38.0, Lng: 8.0})
	conv := e.match(a, b)
	e.mustStatus(e.send(a, conv, "hi"), http.StatusCreated, "message")

	e.mustStatus(e.do(http.MethodPost, "/blocks/"+b.ID, a.Token, nil), http.StatusNoContent, "block")
	e.mustStatus(e.do(http.MethodPost, "/blocks/"+b.ID, a.Token, nil), http.StatusNoContent, "block is idempotent")

	// The conversation vanishes for both people and nobody can write in it.
	for _, p := range []*person{a, b} {
		e.mustStatus(e.do(http.MethodGet, "/chats/"+conv+"/messages", p.Token, nil), http.StatusNotFound, "history after block")
		e.mustStatus(e.send(p, conv, "still there?"), http.StatusNotFound, "send after block")
	}
	// Neither side can see the other any more, even by direct request.
	e.mustStatus(e.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil), http.StatusNotFound, "blocked user views blocker")
	e.mustStatus(e.do(http.MethodGet, "/profiles/"+b.ID, a.Token, nil), http.StatusNotFound, "blocker views blocked")
	if contains(e.discoverIDs(a), b.ID) || contains(e.discoverIDs(b), a.ID) {
		t.Fatal("blocked people must disappear from discovery both ways")
	}
	e.mustStatus(e.swipe(b, a, "like"), http.StatusNotFound, "blocked user swipes")

	list := e.mustStatus(e.do(http.MethodGet, "/blocks", a.Token, nil), http.StatusOK, "blocks").json(t)["blocked"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["userId"] != b.ID || list[0].(map[string]any)["firstName"] != "Bram" {
		t.Fatalf("unexpected block list: %v", list)
	}
	if bl := e.mustStatus(e.do(http.MethodGet, "/blocks", b.Token, nil), http.StatusOK, "blocks B").json(t)["blocked"].([]any); len(bl) != 0 {
		t.Fatal("the blocked person must not learn who blocked them")
	}

	e.mustStatus(e.do(http.MethodDelete, "/blocks/"+b.ID, a.Token, nil), http.StatusNoContent, "unblock")
	if bl := e.mustStatus(e.do(http.MethodGet, "/blocks", a.Token, nil), http.StatusOK, "blocks").json(t)["blocked"].([]any); len(bl) != 0 {
		t.Fatal("unblock should empty the list")
	}

	e.mustStatus(e.do(http.MethodPost, "/blocks/"+a.ID, a.Token, nil), http.StatusBadRequest, "self block")
	e.mustStatus(e.do(http.MethodPost, "/blocks/00000000-0000-4000-8000-000000000000", a.Token, nil), http.StatusNotFound, "unknown user")
	e.mustStatus(e.do(http.MethodPost, "/blocks/zzz", a.Token, nil), http.StatusNotFound, "malformed id")
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Rey", Gender: "woman", Seeks: []string{"man"}, Lat: 37.0, Lng: 9.0})
	b := e.signup(profileSpec{Name: "Rob", Gender: "man", Seeks: []string{"woman"}, Lat: 37.0, Lng: 9.0})

	report := func(from *person, body map[string]any) response {
		return e.do(http.MethodPost, "/reports", from.Token, body)
	}
	e.mustStatus(report(a, map[string]any{"userId": b.ID, "reason": "nonsense"}), http.StatusBadRequest, "bad reason")
	e.mustStatus(report(a, map[string]any{"userId": a.ID, "reason": "spam"}), http.StatusBadRequest, "self report")
	e.mustStatus(report(a, map[string]any{"userId": "00000000-0000-4000-8000-000000000000", "reason": "spam"}), http.StatusNotFound, "unknown user")
	e.mustStatus(report(a, map[string]any{"userId": b.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}), http.StatusBadRequest, "long details")
	e.mustStatus(report(a, map[string]any{"userId": b.ID, "reason": "harassment", "details": "  rude messages  "}), http.StatusAccepted, "report")
	e.mustStatus(report(a, map[string]any{"userId": b.ID, "reason": "harassment"}), http.StatusAccepted, "duplicate report is accepted silently")

	var n int
	var details string
	if err := e.pool.QueryRow(contextBackground(), `SELECT COUNT(*), MAX(details) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n, &details); err != nil {
		t.Fatal(err)
	}
	if n != 1 || details != "rude messages" {
		t.Fatalf("expected one trimmed report, got %d %q", n, details)
	}

	// The reporter is not blocked automatically: reporting and blocking are separate choices.
	e.mustStatus(e.do(http.MethodGet, "/profiles/"+b.ID, a.Token, nil), http.StatusOK, "still visible")
}

func TestAccountDeletionErasesEverything(t *testing.T) {
	e := newEnv(t)
	a := e.signup(profileSpec{Name: "Dora", Gender: "woman", Seeks: []string{"man"}, Lat: 36.0, Lng: 7.0, Interests: []string{"cinema"}})
	b := e.signup(profileSpec{Name: "Dirk", Gender: "man", Seeks: []string{"woman"}, Lat: 36.0, Lng: 7.0})
	conv := e.match(a, b)
	e.mustStatus(e.send(a, conv, "bye soon"), http.StatusCreated, "message")
	e.mustStatus(e.do(http.MethodPost, "/reports", a.Token, map[string]any{"userId": b.ID, "reason": "spam"}), http.StatusAccepted, "report by Dora")
	e.mustStatus(e.do(http.MethodPost, "/reports", b.Token, map[string]any{"userId": a.ID, "reason": "other"}), http.StatusAccepted, "report about Dora")

	var keys []string
	rows, err := e.pool.Query(contextBackground(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) == 0 {
		t.Fatal("test setup should have a photo")
	}

	e.mustStatus(e.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": "not-the-password"}), http.StatusForbidden, "wrong password")
	e.mustStatus(e.do(http.MethodDelete, "/me", a.Token, map[string]string{"password": "Password123"}), http.StatusNoContent, "delete account")

	// The access token dies immediately; the account cannot sign in.
	e.mustStatus(e.do(http.MethodGet, "/me", a.Token, nil), http.StatusUnauthorized, "token after deletion")
	e.mustStatus(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}), http.StatusUnauthorized, "login after deletion")

	// Nothing remains in the database.
	for table, column := range map[string]string{
		"users": "id", "profiles": "user_id", "preferences": "user_id", "photos": "user_id", "user_interests": "user_id",
		"sessions": "user_id", "notifications": "user_id", "blocks": "blocker_id",
	} {
		var n int
		if err := e.pool.QueryRow(contextBackground(), "SELECT COUNT(*) FROM "+table+" WHERE "+column+" = $1", a.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s still holds %d rows for the deleted user (%v)", table, n, err)
		}
	}
	var leftover int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM likes WHERE from_user_id = $1 OR to_user_id = $1`, a.ID).Scan(&leftover)
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) + (SELECT COUNT(*) FROM messages WHERE conversation_id = $2) FROM matches WHERE user_a = $1 OR user_b = $1`, a.ID, conv).Scan(&leftover)
	if leftover != 0 {
		t.Errorf("likes, matches or messages remain: %d", leftover)
	}
	var aboutHer int
	_ = e.pool.QueryRow(contextBackground(), `SELECT COUNT(*) FROM reports WHERE reported_id = $1`, a.ID).Scan(&aboutHer)
	if aboutHer != 0 {
		t.Error("reports about the deleted user must be removed")
	}
	var byHer *string
	_ = e.pool.QueryRow(contextBackground(), `SELECT reporter_id::text FROM reports WHERE reported_id = $1`, b.ID).Scan(&byHer)
	if byHer != nil {
		t.Error("the reporter id of a deleted account must be cleared")
	}

	// Files are gone from disk, and the other person no longer sees her.
	for _, key := range keys {
		for _, name := range []string{key, key + "_t"} {
			if _, err := os.Stat(filepath.Join(e.uploadDir, name[:2], name)); !os.IsNotExist(err) {
				t.Errorf("photo file %s should be deleted (%v)", name, err)
			}
		}
	}
	if list := e.mustStatus(e.do(http.MethodGet, "/chats", b.Token, nil), http.StatusOK, "chats B").json(t)["conversations"].([]any); len(list) != 0 {
		t.Errorf("the other person must lose the conversation: %v", list)
	}
	e.mustStatus(e.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil), http.StatusNotFound, "deleted profile")
}

func TestChangePasswordOverHTTP(t *testing.T) {
	e := newEnv(t)
	p := e.register("changepw")
	e.mustStatus(e.do(http.MethodPut, "/me/password", p.Token, map[string]string{"currentPassword": "wrong-one", "newPassword": "NewPassword456"}), http.StatusForbidden, "wrong current")
	e.mustStatus(e.do(http.MethodPut, "/me/password", p.Token, map[string]string{"currentPassword": "Password123", "newPassword": "short"}), http.StatusBadRequest, "weak new")
	e.mustStatus(e.do(http.MethodPut, "/me/password", p.Token, map[string]string{"currentPassword": "Password123", "newPassword": "NewPassword456"}), http.StatusNoContent, "change")
	e.mustStatus(e.do(http.MethodGet, "/me", p.Token, nil), http.StatusUnauthorized, "sessions are revoked")
	e.mustStatus(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": p.Email, "password": "NewPassword456"}), http.StatusOK, "login with the new password")
}
