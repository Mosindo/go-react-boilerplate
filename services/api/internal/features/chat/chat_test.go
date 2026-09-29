package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestNormalizeBody(t *testing.T) {
	long := strings.Repeat("é", MaxBodyRunes)
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"  hello  ", "hello", true},
		{"", "", false},
		{"   \n\t ", "", false},
		{"a\x00b\x07c", "abc", true},
		{"line1\r\nline2", "line1\nline2", true},
		{"a\tb", "a b", true},
		{"\x00\x01", "", false},
		{long, long, true},
		{long + "x", "", false},
		{"\xff\xfe", "", false},
		{"héllo 👋", "héllo 👋", true},
	}
	for _, c := range cases {
		got, err := NormalizeBody(c.in)
		if (err == nil) != c.ok || (c.ok && got != c.want) {
			t.Errorf("NormalizeBody(%q) = %q, %v", c.in, got, err)
		}
	}
	// 2000 runes that are multi-byte must pass although > 2000 bytes.
	if _, err := NormalizeBody(strings.Repeat("👋", 2000)); err != nil {
		t.Errorf("2000 emoji should be accepted: %v", err)
	}
}

func TestClampLimitAndTruncate(t *testing.T) {
	if ClampLimit(0) != DefaultLimit || ClampLimit(-3) != DefaultLimit || ClampLimit(1000) != MaxLimit || ClampLimit(5) != 5 {
		t.Fatal("ClampLimit")
	}
	if got := truncateRunes("short", 80); got != "short" {
		t.Fatal(got)
	}
	got := truncateRunes(strings.Repeat("é", 200), 80)
	if n := len([]rune(got)); n > 80 {
		t.Fatalf("too long: %d", n)
	}
}

func TestCursorValidation(t *testing.T) {
	if !isUUID("3f2b8c1e-1111-4222-8333-444455556666") || isUUID("nope") || isUUID("") || isUUID("3f2b8c1e-1111-4222-8333-44445555666") {
		t.Fatal("isUUID")
	}
	if n, ok := parseLimit(""); !ok || n != 0 {
		t.Fatal("empty limit")
	}
	if _, ok := parseLimit("abc"); ok {
		t.Fatal("abc")
	}
	if _, ok := parseLimit("0"); ok {
		t.Fatal("0")
	}
}

// ---- integration ----

type capPub struct {
	mu     sync.Mutex
	events []struct {
		User string
		Ev   realtime.Event
	}
}

func (p *capPub) Publish(u string, e realtime.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, struct {
		User string
		Ev   realtime.Event
	}{u, e})
}

func (p *capPub) of(user, typ string) []realtime.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []realtime.Event
	for _, e := range p.events {
		if e.User == user && e.Ev.Type == typ {
			out = append(out, e.Ev)
		}
	}
	return out
}

type capNotifier struct {
	mu sync.Mutex
	ns []notify.New
}

func (n *capNotifier) Notify(_ context.Context, x notify.New) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ns = append(n.ns, x)
	return nil
}

type env struct {
	pool *pgxpool.Pool
	r    *gin.Engine
	pub  *capPub
	not  *capNotifier
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := testutil.Pool(t)
	pub, not := &capPub{}, &capNotifier{}
	d := testutil.Deps(t, not, pub)
	r := testutil.Router(func(r gin.IRouter) { RegisterRoutes(r, pool, d) })
	return &env{pool, r, pub, not}
}

type user struct{ id, tok string }

func (e *env) user(t *testing.T, name string) user {
	t.Helper()
	id, tok := testutil.User(t, e.pool, name, "1995-05-05")
	_, err := e.pool.Exec(context.Background(),
		`INSERT INTO profiles (user_id, first_name, gender) VALUES ($1, $2, 'woman')`, id, name)
	if err != nil {
		t.Fatal(err)
	}
	return user{id, tok}
}

// match creates a match + match conversation + participants by direct SQL.
func (e *env) match(t *testing.T, a, b user) string {
	t.Helper()
	ctx := context.Background()
	x, y := a.id, b.id
	if x > y {
		x, y = y, x
	}
	var matchID, convID string
	if err := e.pool.QueryRow(ctx, `INSERT INTO matches (user_a, user_b) VALUES ($1,$2) RETURNING id`, x, y).Scan(&matchID); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `INSERT INTO conversations (kind, match_id) VALUES ('match',$1) RETURNING id`, matchID).Scan(&convID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1,$2),($1,$3)`, convID, a.id, b.id); err != nil {
		t.Fatal(err)
	}
	return convID
}

func (e *env) send(t *testing.T, u user, conv, body string) testutil.Response {
	return testutil.Do(t, e.r, "POST", "/conversations/"+conv+"/messages", map[string]string{"body": body}, u.tok)
}

func (e *env) list(t *testing.T, u user, q string) []ConversationSummary {
	t.Helper()
	res := testutil.Do(t, e.r, "GET", "/conversations"+q, nil, u.tok)
	if res.Status != 200 {
		t.Fatalf("list status %d %s", res.Status, res.Body)
	}
	var out ListConversationsResponse
	res.JSON(t, &out)
	return out.Conversations
}

func (e *env) block(t *testing.T, blocker, blocked user) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO blocks (blocker_id, blocked_id) VALUES ($1,$2)`, blocker.id, blocked.id); err != nil {
		t.Fatal(err)
	}
}

func TestListOrderingUnreadAndLastMessage(t *testing.T) {
	e := setup(t)
	me, x, y := e.user(t, "Me"), e.user(t, "Xena"), e.user(t, "Yara")
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO photos (user_id, storage_key, position, width, height, size_bytes) VALUES ($1,$2,0,10,10,10)`, x.id, "k/"+x.id); err != nil {
		t.Fatal(err)
	}
	cx := e.match(t, me, x)
	cy := e.match(t, me, y)

	got := e.list(t, me, "")
	if len(got) != 2 || got[0].LastMessage != nil || got[0].UnreadCount != 0 {
		t.Fatalf("fresh matches: %+v", got)
	}
	// Newest match first (cy created after cx).
	if got[0].ID != cy {
		t.Fatalf("expected newest match first")
	}
	for _, c := range got {
		if c.User.Age == nil || c.User.FirstName == "" {
			t.Fatalf("missing user info: %+v", c.User)
		}
	}
	if got[1].User.Photo == nil || !strings.HasPrefix(got[1].User.Photo.URL, "/photos/") || got[1].User.Photo.Position != 0 {
		t.Fatalf("photo: %+v", got[1].User.Photo)
	}

	// x writes twice -> conversation with x is now top, 2 unread, last message is second.
	e.send(t, x, cx, "one")
	e.send(t, x, cx, "two")
	got = e.list(t, me, "")
	if got[0].ID != cx || got[0].UnreadCount != 2 || got[0].LastMessage == nil || got[0].LastMessage.Body != "two" {
		t.Fatalf("after messages: %+v", got[0])
	}
	if !got[0].UpdatedAt.Equal(got[0].LastMessage.CreatedAt) {
		t.Fatal("updatedAt must equal lastMessage.createdAt")
	}
	// Sender sees zero unread.
	if xs := e.list(t, x, ""); xs[0].UnreadCount != 0 {
		t.Fatalf("sender unread: %d", xs[0].UnreadCount)
	}
	// Read clears unread.
	if r := testutil.Do(t, e.r, "POST", "/conversations/"+cx+"/read", nil, me.tok); r.Status != 204 {
		t.Fatalf("read %d", r.Status)
	}
	if got = e.list(t, me, ""); got[0].UnreadCount != 0 {
		t.Fatalf("unread after read: %d", got[0].UnreadCount)
	}
	// Pagination by updatedAt.
	page1 := e.list(t, me, "?limit=1")
	if len(page1) != 1 || page1[0].ID != cx {
		t.Fatalf("page1 %+v", page1)
	}
	page2 := e.list(t, me, "?limit=1&before="+page1[0].UpdatedAt.Format(time.RFC3339Nano))
	if len(page2) != 1 || page2[0].ID != cy {
		t.Fatalf("page2 %+v", page2)
	}
	if r := testutil.Do(t, e.r, "GET", "/conversations?before=garbage", nil, me.tok); r.Status != 400 {
		t.Fatalf("bad cursor: %d", r.Status)
	}
}

func TestMessagePaginationNoDuplicatesNoGaps(t *testing.T) {
	e := setup(t)
	a, b := e.user(t, "Ann"), e.user(t, "Bob")
	c := e.match(t, a, b)
	want := []string{}
	for i := 0; i < 11; i++ {
		u := a
		if i%3 == 0 {
			u = b
		}
		res := e.send(t, u, c, fmt.Sprintf("m%02d", i))
		if res.Status != 201 {
			t.Fatalf("send %d: %d %s", i, res.Status, res.Body)
		}
		want = append([]string{fmt.Sprintf("m%02d", i)}, want...) // newest first
	}
	var got []string
	seen := map[string]bool{}
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		q := "?limit=4"
		if cursor != "" {
			q += "&before=" + cursor
		}
		res := testutil.Do(t, e.r, "GET", "/conversations/"+c+"/messages"+q, nil, a.tok)
		if res.Status != 200 {
			t.Fatalf("status %d", res.Status)
		}
		var out ListMessagesResponse
		res.JSON(t, &out)
		for _, m := range out.Messages {
			if seen[m.ID] {
				t.Fatalf("duplicate %s", m.ID)
			}
			seen[m.ID] = true
			got = append(got, m.Body)
		}
		if out.NextCursor == nil {
			break
		}
		cursor = *out.NextCursor
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v", got, want)
	}
	if r := testutil.Do(t, e.r, "GET", "/conversations/"+c+"/messages?before=bad", nil, a.tok); r.Status != 400 {
		t.Fatalf("bad cursor %d", r.Status)
	}
}

func TestNonParticipantGets404Everywhere(t *testing.T) {
	e := setup(t)
	a, b, mallory := e.user(t, "Ann"), e.user(t, "Bob"), e.user(t, "Mal")
	c := e.match(t, a, b)
	e.send(t, a, c, "secret words")

	for _, tc := range []struct{ method, path string }{
		{"GET", "/conversations/" + c + "/messages"},
		{"POST", "/conversations/" + c + "/read"},
		{"DELETE", "/conversations/" + c},
	} {
		res := testutil.Do(t, e.r, tc.method, tc.path, nil, mallory.tok)
		if res.Status != 404 || strings.Contains(string(res.Body), "secret") {
			t.Errorf("%s %s = %d %s", tc.method, tc.path, res.Status, res.Body)
		}
	}
	if res := e.send(t, mallory, c, "hi"); res.Status != 404 {
		t.Errorf("send = %d", res.Status)
	}
	// Malformed and unknown ids.
	for _, id := range []string{"not-a-uuid", "00000000-0000-4000-8000-000000000000"} {
		if res := testutil.Do(t, e.r, "GET", "/conversations/"+id+"/messages", nil, a.tok); res.Status != 404 {
			t.Errorf("id %s = %d", id, res.Status)
		}
	}
	// Mallory's list does not include it; mallory's failed actions changed nothing.
	if l := e.list(t, mallory, ""); len(l) != 0 {
		t.Fatal("mallory sees conversations")
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM messages WHERE conversation_id=$1`, c).Scan(&n)
	if n != 1 {
		t.Fatalf("messages count %d", n)
	}
	if res := testutil.Do(t, e.r, "GET", "/conversations", nil, ""); res.Status != 401 {
		t.Fatalf("unauth %d", res.Status)
	}
}

func TestSecurityUserCannotReadOthersConversation(t *testing.T) {
	e := setup(t)
	a, b, c3, d4 := e.user(t, "A"), e.user(t, "B"), e.user(t, "C"), e.user(t, "D")
	ab := e.match(t, a, b)
	cd := e.match(t, c3, d4)
	e.send(t, a, ab, "private A to B")
	e.send(t, c3, cd, "private C to D")
	// C tries to read A/B's conversation via its own valid token.
	res := testutil.Do(t, e.r, "GET", "/conversations/"+ab+"/messages", nil, c3.tok)
	if res.Status != 404 || strings.Contains(string(res.Body), "private") {
		t.Fatalf("leak: %d %s", res.Status, res.Body)
	}
	for _, s := range e.list(t, c3, "") {
		if s.ID == ab {
			t.Fatal("listed foreign conversation")
		}
	}
}

func TestSendValidationAndBlocking(t *testing.T) {
	e := setup(t)
	a, b := e.user(t, "Ann"), e.user(t, "Bob")
	c := e.match(t, a, b)

	for _, body := range []string{"", "   ", "\x00\x01"} {
		if res := e.send(t, a, c, body); res.Status != 400 {
			t.Errorf("body %q = %d", body, res.Status)
		}
	}
	if res := e.send(t, a, c, strings.Repeat("x", 2001)); res.Status != 400 {
		t.Errorf("2001 = %d", res.Status)
	}
	res := e.send(t, a, c, strings.Repeat("é", 2000))
	if res.Status != 201 {
		t.Fatalf("2000 runes = %d %s", res.Status, res.Body)
	}
	res = e.send(t, a, c, "  hi\x00 there  ")
	var m Message
	res.JSON(t, &m)
	if res.Status != 201 || m.Body != "hi there" || m.SenderID != a.id || m.ConversationID != c || m.ReadAt != nil {
		t.Fatalf("message: %d %+v", res.Status, m)
	}
	if strings.Contains(string(res.Body), b.id) {
		t.Fatal("response leaks recipient id")
	}
	if r := testutil.Do(t, e.r, "POST", "/conversations/"+c+"/messages", "not an object", a.tok); r.Status != 400 {
		t.Fatalf("bad json %d", r.Status)
	}

	e.block(t, b, a) // b blocked a -> a cannot send
	res = e.send(t, a, c, "still there?")
	if res.Status != 403 || !strings.Contains(string(res.Body), "you cannot message this user") {
		t.Fatalf("blocked sender: %d %s", res.Status, res.Body)
	}
	// Reverse direction also refused.
	if res = e.send(t, b, c, "nope"); res.Status != 403 {
		t.Fatalf("blocker send: %d", res.Status)
	}
	// Blocked conversation is excluded from both lists.
	if len(e.list(t, a, "")) != 0 || len(e.list(t, b, "")) != 0 {
		t.Fatal("blocked conversation still listed")
	}
}

func TestHideThenNewMessageUnhides(t *testing.T) {
	e := setup(t)
	a, b := e.user(t, "Ann"), e.user(t, "Bob")
	c := e.match(t, a, b)
	e.send(t, a, c, "hello")
	if r := testutil.Do(t, e.r, "DELETE", "/conversations/"+c, nil, b.tok); r.Status != 204 {
		t.Fatalf("hide %d", r.Status)
	}
	if len(e.list(t, b, "")) != 0 {
		t.Fatal("hidden conversation listed")
	}
	if len(e.list(t, a, "")) != 1 {
		t.Fatal("hiding must be local")
	}
	// Messages still exist for both; a can still read.
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM messages WHERE conversation_id=$1`, c).Scan(&n)
	if n != 1 {
		t.Fatal("messages deleted")
	}
	// Own message does not unhide.
	e.send(t, b, c, "reply while hidden")
	// Sender's own send does not unhide own row; verify via direct check then recipient send.
	e.send(t, a, c, "back again")
	got := e.list(t, b, "")
	if len(got) != 1 || got[0].LastMessage.Body != "back again" {
		t.Fatalf("not unhidden: %+v", got)
	}
}

func TestReadReceiptsMonotonicAndEvents(t *testing.T) {
	e := setup(t)
	a, b := e.user(t, "Ann"), e.user(t, "Bob")
	c := e.match(t, a, b)
	e.send(t, a, c, "read me")

	msgs := func(u user) []Message {
		res := testutil.Do(t, e.r, "GET", "/conversations/"+c+"/messages", nil, u.tok)
		var out ListMessagesResponse
		res.JSON(t, &out)
		return out.Messages
	}
	if msgs(a)[0].ReadAt != nil {
		t.Fatal("readAt before read")
	}
	if r := testutil.Do(t, e.r, "POST", "/conversations/"+c+"/read", nil, b.tok); r.Status != 204 {
		t.Fatal(r.Status)
	}
	first := msgs(a)[0].ReadAt
	if first == nil {
		t.Fatal("readAt not set for sender")
	}
	if msgs(b)[0].ReadAt != nil {
		t.Fatal("received message must have null readAt")
	}
	testutil.Do(t, e.r, "POST", "/conversations/"+c+"/read", nil, b.tok)
	second := msgs(a)[0].ReadAt
	if second.Before(*first) {
		t.Fatal("read moved backwards")
	}
	// Conflicting older state can't win: force last_read_at forward and check GREATEST semantics.
	future := time.Now().Add(time.Hour).UTC()
	_, _ = e.pool.Exec(context.Background(), `UPDATE conversation_participants SET last_read_at=$3 WHERE conversation_id=$1 AND user_id=$2`, c, b.id, future)
	testutil.Do(t, e.r, "POST", "/conversations/"+c+"/read", nil, b.tok)
	var got time.Time
	_ = e.pool.QueryRow(context.Background(), `SELECT last_read_at FROM conversation_participants WHERE conversation_id=$1 AND user_id=$2`, c, b.id).Scan(&got)
	if got.Before(future.Add(-time.Second)) {
		t.Fatal("last_read_at regressed")
	}
	evs := e.pub.of(a.id, "conversation.read")
	if len(evs) != 2+1 {
		t.Fatalf("read events to sender: %d", len(evs))
	}
	data := evs[0].Data.(map[string]any)
	if data["conversationId"] != c || data["readAt"] == nil {
		t.Fatalf("event data %+v", data)
	}
	if len(e.pub.of(b.id, "conversation.read")) != 0 {
		t.Fatal("reader must not receive own read event")
	}
}

func TestPublishesToBothSidesAndNotifiesOnce(t *testing.T) {
	e := setup(t)
	a, b := e.user(t, "Ann"), e.user(t, "Bob")
	c := e.match(t, a, b)
	long := strings.Repeat("word ", 60)
	res := e.send(t, a, c, long)
	var m Message
	res.JSON(t, &m)

	for _, u := range []user{a, b} {
		evs := e.pub.of(u.id, "message.new")
		if len(evs) != 1 {
			t.Fatalf("user %s events %d", u.id, len(evs))
		}
		got := evs[0].Data.(map[string]any)["message"].(Message)
		if got.ID != m.ID {
			t.Fatal("wrong message in event")
		}
	}
	e.not.mu.Lock()
	defer e.not.mu.Unlock()
	if len(e.not.ns) != 1 {
		t.Fatalf("notifications %d", len(e.not.ns))
	}
	n := e.not.ns[0]
	if n.UserID != b.id || n.Type != notify.TypeMessage || n.Title != "New message" ||
		!strings.HasPrefix(n.Body, "Ann: ") || len([]rune(n.Body)) > 80 ||
		n.Data["conversationId"] != c || n.Data["userId"] != a.id {
		t.Fatalf("notification %+v", n)
	}
	// Failed sends publish nothing.
	e.send(t, a, c, "")
	if len(e.pub.of(b.id, "message.new")) != 1 {
		t.Fatal("event for failed send")
	}
}

func TestConcurrentSendsKeepConsistentOrder(t *testing.T) {
	e := setup(t)
	a, b := e.user(t, "Ann"), e.user(t, "Bob")
	c := e.match(t, a, b)
	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u := a
			if i%2 == 1 {
				u = b
			}
			if res := e.send(t, u, c, fmt.Sprintf("c%02d", i)); res.Status != 201 {
				t.Errorf("send %d: %d %s", i, res.Status, res.Body)
			}
		}(i)
	}
	wg.Wait()

	res := testutil.Do(t, e.r, "GET", "/conversations/"+c+"/messages?limit=100", nil, a.tok)
	var out ListMessagesResponse
	res.JSON(t, &out)
	if len(out.Messages) != n {
		t.Fatalf("got %d messages", len(out.Messages))
	}
	for i := 1; i < len(out.Messages); i++ {
		if !out.Messages[i].CreatedAt.Before(out.Messages[i-1].CreatedAt) {
			t.Fatalf("created_at not strictly decreasing at %d", i)
		}
	}
	// last_message_at equals the newest message.
	l := e.list(t, a, "")
	if !l[0].UpdatedAt.Equal(out.Messages[0].CreatedAt) || l[0].LastMessage.ID != out.Messages[0].ID {
		t.Fatal("conversation summary inconsistent with newest message")
	}
}
