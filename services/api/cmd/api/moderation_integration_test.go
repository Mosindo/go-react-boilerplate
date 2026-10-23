package main

import (
	"net/http"
	"testing"
)

func TestBlockUnmatchesAndHidesBothWays(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)
	e.expect(send(e, a, matchID, "hi"), http.StatusCreated)

	e.expect(e.do(http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent)
	e.expect(e.do(http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent) // idempotent

	e.expect(send(e, b, matchID, "wait"), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/matches/"+matchID+"/messages", nil, a.Token), http.StatusNotFound)
	if matchSummary(e, a, matchID) != nil || matchSummary(e, b, matchID) != nil {
		t.Fatal("block must close the match for both")
	}
	if contains(e.discoverIDs(a), b.ID) || contains(e.discoverIDs(b), a.ID) {
		t.Fatal("blocked users must be invisible in both directions")
	}
	e.expect(e.do(http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/"+a.ID, nil, b.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/swipes", map[string]string{"targetUserId": a.ID, "action": "like"}, b.Token), http.StatusNotFound)

	list := e.expect(e.do(http.MethodGet, "/blocks", nil, a.Token), http.StatusOK).JSON(t)["blocks"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["userId"] != b.ID || list[0].(map[string]any)["firstName"] != "Berta" {
		t.Fatalf("unexpected block list %v", list)
	}
	if n := len(e.expect(e.do(http.MethodGet, "/blocks", nil, b.Token), http.StatusOK).JSON(t)["blocks"].([]any)); n != 0 {
		t.Fatal("the blocked user has no block of their own")
	}

	// Unblocking restores visibility only for profiles never swiped; the match stays closed.
	e.expect(e.do(http.MethodDelete, "/blocks/"+b.ID, nil, a.Token), http.StatusNoContent)
	e.expect(e.do(http.MethodDelete, "/blocks/"+b.ID, nil, a.Token), http.StatusNoContent)
	if matchSummary(e, a, matchID) != nil {
		t.Fatal("unblock must not restore the match")
	}
	e.expect(e.do(http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusOK)
}

func TestBlockValidation(t *testing.T) {
	e := newEnv(t)
	a, _ := e.pair()
	e.expect(e.do(http.MethodPost, "/blocks", map[string]string{"userId": a.ID}, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/blocks", map[string]string{"userId": "123e4567-e89b-12d3-a456-426614174000"}, a.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/blocks", map[string]string{"userId": "junk"}, a.Token), http.StatusNotFound)
}

func TestReportValidationAndStorage(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	matchID := e.match(a, b)

	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "boring"}, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": a.ID, "reason": "spam"}, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "spam", "details": string(make([]rune, 1001))}, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": "123e4567-e89b-12d3-a456-426614174000", "reason": "spam"}, a.Token), http.StatusNotFound)

	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "harassment", "details": "insultes", "matchId": matchID, "block": true}, a.Token), http.StatusCreated)
	var reason, status string
	var linked *string
	if err := e.pool.QueryRow(t.Context(), `SELECT reason, status, match_id::text FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&reason, &status, &linked); err != nil {
		t.Fatal(err)
	}
	if reason != "harassment" || status != "open" || linked == nil || *linked != matchID {
		t.Fatalf("report not stored properly: %s %s %v", reason, status, linked)
	}
	if matchSummary(e, a, matchID) != nil {
		t.Fatal("block flag must unmatch")
	}
	// Reporting twice refreshes the same row.
	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "spam"}, a.Token), http.StatusCreated)
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("one report per pair, got %d", n)
	}
	// A match id the reporter does not share is silently ignored (no IDOR on evidence links).
	other := e.register("rep-other")
	e.onboard(other, "Otto", profileOpts{gender: "man", interestedIn: []string{"woman"}})
	e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "spam", "matchId": matchID}, other.Token), http.StatusCreated)
	_ = e.pool.QueryRow(t.Context(), `SELECT match_id::text FROM reports WHERE reporter_id = $1`, other.ID).Scan(&linked)
	if linked != nil {
		t.Fatal("evidence link must require a shared match")
	}
}

func TestRepeatedReportsSuspendAccount(t *testing.T) {
	e := newEnv(t)
	target := e.register("villain")
	e.onboard(target, "Vlad", profileOpts{gender: "man", interestedIn: []string{"woman"}})
	viewer := e.register("viewer")
	e.onboard(viewer, "Vera", profileOpts{gender: "woman", interestedIn: []string{"man"}})
	if !contains(e.discoverIDs(viewer), target.ID) {
		t.Fatal("target should be discoverable before reports")
	}

	// The same reporter cannot suspend someone alone.
	for i := 0; i < 3; i++ {
		e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": target.ID, "reason": "spam"}, viewer.Token), http.StatusCreated)
	}
	e.expect(e.do(http.MethodGet, "/me", nil, target.Token), http.StatusOK)

	for i := 0; i < 4; i++ {
		r := e.register("reporter")
		e.expect(e.do(http.MethodPost, "/reports", map[string]any{"userId": target.ID, "reason": "fake_profile"}, r.Token), http.StatusCreated)
	}
	// Suspension revokes live sessions (401) and blocks new logins (403).
	e.expect(e.do(http.MethodGet, "/me", nil, target.Token), http.StatusUnauthorized)
	e.expect(e.do(http.MethodPost, "/auth/login", map[string]string{"email": target.Email, "password": testPassword}, ""), http.StatusForbidden)
	e.expect(e.do(http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": target.Refresh}, ""), http.StatusUnauthorized)
	if contains(e.discoverIDs(viewer), target.ID) {
		t.Fatal("suspended accounts must be hidden")
	}
}
