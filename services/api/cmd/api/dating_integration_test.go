package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type ownProfile struct {
	FirstName  string   `json:"firstName"`
	Age        int      `json:"age"`
	IsComplete bool     `json:"isComplete"`
	Missing    []string `json:"missing"`
	Photos     []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"photos"`
	Interests []struct {
		ID int `json:"id"`
	} `json:"interests"`
}

type photosPayload struct {
	Photos []struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Position int    `json:"position"`
	} `json:"photos"`
}

type messagesPayload struct {
	Messages []struct {
		ID           string `json:"id"`
		SenderUserID string `json:"senderUserId"`
		Content      string `json:"content"`
	} `json:"messages"`
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	HasMore         bool       `json:"hasMore"`
}

type chatsPayload struct {
	Chats []struct {
		MatchID     string        `json:"matchId"`
		User        publicProfile `json:"user"`
		UnreadCount int           `json:"unreadCount"`
	} `json:"chats"`
}

type notificationsPayload struct {
	Notifications []struct {
		ID   string            `json:"id"`
		Type string            `json:"type"`
		Data map[string]string `json:"data"`
	} `json:"notifications"`
	UnreadCount int `json:"unreadCount"`
}

func TestOnboardingValidation(t *testing.T) {
	env := newEnv(t)
	m := env.register(t, "onboard")

	expect(t, env.do(t, http.MethodGet, "/profile", nil, m.Token), http.StatusNotFound, "profile before creation")
	expect(t, env.upload(t, http.MethodPost, "/profile/photos", pngImage(t, 400, 400, 1), m.Token), http.StatusBadRequest, "photo before profile")
	expect(t, env.do(t, http.MethodGet, "/discover", nil, m.Token), http.StatusConflict, "discover before profile")

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	expect(t, env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "Lea", "birthdate": minor, "gender": "woman"}, m.Token), http.StatusBadRequest, "underage")
	expect(t, env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "Lea", "birthdate": "1990-02-30", "gender": "woman"}, m.Token), http.StatusBadRequest, "invalid date")
	expect(t, env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "Lea", "birthdate": "1990-02-01", "gender": "robot"}, m.Token), http.StatusBadRequest, "invalid gender")
	expect(t, env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "<script>", "birthdate": "1990-02-01", "gender": "woman"}, m.Token), http.StatusBadRequest, "invalid name")
	expect(t, env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "Léa", "birthdate": "1990-02-01", "gender": "woman", "bio": strings.Repeat("a", 501)}, m.Token), http.StatusBadRequest, "bio too long")

	r := env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "  Léa  ", "birthdate": "1990-02-01", "gender": "woman", "bio": "Bonjour"}, m.Token)
	expect(t, r, http.StatusOK, "create profile")
	p := decode[ownProfile](t, r)
	if p.FirstName != "Léa" || p.IsComplete || len(p.Missing) != 2 {
		t.Fatalf("unexpected profile state: %+v", p)
	}
	expect(t, env.do(t, http.MethodPut, "/profile", map[string]any{"firstName": "Léa", "birthdate": "1991-02-01", "gender": "woman"}, m.Token), http.StatusBadRequest, "birthdate locked")

	expect(t, env.do(t, http.MethodPut, "/profile/preferences", map[string]any{"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 30}, m.Token), http.StatusBadRequest, "empty interested in")
	expect(t, env.do(t, http.MethodPut, "/profile/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 30}, m.Token), http.StatusBadRequest, "bad age range")
	expect(t, env.do(t, http.MethodPut, "/profile/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 45, "maxDistanceKm": 30}, m.Token), http.StatusOK, "preferences")

	interests := decode[struct {
		Interests []struct {
			ID int `json:"id"`
		} `json:"interests"`
	}](t, env.do(t, http.MethodGet, "/interests", nil, m.Token)).Interests
	if len(interests) < 20 {
		t.Fatalf("expected seeded interests, got %d", len(interests))
	}
	ids := []int{}
	for _, i := range interests[:11] {
		ids = append(ids, i.ID)
	}
	expect(t, env.do(t, http.MethodPut, "/profile/interests", map[string]any{"interestIds": ids}, m.Token), http.StatusBadRequest, "too many interests")
	expect(t, env.do(t, http.MethodPut, "/profile/interests", map[string]any{"interestIds": []int{32000}}, m.Token), http.StatusBadRequest, "unknown interest")
	r = env.do(t, http.MethodPut, "/profile/interests", map[string]any{"interestIds": ids[:3]}, m.Token)
	expect(t, r, http.StatusOK, "interests")
	if got := len(decode[ownProfile](t, r).Interests); got != 3 {
		t.Fatalf("expected 3 interests, got %d", got)
	}

	expect(t, env.do(t, http.MethodPut, "/profile/location", map[string]any{"latitude": 123.0, "longitude": 2.0}, m.Token), http.StatusBadRequest, "invalid latitude")
	expect(t, env.upload(t, http.MethodPost, "/profile/photos", []byte("<?php system($_GET['c']); ?>"), m.Token), http.StatusUnsupportedMediaType, "script upload")
	expect(t, env.upload(t, http.MethodPost, "/profile/photos", pngImage(t, 50, 50, 1), m.Token), http.StatusBadRequest, "tiny image")
	r = env.upload(t, http.MethodPost, "/profile/photos", pngImage(t, 600, 800, 1), m.Token)
	expect(t, r, http.StatusCreated, "photo")

	p = decode[ownProfile](t, env.do(t, http.MethodGet, "/profile", nil, m.Token))
	if !p.IsComplete || len(p.Missing) != 0 {
		t.Fatalf("profile should be complete: %+v", p)
	}
	expect(t, env.do(t, http.MethodGet, "/discover", nil, m.Token), http.StatusOK, "discover once complete")
}

func TestPhotoManagementAndSignedMedia(t *testing.T) {
	env := newEnv(t)
	m := env.onboard(t, "photos", profileSpec{Name: "Nina", Gender: "woman", InterestedIn: []string{"man"}})

	var list photosPayload
	for i := 0; i < 5; i++ {
		r := env.upload(t, http.MethodPost, "/profile/photos", pngImage(t, 300, 300, uint8(i*40)), m.Token)
		expect(t, r, http.StatusCreated, "upload")
		list = decode[photosPayload](t, r)
	}
	if len(list.Photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(list.Photos))
	}
	expect(t, env.upload(t, http.MethodPost, "/profile/photos", pngImage(t, 300, 300, 1), m.Token), http.StatusBadRequest, "7th photo")

	// Signed URLs serve sanitized JPEGs; tampering is rejected.
	url := list.Photos[0].URL
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("media fetch failed: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	rec = httptest.NewRecorder()
	env.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url[:len(url)-4]+"beef", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("tampered media url should be 403, got %d", rec.Code)
	}

	// Reorder: last becomes primary.
	ids := []string{}
	for i := len(list.Photos) - 1; i >= 0; i-- {
		ids = append(ids, list.Photos[i].ID)
	}
	r := env.do(t, http.MethodPut, "/profile/photos/order", map[string]any{"photoIds": ids}, m.Token)
	expect(t, r, http.StatusOK, "reorder")
	if decode[photosPayload](t, r).Photos[0].ID != ids[0] {
		t.Fatal("reorder did not change primary photo")
	}
	expect(t, env.do(t, http.MethodPut, "/profile/photos/order", map[string]any{"photoIds": ids[:2]}, m.Token), http.StatusBadRequest, "partial reorder")

	r = env.upload(t, http.MethodPut, "/profile/photos/"+ids[0], pngImage(t, 500, 500, 200), m.Token)
	expect(t, r, http.StatusOK, "replace")
	replaced := decode[photosPayload](t, r)
	if replaced.Photos[0].ID != ids[0] || replaced.Photos[0].URL == list.Photos[len(list.Photos)-1].URL {
		t.Fatal("replace should keep id/position and change file")
	}

	r = env.do(t, http.MethodDelete, "/profile/photos/"+ids[0], nil, m.Token)
	expect(t, r, http.StatusOK, "delete")
	after := decode[photosPayload](t, r)
	if len(after.Photos) != 5 || after.Photos[0].Position != 0 || after.Photos[0].ID != ids[1] {
		t.Fatalf("positions not compacted after delete: %+v", after)
	}

	other := env.onboard(t, "photos_other", profileSpec{Name: "Omar", Gender: "man", InterestedIn: []string{"woman"}})
	expect(t, env.do(t, http.MethodDelete, "/profile/photos/"+ids[1], nil, other.Token), http.StatusNotFound, "delete someone else's photo")
}

func TestDiscoverLikeMatchChatFlow(t *testing.T) {
	env := newEnv(t)
	alice := env.onboard(t, "alice", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Lat: 45.7640, Lng: 4.8357})
	bob := env.onboard(t, "bob", profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Lat: 45.7500, Lng: 4.8500})
	carol := env.onboard(t, "carol", profileSpec{Name: "Carol", Gender: "woman", InterestedIn: []string{"woman"}})
	far := env.onboard(t, "far", profileSpec{Name: "Fanny", Gender: "woman", InterestedIn: []string{"man"}, Lat: 48.8566, Lng: 2.3522})

	deck := env.discover(t, bob)
	if !containsProfile(deck, alice.ID) {
		t.Fatal("bob should discover alice")
	}
	if containsProfile(deck, carol.ID) {
		t.Fatal("carol is not interested in men: must not be shown to bob")
	}
	if containsProfile(deck, far.ID) {
		t.Fatal("profiles beyond max distance must be filtered")
	}
	for _, p := range deck {
		if p.UserID == alice.ID {
			if p.DistanceKm == nil || *p.DistanceKm < 1 || *p.DistanceKm > 5 || p.Age == nil || len(p.Photos) == 0 {
				t.Fatalf("unexpected public card: %+v", p)
			}
		}
	}
	raw := string(env.do(t, http.MethodGet, "/profiles/"+alice.ID, nil, bob.Token).Body)
	for _, leak := range []string{"latitude", "longitude", "birthdate", alice.Email} {
		if strings.Contains(raw, leak) {
			t.Fatalf("public profile leaks %q: %s", leak, raw)
		}
	}

	expect(t, env.swipe(t, bob, bob.ID, "like"), http.StatusBadRequest, "self swipe")
	expect(t, env.swipe(t, bob, alice.ID, "superlike"), http.StatusBadRequest, "invalid decision")
	r := env.swipe(t, bob, alice.ID, "like")
	expect(t, r, http.StatusOK, "bob likes alice")
	if decode[swipeResp](t, r).Matched {
		t.Fatal("one-sided like must not match")
	}
	expect(t, env.swipe(t, bob, alice.ID, "like"), http.StatusConflict, "duplicate like")
	if containsProfile(env.discover(t, bob), alice.ID) {
		t.Fatal("already-swiped profile must not reappear")
	}
	expect(t, env.do(t, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "hi"}, bob.Token), http.StatusNotFound, "chat before match")

	aliceDeck := env.discover(t, alice)
	if len(aliceDeck) == 0 || !containsProfile(aliceDeck, bob.ID) {
		t.Fatal("alice should discover bob")
	}
	r = env.swipe(t, alice, bob.ID, "like")
	expect(t, r, http.StatusOK, "alice likes bob")
	res := decode[swipeResp](t, r)
	if !res.Matched || res.Match == nil {
		t.Fatalf("mutual like must match: %s", r)
	}
	var matches int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid,$2::uuid) AND user_high_id = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID).Scan(&matches)
	if matches != 1 {
		t.Fatalf("expected exactly one match row, got %d", matches)
	}

	for _, m := range []member{alice, bob} {
		n := decode[notificationsPayload](t, env.do(t, http.MethodGet, "/notifications", nil, m.Token))
		if n.UnreadCount < 1 || n.Notifications[0].Type != "match" {
			t.Fatalf("expected a match notification: %+v", n)
		}
	}
	ml := decode[struct {
		Matches []struct {
			ID string `json:"id"`
		} `json:"matches"`
	}](t, env.do(t, http.MethodGet, "/matches", nil, bob.Token))
	if len(ml.Matches) != 1 || ml.Matches[0].ID != res.Match.ID {
		t.Fatalf("unexpected matches list: %+v", ml)
	}

	// Chat.
	expect(t, env.do(t, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "   "}, bob.Token), http.StatusBadRequest, "blank message")
	expect(t, env.do(t, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": strings.Repeat("x", 2001)}, bob.Token), http.StatusBadRequest, "long message")
	expect(t, env.do(t, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "Salut Alice !"}, bob.Token), http.StatusCreated, "send")
	expect(t, env.do(t, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "Ça va ?"}, bob.Token), http.StatusCreated, "send 2")

	chats := decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, alice.Token))
	if len(chats.Chats) != 1 || chats.Chats[0].User.UserID != bob.ID || chats.Chats[0].UnreadCount != 2 {
		t.Fatalf("unexpected chats for alice: %+v", chats)
	}
	n := decode[notificationsPayload](t, env.do(t, http.MethodGet, "/notifications", nil, alice.Token))
	messageNotifs := 0
	for _, item := range n.Notifications {
		if item.Type == "message" {
			messageNotifs++
		}
	}
	if messageNotifs != 1 {
		t.Fatalf("message notifications must be deduplicated per conversation, got %d", messageNotifs)
	}

	msgs := decode[messagesPayload](t, env.do(t, http.MethodGet, "/chats/"+bob.ID+"/messages", nil, alice.Token))
	if len(msgs.Messages) != 2 || msgs.Messages[0].Content != "Salut Alice !" || msgs.OtherLastReadAt == nil {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
	expect(t, env.do(t, http.MethodPost, "/chats/"+bob.ID+"/read", nil, alice.Token), http.StatusOK, "mark read")
	if decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, alice.Token)).Chats[0].UnreadCount != 0 {
		t.Fatal("unread count must reset after read")
	}
	bobView := decode[messagesPayload](t, env.do(t, http.MethodGet, "/chats/"+alice.ID+"/messages", nil, bob.Token))
	if bobView.OtherLastReadAt == nil {
		t.Fatal("bob should see alice's read receipt")
	}

	// Pagination with keyset cursor.
	for i := 0; i < 5; i++ {
		expect(t, env.do(t, http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": fmt.Sprintf("m%d", i)}, alice.Token), http.StatusCreated, "send page")
	}
	page := decode[messagesPayload](t, env.do(t, http.MethodGet, "/chats/"+bob.ID+"/messages?limit=3", nil, alice.Token))
	if len(page.Messages) != 3 || !page.HasMore || page.Messages[2].Content != "m4" {
		t.Fatalf("unexpected first page: %+v", page)
	}
	older := decode[messagesPayload](t, env.do(t, http.MethodGet, "/chats/"+bob.ID+"/messages?limit=3&before="+page.Messages[0].ID, nil, alice.Token))
	if len(older.Messages) != 3 || older.Messages[2].Content != "m1" {
		t.Fatalf("unexpected older page: %+v", older)
	}

	// An outsider matched with alice still cannot read the alice/bob conversation.
	dave := env.onboard(t, "dave", profileSpec{Name: "Dave", Gender: "man", InterestedIn: []string{"woman"}})
	env.match(t, alice, dave)
	expect(t, env.do(t, http.MethodGet, "/chats/"+bob.ID+"/messages", nil, dave.Token), http.StatusNotFound, "outsider reads conversation")
	expect(t, env.do(t, http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": "intrusion"}, dave.Token), http.StatusNotFound, "outsider writes")

	// Local hide: disappears for alice only, comes back on new message.
	expect(t, env.do(t, http.MethodDelete, "/chats/"+bob.ID, nil, alice.Token), http.StatusNoContent, "hide")
	if len(decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, alice.Token)).Chats) != 0 {
		t.Fatal("hidden conversation must disappear for alice")
	}
	if len(decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, bob.Token)).Chats) != 1 {
		t.Fatal("hide must not affect bob")
	}
	expect(t, env.do(t, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "Toujours là ?"}, bob.Token), http.StatusCreated, "send after hide")
	if len(decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, alice.Token)).Chats) != 1 {
		t.Fatal("new message must resurface the conversation")
	}

	n = decode[notificationsPayload](t, env.do(t, http.MethodGet, "/notifications", nil, alice.Token))
	expect(t, env.do(t, http.MethodPost, "/notifications/"+n.Notifications[0].ID+"/read", nil, bob.Token), http.StatusNotFound, "mark other user's notification")
	expect(t, env.do(t, http.MethodPost, "/notifications/read-all", nil, alice.Token), http.StatusNoContent, "read all")
	if decode[notificationsPayload](t, env.do(t, http.MethodGet, "/notifications", nil, alice.Token)).UnreadCount != 0 {
		t.Fatal("all notifications must be read")
	}
}

func TestPassAndUnmatch(t *testing.T) {
	env := newEnv(t)
	a := env.onboard(t, "pass_a", profileSpec{Name: "Anna", Gender: "woman", InterestedIn: []string{"man"}})
	b := env.onboard(t, "pass_b", profileSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}})
	c := env.onboard(t, "pass_c", profileSpec{Name: "Chris", Gender: "man", InterestedIn: []string{"woman"}})

	expect(t, env.swipe(t, a, c.ID, "pass"), http.StatusOK, "pass")
	if containsProfile(env.discover(t, a), c.ID) {
		t.Fatal("passed profile must not reappear")
	}
	r := env.swipe(t, c, a.ID, "like")
	if decode[swipeResp](t, r).Matched {
		t.Fatal("like on someone who passed must not match")
	}

	matchID := env.match(t, a, b)
	expect(t, env.do(t, http.MethodPost, "/chats/"+b.ID+"/messages", map[string]string{"content": "hey"}, a.Token), http.StatusCreated, "send")
	expect(t, env.do(t, http.MethodDelete, "/matches/"+matchID, nil, c.Token), http.StatusNotFound, "unmatch someone else's match")
	expect(t, env.do(t, http.MethodDelete, "/matches/"+matchID, nil, a.Token), http.StatusNoContent, "unmatch")
	expect(t, env.do(t, http.MethodGet, "/chats/"+a.ID+"/messages", nil, b.Token), http.StatusNotFound, "chat after unmatch")
	if containsProfile(env.discover(t, a), b.ID) || containsProfile(env.discover(t, b), a.ID) {
		t.Fatal("unmatched pair must not resurface")
	}
	var msgs int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE sender_user_id = $1`, a.ID).Scan(&msgs)
	if msgs != 0 {
		t.Fatal("unmatch must delete the conversation")
	}
}

func TestConcurrentMutualLikesCreateExactlyOneMatch(t *testing.T) {
	env := newEnv(t)
	for i := 0; i < 5; i++ {
		a := env.onboard(t, "race_a", profileSpec{Name: "Ada", Gender: "woman", InterestedIn: []string{"man"}})
		b := env.onboard(t, "race_b", profileSpec{Name: "Bao", Gender: "man", InterestedIn: []string{"woman"}})
		var wg sync.WaitGroup
		results := make([]bool, 2)
		for j, pair := range [][2]member{{a, b}, {b, a}} {
			wg.Add(1)
			go func(j int, actor, target member) {
				defer wg.Done()
				r := env.swipe(t, actor, target.ID, "like")
				if r.Status == http.StatusOK {
					var s swipeResp
					_ = json.Unmarshal(r.Body, &s)
					results[j] = s.Matched
				}
			}(j, pair[0], pair[1])
		}
		wg.Wait()
		var count int
		_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid,$2::uuid) AND user_high_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&count)
		if count != 1 || results[0] == results[1] {
			t.Fatalf("iteration %d: expected exactly one match reported once, rows=%d results=%v", i, count, results)
		}
	}
}

func TestBlockAndReport(t *testing.T) {
	env := newEnv(t)
	a := env.onboard(t, "block_a", profileSpec{Name: "Amel", Gender: "woman", InterestedIn: []string{"man"}})
	b := env.onboard(t, "block_b", profileSpec{Name: "Basile", Gender: "man", InterestedIn: []string{"woman"}})
	c := env.onboard(t, "block_c", profileSpec{Name: "Cyril", Gender: "man", InterestedIn: []string{"woman"}})

	env.match(t, a, b)
	expect(t, env.do(t, http.MethodPost, "/blocks", map[string]string{"userId": a.ID}, a.Token), http.StatusBadRequest, "self block")
	expect(t, env.do(t, http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent, "block")
	expect(t, env.do(t, http.MethodPost, "/chats/"+a.ID+"/messages", map[string]string{"content": "?"}, b.Token), http.StatusNotFound, "blocked user messages")
	expect(t, env.do(t, http.MethodGet, "/profiles/"+a.ID, nil, b.Token), http.StatusNotFound, "blocked user views profile")
	expect(t, env.do(t, http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusNotFound, "blocker views blocked profile")
	expect(t, env.do(t, http.MethodGet, "/users/"+a.ID, nil, b.Token), http.StatusNotFound, "blocked user in directory")
	expect(t, env.swipe(t, b, a.ID, "like"), http.StatusNotFound, "swipe on blocker")
	if containsProfile(env.discover(t, b), a.ID) {
		t.Fatal("blocker must not appear in discovery")
	}
	if len(decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, b.Token)).Chats) != 0 {
		t.Fatal("conversation must be gone after block")
	}
	blocks := decode[struct {
		Blocks []struct {
			UserID string `json:"userId"`
		} `json:"blocks"`
	}](t, env.do(t, http.MethodGet, "/blocks", nil, a.Token))
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != b.ID {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
	expect(t, env.do(t, http.MethodDelete, "/blocks/"+b.ID, nil, a.Token), http.StatusNoContent, "unblock")

	expect(t, env.do(t, http.MethodPost, "/reports", map[string]string{"userId": c.ID, "reason": "nope"}, a.Token), http.StatusBadRequest, "invalid reason")
	expect(t, env.do(t, http.MethodPost, "/reports", map[string]string{"userId": c.ID, "reason": "harassment", "details": "Messages insistants"}, a.Token), http.StatusAccepted, "report")
	expect(t, env.do(t, http.MethodPost, "/reports", map[string]string{"userId": c.ID, "reason": "harassment"}, a.Token), http.StatusAccepted, "duplicate report absorbed")
	var reports int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_user_id = $1 AND reported_user_id = $2`, a.ID, c.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected one open report, got %d", reports)
	}
	if containsProfile(env.discover(t, c), a.ID) {
		t.Fatal("reporting must also block")
	}
}

func TestUsersDirectoryIsPrivate(t *testing.T) {
	env := newEnv(t)
	a := env.onboard(t, "dir_a", profileSpec{Name: "Ines", Gender: "woman", InterestedIn: []string{"man"}})
	b := env.onboard(t, "dir_b", profileSpec{Name: "Hugo", Gender: "man", InterestedIn: []string{"woman"}})
	r := env.do(t, http.MethodGet, "/users?limit=100", nil, a.Token)
	expect(t, r, http.StatusOK, "users")
	if strings.Contains(string(r.Body), "@") {
		t.Fatalf("directory must not expose emails: %s", r)
	}
	expect(t, env.do(t, http.MethodPut, "/profile/privacy", map[string]bool{"discoverable": false, "showAge": false, "showDistance": false}, b.Token), http.StatusOK, "privacy")
	expect(t, env.do(t, http.MethodGet, "/users/"+b.ID, nil, a.Token), http.StatusNotFound, "hidden profile in directory")
	expect(t, env.do(t, http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusNotFound, "hidden profile")
	if containsProfile(env.discover(t, a), b.ID) {
		t.Fatal("non-discoverable profile must not be shown")
	}
}

func TestDeleteAccountErasesData(t *testing.T) {
	env := newEnv(t)
	a := env.onboard(t, "del_a", profileSpec{Name: "Zoé", Gender: "woman", InterestedIn: []string{"man"}})
	b := env.onboard(t, "del_b", profileSpec{Name: "Yann", Gender: "man", InterestedIn: []string{"woman"}})
	env.match(t, a, b)
	expect(t, env.do(t, http.MethodPost, "/chats/"+b.ID+"/messages", map[string]string{"content": "coucou"}, a.Token), http.StatusCreated, "send")
	expect(t, env.do(t, http.MethodPost, "/reports", map[string]string{"userId": b.ID, "reason": "spam"}, a.Token), http.StatusAccepted, "report")
	photoURL := decode[ownProfile](t, env.do(t, http.MethodGet, "/profile", nil, a.Token)).Photos[0].URL

	expect(t, env.do(t, http.MethodDelete, "/me", map[string]string{"password": "Wrong12345"}, a.Token), http.StatusForbidden, "delete with wrong password")
	expect(t, env.do(t, http.MethodDelete, "/me", map[string]string{"password": testPassword}, a.Token), http.StatusNoContent, "delete account")
	expect(t, env.do(t, http.MethodGet, "/me", nil, a.Token), http.StatusUnauthorized, "token after deletion")
	expect(t, env.do(t, http.MethodPost, "/auth/login", map[string]string{"email": a.Email, "password": testPassword}, ""), http.StatusUnauthorized, "login after deletion")

	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, photoURL, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("photo file must be deleted, got %d", rec.Code)
	}
	for table, column := range map[string]string{"profiles": "user_id", "photos": "user_id", "swipes": "actor_user_id", "messages": "sender_user_id", "sessions": "user_id"} {
		var n int
		_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM `+table+` WHERE `+column+` = $1`, a.ID).Scan(&n)
		if n != 0 {
			t.Fatalf("%s still has %d rows for deleted user", table, n)
		}
	}
	if len(decode[chatsPayload](t, env.do(t, http.MethodGet, "/chats", nil, b.Token)).Chats) != 0 {
		t.Fatal("partner must lose the conversation")
	}
	var orphanReports int
	_ = env.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_user_id IS NULL AND reported_user_id = $1`, b.ID).Scan(&orphanReports)
	if orphanReports != 1 {
		t.Fatal("reports must be kept (anonymized) for moderation")
	}
}

func TestRealtimeWebSocketDeliversMessages(t *testing.T) {
	env := newEnv(t)
	a := env.onboard(t, "ws_a", profileSpec{Name: "Maya", Gender: "woman", InterestedIn: []string{"man"}})
	b := env.onboard(t, "ws_b", profileSpec{Name: "Noah", Gender: "man", InterestedIn: []string{"woman"}})
	env.match(t, a, b)

	server := httptest.NewServer(env.router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Unauthenticated sockets are closed.
	bad, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = bad.Write(ctx, websocket.MessageText, []byte(`{"type":"auth","token":"nope"}`))
	if _, _, err := bad.Read(ctx); err == nil {
		t.Fatal("expected unauthenticated socket to be closed")
	}

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	auth, _ := json.Marshal(map[string]string{"type": "auth", "token": a.Token})
	if err := conn.Write(ctx, websocket.MessageText, auth); err != nil {
		t.Fatal(err)
	}
	readEvent := func() map[string]any {
		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var ev map[string]any
		_ = json.Unmarshal(data, &ev)
		return ev
	}
	if ev := readEvent(); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}

	expect(t, env.do(t, http.MethodPost, "/chats/"+a.ID+"/messages", map[string]string{"content": "temps réel"}, b.Token), http.StatusCreated, "send")
	seenMessage := false
	for i := 0; i < 3 && !seenMessage; i++ {
		ev := readEvent()
		if ev["type"] == "message.new" {
			data := ev["data"].(map[string]any)
			if data["content"] != "temps réel" {
				t.Fatalf("unexpected payload %v", data)
			}
			seenMessage = true
		}
	}
	if !seenMessage {
		t.Fatal("message.new not delivered over websocket")
	}
}
