package main

import (
	"context"
	"net/http"
	"testing"
)

func TestBlockDissolvesMatchAndHidesUsers(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)
	conv := e.forceMatch(a, b)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "hey"}, a.Token), http.StatusCreated)

	e.expect(e.do("POST", "/blocks", map[string]string{"userId": a.ID}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": "11111111-1111-1111-1111-111111111111"}, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": a.ID}, b.Token), http.StatusNoContent)
	e.expect(e.do("POST", "/blocks", map[string]string{"userId": a.ID}, b.Token), http.StatusNoContent) // idempotent

	// Conversation, messages and visibility vanish for both sides.
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", nil, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "still there?"}, a.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+a.ID, nil, b.Token), http.StatusNotFound)
	if containsProfile(e.discover(a), b.ID) || containsProfile(e.discover(b), a.ID) {
		t.Fatal("blocked users must not see each other in discovery")
	}
	e.expect(e.do("POST", "/swipes", map[string]string{"targetUserId": b.ID, "action": "like"}, a.Token), http.StatusNotFound)

	var list struct {
		Blocked []struct {
			UserID      string `json:"userId"`
			DisplayName string `json:"displayName"`
		} `json:"blocked"`
	}
	e.expect(e.do("GET", "/blocks", nil, b.Token), http.StatusOK).decode(t, &list)
	if len(list.Blocked) != 1 || list.Blocked[0].UserID != a.ID || list.Blocked[0].DisplayName != "Alice" {
		t.Fatalf("unexpected block list %+v", list)
	}

	e.expect(e.do("DELETE", "/blocks/"+a.ID, nil, b.Token), http.StatusNoContent)
	e.expect(e.do("GET", "/profiles/"+a.ID, nil, b.Token), http.StatusOK)
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)

	e.expect(e.do("POST", "/reports", map[string]any{"userId": b.ID, "reason": "dislike"}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, a.Token), http.StatusBadRequest)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": "11111111-1111-1111-1111-111111111111", "reason": "spam"}, a.Token), http.StatusNotFound)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": b.ID, "reason": "harassment", "details": "insultes", "block": true}, a.Token), http.StatusCreated)

	var count int
	if err := e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'`, a.ID, b.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("expected one open report, got %d err=%v", count, err)
	}
	if containsProfile(e.discover(a), b.ID) {
		t.Fatal("report with block must hide the user")
	}
}

func TestDeleteAccountRemovesPersonalData(t *testing.T) {
	e := newEnv(t)
	at := uniqueGeo()
	alice, bob := pair(at)
	a := e.onboard("alice", alice)
	b := e.onboard("bob", bob)
	conv := e.forceMatch(a, b)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", map[string]string{"body": "avant"}, a.Token), http.StatusCreated)
	e.expect(e.do("POST", "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, b.Token), http.StatusCreated)

	var key, photoURL string
	if err := e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	var photos struct {
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
	}
	e.expect(e.do("GET", "/photos", nil, a.Token), http.StatusOK).decode(t, &photos)
	photoURL = photos.Photos[0].URL

	e.expect(e.do("DELETE", "/me", map[string]string{"password": "wrong-password-1"}, a.Token), http.StatusForbidden)
	e.expect(e.do("DELETE", "/me", map[string]string{"password": testPassword}, a.Token), http.StatusNoContent)

	e.expect(e.do("POST", "/auth/login", map[string]string{"email": a.Email, "password": testPassword}, ""), http.StatusUnauthorized)
	e.expect(e.do("GET", photoURL, nil, ""), http.StatusNotFound)
	if _, err := e.store.Open(key); err == nil {
		t.Fatal("photo file must be deleted from storage")
	}
	for table, col := range map[string]string{"profiles": "user_id", "preferences": "user_id", "photos": "user_id", "swipes": "user_id", "messages": "sender_id", "notifications": "user_id", "sessions": "user_id"} {
		var n int
		if err := e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM `+table+` WHERE `+col+` = $1`, a.ID).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s still holds %d rows of the deleted user (err=%v)", table, n, err)
		}
	}
	var convs, matches int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations WHERE id = $1`, conv).Scan(&convs)
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = $1 OR user_b = $1`, a.ID).Scan(&matches)
	if convs != 0 || matches != 0 {
		t.Fatalf("conversation and matches must be gone: convs=%d matches=%d", convs, matches)
	}
	// The report survives for moderation but no longer points at a person.
	var kept int
	if err := e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id IS NULL`, b.ID).Scan(&kept); err != nil || kept != 1 {
		t.Fatalf("report should be kept with the reported user cleared, got %d err=%v", kept, err)
	}
	if containsProfile(e.discover(b), a.ID) {
		t.Fatal("deleted users must disappear from discovery")
	}
}
