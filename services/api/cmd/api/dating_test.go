package main

import (
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var (
	alice = userSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 29, Interests: []string{"travel", "music"}}
	bob   = userSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Age: 31, Interests: []string{"music", "sport"}}
)

func TestProfileValidationAndOnboarding(t *testing.T) {
	e := newEnv(t)
	r := e.expect(e.do("POST", "/auth/register", "", map[string]string{"email": "n@example.test", "password": "correct-horse-1"}), 201)
	tok := r.json(t)["accessToken"].(string)

	o := e.expect(e.do("GET", "/me/profile", tok, nil), 200).json(t)
	if o["onboardingComplete"] != false || len(o["missing"].([]any)) != 3 {
		t.Fatalf("fresh account should miss profile/preferences/photos: %v", o)
	}
	// Photos need a profile first.
	e.expect(e.upload("POST", "/me/photos", tok, pngBytes(50, 50), "photo"), 409)

	adult := time.Now().AddDate(-25, 0, 0).Format("2006-01-02")
	minor := time.Now().AddDate(-17, 0, -1).Format("2006-01-02")
	justAdult := time.Now().AddDate(-18, 0, -1).Format("2006-01-02")
	base := func(m map[string]any) map[string]any {
		out := map[string]any{"firstName": "Nina", "birthDate": adult, "gender": "woman", "bio": "", "city": "Paris", "interests": []string{}}
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	for name, body := range map[string]map[string]any{
		"minor":           base(map[string]any{"birthDate": minor}),
		"future":          base(map[string]any{"birthDate": time.Now().AddDate(1, 0, 0).Format("2006-01-02")}),
		"ancient":         base(map[string]any{"birthDate": "1900-01-01"}),
		"bad date":        base(map[string]any{"birthDate": "12/03/1990"}),
		"empty name":      base(map[string]any{"firstName": "   "}),
		"long name":       base(map[string]any{"firstName": strings.Repeat("a", 41)}),
		"bad gender":      base(map[string]any{"gender": "robot"}),
		"long bio":        base(map[string]any{"bio": strings.Repeat("a", 501)}),
		"unknown int":     base(map[string]any{"interests": []string{"nope"}}),
		"too many int":    base(map[string]any{"interests": []string{"travel", "cooking", "music", "cinema", "reading", "sport", "hiking", "art", "photography", "gaming", "dancing"}}),
		"control in name": base(map[string]any{"firstName": "Ni\x00na"}),
	} {
		if r := e.do("PUT", "/me/profile", tok, body); r.Status != 400 {
			t.Errorf("%s: expected 400, got %d (%s)", name, r.Status, r.Body)
		}
	}
	e.expect(e.do("PUT", "/me/profile", tok, base(map[string]any{"birthDate": justAdult, "interests": []string{"travel", "travel"}})), 200)

	for name, body := range map[string]map[string]any{
		"no gender":  {"interestedIn": []string{}, "minAge": 18, "maxAge": 40},
		"bad gender": {"interestedIn": []string{"x"}, "minAge": 18, "maxAge": 40},
		"under 18":   {"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40},
		"inverted":   {"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30},
		"over 99":    {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 120},
		"distance":   {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 0},
	} {
		if r := e.do("PUT", "/me/preferences", tok, body); r.Status != 400 {
			t.Errorf("prefs %s: expected 400, got %d", name, r.Status)
		}
	}
	e.expect(e.do("PUT", "/me/location", tok, map[string]any{"latitude": 91, "longitude": 0}), 400)
	e.expect(e.do("PUT", "/me/preferences", tok, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 50}), 200)
	e.expect(e.upload("POST", "/me/photos", tok, pngBytes(50, 50), "photo"), 201)
	o = e.expect(e.do("GET", "/me/profile", tok, nil), 200).json(t)
	if o["onboardingComplete"] != true {
		t.Fatalf("onboarding should be complete: %v", o)
	}
	if p := o["profile"].(map[string]any); p["age"].(float64) != 18 || len(p["interests"].([]any)) != 1 {
		t.Fatalf("unexpected profile: %v", p)
	}
}

func TestLocationIsCoarseAndNeverExposed(t *testing.T) {
	e := newEnv(t)
	a := e.onboard(userSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 29, Lat: ptr(45.764043), Lng: ptr(4.835659)})
	b := e.onboard(userSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Age: 31, Lat: ptr(45.78), Lng: ptr(4.86)})
	var lat float64
	if err := e.pool.QueryRow(t.Context(), `SELECT latitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat); err != nil || lat != 45.76 {
		t.Fatalf("stored latitude should be rounded to 2 decimals, got %v (%v)", lat, err)
	}
	r := e.expect(e.do("GET", "/discover", a.Token, nil), 200)
	body := string(r.Body)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email", "45.7"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("feed leaks %q: %s", forbidden, body)
		}
	}
	p := e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200).json(t)
	if d, ok := p["distanceKm"].(float64); !ok || d < 1 || d > 5 {
		t.Fatalf("expected approximate distance, got %v", p["distanceKm"])
	}
	// Hiding distance removes it.
	e.expect(e.do("PUT", "/me/privacy", b.Token, map[string]bool{"isDiscoverable": true, "showDistance": false}), 200)
	p = e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200).json(t)
	if p["distanceKm"] != nil {
		t.Fatalf("distance should be hidden: %v", p["distanceKm"])
	}
	// Clearing location removes coordinates.
	e.expect(e.do("DELETE", "/me/location", a.Token, nil), 200)
	if err := e.pool.QueryRow(t.Context(), `SELECT COALESCE(latitude, -1) FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat); err != nil || lat != -1 {
		t.Fatalf("location should be cleared: %v", lat)
	}
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	u := e.onboard(userSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Age: 29, NoPhoto: true})
	other := e.onboard(bob)

	// Not an image, even with an image name/type.
	e.expect(e.upload("POST", "/me/photos", u.Token, []byte("<?php echo 1; ?>"), "photo"), 415)
	e.expect(e.upload("POST", "/me/photos", u.Token, append([]byte("GIF89a"), make([]byte, 100)...), "photo"), 415)
	// Truncated PNG: right magic, undecodable.
	e.expect(e.upload("POST", "/me/photos", u.Token, pngBytes(40, 40)[:30], "photo"), 400)
	e.expect(e.upload("POST", "/me/photos", u.Token, pngBytes(40, 40), "wrong"), 400)
	// Oversized upload.
	e.expect(e.upload("POST", "/me/photos", u.Token, make([]byte, 9<<20), "photo"), 413)

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.expect(e.upload("POST", "/me/photos", u.Token, pngBytes(2000+i, 1500), "photo"), 201)
		ids = append(ids, r.json(t)["id"].(string))
	}
	e.expect(e.upload("POST", "/me/photos", u.Token, pngBytes(50, 50), "photo"), 409)

	// Stored image is a re-encoded, downsized JPEG.
	img := e.expect(e.do("GET", "/photos/"+ids[0], u.Token, nil), 200)
	if len(img.Body) < 3 || img.Body[0] != 0xFF || img.Body[1] != 0xD8 {
		t.Fatal("stored photo is not a JPEG")
	}
	var w int
	_ = e.pool.QueryRow(t.Context(), `SELECT width FROM photos WHERE id = $1`, ids[0]).Scan(&w)
	if w != 1280 {
		t.Fatalf("expected downscale to 1280, got %d", w)
	}

	// Reorder: main photo = position 0.
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	e.expect(e.do("PUT", "/me/photos/order", u.Token, map[string]any{"ids": rev}), 200)
	o := e.expect(e.do("GET", "/me/profile", u.Token, nil), 200).json(t)
	if first := o["profile"].(map[string]any)["photos"].([]any)[0].(map[string]any)["id"]; first != ids[5] {
		t.Fatalf("main photo should be %s, got %v", ids[5], first)
	}
	e.expect(e.do("PUT", "/me/photos/order", u.Token, map[string]any{"ids": rev[:5]}), 400)
	e.expect(e.do("PUT", "/me/photos/order", u.Token, map[string]any{"ids": append(rev[:5:5], rev[0])}), 400)

	// Replace keeps id and position; others cannot replace/delete.
	e.expect(e.upload("PUT", "/me/photos/"+ids[5], u.Token, pngBytes(100, 100), "photo"), 200)
	e.expect(e.upload("PUT", "/me/photos/"+ids[5], other.Token, pngBytes(100, 100), "photo"), 404)
	e.expect(e.do("DELETE", "/me/photos/"+ids[5], other.Token, nil), 404)
	e.expect(e.do("DELETE", "/me/photos/not-a-uuid", u.Token, nil), 404)

	// Delete compacts positions and promotes the next photo.
	e.expect(e.do("DELETE", "/me/photos/"+ids[5], u.Token, nil), 204)
	o = e.expect(e.do("GET", "/me/profile", u.Token, nil), 200).json(t)
	photos := o["profile"].(map[string]any)["photos"].([]any)
	if len(photos) != 5 || photos[0].(map[string]any)["id"] != ids[4] || photos[4].(map[string]any)["position"].(float64) != 4 {
		t.Fatalf("positions not compacted: %v", photos)
	}

	// Visibility: anyone eligible can see it unauthenticated cannot; blocked cannot.
	e.expect(e.do("GET", "/photos/"+ids[4], "", nil), 401)
	e.expect(e.do("GET", "/photos/"+ids[4], other.Token, nil), 200)
	e.expect(e.do("POST", "/blocks", u.Token, map[string]string{"userId": other.ID}), 204)
	e.expect(e.do("GET", "/photos/"+ids[4], other.Token, nil), 404)
	e.expect(e.do("GET", "/photos/"+ids[4], u.Token, nil), 200)
}

func TestDiscoveryRules(t *testing.T) {
	e := newEnv(t)
	lyon := userSpec{Lat: ptr(45.76), Lng: ptr(4.84)}
	mk := func(s userSpec) user {
		s.Lat, s.Lng = lyon.Lat, lyon.Lng
		return e.onboard(s)
	}
	me := mk(userSpec{Name: "Me", Gender: "woman", InterestedIn: []string{"man"}, Age: 30, MinAge: 25, MaxAge: 40, MaxDistance: ptr(50), Interests: []string{"travel"}})
	ok := mk(userSpec{Name: "Ok", Gender: "man", InterestedIn: []string{"woman"}, Age: 33, MinAge: 20, MaxAge: 40, Interests: []string{"travel"}})
	wrongGender := mk(userSpec{Name: "Wrongg", Gender: "woman", InterestedIn: []string{"woman", "man"}, Age: 30})
	notInterested := mk(userSpec{Name: "Notint", Gender: "man", InterestedIn: []string{"man"}, Age: 30})
	tooYoung := mk(userSpec{Name: "Young", Gender: "man", InterestedIn: []string{"woman"}, Age: 22})
	tooOldForThem := mk(userSpec{Name: "Picky", Gender: "man", InterestedIn: []string{"woman"}, Age: 33, MinAge: 18, MaxAge: 28})
	paused := mk(userSpec{Name: "Paused", Gender: "man", InterestedIn: []string{"woman"}, Age: 33})
	e.expect(e.do("PUT", "/me/privacy", paused.Token, map[string]bool{"isDiscoverable": false, "showDistance": true}), 200)
	nophoto := mk(userSpec{Name: "Nophoto", Gender: "man", InterestedIn: []string{"woman"}, Age: 33, NoPhoto: true})
	far := e.onboard(userSpec{Name: "Far", Gender: "man", InterestedIn: []string{"woman"}, Age: 33, Lat: ptr(48.85), Lng: ptr(2.35)})         // Paris ~390km
	noloc := e.onboard(userSpec{Name: "Noloc", Gender: "man", InterestedIn: []string{"woman"}, Age: 33})                                      // no coords, I require ≤50km
	themLocal := e.onboard(userSpec{Name: "Local", Gender: "man", InterestedIn: []string{"woman"}, Age: 33, Lat: ptr(45.77), Lng: ptr(4.85)}) // ok
	blocked := mk(userSpec{Name: "Blocked", Gender: "man", InterestedIn: []string{"woman"}, Age: 33})
	blocker := mk(userSpec{Name: "Blocker", Gender: "man", InterestedIn: []string{"woman"}, Age: 33})
	e.expect(e.do("POST", "/blocks", me.Token, map[string]string{"userId": blocked.ID}), 204)
	e.expect(e.do("POST", "/blocks", blocker.Token, map[string]string{"userId": me.ID}), 204)

	ids := feedIDs(t, e, me)
	for name, want := range map[string]bool{"ok": true, "local": true} {
		id := map[string]string{"ok": ok.ID, "local": themLocal.ID}[name]
		if contains(ids, id) != want {
			t.Errorf("%s should be in feed (%v)", name, ids)
		}
	}
	for name, u := range map[string]user{"wrongGender": wrongGender, "notInterested": notInterested, "tooYoung": tooYoung,
		"tooOldForThem": tooOldForThem, "paused": paused, "nophoto": nophoto, "far": far, "noloc": noloc, "blocked": blocked, "blocker": blocker, "self": me} {
		if contains(ids, u.ID) {
			t.Errorf("%s must not be in feed", name)
		}
	}
	// Shared interests rank first.
	if ids[0] != ok.ID {
		t.Errorf("profile with shared interest should rank first, got %v", ids)
	}

	// Acting on a card removes it; pass is final and never reappears.
	e.swipe(me, ok, "pass")
	if contains(feedIDs(t, e, me), ok.ID) {
		t.Error("passed profile reappeared")
	}
	// Someone who passed me disappears from my feed.
	e.swipe(themLocal, me, "pass")
	if contains(feedIDs(t, e, me), themLocal.ID) {
		t.Error("profile that passed me still shown")
	}

	// Swiping on ineligible users is refused server-side.
	for _, u := range []user{wrongGender, tooYoung, blocked, far} {
		e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"userId": u.ID, "action": "like"}), 403)
	}
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"userId": me.ID, "action": "like"}), 403)
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"userId": "nope", "action": "like"}), 400)
	e.expect(e.do("POST", "/swipes", me.Token, map[string]string{"userId": ok.ID, "action": "super"}), 400)
	// Hidden/ineligible profiles cannot be opened either.
	e.expect(e.do("GET", "/profiles/"+paused.ID, me.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+blocked.ID, me.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+me.ID, me.Token, nil), 404)
	e.expect(e.do("GET", "/discover?limit=999", me.Token, nil), 400)
}

func TestMatchingNoDuplicates(t *testing.T) {
	e := newEnv(t)
	a, b := e.onboard(alice), e.onboard(bob)

	r := e.swipe(a, b, "like")
	if r["matched"] != false || r["alreadySwiped"] != false {
		t.Fatalf("first like must not match: %v", r)
	}
	// Repeated like is idempotent.
	r = e.swipe(a, b, "like")
	if r["alreadySwiped"] != true || r["matched"] != false {
		t.Fatalf("repeat like should be flagged: %v", r)
	}
	// A pass after a like cannot overturn it.
	if r = e.swipe(a, b, "pass"); r["alreadySwiped"] != true {
		t.Fatalf("swipes are final: %v", r)
	}
	// Liked-me profiles are boosted in B's feed, and A is not hidden from B.
	if !contains(feedIDs(t, e, b), a.ID) {
		t.Fatal("A should be in B's feed")
	}
	r = e.swipe(b, a, "like")
	if r["matched"] != true || r["matchId"] == "" || r["conversationId"] == "" {
		t.Fatalf("mutual like must match: %v", r)
	}
	r2 := e.swipe(b, a, "like")
	if r2["matched"] != true || r2["matchId"] != r["matchId"] || r2["alreadySwiped"] != true {
		t.Fatalf("repeat after match must return same match: %v", r2)
	}
	var matches, convs, parts int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches`).Scan(&matches)
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM conversations`).Scan(&convs)
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM conversation_participants`).Scan(&parts)
	if matches != 1 || convs != 1 || parts != 2 {
		t.Fatalf("expected 1 match/1 conversation/2 participants, got %d/%d/%d", matches, convs, parts)
	}
	// Both disappear from each other's feed.
	if contains(feedIDs(t, e, a), b.ID) || contains(feedIDs(t, e, b), a.ID) {
		t.Fatal("matched users must leave the feed")
	}
	// A pass never produces a match.
	c := e.onboard(userSpec{Name: "Carl", Gender: "man", InterestedIn: []string{"woman"}, Age: 33})
	e.swipe(c, a, "pass")
	e.expect(e.do("POST", "/swipes", a.Token, map[string]string{"userId": c.ID, "action": "like"}), 403) // c passed a → ineligible
}

func TestSimultaneousMutualLikesCreateExactlyOneMatch(t *testing.T) {
	e := newEnv(t)
	for round := 0; round < 5; round++ {
		a := e.onboard(userSpec{Name: fmtID("Anna", round), Gender: "woman", InterestedIn: []string{"man"}, Age: 29})
		b := e.onboard(userSpec{Name: fmtID("Ben", round), Gender: "man", InterestedIn: []string{"woman"}, Age: 31})
		var wg sync.WaitGroup
		results := make([]map[string]any, 2)
		for i, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = e.do("POST", "/swipes", pair[0].Token, map[string]string{"userId": pair[1].ID, "action": "like"}).json(t)
			}()
		}
		wg.Wait()
		if results[0]["matched"] != true && results[1]["matched"] != true {
			t.Fatalf("round %d: simultaneous mutual likes produced no match: %v", round, results)
		}
		var n int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: %d matches", round, n)
		}
	}
}

func TestChatPermissionsAndReadState(t *testing.T) {
	e := newEnv(t)
	a, b := e.onboard(alice), e.onboard(bob)
	outsider := e.onboard(userSpec{Name: "Eve", Gender: "man", InterestedIn: []string{"woman"}, Age: 40})
	conv := e.match(a, b)

	// A new match shows up as an empty conversation for both.
	for _, u := range []user{a, b} {
		r := e.expect(e.do("GET", "/conversations", u.Token, nil), 200).json(t)
		cs := r["conversations"].([]any)
		if len(cs) != 1 || cs[0].(map[string]any)["lastMessage"] != nil {
			t.Fatalf("expected one empty conversation: %v", cs)
		}
	}
	// Outsider can neither read, write, mark read nor hide.
	e.expect(e.do("GET", "/conversations/"+conv+"/messages", outsider.Token, nil), 404)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", outsider.Token, map[string]string{"body": "hi"}), 404)
	e.expect(e.do("POST", "/conversations/"+conv+"/read", outsider.Token, nil), 404)
	e.expect(e.do("DELETE", "/conversations/"+conv, outsider.Token, nil), 404)
	e.expect(e.do("GET", "/conversations/not-a-uuid/messages", a.Token, nil), 404)

	// Message validation.
	for _, body := range []string{"", "   ", strings.Repeat("a", 2001), "bad\x00char"} {
		e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": body}), 400)
	}
	m1 := e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "  Salut Bob !  "}), 201).json(t)
	if m1["body"] != "Salut Bob !" || m1["read"] != false {
		t.Fatalf("unexpected message: %v", m1)
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Ça va ?"}), 201)

	// Unread count for B, none for A.
	cs := e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t)["conversations"].([]any)
	if cs[0].(map[string]any)["unreadCount"].(float64) != 2 {
		t.Fatalf("B should have 2 unread: %v", cs[0])
	}
	cs = e.expect(e.do("GET", "/conversations", a.Token, nil), 200).json(t)["conversations"].([]any)
	if cs[0].(map[string]any)["unreadCount"].(float64) != 0 {
		t.Fatalf("A should have 0 unread: %v", cs[0])
	}

	// Read receipts.
	msgs := e.expect(e.do("GET", "/conversations/"+conv+"/messages", a.Token, nil), 200).json(t)["messages"].([]any)
	if msgs[0].(map[string]any)["body"] != "Ça va ?" || msgs[0].(map[string]any)["read"] != false {
		t.Fatalf("newest first, unread: %v", msgs)
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 204)
	msgs = e.expect(e.do("GET", "/conversations/"+conv+"/messages", a.Token, nil), 200).json(t)["messages"].([]any)
	if msgs[0].(map[string]any)["read"] != true {
		t.Fatalf("message should be read: %v", msgs)
	}
	cs = e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t)["conversations"].([]any)
	if cs[0].(map[string]any)["unreadCount"].(float64) != 0 {
		t.Fatal("unread should be cleared")
	}

	// Pagination: before=<id> returns only older messages.
	id := msgs[0].(map[string]any)["id"].(float64)
	older := e.expect(e.do("GET", "/conversations/"+conv+"/messages?limit=1&before="+itoa(int(id)), a.Token, nil), 200).json(t)["messages"].([]any)
	if len(older) != 1 || older[0].(map[string]any)["body"] != "Salut Bob !" {
		t.Fatalf("pagination failed: %v", older)
	}

	// Local delete hides history for the deleter only, and a new message revives it.
	e.expect(e.do("DELETE", "/conversations/"+conv, b.Token, nil), 204)
	if cs = e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t)["conversations"].([]any); len(cs) != 0 {
		t.Fatal("hidden conversation still listed")
	}
	if cs = e.expect(e.do("GET", "/conversations", a.Token, nil), 200).json(t)["conversations"].([]any); len(cs) != 1 {
		t.Fatal("other side must keep conversation")
	}
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "Tu es là ?"}), 201)
	cs = e.expect(e.do("GET", "/conversations", b.Token, nil), 200).json(t)["conversations"].([]any)
	if len(cs) != 1 {
		t.Fatal("new message should revive the conversation")
	}
	bm := e.expect(e.do("GET", "/conversations/"+conv+"/messages", b.Token, nil), 200).json(t)["messages"].([]any)
	if len(bm) != 1 {
		t.Fatalf("B should only see messages after local delete, got %d", len(bm))
	}
}

func TestBlockUnmatchAndReport(t *testing.T) {
	e := newEnv(t)
	a, b, c := e.onboard(alice), e.onboard(bob), e.onboard(userSpec{Name: "Carl", Gender: "man", InterestedIn: []string{"woman"}, Age: 35})
	convAB := e.match(a, b)
	convAC := e.match(a, c)

	// Unmatch: only a party, ends access for both, cannot be re-liked.
	matchID := e.pool
	var mid string
	_ = matchID.QueryRow(t.Context(), `SELECT match_id FROM conversations WHERE id = $1`, convAC).Scan(&mid)
	e.expect(e.do("DELETE", "/matches/"+mid, b.Token, nil), 404)
	e.expect(e.do("DELETE", "/matches/"+mid, c.Token, nil), 204)
	e.expect(e.do("DELETE", "/matches/"+mid, c.Token, nil), 404)
	for _, u := range []user{a, c} {
		e.expect(e.do("GET", "/conversations/"+convAC+"/messages", u.Token, nil), 404)
		e.expect(e.do("POST", "/conversations/"+convAC+"/messages", u.Token, map[string]string{"body": "x"}), 404)
	}
	if contains(feedIDs(t, e, a), c.ID) {
		t.Fatal("unmatched user must not reappear")
	}

	// Block ends the match and hides everything in both directions.
	e.expect(e.do("POST", "/conversations/"+convAB+"/messages", b.Token, map[string]string{"body": "hello"}), 201)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": a.ID}), 400)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": "0b1c2d3e-0000-4000-8000-000000000000"}), 404)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), 204)
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), 204) // idempotent
	for _, u := range []user{a, b} {
		e.expect(e.do("GET", "/conversations/"+convAB+"/messages", u.Token, nil), 404)
		e.expect(e.do("POST", "/conversations/"+convAB+"/messages", u.Token, map[string]string{"body": "x"}), 404)
		cs := e.expect(e.do("GET", "/conversations", u.Token, nil), 200).json(t)["conversations"].([]any)
		if len(cs) != 0 {
			t.Fatalf("blocked conversation still listed: %v", cs)
		}
	}
	bl := e.expect(e.do("GET", "/blocks", a.Token, nil), 200).json(t)["blocks"].([]any)
	if len(bl) != 1 || bl[0].(map[string]any)["id"] != b.ID {
		t.Fatalf("block list: %v", bl)
	}
	if got := e.expect(e.do("GET", "/blocks", b.Token, nil), 200).json(t)["blocks"].([]any); len(got) != 0 {
		t.Fatal("the blocked user must not see who blocked them")
	}
	// Unblock does not restore the ended match.
	e.expect(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), 204)
	e.expect(e.do("GET", "/conversations/"+convAB+"/messages", a.Token, nil), 404)

	// Reports.
	rep := func(tok string, body map[string]any) resp { return e.do("POST", "/reports", tok, body) }
	e.expect(rep(a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "insultes"}), 204)
	e.expect(rep(a.Token, map[string]any{"userId": b.ID, "reason": "harassment", "details": "insultes"}), 204) // deduped
	e.expect(rep(a.Token, map[string]any{"userId": b.ID, "reason": "bogus"}), 400)
	e.expect(rep(a.Token, map[string]any{"userId": a.ID, "reason": "spam"}), 400)
	e.expect(rep(a.Token, map[string]any{"userId": "0b1c2d3e-0000-4000-8000-000000000000", "reason": "spam"}), 404)
	e.expect(rep(a.Token, map[string]any{"userId": c.ID, "reason": "spam", "alsoBlock": true}), 204)
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1`, a.ID).Scan(&n)
	if n != 2 {
		t.Fatalf("expected 2 stored reports, got %d", n)
	}
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM blocks WHERE blocker_id = $1 AND blocked_id = $2`, a.ID, c.ID).Scan(&n)
	if n != 1 {
		t.Fatal("alsoBlock should block")
	}
}

func TestNotifications(t *testing.T) {
	e := newEnv(t)
	a, b := e.onboard(alice), e.onboard(bob)
	conv := e.match(a, b)
	waitFor(t, func() bool { return unread(t, e, a) == 1 && unread(t, e, b) == 1 }, "match notifications")

	for i := 0; i < 3; i++ {
		e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "msg"}), 201)
	}
	// Message notifications are coalesced per conversation: match + 1 message.
	waitFor(t, func() bool { return unread(t, e, b) == 2 }, "coalesced message notification")
	if unread(t, e, a) != 1 {
		t.Fatal("sender must not be notified of own messages")
	}

	page := e.expect(e.do("GET", "/notifications?limit=1", b.Token, nil), 200).json(t)
	if len(page["notifications"].([]any)) != 1 || page["nextCursor"] == nil {
		t.Fatalf("expected paginated list: %v", page)
	}
	next := page["nextCursor"].(string)
	page2 := e.expect(e.do("GET", "/notifications?limit=1&cursor="+next, b.Token, nil), 200).json(t)
	if len(page2["notifications"].([]any)) != 1 || page2["notifications"].([]any)[0].(map[string]any)["id"] == page["notifications"].([]any)[0].(map[string]any)["id"] {
		t.Fatalf("second page wrong: %v", page2)
	}
	e.expect(e.do("GET", "/notifications?cursor=garbage", b.Token, nil), 400)

	// Reading a conversation clears its message notification; others can't touch mine.
	e.expect(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 204)
	waitFor(t, func() bool { return unread(t, e, b) == 1 }, "message notification read with conversation")
	id := page["notifications"].([]any)[0].(map[string]any)["id"].(string)
	e.expect(e.do("POST", "/notifications/"+id+"/read", a.Token, nil), 404)
	e.expect(e.do("POST", "/notifications/read-all", b.Token, nil), 204)
	if unread(t, e, b) != 0 {
		t.Fatal("read-all should clear")
	}
}

func TestRealtimeWebSocket(t *testing.T) {
	e := newEnv(t)
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	a, b := e.onboard(alice), e.onboard(bob)

	dial := func(u user) *websocket.Conn {
		tk := e.expect(e.do("POST", "/ws/ticket", u.Token, nil), 200).json(t)["ticket"].(string)
		url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?ticket=" + tk
		c, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		// Tickets are single use.
		if _, _, err := websocket.DefaultDialer.Dial(url, nil); err == nil {
			t.Fatal("ticket reused")
		}
		return c
	}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket=forged", nil); err == nil {
		t.Fatal("forged ticket accepted")
	}
	e.expect(e.do("POST", "/ws/ticket", "", nil), 401)

	wa, wb := dial(a), dial(b)
	defer wa.Close()
	defer wb.Close()
	next := func(c *websocket.Conn, typ string) map[string]any {
		t.Helper()
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var ev struct {
				Type string         `json:"type"`
				Data map[string]any `json:"data"`
			}
			if err := c.ReadJSON(&ev); err != nil {
				t.Fatalf("waiting for %s: %v", typ, err)
			}
			if ev.Type == typ {
				return ev.Data
			}
		}
	}

	conv := e.match(a, b)
	if d := next(wa, "match.new"); d["conversationId"] != conv || d["peerName"] != "Bob" {
		t.Fatalf("A match.new: %v", d)
	}
	next(wb, "match.new")
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "temps réel"}), 201)
	if d := next(wb, "message.new"); d["body"] != "temps réel" || d["senderId"] != a.ID {
		t.Fatalf("B message.new: %v", d)
	}
	next(wb, "notification.new")
	e.expect(e.do("POST", "/conversations/"+conv+"/read", b.Token, nil), 204)
	if d := next(wa, "message.read"); d["conversationId"] != conv {
		t.Fatalf("A message.read: %v", d)
	}
	e.expect(e.do("POST", "/blocks", a.Token, map[string]string{"userId": b.ID}), 204)
	next(wb, "match.ended")
}

func TestPrivacyAndAccountIsolation(t *testing.T) {
	e := newEnv(t)
	a, b := e.onboard(alice), e.onboard(bob)
	// Every authenticated route refuses anonymous access.
	for _, route := range [][2]string{
		{"GET", "/me/profile"}, {"PUT", "/me/profile"}, {"PUT", "/me/preferences"}, {"PUT", "/me/privacy"}, {"PUT", "/me/location"},
		{"GET", "/me/photos"}, {"POST", "/me/photos"}, {"GET", "/photos/x"}, {"GET", "/discover"}, {"GET", "/profiles/x"},
		{"POST", "/swipes"}, {"GET", "/conversations"}, {"GET", "/blocks"}, {"POST", "/reports"}, {"GET", "/notifications"}, {"GET", "/interests"},
	} {
		if r := e.do(route[0], route[1], "", nil); r.Status != 401 {
			t.Errorf("%s %s without token: %d", route[0], route[1], r.Status)
		}
	}
	// Pausing the profile removes it from feeds but keeps matches working.
	conv := e.match(a, b)
	e.expect(e.do("PUT", "/me/privacy", b.Token, map[string]bool{"isDiscoverable": false, "showDistance": true}), 200)
	e.expect(e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]string{"body": "toujours là"}), 201)
	e.expect(e.do("GET", "/profiles/"+b.ID, a.Token, nil), 200) // matched → visible
	// Interests catalog.
	list := e.expect(e.do("GET", "/interests", a.Token, nil), 200).json(t)["interests"].([]any)
	if len(list) < 10 {
		t.Fatal("interests catalog missing")
	}
}

func unread(t *testing.T, e *env, u user) int {
	t.Helper()
	return int(e.expect(e.do("GET", "/notifications/unread-count", u.Token, nil), 200).json(t)["unread"].(float64))
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func itoa(n int) string { return strconv.Itoa(n) }
