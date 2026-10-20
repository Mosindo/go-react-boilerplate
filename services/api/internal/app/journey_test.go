package app_test

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
)

func man(name string) profileSpec {
	return profileSpec{Name: name, Gender: "man", InterestedIn: []string{"woman"}}
}

func woman(name string) profileSpec {
	return profileSpec{Name: name, Gender: "woman", InterestedIn: []string{"man"}}
}

func ids(list []any, path ...string) []string {
	out := []string{}
	for _, item := range list {
		cur := item
		for _, p := range path {
			m, _ := cur.(map[string]any)
			cur = m[p]
		}
		s, _ := cur.(string)
		out = append(out, s)
	}
	return out
}

func has(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Register -> profile -> discover -> like -> match -> chat -> notifications.
func TestCriticalJourney(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard("alice", woman("Alice"))
	bob := e.onboard("bob", man("Bob"))

	// Alice discovers Bob, with coarse distance and no exact location.
	res := e.do("GET", "/discover?limit=10", alice.Token, nil)
	if res.Status != 200 {
		t.Fatalf("discover: %d %s", res.Status, res.Raw)
	}
	profiles := res.list("profiles")
	if len(profiles) != 1 || ids(profiles, "userId")[0] != bob.ID {
		t.Fatalf("expected only bob, got %s", res.Raw)
	}
	raw := string(res.Raw)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "48.85"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("discover leaks %q: %s", forbidden, raw)
		}
	}
	first := profiles[0].(map[string]any)
	if first["distanceKm"] != float64(1) || first["age"] == nil {
		t.Fatalf("unexpected profile fields: %v", first)
	}

	// One-sided like: no match yet.
	res = e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"})
	if res.Status != 200 || res.Body["matched"] != false {
		t.Fatalf("first like: %d %s", res.Status, res.Raw)
	}
	// Already-processed profiles never come back, repeated swipes are rejected.
	if got := e.do("GET", "/discover", alice.Token, nil); len(got.list("profiles")) != 0 {
		t.Fatalf("alice should not see bob again: %s", got.Raw)
	}
	if res = e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"}); res.Status != http.StatusConflict {
		t.Fatalf("duplicate like should be 409, got %d", res.Status)
	}
	if res = e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "pass"}); res.Status != http.StatusConflict {
		t.Fatalf("switching reaction should be 409, got %d", res.Status)
	}
	if got := e.do("GET", "/matches", bob.Token, nil); len(got.list("matches")) != 0 {
		t.Fatalf("no match expected yet")
	}

	// Reciprocal like: match.
	res = e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"})
	if res.Status != 200 || res.Body["matched"] != true {
		t.Fatalf("expected match: %d %s", res.Status, res.Raw)
	}
	conversationID := res.str("match", "conversationId")
	if conversationID == "" || res.str("match", "user", "userId") != alice.ID {
		t.Fatalf("bad match payload: %s", res.Raw)
	}
	for _, u := range []user{alice, bob} {
		matches := e.do("GET", "/matches", u.Token, nil)
		if len(matches.list("matches")) != 1 {
			t.Fatalf("expected one match for %s: %s", u.Email, matches.Raw)
		}
	}

	// Chat.
	res = e.do("POST", "/conversations/"+conversationID+"/messages", alice.Token, map[string]any{"body": "  Hello Bob!  "})
	if res.Status != 201 || res.str("body") != "Hello Bob!" || res.str("senderId") != alice.ID {
		t.Fatalf("send: %d %s", res.Status, res.Raw)
	}
	convs := e.do("GET", "/conversations", bob.Token, nil)
	conv := convs.list("conversations")[0].(map[string]any)
	if conv["unreadCount"] != float64(1) || conv["lastMessage"].(map[string]any)["body"] != "Hello Bob!" {
		t.Fatalf("bob conversation: %s", convs.Raw)
	}
	if aliceConvs := e.do("GET", "/conversations", alice.Token, nil); aliceConvs.list("conversations")[0].(map[string]any)["unreadCount"] != float64(0) {
		t.Fatalf("sender should have nothing unread: %s", aliceConvs.Raw)
	}

	msgs := e.do("GET", "/conversations/"+conversationID+"/messages", alice.Token, nil)
	if len(msgs.list("messages")) != 1 || msgs.Body["otherLastReadAt"] != nil {
		t.Fatalf("alice messages: %s", msgs.Raw)
	}
	if res = e.do("POST", "/conversations/"+conversationID+"/read", bob.Token, nil); res.Status != 204 {
		t.Fatalf("read: %d", res.Status)
	}
	msgs = e.do("GET", "/conversations/"+conversationID+"/messages", alice.Token, nil)
	if msgs.Body["otherLastReadAt"] == nil {
		t.Fatalf("expected read receipt: %s", msgs.Raw)
	}
	convs = e.do("GET", "/conversations", bob.Token, nil)
	if convs.list("conversations")[0].(map[string]any)["unreadCount"] != float64(0) {
		t.Fatalf("unread should be cleared")
	}

	// Pagination of messages.
	for i := 0; i < 4; i++ {
		e.do("POST", "/conversations/"+conversationID+"/messages", bob.Token, map[string]any{"body": "msg"})
	}
	page := e.do("GET", "/conversations/"+conversationID+"/messages?limit=2", bob.Token, nil)
	if len(page.list("messages")) != 2 || page.Body["nextCursor"] == nil {
		t.Fatalf("page 1: %s", page.Raw)
	}
	page2 := e.do("GET", "/conversations/"+conversationID+"/messages?limit=10&cursor="+page.str("nextCursor"), bob.Token, nil)
	if len(page2.list("messages")) != 3 || page2.Body["nextCursor"] != nil {
		t.Fatalf("page 2: %s", page2.Raw)
	}

	// Notifications: Alice got a match + (unread) message from Bob; Bob got match.
	notes := e.do("GET", "/notifications", alice.Token, nil)
	types := ids(notes.list("notifications"), "type")
	if !has(types, "match") || !has(types, "message") || len(types) != 2 || notes.Body["unreadCount"] != float64(2) {
		t.Fatalf("alice notifications (message notifications must coalesce): %s", notes.Raw)
	}
	if res = e.do("POST", "/notifications/read-all", alice.Token, nil); res.Status != 204 {
		t.Fatalf("read-all: %d", res.Status)
	}
	if notes = e.do("GET", "/notifications", alice.Token, nil); notes.Body["unreadCount"] != float64(0) {
		t.Fatalf("expected all read: %s", notes.Raw)
	}
}

func TestConversationPermissions(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard("alice", woman("Alice"))
	bob := e.onboard("bob", man("Bob"))
	carol := e.onboard("carol", woman("Carol"))
	e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"})
	conv := e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}).str("match", "conversationId")
	e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": "secret"})

	for _, tc := range []struct{ method, path string }{
		{"GET", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/messages"},
		{"POST", "/conversations/" + conv + "/read"},
		{"DELETE", "/conversations/" + conv},
	} {
		if res := e.do(tc.method, tc.path, carol.Token, map[string]any{"body": "intruder"}); res.Status != 404 {
			t.Errorf("%s %s by outsider: want 404 got %d", tc.method, tc.path, res.Status)
		}
	}
	if res := e.do("GET", "/conversations", carol.Token, nil); len(res.list("conversations")) != 0 {
		t.Errorf("outsider lists conversations: %s", res.Raw)
	}
	if res := e.do("GET", "/conversations/not-a-uuid/messages", alice.Token, nil); res.Status != 404 {
		t.Errorf("bad id: %d", res.Status)
	}
	if res := e.do("GET", "/conversations/"+conv+"/messages", "", nil); res.Status != 401 {
		t.Errorf("anonymous: %d", res.Status)
	}
	// Message validation.
	for _, body := range []string{"", "   ", strings.Repeat("x", 2001)} {
		if res := e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": body}); res.Status != 400 {
			t.Errorf("body len %d: want 400 got %d", len(body), res.Status)
		}
	}
	// Local deletion hides history for the caller only; a new message revives it.
	if res := e.do("DELETE", "/conversations/"+conv, bob.Token, nil); res.Status != 204 {
		t.Fatalf("hide: %d", res.Status)
	}
	if res := e.do("GET", "/conversations", bob.Token, nil); len(res.list("conversations")) != 0 {
		t.Errorf("hidden conversation still listed")
	}
	if res := e.do("GET", "/conversations", alice.Token, nil); len(res.list("conversations")) != 1 {
		t.Errorf("hiding must not affect the other user")
	}
	e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": "still there?"})
	res := e.do("GET", "/conversations/"+conv+"/messages", bob.Token, nil)
	if got := res.list("messages"); len(got) != 1 || got[0].(map[string]any)["body"] != "still there?" {
		t.Errorf("hidden history leaked or new message missing: %s", res.Raw)
	}
}

func TestBlockUnmatchesAndHides(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard("alice", woman("Alice"))
	bob := e.onboard("bob", man("Bob"))
	e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"})
	conv := e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}).str("match", "conversationId")
	e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]any{"body": "hi"})

	if res := e.do("POST", "/blocks", alice.Token, map[string]any{"userId": bob.ID}); res.Status != 204 {
		t.Fatalf("block: %d %s", res.Status, res.Raw)
	}
	if res := e.do("POST", "/blocks", alice.Token, map[string]any{"userId": bob.ID}); res.Status != 204 {
		t.Fatalf("block is idempotent: %d", res.Status)
	}
	checks := []struct {
		name string
		res  result
		want int
	}{
		{"bob reads conversation", e.do("GET", "/conversations/"+conv+"/messages", bob.Token, nil), 404},
		{"bob sends", e.do("POST", "/conversations/"+conv+"/messages", bob.Token, map[string]any{"body": "x"}), 404},
		{"bob views alice", e.do("GET", "/profiles/"+alice.ID, bob.Token, nil), 404},
		{"alice views bob", e.do("GET", "/profiles/"+bob.ID, alice.Token, nil), 404},
		{"bob swipes alice", e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}), 409},
	}
	for _, c := range checks {
		if c.res.Status != c.want {
			t.Errorf("%s: want %d got %d (%s)", c.name, c.want, c.res.Status, c.res.Raw)
		}
	}
	for _, u := range []user{alice, bob} {
		if res := e.do("GET", "/matches", u.Token, nil); len(res.list("matches")) != 0 {
			t.Errorf("match must be dissolved for %s", u.Email)
		}
		if res := e.do("GET", "/notifications", u.Token, nil); len(res.list("notifications")) != 0 {
			t.Errorf("notifications linking blocked users must be removed: %s", res.Raw)
		}
	}
	list := e.do("GET", "/blocks", alice.Token, nil)
	if ids(list.list("blocks"), "userId")[0] != bob.ID {
		t.Fatalf("blocks list: %s", list.Raw)
	}
	// A fresh pair that never swiped: block hides them from discovery both ways.
	carol := e.onboard("carol", man("Carol"))
	dave := e.onboard("dave", woman("Dave"))
	e.do("POST", "/blocks", dave.Token, map[string]any{"userId": carol.ID})
	for _, u := range []user{carol, dave} {
		for _, p := range ids(e.do("GET", "/discover", u.Token, nil).list("profiles"), "userId") {
			if (u.ID == carol.ID && p == dave.ID) || (u.ID == dave.ID && p == carol.ID) {
				t.Errorf("blocked user visible in discovery")
			}
		}
	}
	if res := e.do("DELETE", "/blocks/"+bob.ID, alice.Token, nil); res.Status != 204 {
		t.Fatalf("unblock: %d", res.Status)
	}
	if res := e.do("POST", "/blocks", alice.Token, map[string]any{"userId": alice.ID}); res.Status != 400 {
		t.Errorf("self block: %d", res.Status)
	}
	if res := e.do("POST", "/blocks", alice.Token, map[string]any{"userId": "00000000-0000-0000-0000-000000000000"}); res.Status != 404 {
		t.Errorf("unknown block: %d", res.Status)
	}
}

func TestReports(t *testing.T) {
	e := newEnv(t)
	alice := e.register("alice")
	bob := e.register("bob")
	if res := e.do("POST", "/reports", alice.Token, map[string]any{"userId": bob.ID, "reason": "harassment", "details": "rude"}); res.Status != 201 {
		t.Fatalf("report: %d %s", res.Status, res.Raw)
	}
	for name, body := range map[string]map[string]any{
		"bad reason": {"userId": bob.ID, "reason": "dislike"},
		"self":       {"userId": alice.ID, "reason": "spam"},
		"unknown":    {"userId": "00000000-0000-0000-0000-000000000000", "reason": "spam"},
		"long":       {"userId": bob.ID, "reason": "spam", "details": strings.Repeat("a", 1001)},
	} {
		if res := e.do("POST", "/reports", alice.Token, body); res.Status < 400 {
			t.Errorf("%s accepted: %d", name, res.Status)
		}
	}
}

func TestDiscoveryHonoursPreferences(t *testing.T) {
	e := newEnv(t)
	me := e.onboard("me", profileSpec{Name: "Me", Gender: "woman", InterestedIn: []string{"man"}, MinAge: 25, MaxAge: 40, MaxDistance: 30})

	ok := e.onboard("ok", profileSpec{Name: "Okay", Gender: "man", InterestedIn: []string{"woman"}, BirthDate: "1992-01-01", Lat: 48.90, Lng: 2.40})
	wrongGender := e.onboard("wg", profileSpec{Name: "Wg", Gender: "woman", InterestedIn: []string{"woman"}})
	tooYoung := e.onboard("young", profileSpec{Name: "Young", Gender: "man", InterestedIn: []string{"woman"}, BirthDate: "2005-01-01"})
	tooFar := e.onboard("far", profileSpec{Name: "Far", Gender: "man", InterestedIn: []string{"woman"}, Lat: 45.76, Lng: 4.83}) // Lyon
	notInterested := e.onboard("ni", profileSpec{Name: "Ni", Gender: "man", InterestedIn: []string{"man"}})
	agePicky := e.onboard("picky", profileSpec{Name: "Picky", Gender: "man", InterestedIn: []string{"woman"}, MinAge: 18, MaxAge: 25})
	hidden := e.onboard("hid", man("Hid"))
	e.do("PUT", "/profile/preferences", hidden.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": false, "showDistance": true})
	noPhoto := e.register("nophoto")
	e.do("PUT", "/profile/me", noPhoto.Token, map[string]any{"firstName": "Nop", "birthDate": "1992-01-01", "gender": "man"})
	e.do("PUT", "/profile/location", noPhoto.Token, map[string]any{"latitude": 48.85, "longitude": 2.35})

	got := ids(e.do("GET", "/discover", me.Token, nil).list("profiles"), "userId")
	if len(got) != 1 || got[0] != ok.ID {
		t.Fatalf("only 'ok' expected, got %v (wg=%s young=%s far=%s ni=%s picky=%s hid=%s)", got, wrongGender.ID, tooYoung.ID, tooFar.ID, notInterested.ID, agePicky.ID, hidden.ID)
	}

	// The server rejects swipes on anyone not eligible, even with a known id.
	for name, target := range map[string]string{"wrong gender": wrongGender.ID, "too far": tooFar.ID, "hidden": hidden.ID, "no photo": noPhoto.ID} {
		if res := e.do("POST", "/swipes", me.Token, map[string]any{"targetUserId": target, "action": "like"}); res.Status != 404 {
			t.Errorf("swipe on %s: want 404 got %d", name, res.Status)
		}
	}
	// Incomplete viewer gets a clear error.
	if res := e.do("GET", "/discover", noPhoto.Token, nil); res.Status != 409 {
		t.Errorf("incomplete viewer: %d", res.Status)
	}
}

func TestDiscoveryRanksSharedInterestsFirstAndBatches(t *testing.T) {
	e := newEnv(t)
	me := e.onboard("me", profileSpec{Name: "Me", Gender: "woman", InterestedIn: []string{"man"}, Interests: []string{"hiking", "music", "art"}})
	none := e.onboard("none", profileSpec{Name: "None", Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"gaming"}})
	many := e.onboard("many", profileSpec{Name: "Many", Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"hiking", "music", "art"}})
	some := e.onboard("some", profileSpec{Name: "Some", Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"hiking"}})
	got := ids(e.do("GET", "/discover", me.Token, nil).list("profiles"), "userId")
	want := []string{many.ID, some.ID, none.ID}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ranking: got %v want %v", got, want)
		}
	}
	if len(ids(e.do("GET", "/discover?limit=2", me.Token, nil).list("profiles"), "userId")) != 2 {
		t.Fatal("limit not applied")
	}
	if res := e.do("GET", "/discover?limit=1000", me.Token, nil); res.Status != 400 {
		t.Fatalf("limit validation: %d", res.Status)
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.onboard("a", woman("A"))
		b := e.onboard("b", man("B"))
		var wg sync.WaitGroup
		results := make([]result, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			results[0] = e.do("POST", "/swipes", a.Token, map[string]any{"targetUserId": b.ID, "action": "like"})
		}()
		go func() {
			defer wg.Done()
			results[1] = e.do("POST", "/swipes", b.Token, map[string]any{"targetUserId": a.ID, "action": "like"})
		}()
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r.Status != 200 {
				t.Fatalf("swipe failed: %d %s", r.Status, r.Raw)
			}
			if r.Body["matched"] == true {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("round %d: expected exactly one side to observe the match, got %d", round, matched)
		}
		if n := len(e.do("GET", "/matches", a.Token, nil).list("matches")); n != 1 {
			t.Fatalf("round %d: %d matches", round, n)
		}
	}
}

func TestUnmatchKeepsPairOutOfDiscovery(t *testing.T) {
	e := newEnv(t)
	a := e.onboard("a", woman("A"))
	b := e.onboard("b", man("B"))
	e.do("POST", "/swipes", a.Token, map[string]any{"targetUserId": b.ID, "action": "like"})
	m := e.do("POST", "/swipes", b.Token, map[string]any{"targetUserId": a.ID, "action": "like"})
	matchID := m.str("match", "id")
	if res := e.do("DELETE", "/matches/"+matchID, b.Token, nil); res.Status != 204 {
		t.Fatalf("unmatch: %d", res.Status)
	}
	if res := e.do("DELETE", "/matches/"+matchID, b.Token, nil); res.Status != 404 {
		t.Fatalf("second unmatch: %d", res.Status)
	}
	for _, u := range []user{a, b} {
		if len(e.do("GET", "/discover", u.Token, nil).list("profiles")) != 0 {
			t.Fatal("unmatched pair reappeared")
		}
	}
}

func TestPhotoVisibilityAndAccountDeletion(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard("alice", woman("Alice"))
	bob := e.onboard("bob", man("Bob"))
	carol := e.onboard("carol", woman("Carol"))

	mine := e.do("GET", "/profile/me", alice.Token, nil)
	photoURL := ids(mine.Body["profile"].(map[string]any)["photos"].([]any), "url")[0]

	if res := e.do("GET", photoURL, bob.Token, nil); res.Status != 200 || len(res.Raw) < 100 || res.Raw[0] != 0xFF {
		t.Fatalf("discoverable photo should be served as JPEG: %d", res.Status)
	}
	if res := e.do("GET", photoURL, "", nil); res.Status != 401 {
		t.Fatalf("anonymous photo: %d", res.Status)
	}
	// Hide Alice: strangers get 404, a match partner still sees her.
	e.do("PUT", "/profile/preferences", alice.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": false, "showDistance": true})
	if res := e.do("GET", photoURL, carol.Token, nil); res.Status != 404 {
		t.Fatalf("hidden photo visible to stranger: %d", res.Status)
	}
	if res := e.do("GET", "/profiles/"+alice.ID, carol.Token, nil); res.Status != 404 {
		t.Fatalf("hidden profile visible: %d", res.Status)
	}
	e.do("PUT", "/profile/preferences", alice.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": true, "showDistance": true})
	e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"})
	conv := e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"}).str("match", "conversationId")
	e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": "hello"})
	e.do("POST", "/reports", alice.Token, map[string]any{"userId": bob.ID, "reason": "spam"})
	e.do("POST", "/blocks", alice.Token, map[string]any{"userId": carol.ID})

	// Account deletion requires the password...
	if res := e.do("DELETE", "/me", alice.Token, map[string]any{"password": "wrong-password"}); res.Status != 401 {
		t.Fatalf("delete with wrong password: %d", res.Status)
	}
	entries, _ := os.ReadDir(e.dir)
	if len(entries) != 3 {
		t.Fatalf("expected 3 stored photos, got %d", len(entries))
	}
	if res := e.do("DELETE", "/me", alice.Token, map[string]any{"password": testPassword}); res.Status != 204 {
		t.Fatalf("delete: %d %s", res.Status, res.Raw)
	}
	// ... and erases everything about her.
	if entries, _ = os.ReadDir(e.dir); len(entries) != 2 {
		t.Fatalf("photo file not removed: %d left", len(entries))
	}
	if res := e.do("GET", "/me", alice.Token, nil); res.Status != 401 {
		t.Fatalf("deleted user still resolves: %d", res.Status)
	}
	if res := e.do("POST", "/auth/login", "", map[string]any{"email": alice.Email, "password": testPassword}); res.Status != 401 {
		t.Fatalf("deleted user can login: %d", res.Status)
	}
	if res := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": alice.Refresh}); res.Status != 401 {
		t.Fatalf("refresh after delete: %d", res.Status)
	}
	if res := e.do("GET", "/matches", bob.Token, nil); len(res.list("matches")) != 0 {
		t.Fatalf("match survived deletion: %s", res.Raw)
	}
	if res := e.do("GET", "/conversations/"+conv+"/messages", bob.Token, nil); res.Status != 404 {
		t.Fatalf("conversation survived deletion: %d", res.Status)
	}
	if res := e.do("GET", photoURL, bob.Token, nil); res.Status != 404 {
		t.Fatalf("photo survived deletion: %d", res.Status)
	}
}
