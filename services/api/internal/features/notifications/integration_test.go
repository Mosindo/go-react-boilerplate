package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"example.com/api/internal/platform/push"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/testutil"
)

func TestNotifyDeliversPushAndForgetsDeadTokens(t *testing.T) {
	pool := testutil.DB(t)
	ctx := context.Background()
	email := testutil.Email(t, pool, "push")
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id`, email).Scan(&userID); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var received []push.Message
	expo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var batch []push.Message
		_ = json.NewDecoder(r.Body).Decode(&batch)
		mu.Lock()
		received = append(received, batch...)
		mu.Unlock()
		tickets := make([]map[string]any, len(batch))
		for i, m := range batch {
			if m.To == "ExponentPushToken[deaddeaddead]" {
				tickets[i] = map[string]any{"status": "error", "details": map[string]string{"error": "DeviceNotRegistered"}}
			} else {
				tickets[i] = map[string]any{"status": "ok"}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": tickets})
	}))
	defer expo.Close()

	svc := NewService(NewPGRepository(pool), realtime.NopPublisher{}, push.NewExpoSender(expo.URL, ""))
	if err := svc.RegisterPushToken(ctx, userID, "not-a-token", "ios"); err == nil {
		t.Fatal("invalid token must be rejected")
	}
	for _, token := range []string{"ExponentPushToken[alivealivealive]", "ExponentPushToken[deaddeaddead]"} {
		if err := svc.RegisterPushToken(ctx, userID, token, "android"); err != nil {
			t.Fatal(err)
		}
	}

	svc.Notify(ctx, userID, TypeMatch, "Nouveau match", "Vous avez un match avec Sam !", map[string]string{"conversationId": "c1", "userId": "u2"})
	svc.WaitForPushes()

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 || received[0].Title != "Nouveau match" || received[0].Data["conversationId"] != "c1" {
		t.Fatalf("unexpected pushes: %+v", received)
	}
	if _, leaked := received[0].Data["userId"]; leaked {
		t.Fatal("push payload must only carry navigation identifiers")
	}
	tokens, err := svc.repo.PushTokens(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0] != "ExponentPushToken[alivealivealive]" {
		t.Fatalf("dead token must be forgotten, got %v", tokens)
	}
}
