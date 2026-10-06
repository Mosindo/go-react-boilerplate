package main

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestBlockEndsMatchAndHidesBothWays(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)
	f.sendAs(e, f.a, "hello")

	e.expect(e.do(http.MethodPost, "/blocks", f.a.Token, map[string]string{"userId": f.b.ID}), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/blocks", f.a.Token, map[string]string{"userId": f.b.ID}), http.StatusNoContent) // idempotent

	// The conversation is gone for both, nobody can write.
	e.send(f.b, f.matchID, "why?", http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/matches/"+f.matchID+"/messages", f.a.Token, nil), http.StatusNotFound)

	// Neither can see the other's profile or photos, in discovery or directly.
	e.expect(e.do(http.MethodGet, "/profiles/"+f.b.ID, f.a.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/"+f.a.ID, f.b.Token, nil), http.StatusNotFound)
	photo := photoIDs(t, e.expect(e.do(http.MethodGet, "/photos", f.a.Token, nil), http.StatusOK))[0]
	e.expect(e.do(http.MethodGet, "/photos/"+photo+"/content", f.b.Token, nil), http.StatusNotFound)
	// Swiping a blocked user is refused.
	e.swipe(f.b, f.a, "like", http.StatusNotFound)

	// Another user pair is unaffected.
	if !contains(e.discoverIDs(f.outsider), f.a.ID) {
		t.Fatal("blocking must not affect third parties")
	}

	// Listing and unblocking. Unblocking restores visibility but not the match.
	list := e.expect(e.do(http.MethodGet, "/blocks", f.a.Token, nil), http.StatusOK).json(t)["blocks"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["userId"] != f.b.ID {
		t.Fatalf("unexpected block list %v", list)
	}
	e.expect(e.do(http.MethodDelete, "/blocks/"+f.b.ID, f.a.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, "/matches/"+f.matchID+"/messages", f.a.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/"+f.b.ID, f.a.Token, nil), http.StatusOK)

	// Invalid targets.
	e.expect(e.do(http.MethodPost, "/blocks", f.a.Token, map[string]string{"userId": f.a.ID}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/blocks", f.a.Token, map[string]string{"userId": "00000000-0000-0000-0000-000000000000"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/blocks", f.a.Token, map[string]string{"userId": "x"}), http.StatusBadRequest)
}

func TestReportingAndAutoHide(t *testing.T) {
	e := newEnv(t)
	target := e.newPerson(persona{Name: "Target", Gender: "woman", InterestedIn: []string{"man"}})
	var reporters []user
	for i := 0; i < 4; i++ {
		reporters = append(reporters, e.newPerson(persona{Name: "Rep", Gender: "man", InterestedIn: []string{"woman"}}))
	}
	viewer := reporters[3]

	report := func(from user, reason string, want int) {
		e.expect(e.do(http.MethodPost, "/reports", from.Token, map[string]any{"userId": target.ID, "reason": reason, "details": "bad behaviour"}), want)
	}
	report(reporters[0], "nonsense", http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", reporters[0].Token, map[string]any{"userId": reporters[0].ID, "reason": "spam"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", reporters[0].Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"}), http.StatusNotFound)

	report(reporters[0], "harassment", http.StatusAccepted)
	report(reporters[0], "spam", http.StatusAccepted) // duplicate by same reporter: no inflation
	if n := countRows(t, e.pool, `SELECT count(*) FROM reports WHERE reported_id=$1`, target.ID); n != 1 {
		t.Fatalf("a reporter may have one open report per user, got %d", n)
	}
	if !contains(e.discoverIDs(viewer), target.ID) {
		t.Fatal("one report must not hide a profile")
	}
	report(reporters[1], "fake_profile", http.StatusAccepted)
	report(reporters[2], "inappropriate_content", http.StatusAccepted)
	if contains(e.discoverIDs(viewer), target.ID) {
		t.Fatal("3 distinct reporters must pull the profile from discovery pending review")
	}

	// Report + block in one step.
	e.expect(e.do(http.MethodPost, "/reports", viewer.Token, map[string]any{"userId": reporters[0].ID, "reason": "spam", "alsoBlock": true}), http.StatusAccepted)
	if n := countRows(t, e.pool, `SELECT count(*) FROM blocks WHERE blocker_id=$1`, viewer.ID); n != 1 {
		t.Fatalf("alsoBlock must block, got %d", n)
	}
}

func TestAccountDeletionErasesEverything(t *testing.T) {
	e := newEnv(t)
	f := newChatFixture(t, e)
	f.sendAs(e, f.a, "hello")
	f.sendAs(e, f.b, "hi")
	e.expect(e.do(http.MethodPost, "/reports", f.b.Token, map[string]any{"userId": f.a.ID, "reason": "spam"}), http.StatusAccepted)

	var key string
	if err := e.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE user_id=$1 LIMIT 1`, f.a.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(e.dir, filepath.FromSlash(key))
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("photo file should exist before deletion: %v", err)
	}

	// Requires the password.
	e.expect(e.do(http.MethodDelete, "/account", f.a.Token, map[string]string{"password": "WrongPassword1"}), http.StatusForbidden)
	e.expect(e.do(http.MethodDelete, "/account", f.a.Token, map[string]string{}), http.StatusBadRequest)
	e.expect(e.do(http.MethodDelete, "/account", "", map[string]string{"password": "Password123!"}), http.StatusUnauthorized)
	e.expect(e.do(http.MethodDelete, "/account", f.a.Token, map[string]string{"password": "Password123!"}), http.StatusNoContent)

	// The token dies immediately, credentials no longer work.
	e.expect(e.do(http.MethodGet, "/me", f.a.Token, nil), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", "", map[string]string{"email": f.a.Email, "password": "Password123!"}), http.StatusUnauthorized)

	// Nothing about the user remains, in any table or on disk.
	for name, query := range map[string]string{
		"users":         `SELECT count(*) FROM users WHERE id=$1`,
		"profiles":      `SELECT count(*) FROM profiles WHERE user_id=$1`,
		"preferences":   `SELECT count(*) FROM preferences WHERE user_id=$1`,
		"photos":        `SELECT count(*) FROM photos WHERE user_id=$1`,
		"swipes":        `SELECT count(*) FROM swipes WHERE from_user_id=$1 OR to_user_id=$1`,
		"matches":       `SELECT count(*) FROM matches WHERE user_a=$1 OR user_b=$1`,
		"messages":      `SELECT count(*) FROM messages WHERE sender_user_id=$1 OR recipient_user_id=$1`,
		"participants":  `SELECT count(*) FROM conversation_participants WHERE user_id=$1`,
		"notifications": `SELECT count(*) FROM notifications WHERE user_id=$1`,
		"sessions":      `SELECT count(*) FROM sessions WHERE user_id=$1`,
		"reports":       `SELECT count(*) FROM reports WHERE reported_id=$1 OR reporter_id=$1`,
	} {
		if n := countRows(t, e.pool, query, f.a.ID); n != 0 {
			t.Errorf("%s: %d rows survive account deletion", name, n)
		}
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("photo files must be deleted from storage")
	}
	if n := countRows(t, e.pool, `SELECT count(*) FROM conversations`); n != 0 {
		t.Fatalf("the shared conversation must be erased too, got %d", n)
	}

	// The other person is intact and simply no longer has that match.
	list := e.expect(e.do(http.MethodGet, "/matches", f.b.Token, nil), http.StatusOK).json(t)
	if len(list["matches"].([]any)) != 0 {
		t.Fatalf("the peer must not see a ghost match: %v", list)
	}
	e.expect(e.do(http.MethodGet, "/me", f.b.Token, nil), http.StatusOK)
}
