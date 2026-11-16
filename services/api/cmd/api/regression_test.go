package main

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Parallel wrong guesses must never exceed the 5-attempt budget.
func TestResetAttemptsCappedUnderConcurrency(t *testing.T) {
	env := newTestEnv(t)
	u := env.register("reset-race")
	env.expect(env.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}), 202)
	code := env.mailer.code(t, u.Email)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "NewPassword9"})
		}()
	}
	wg.Wait()
	if n := env.countRows(`SELECT attempts FROM password_resets WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, u.ID); n > 5 {
		t.Fatalf("attempts must be capped at 5, got %d", n)
	}
	env.expect(env.do("POST", "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "NewPassword9"}), 400)
}

// A block racing a mutual like must never leave a visible match.
func TestBlockRacingLikeLeavesNoMatch(t *testing.T) {
	env := newTestEnv(t)
	for round := 0; round < 8; round++ {
		a := env.onboard("blkrace-a", profileSpec{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}})
		b := env.onboard("blkrace-b", profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}})
		env.swipe(a, b.ID, "like")
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			env.do("POST", "/swipes", b.Token, map[string]string{"targetUserId": a.ID, "action": "like"})
		}()
		go func() {
			defer wg.Done()
			env.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID})
		}()
		wg.Wait()
		if n := env.countRows(`SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid) AND unmatched_at IS NULL`, a.ID, b.ID); n != 0 {
			t.Fatalf("round %d: active match survived a block", round)
		}
		for _, u := range []testUser{a, b} {
			var ml struct {
				Matches []any `json:"matches"`
			}
			env.expect(env.do("GET", "/matches", u.Token, nil), 200).decode(t, &ml)
			if len(ml.Matches) != 0 {
				t.Fatalf("round %d: blocked pair still listed as match", round)
			}
		}
	}
}

// A multipart Content-Type must not lift the body size limit on JSON routes.
func TestMultipartHeaderDoesNotBypassBodyLimit(t *testing.T) {
	env := newTestEnv(t)
	big := bytes.Repeat([]byte("a"), 20<<20)
	body := append([]byte(`{"refreshToken":"`), big...)
	body = append(body, []byte(`"}`)...)
	req := httptest.NewRequest("POST", "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	if w.Code < 400 {
		t.Fatalf("oversized body must be rejected, got %d", w.Code)
	}
}

// Changing the password closes realtime sockets of every session.
func TestPasswordChangeClosesRealtime(t *testing.T) {
	env := newTestEnv(t)
	srv := httptest.NewServer(env.router)
	defer srv.Close()
	u := env.register("rt-pw")
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	env.expect(env.do("POST", "/realtime/ticket", u.Token, nil), 200).decode(t, &ticket)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/realtime?ticket="+ticket.Ticket, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	deadline := time.Now().Add(2 * time.Second)
	for env.hub.ConnectionCount(u.ID) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	env.expect(env.do("POST", "/me/password", u.Token, map[string]string{"currentPassword": testPassword, "newPassword": "Changed123"}), 204)
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("socket should be closed after a password change")
	}
	if env.hub.ConnectionCount(u.ID) != 0 {
		t.Fatal("hub still holds the connection")
	}
}
