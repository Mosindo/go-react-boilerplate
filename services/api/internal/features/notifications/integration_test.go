package notifications

import (
	"context"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pub struct {
	mu     sync.Mutex
	events []realtime.Event
	users  []string
}

func (p *pub) Publish(uid string, e realtime.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.users = append(p.users, uid)
	p.events = append(p.events, e)
}

func setup(t *testing.T) (*pgxpool.Pool, *gin.Engine, notify.Notifier, *pub) {
	pool := testutil.Pool(t)
	p := &pub{}
	d := testutil.Deps(t, nil, p)
	r := testutil.Router(func(r gin.IRouter) { RegisterRoutes(r, pool, d) })
	return pool, r, NewNotifier(pool, p), p
}

func list(t *testing.T, r *gin.Engine, token, query string) ListResult {
	t.Helper()
	res := testutil.Do(t, r, "GET", "/notifications"+query, nil, token)
	if res.Status != 200 {
		t.Fatalf("list %d %s", res.Status, res.Body)
	}
	var out ListResult
	res.JSON(t, &out)
	return out
}

func TestNotifyListReadFlow(t *testing.T) {
	pool, r, n, p := setup(t)
	u, tok := testutil.User(t, pool, "notif", "1990-01-01")
	other, otherTok := testutil.User(t, pool, "notif", "1990-01-01")

	if got := list(t, r, tok, ""); len(got.Notifications) != 0 || got.UnreadCount != 0 || got.Notifications == nil {
		t.Fatalf("empty list: %+v", got)
	}
	for i := 0; i < 3; i++ {
		if err := n.Notify(context.Background(), notify.New{UserID: u, Type: "match", Title: "New match", Body: "b",
			Data: map[string]any{"conversationId": "c" + string(rune('0'+i)), "userId": other}}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(p.events) != 3 || p.events[0].Type != "notification.new" || p.users[0] != u {
		t.Fatalf("realtime: %+v", p.events)
	}
	got := list(t, r, tok, "")
	if len(got.Notifications) != 3 || got.UnreadCount != 3 {
		t.Fatalf("list: %+v", got)
	}
	first := got.Notifications[0]
	if first.Data["conversationId"] != "c2" || first.IsRead || first.Type != "match" {
		t.Errorf("newest first expected: %+v", first)
	}

	// pagination
	page1 := list(t, r, tok, "?limit=2")
	if len(page1.Notifications) != 2 || page1.UnreadCount != 3 {
		t.Fatalf("page1: %+v", page1)
	}
	before := page1.Notifications[1].CreatedAt.Format(time.RFC3339Nano)
	page2 := list(t, r, tok, "?limit=2&before="+before)
	if len(page2.Notifications) != 1 || page2.Notifications[0].ID != got.Notifications[2].ID {
		t.Fatalf("page2: %+v", page2)
	}
	for _, q := range []string{"?limit=0", "?limit=101", "?limit=x", "?before=yesterday"} {
		if res := testutil.Do(t, r, "GET", "/notifications"+q, nil, tok); res.Status != 400 {
			t.Errorf("%s status %d", q, res.Status)
		}
	}

	// ownership isolation
	if res := testutil.Do(t, r, "POST", "/notifications/"+first.ID+"/read", nil, otherTok); res.Status != 404 {
		t.Errorf("foreign read %d", res.Status)
	}
	if list(t, r, tok, "").UnreadCount != 3 {
		t.Error("foreign read must not change state")
	}
	if got := list(t, r, otherTok, ""); len(got.Notifications) != 0 {
		t.Error("other user sees my notifications")
	}
	if res := testutil.Do(t, r, "POST", "/notifications/not-a-uuid/read", nil, tok); res.Status != 404 {
		t.Errorf("bad id %d", res.Status)
	}
	if res := testutil.Do(t, r, "POST", "/notifications/"+first.ID+"/read", nil, tok); res.Status != 204 {
		t.Errorf("read %d", res.Status)
	}
	got = list(t, r, tok, "")
	if got.UnreadCount != 2 || !got.Notifications[0].IsRead {
		t.Errorf("after read: %+v", got)
	}
	var readAt *time.Time
	_ = pool.QueryRow(context.Background(), `SELECT read_at FROM notifications WHERE id=$1`, first.ID).Scan(&readAt)
	if readAt == nil {
		t.Error("read_at must be set consistently with is_read")
	}
	if res := testutil.Do(t, r, "POST", "/notifications/read-all", nil, otherTok); res.Status != 204 {
		t.Errorf("read-all other %d", res.Status)
	}
	if list(t, r, tok, "").UnreadCount != 2 {
		t.Error("read-all must only affect the caller")
	}
	if res := testutil.Do(t, r, "POST", "/notifications/read-all", nil, tok); res.Status != 204 {
		t.Errorf("read-all %d", res.Status)
	}
	if got := list(t, r, tok, ""); got.UnreadCount != 0 {
		t.Errorf("unread after read-all: %d", got.UnreadCount)
	}
	// clients cannot create notifications
	if res := testutil.Do(t, r, "POST", "/notifications", map[string]string{"title": "x"}, tok); res.Status != 404 && res.Status != 405 {
		t.Errorf("create endpoint must not exist: %d", res.Status)
	}
	if res := testutil.Do(t, r, "GET", "/notifications", nil, ""); res.Status != 401 {
		t.Errorf("unauth %d", res.Status)
	}
}

func TestMessageCoalescing(t *testing.T) {
	pool, r, n, _ := setup(t)
	u, tok := testutil.User(t, pool, "coal", "1990-01-01")
	msg := func(conv, body string) {
		if err := n.Notify(context.Background(), notify.New{UserID: u, Type: "message", Title: "New message", Body: body,
			Data: map[string]any{"conversationId": conv}}); err != nil {
			t.Error(err)
		}
	}
	msg("conv-1", "one")
	msg("conv-2", "other conversation")
	msg("conv-1", "two")
	got := list(t, r, tok, "")
	if len(got.Notifications) != 2 || got.UnreadCount != 2 {
		t.Fatalf("expected coalescing per conversation: %+v", got)
	}
	if got.Notifications[0].Body != "two" || got.Notifications[0].Data["conversationId"] != "conv-1" {
		t.Errorf("updated notification should be newest: %+v", got.Notifications[0])
	}

	// Once read, a new message creates a fresh notification.
	testutil.Do(t, r, "POST", "/notifications/"+got.Notifications[0].ID+"/read", nil, tok)
	msg("conv-1", "three")
	if got := list(t, r, tok, ""); len(got.Notifications) != 3 || got.UnreadCount != 2 {
		t.Fatalf("after read: %+v", got)
	}

	// Concurrent messages in one conversation still yield a single unread row.
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); msg("conv-race", "x") }()
	}
	wg.Wait()
	var cnt int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND data->>'conversationId'='conv-race'`, u).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("race produced %d rows", cnt)
	}

	// Match notifications are never coalesced.
	for i := 0; i < 2; i++ {
		_ = n.Notify(context.Background(), notify.New{UserID: u, Type: "match", Title: "m", Body: "m", Data: map[string]any{"conversationId": "conv-1"}})
	}
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM notifications WHERE user_id=$1 AND type='match'`, u).Scan(&cnt)
	if cnt != 2 {
		t.Errorf("match rows %d", cnt)
	}
	if err := n.Notify(context.Background(), notify.New{}); err == nil {
		t.Error("empty notification must be rejected")
	}
}

func TestParseLimitAndUUID(t *testing.T) {
	for raw, want := range map[string]int{"": 30, "1": 1, "100": 100} {
		if got, err := parseLimit(raw); err != nil || got != want {
			t.Errorf("parseLimit(%q)=%d,%v", raw, got, err)
		}
	}
	for _, raw := range []string{"0", "101", "-5", "1e2", "99999"} {
		if _, err := parseLimit(raw); err == nil {
			t.Errorf("parseLimit(%q) should fail", raw)
		}
	}
	if !isUUID("123e4567-e89b-12d3-a456-426614174000") || isUUID("123e4567e89b12d3a456426614174000") || isUUID("zzze4567-e89b-12d3-a456-426614174000") {
		t.Error("isUUID")
	}
}
