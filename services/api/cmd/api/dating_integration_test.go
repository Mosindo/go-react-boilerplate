package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func birthFor(age int) string { return time.Now().AddDate(-age, 0, -2).Format("2006-01-02") }

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newTestEnv(t)
	u := e.register(t, "prof")
	base := func() map[string]any {
		return map[string]any{"firstName": "Camille", "birthDate": birthFor(29), "gender": "woman", "bio": "Hello", "city": "Lyon"}
	}

	e.expect(t, e.do(t, http.MethodGet, "/me/profile", nil, u.Token), http.StatusNotFound, "profile before onboarding")

	bad := func(mutate func(map[string]any), what string) {
		t.Helper()
		p := base()
		mutate(p)
		e.expect(t, e.do(t, http.MethodPut, "/me/profile", p, u.Token), http.StatusBadRequest, what)
	}
	bad(func(p map[string]any) { p["birthDate"] = birthFor(17) }, "underage")
	bad(func(p map[string]any) { p["birthDate"] = time.Now().AddDate(-18, 0, 1).Format("2006-01-02") }, "turns 18 tomorrow")
	bad(func(p map[string]any) { p["birthDate"] = "12/05/1990" }, "bad date format")
	bad(func(p map[string]any) { p["birthDate"] = birthFor(120) }, "implausible age")
	bad(func(p map[string]any) { p["gender"] = "robot" }, "bad gender")
	bad(func(p map[string]any) { p["firstName"] = "   " }, "blank name")
	bad(func(p map[string]any) { p["firstName"] = "1234" }, "name without letters")
	bad(func(p map[string]any) { p["bio"] = strings.Repeat("é", 501) }, "bio too long")
	bad(func(p map[string]any) { p["interests"] = []string{"does-not-exist"} }, "unknown interest")
	bad(func(p map[string]any) {
		p["interests"] = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	}, "too many interests")

	p := base()
	p["firstName"] = "  Camille\t  Dupont "
	p["interests"] = []string{"music", "music", "art"}
	resp := e.expect(t, e.do(t, http.MethodPut, "/me/profile", p, u.Token), http.StatusOK, "valid profile")
	var own struct {
		FirstName    string `json:"firstName"`
		Age          int    `json:"age"`
		Complete     bool   `json:"complete"`
		Discoverable bool   `json:"discoverable"`
		HasLocation  bool   `json:"hasLocation"`
		Interests    []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
	}
	resp.decode(t, &own)
	if own.FirstName != "Camille Dupont" || own.Age != 29 || own.Complete || !own.Discoverable || len(own.Interests) != 2 {
		t.Fatalf("unexpected own profile: %+v", own)
	}

	// birth date is immutable
	p["birthDate"] = birthFor(40)
	e.expect(t, e.do(t, http.MethodPut, "/me/profile", p, u.Token), http.StatusBadRequest, "birth date change")
	delete(p, "birthDate")
	e.expect(t, e.do(t, http.MethodPut, "/me/profile", p, u.Token), http.StatusOK, "update without birth date")

	// preferences validation
	prefs := func(m map[string]any, status int, what string) {
		t.Helper()
		e.expect(t, e.do(t, http.MethodPut, "/me/preferences", m, u.Token), status, what)
	}
	prefs(map[string]any{"interestedIn": []string{}, "ageMin": 18, "ageMax": 40, "maxDistanceKm": 20}, 400, "no genders")
	prefs(map[string]any{"interestedIn": []string{"man"}, "ageMin": 17, "ageMax": 40, "maxDistanceKm": 20}, 400, "min age < 18")
	prefs(map[string]any{"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 25, "maxDistanceKm": 20}, 400, "min > max")
	prefs(map[string]any{"interestedIn": []string{"alien"}, "ageMin": 18, "ageMax": 40, "maxDistanceKm": 20}, 400, "bad gender")
	prefs(map[string]any{"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 40, "maxDistanceKm": -3}, 400, "bad distance")
	prefs(map[string]any{"interestedIn": []string{"man", "woman"}, "ageMin": 25, "ageMax": 45, "maxDistanceKm": 30}, 200, "valid prefs")

	// location: validated, rounded server side, never exposed
	e.expect(t, e.do(t, http.MethodPut, "/me/location", map[string]any{"latitude": 95, "longitude": 2}, u.Token), http.StatusBadRequest, "latitude out of range")
	e.expect(t, e.do(t, http.MethodPut, "/me/location", map[string]any{"latitude": 45.764043, "longitude": 4.835659, "city": "Lyon"}, u.Token), http.StatusNoContent, "set location")
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 45.76 || lng != 4.84 {
		t.Fatalf("coordinates must be rounded to 0.01, got %v,%v", lat, lng)
	}
	raw := e.expect(t, e.do(t, http.MethodGet, "/me/profile", nil, u.Token), http.StatusOK, "own profile")
	for _, forbidden := range []string{"latitude", "longitude", "45.76", "password"} {
		if strings.Contains(string(raw.Body), forbidden) {
			t.Fatalf("own profile leaks %q: %s", forbidden, raw.Body)
		}
	}
}

func TestPhotoManagement(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	owner := e.onboard(t, ar, userOpts{Name: "Owner", NoPhoto: true})
	stranger := e.onboard(t, ar, userOpts{Name: "Stranger"})

	// rejected inputs
	e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, []byte("not an image at all")), http.StatusBadRequest, "text file")
	e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, append([]byte("GIF89a"), make([]byte, 100)...)), http.StatusBadRequest, "gif")
	e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, jpegBytes(t, 100, 100)), http.StatusBadRequest, "too small")
	e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, append(jpegBytes(t, 400, 400), make([]byte, 6<<20)...)), http.StatusRequestEntityTooLarge, "over 5MB")

	// accepted: re-encoded, downscaled, EXIF-free JPEG capped at 1080px
	first := e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, jpegBytes(t, 3000, 2000)), http.StatusCreated, "upload big photo")
	var p1 struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
		Width    int    `json:"width"`
		URL      string `json:"url"`
	}
	first.decode(t, &p1)
	if p1.Position != 0 || p1.Width != 1080 {
		t.Fatalf("expected position 0 and width 1080, got %+v", p1)
	}
	file := e.expect(t, e.do(t, http.MethodGet, p1.URL, nil, owner.Token), http.StatusOK, "owner reads own photo")
	if file.Header.Get("Content-Type") != "image/jpeg" || len(file.Body) < 3 || file.Body[0] != 0xFF || file.Body[1] != 0xD8 {
		t.Fatal("photo must be served as JPEG")
	}

	ids := []string{p1.ID}
	for i := 1; i < 6; i++ {
		r := e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, jpegBytes(t, 400, 500)), http.StatusCreated, "upload")
		var p struct {
			ID string `json:"id"`
		}
		r.decode(t, &p)
		ids = append(ids, p.ID)
	}
	e.expect(t, e.upload(t, http.MethodPost, "/me/photos", owner.Token, jpegBytes(t, 400, 500)), http.StatusConflict, "7th photo")

	// reorder: make the last photo the primary one
	order := append([]string{ids[5]}, ids[:5]...)
	e.expect(t, e.do(t, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order[:5]}, owner.Token), http.StatusBadRequest, "incomplete order")
	e.expect(t, e.do(t, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": []string{order[0], order[0], order[2], order[3], order[4], order[5]}}, owner.Token), http.StatusBadRequest, "duplicate in order")
	e.expect(t, e.do(t, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": []string{order[0], order[1], order[2], order[3], order[4], "11111111-1111-1111-1111-111111111111"}}, owner.Token), http.StatusBadRequest, "foreign id in order")
	listed := e.expect(t, e.do(t, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order}, owner.Token), http.StatusOK, "reorder")
	var list struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	listed.decode(t, &list)
	for i, p := range list.Photos {
		if p.ID != order[i] || p.Position != i {
			t.Fatalf("unexpected order at %d: %+v", i, list.Photos)
		}
	}

	// a stranger cannot modify or delete it
	e.expect(t, e.do(t, http.MethodDelete, "/me/photos/"+ids[0], nil, stranger.Token), http.StatusNotFound, "delete foreign photo")
	e.expect(t, e.upload(t, http.MethodPut, "/me/photos/"+ids[0], stranger.Token, jpegBytes(t, 400, 400)), http.StatusNotFound, "replace foreign photo")

	// replace keeps the slot, changes the id and removes the old file
	var oldKey string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[1]).Scan(&oldKey)
	rep := e.expect(t, e.upload(t, http.MethodPut, "/me/photos/"+ids[1], owner.Token, jpegBytes(t, 500, 500)), http.StatusOK, "replace")
	var replaced struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
	}
	rep.decode(t, &replaced)
	if replaced.ID == ids[1] || replaced.Position != 2 {
		t.Fatalf("replace should keep position and mint a new id: %+v", replaced)
	}
	if _, err := os.Stat(filepath.Join(e.dir, oldKey)); !os.IsNotExist(err) {
		t.Fatal("old file must be deleted after replace")
	}

	// delete compacts positions
	e.expect(t, e.do(t, http.MethodDelete, "/me/photos/"+order[0], nil, owner.Token), http.StatusNoContent, "delete primary")
	after := e.expect(t, e.do(t, http.MethodGet, "/me/photos", nil, owner.Token), http.StatusOK, "list")
	list.Photos = nil
	after.decode(t, &list)
	if len(list.Photos) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(list.Photos))
	}
	for i, p := range list.Photos {
		if p.Position != i {
			t.Fatalf("positions must stay contiguous: %+v", list.Photos)
		}
	}
	e.expect(t, e.do(t, http.MethodGet, "/photos/not-a-uuid/file", nil, owner.Token), http.StatusNotFound, "bad id")
}

func TestPhotoAccessControl(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	visible := e.onboard(t, ar, userOpts{Name: "Visible"})
	hidden := e.onboard(t, ar, userOpts{Name: "Hidden", Hidden: true})
	viewer := e.onboard(t, ar, userOpts{Name: "Viewer"})

	photoOf := func(u testUser) string {
		var id string
		if err := e.pool.QueryRow(context.Background(), `SELECT id FROM photos WHERE user_id = $1`, u.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return "/photos/" + id + "/file"
	}
	e.expect(t, e.do(t, http.MethodGet, photoOf(visible), nil, viewer.Token), http.StatusOK, "discoverable user's photo")
	e.expect(t, e.do(t, http.MethodGet, photoOf(hidden), nil, viewer.Token), http.StatusNotFound, "hidden user's photo to stranger")
	e.expect(t, e.do(t, http.MethodGet, photoOf(hidden), nil, hidden.Token), http.StatusOK, "hidden user's own photo")
	e.expect(t, e.do(t, http.MethodGet, "/profiles/"+hidden.ID, nil, viewer.Token), http.StatusNotFound, "hidden profile")

	// matched users can see each other even when hidden
	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": visible.ID}, viewer.Token), http.StatusNoContent, "block")
	e.expect(t, e.do(t, http.MethodGet, photoOf(visible), nil, viewer.Token), http.StatusNotFound, "blocked user's photo")
	e.expect(t, e.do(t, http.MethodGet, photoOf(viewer), nil, visible.Token), http.StatusNotFound, "photo of someone who blocked you")
}

func TestDiscoveryFilters(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	me := e.onboard(t, ar, userOpts{Name: "Me", Gender: "woman", InterestedIn: []string{"man"}, Age: 30, AgeMin: 25, AgeMax: 40})

	ok := e.onboard(t, ar, userOpts{Name: "Ok", Gender: "man", InterestedIn: []string{"woman"}, Age: 33})
	wrongGender := e.onboard(t, ar, userOpts{Name: "WrongGender", Gender: "woman", InterestedIn: []string{"woman"}, Age: 33})
	tooYoung := e.onboard(t, ar, userOpts{Name: "TooYoung", Gender: "man", Age: 22})
	tooOld := e.onboard(t, ar, userOpts{Name: "TooOld", Gender: "man", Age: 55})
	notInterested := e.onboard(t, ar, userOpts{Name: "NotInterested", Gender: "man", InterestedIn: []string{"man"}, Age: 33})
	theirAgeRange := e.onboard(t, ar, userOpts{Name: "TheirAgeRange", Gender: "man", InterestedIn: []string{"woman"}, Age: 33, AgeMin: 18, AgeMax: 25})
	hidden := e.onboard(t, ar, userOpts{Name: "Hidden", Gender: "man", Age: 33, Hidden: true})
	noPhoto := e.onboard(t, ar, userOpts{Name: "NoPhoto", Gender: "man", Age: 33, NoPhoto: true})
	far := e.onboard(t, area{lat: ar.lat + 2, lng: ar.lng}, userOpts{Name: "Far", Gender: "man", Age: 33}) // ~220 km away
	blocked := e.onboard(t, ar, userOpts{Name: "Blocked", Gender: "man", Age: 33})
	blocker := e.onboard(t, ar, userOpts{Name: "Blocker", Gender: "man", Age: 33})
	suspended := e.onboard(t, ar, userOpts{Name: "Suspended", Gender: "man", Age: 33})

	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": blocked.ID}, me.Token), http.StatusNoContent, "block")
	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": me.ID}, blocker.Token), http.StatusNoContent, "blocked by")
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET status = 'suspended' WHERE id = $1`, suspended.ID); err != nil {
		t.Fatal(err)
	}

	list := e.discover(t, me)
	if !containsProfile(list, ok.ID) {
		t.Fatal("eligible candidate missing from discovery")
	}
	for name, u := range map[string]testUser{
		"wrong gender": wrongGender, "too young": tooYoung, "too old": tooOld, "not interested in me": notInterested,
		"their age range excludes me": theirAgeRange, "hidden": hidden, "no photo": noPhoto, "too far": far,
		"blocked by me": blocked, "blocked me": blocker, "suspended": suspended, "myself": me,
	} {
		if containsProfile(list, u.ID) {
			t.Fatalf("%s must not appear in discovery", name)
		}
	}
	for _, p := range list {
		if p.ID == ok.ID {
			if p.Age != 33 || p.DistanceKm == nil || *p.DistanceKm != 1 {
				t.Fatalf("expected age 33 and distance 1 km, got %+v", p)
			}
			if len(p.Photos) != 1 || !strings.HasPrefix(p.Photos[0].URL, "/photos/") {
				t.Fatalf("expected one photo url, got %+v", p.Photos)
			}
		}
	}

	// the public payload must never include private fields
	raw := e.expect(t, e.do(t, http.MethodGet, "/discover", nil, me.Token), http.StatusOK, "discover raw")
	for _, forbidden := range []string{"email", "birthDate", "latitude", "longitude", "passwordHash", "@test.invalid"} {
		if strings.Contains(string(raw.Body), forbidden) {
			t.Fatalf("discovery leaks %q", forbidden)
		}
	}

	// a user with no location sees candidates regardless of distance
	nomad := e.register(t, "nomad")
	e.expect(t, e.do(t, http.MethodPut, "/me/profile", map[string]any{"firstName": "Nomad", "birthDate": birthFor(30), "gender": "woman"}, nomad.Token), http.StatusOK, "nomad profile")
	e.expect(t, e.upload(t, http.MethodPost, "/me/photos", nomad.Token, jpegBytes(t, 400, 400)), http.StatusCreated, "nomad photo")
	e.expect(t, e.do(t, http.MethodGet, "/discover?limit=5", nil, nomad.Token), http.StatusOK, "discover without location")
}

func TestSwipeMatchAndChatFlow(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	alice := e.onboard(t, ar, userOpts{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}})
	bob := e.onboard(t, ar, userOpts{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}})
	eve := e.onboard(t, ar, userOpts{Name: "Eve", Gender: "woman", InterestedIn: []string{"man"}})

	if !containsProfile(e.discover(t, alice), bob.ID) || !containsProfile(e.discover(t, bob), alice.ID) {
		t.Fatal("alice and bob should discover each other")
	}

	// one-sided like: no match, and Alice never sees Bob again
	first := e.swipe(t, alice, bob, "like")
	if first.Matched || first.AlreadySwiped {
		t.Fatalf("unexpected first swipe result %+v", first)
	}
	if containsProfile(e.discover(t, alice), bob.ID) {
		t.Fatal("a swiped profile must not reappear")
	}
	// Bob sees Alice first (she liked him: ranking boost) and can't chat yet
	e.expect(t, e.do(t, http.MethodGet, "/conversations", nil, bob.Token), http.StatusOK, "conversations before match")

	// repeated like is idempotent
	again := e.swipe(t, alice, bob, "like")
	if !again.AlreadySwiped || again.Matched {
		t.Fatalf("repeat like must be idempotent: %+v", again)
	}
	// a changed mind (pass after like) does not rewrite history
	flip := e.swipe(t, alice, bob, "pass")
	if flip.Action != "like" || !flip.AlreadySwiped {
		t.Fatalf("swipes are immutable: %+v", flip)
	}

	// reciprocal like creates exactly one match
	second := e.swipe(t, bob, alice, "like")
	if !second.Matched || second.Match == nil || second.Match.ConversationID == "" {
		t.Fatalf("expected match: %+v", second)
	}
	convo := second.Match.ConversationID
	replay := e.swipe(t, bob, alice, "like")
	if !replay.Matched || replay.Match.ID != second.Match.ID {
		t.Fatalf("replay must return the same match: %+v", replay)
	}
	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, alice.ID, bob.ID).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}

	// swiping on yourself / invalid input / ineligible target
	e.expect(t, e.do(t, http.MethodPost, "/swipes", map[string]string{"userId": alice.ID, "action": "like"}, alice.Token), http.StatusNotFound, "self swipe")
	e.expect(t, e.do(t, http.MethodPost, "/swipes", map[string]string{"userId": bob.ID, "action": "superlike"}, alice.Token), http.StatusBadRequest, "bad action")
	e.expect(t, e.do(t, http.MethodPost, "/swipes", map[string]string{"userId": "nope", "action": "like"}, alice.Token), http.StatusBadRequest, "bad uuid")
	e.expect(t, e.do(t, http.MethodPost, "/swipes", map[string]string{"userId": eve.ID, "action": "like"}, alice.Token), http.StatusNotFound, "same-gender ineligible target")

	// both see the match; both got a notification
	for _, u := range []testUser{alice, bob} {
		var matches struct {
			Matches []struct {
				ConversationID string `json:"conversationId"`
				User           struct {
					FirstName string `json:"firstName"`
				} `json:"user"`
			} `json:"matches"`
		}
		e.expect(t, e.do(t, http.MethodGet, "/matches", nil, u.Token), http.StatusOK, "matches").decode(t, &matches)
		if len(matches.Matches) != 1 || matches.Matches[0].ConversationID != convo {
			t.Fatalf("unexpected matches: %+v", matches)
		}
		var notifs struct {
			Notifications []struct {
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		e.expect(t, e.do(t, http.MethodGet, "/notifications", nil, u.Token), http.StatusOK, "notifications").decode(t, &notifs)
		if notifs.UnreadCount != 1 || notifs.Notifications[0].Type != "match" {
			t.Fatalf("expected one unread match notification: %+v", notifs)
		}
	}

	// ----- chat -----
	msgPath := "/conversations/" + convo + "/messages"
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "   "}, alice.Token), http.StatusBadRequest, "blank message")
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": strings.Repeat("x", 2001)}, alice.Token), http.StatusBadRequest, "long message")
	sent := e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "  Salut Bob !  "}, alice.Token), http.StatusCreated, "send")
	var msg struct {
		ID       string     `json:"id"`
		SenderID string     `json:"senderId"`
		Body     string     `json:"body"`
		ReadAt   *time.Time `json:"readAt"`
	}
	sent.decode(t, &msg)
	if msg.Body != "Salut Bob !" || msg.SenderID != alice.ID || msg.ReadAt != nil {
		t.Fatalf("unexpected message %+v", msg)
	}
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "Ca va ?"}, alice.Token), http.StatusCreated, "send 2")

	// outsider (not part of the match) can neither read, write, mark read nor clear
	e.expect(t, e.do(t, http.MethodGet, msgPath, nil, eve.Token), http.StatusNotFound, "outsider reads")
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "hi"}, eve.Token), http.StatusNotFound, "outsider writes")
	e.expect(t, e.do(t, http.MethodPost, "/conversations/"+convo+"/read", nil, eve.Token), http.StatusNotFound, "outsider marks read")
	e.expect(t, e.do(t, http.MethodDelete, "/conversations/"+convo, nil, eve.Token), http.StatusNotFound, "outsider clears")
	e.expect(t, e.do(t, http.MethodGet, "/conversations/"+convo+"/messages?before=bad", nil, bob.Token), http.StatusBadRequest, "bad cursor")

	// Bob's inbox: unread count, ordering, and notification collapsed to one entry
	var inbox struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
			User struct {
				FirstName string `json:"firstName"`
			} `json:"user"`
		} `json:"conversations"`
		TotalUnread int `json:"totalUnread"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/conversations", nil, bob.Token), http.StatusOK, "bob inbox").decode(t, &inbox)
	if len(inbox.Conversations) != 1 || inbox.Conversations[0].UnreadCount != 2 || inbox.TotalUnread != 2 ||
		inbox.Conversations[0].LastMessage.Body != "Ca va ?" || inbox.Conversations[0].User.FirstName != "Alice" {
		t.Fatalf("unexpected inbox %+v", inbox)
	}
	var notifs struct {
		Notifications []struct{ Type, Body string } `json:"notifications"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/notifications", nil, bob.Token), http.StatusOK, "bob notifications").decode(t, &notifs)
	messageNotifs := 0
	for _, n := range notifs.Notifications {
		if n.Type == "message" {
			messageNotifs++
			if strings.Contains(n.Body, "Salut") {
				t.Fatal("notifications must not contain message content")
			}
		}
	}
	if messageNotifs != 1 {
		t.Fatalf("message notifications should collapse to 1, got %d", messageNotifs)
	}

	// reading marks as read; sender sees readAt; unread drops to 0
	var marked struct {
		Marked int `json:"marked"`
	}
	e.expect(t, e.do(t, http.MethodPost, "/conversations/"+convo+"/read", nil, bob.Token), http.StatusOK, "mark read").decode(t, &marked)
	if marked.Marked != 2 {
		t.Fatalf("expected 2 marked, got %d", marked.Marked)
	}
	var page struct {
		Messages []struct {
			Body   string     `json:"body"`
			ReadAt *time.Time `json:"readAt"`
		} `json:"messages"`
		HasMore bool `json:"hasMore"`
	}
	e.expect(t, e.do(t, http.MethodGet, msgPath, nil, alice.Token), http.StatusOK, "alice reads").decode(t, &page)
	if len(page.Messages) != 2 || page.Messages[0].Body != "Salut Bob !" || page.Messages[1].Body != "Ca va ?" || page.Messages[0].ReadAt == nil {
		t.Fatalf("unexpected page %+v", page)
	}
	e.expect(t, e.do(t, http.MethodGet, "/conversations", nil, bob.Token), http.StatusOK, "bob inbox").decode(t, &inbox)
	if inbox.TotalUnread != 0 {
		t.Fatalf("unread should be 0 after read, got %d", inbox.TotalUnread)
	}

	// pagination: cursor returns older messages, oldest first
	for i := 0; i < 3; i++ {
		e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "m" + string(rune('A'+i))}, bob.Token), http.StatusCreated, "bulk")
	}
	e.expect(t, e.do(t, http.MethodGet, msgPath+"?limit=2", nil, alice.Token), http.StatusOK, "page 1").decode(t, &page)
	if len(page.Messages) != 2 || !page.HasMore || page.Messages[1].Body != "mC" {
		t.Fatalf("unexpected newest page: %+v", page)
	}
	var cursor struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	e.expect(t, e.do(t, http.MethodGet, msgPath+"?limit=2", nil, alice.Token), http.StatusOK, "page 1 ids").decode(t, &cursor)
	e.expect(t, e.do(t, http.MethodGet, msgPath+"?limit=50&before="+cursor.Messages[0].ID, nil, alice.Token), http.StatusOK, "page 2").decode(t, &page)
	if len(page.Messages) != 3 || page.HasMore {
		t.Fatalf("unexpected older page: %+v", page)
	}

	// local deletion hides history for one side only
	e.expect(t, e.do(t, http.MethodDelete, "/conversations/"+convo, nil, bob.Token), http.StatusNoContent, "bob clears")
	e.expect(t, e.do(t, http.MethodGet, msgPath, nil, bob.Token), http.StatusOK, "bob after clear").decode(t, &page)
	if len(page.Messages) != 0 {
		t.Fatalf("cleared history must be empty for bob, got %d", len(page.Messages))
	}
	e.expect(t, e.do(t, http.MethodGet, msgPath, nil, alice.Token), http.StatusOK, "alice unaffected").decode(t, &page)
	if len(page.Messages) != 5 {
		t.Fatalf("alice should still see 5 messages, got %d", len(page.Messages))
	}
	e.expect(t, e.do(t, http.MethodGet, "/conversations", nil, bob.Token), http.StatusOK, "bob inbox after clear").decode(t, &inbox)
	if len(inbox.Conversations) != 0 {
		t.Fatal("cleared conversation must leave bob's inbox until a new message arrives")
	}
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "coucou"}, alice.Token), http.StatusCreated, "alice writes again")
	e.expect(t, e.do(t, http.MethodGet, "/conversations", nil, bob.Token), http.StatusOK, "bob inbox").decode(t, &inbox)
	if len(inbox.Conversations) != 1 || inbox.Conversations[0].LastMessage.Body != "coucou" {
		t.Fatalf("new message should resurface the conversation: %+v", inbox)
	}

	// unmatch closes the conversation for both and removes the match
	var matches struct {
		Matches []struct {
			ID string `json:"id"`
		} `json:"matches"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/matches", nil, alice.Token), http.StatusOK, "matches").decode(t, &matches)
	e.expect(t, e.do(t, http.MethodDelete, "/matches/"+matches.Matches[0].ID, nil, eve.Token), http.StatusNotFound, "outsider unmatch")
	e.expect(t, e.do(t, http.MethodDelete, "/matches/"+matches.Matches[0].ID, nil, alice.Token), http.StatusNoContent, "unmatch")
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "still there?"}, bob.Token), http.StatusNotFound, "send after unmatch")
	e.expect(t, e.do(t, http.MethodGet, msgPath, nil, alice.Token), http.StatusNotFound, "read after unmatch")
	if containsProfile(e.discover(t, alice), bob.ID) || containsProfile(e.discover(t, bob), alice.ID) {
		t.Fatal("unmatched people must not resurface in discovery")
	}
}

func TestSimultaneousLikesCreateExactlyOneMatch(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	for round := 0; round < 5; round++ {
		a := e.onboard(t, ar, userOpts{Name: "RaceA", Gender: "woman", InterestedIn: []string{"man"}})
		b := e.onboard(t, ar, userOpts{Name: "RaceB", Gender: "man", InterestedIn: []string{"woman"}})

		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for i, pair := range [][2]testUser{{a, b}, {b, a}} {
			wg.Add(1)
			go func(i int, from, to testUser) {
				defer wg.Done()
				results[i] = e.swipe(t, from, to, "like")
			}(i, pair[0], pair[1])
		}
		wg.Wait()

		matched := 0
		for _, r := range results {
			if r.Matched {
				matched++
			}
		}
		if matched == 0 {
			t.Fatalf("round %d: at least the second like must see the match: %+v", round, results)
		}
		var matches, convos, participants int
		ctx := context.Background()
		_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&matches)
		_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a IN ($1,$2) AND m.user_b IN ($1,$2)`, a.ID, b.ID).Scan(&convos)
		_ = e.pool.QueryRow(ctx, `SELECT COUNT(*) FROM conversation_participants cp JOIN conversations c ON c.id = cp.conversation_id JOIN matches m ON m.id = c.match_id WHERE m.user_a IN ($1,$2) AND m.user_b IN ($1,$2)`, a.ID, b.ID).Scan(&participants)
		if matches != 1 || convos != 1 || participants != 2 {
			t.Fatalf("round %d: matches=%d conversations=%d participants=%d", round, matches, convos, participants)
		}
	}
}

func TestBlockingAndReporting(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	a := e.onboard(t, ar, userOpts{Name: "BlockA", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.onboard(t, ar, userOpts{Name: "BlockB", Gender: "man", InterestedIn: []string{"woman"}})
	convo := e.matchUsers(t, a, b)
	msgPath := "/conversations/" + convo + "/messages"
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "hello"}, a.Token), http.StatusCreated, "pre-block message")

	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": a.ID}, a.Token), http.StatusBadRequest, "block self")
	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": "11111111-1111-1111-1111-111111111111"}, a.Token), http.StatusNotFound, "block unknown user")
	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent, "block")
	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent, "block is idempotent")

	// both directions are cut: chat, matches, profile, discovery
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "hey"}, b.Token), http.StatusNotFound, "blocked user cannot write")
	e.expect(t, e.do(t, http.MethodGet, msgPath, nil, b.Token), http.StatusNotFound, "blocked user cannot read")
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "hey"}, a.Token), http.StatusNotFound, "blocker cannot write while blocked")
	e.expect(t, e.do(t, http.MethodGet, "/profiles/"+b.ID, nil, a.Token), http.StatusNotFound, "blocked profile")
	e.expect(t, e.do(t, http.MethodGet, "/profiles/"+a.ID, nil, b.Token), http.StatusNotFound, "blocker profile")
	var matches struct {
		Matches []struct{} `json:"matches"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/matches", nil, b.Token), http.StatusOK, "matches").decode(t, &matches)
	if len(matches.Matches) != 0 {
		t.Fatal("blocked match must disappear for the other side")
	}
	if containsProfile(e.discover(t, a), b.ID) || containsProfile(e.discover(t, b), a.ID) {
		t.Fatal("blocked users must not see each other")
	}
	var blocks struct {
		Blocks []struct {
			UserID    string `json:"userId"`
			FirstName string `json:"firstName"`
		} `json:"blocks"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/blocks", nil, a.Token), http.StatusOK, "list blocks").decode(t, &blocks)
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].UserID != b.ID || blocks.Blocks[0].FirstName != "BlockB" {
		t.Fatalf("unexpected blocks %+v", blocks)
	}
	e.expect(t, e.do(t, http.MethodDelete, "/blocks/"+b.ID, nil, a.Token), http.StatusNoContent, "unblock")
	// unblocking does not resurrect the match (the user must re-discover each other)
	e.expect(t, e.do(t, http.MethodPost, msgPath, map[string]string{"body": "back?"}, a.Token), http.StatusNotFound, "match stays closed after unblock")

	// ----- reports -----
	c := e.onboard(t, ar, userOpts{Name: "Reported", Gender: "man", InterestedIn: []string{"woman"}})
	report := func(payload map[string]any, status int, what string) testResponse {
		t.Helper()
		return e.expect(t, e.do(t, http.MethodPost, "/reports", payload, a.Token), status, what)
	}
	report(map[string]any{"userId": a.ID, "reason": "spam"}, 400, "report self")
	report(map[string]any{"userId": c.ID, "reason": "because"}, 400, "bad reason")
	report(map[string]any{"userId": c.ID, "reason": "spam", "details": strings.Repeat("x", 1001)}, 400, "details too long")
	report(map[string]any{"userId": "11111111-1111-1111-1111-111111111111", "reason": "spam"}, 404, "unknown user")
	var created struct {
		ID string `json:"id"`
	}
	report(map[string]any{"userId": c.ID, "reason": "harassment", "details": "insultes"}, 201, "report").decode(t, &created)
	if containsProfile(e.discover(t, a), c.ID) {
		t.Fatal("reporting blocks by default")
	}

	// moderation endpoints are admin only
	e.expect(t, e.do(t, http.MethodGet, "/admin/reports", nil, a.Token), http.StatusForbidden, "non admin list")
	e.expect(t, e.do(t, http.MethodPost, "/admin/reports/"+created.ID+"/resolve", map[string]any{"status": "reviewed"}, a.Token), http.StatusForbidden, "non admin resolve")
	if _, err := e.pool.Exec(context.Background(), `UPDATE users SET role = 'admin' WHERE id = $1`, b.ID); err != nil {
		t.Fatal(err)
	}
	var reports struct {
		Reports []struct {
			ID             string `json:"id"`
			ReportedUserID string `json:"reportedUserId"`
			Reason         string `json:"reason"`
		} `json:"reports"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/admin/reports?status=open&limit=100", nil, b.Token), http.StatusOK, "admin list").decode(t, &reports)
	found := false
	for _, r := range reports.Reports {
		if r.ID == created.ID && r.ReportedUserID == c.ID && r.Reason == "harassment" {
			found = true
		}
	}
	if !found {
		t.Fatal("report missing in admin list")
	}
	e.expect(t, e.do(t, http.MethodPost, "/admin/reports/"+created.ID+"/resolve", map[string]any{"status": "banned"}, b.Token), http.StatusBadRequest, "bad status")
	e.expect(t, e.do(t, http.MethodPost, "/admin/reports/"+created.ID+"/resolve", map[string]any{"status": "reviewed", "suspendUser": true}, b.Token), http.StatusOK, "resolve + suspend")
	e.expect(t, e.do(t, http.MethodGet, "/me", nil, c.Token), http.StatusUnauthorized, "suspended user's sessions are revoked")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": c.Email, "password": c.Password}, ""), http.StatusForbidden, "suspended user cannot log in")
}

func TestAccountDeletionErasesEverything(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	a := e.onboard(t, ar, userOpts{Name: "DelA", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.onboard(t, ar, userOpts{Name: "DelB", Gender: "man", InterestedIn: []string{"woman"}})
	convo := e.matchUsers(t, a, b)
	e.expect(t, e.do(t, http.MethodPost, "/conversations/"+convo+"/messages", map[string]string{"body": "bye"}, a.Token), http.StatusCreated, "message")
	e.expect(t, e.do(t, http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent, "block")

	var key string
	if err := e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(e.dir, key)); err != nil {
		t.Fatalf("photo file should exist: %v", err)
	}

	e.expect(t, e.do(t, http.MethodDelete, "/me", map[string]string{"password": "wrong-password"}, a.Token), http.StatusForbidden, "wrong password")
	e.expect(t, e.do(t, http.MethodDelete, "/me", map[string]string{"password": a.Password}, a.Token), http.StatusNoContent, "delete account")

	e.expect(t, e.do(t, http.MethodGet, "/me", nil, a.Token), http.StatusUnauthorized, "token after deletion")
	e.expect(t, e.do(t, http.MethodPost, "/auth/login", map[string]string{"email": a.Email, "password": a.Password}, ""), http.StatusUnauthorized, "login after deletion")
	if _, err := os.Stat(filepath.Join(e.dir, key)); !os.IsNotExist(err) {
		t.Fatal("photo file must be removed with the account")
	}
	for table, col := range map[string]string{
		"profiles": "user_id", "preferences": "user_id", "photos": "user_id", "swipes": "from_user_id",
		"blocks": "blocker_id", "notifications": "user_id", "sessions": "user_id", "user_interests": "user_id",
	} {
		var n int
		if err := e.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s still holds %d rows for the deleted user (err=%v)", table, n, err)
		}
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`, convo).Scan(&n)
	if n != 0 {
		t.Fatal("messages of the deleted account's conversations must be erased")
	}
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = $1 OR user_b = $1`, a.ID).Scan(&n)
	if n != 0 {
		t.Fatal("matches must be erased")
	}
	// the partner is unaffected and no longer sees the match
	var matches struct {
		Matches []struct{} `json:"matches"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/matches", nil, b.Token), http.StatusOK, "partner matches").decode(t, &matches)
	if len(matches.Matches) != 0 {
		t.Fatal("partner should have no match left")
	}
}

func TestInterestsAndRanking(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	me := e.onboard(t, ar, userOpts{Name: "RankMe", Gender: "woman", InterestedIn: []string{"man"}, Interests: []string{"music", "art", "travel"}})
	shared := e.onboard(t, ar, userOpts{Name: "Shared", Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"music", "art", "travel"}})
	none := e.onboard(t, ar, userOpts{Name: "None", Gender: "man", InterestedIn: []string{"woman"}, Interests: []string{"gardening"}})

	var catalog struct {
		Interests []struct{ Slug, Label string } `json:"interests"`
	}
	e.expect(t, e.do(t, http.MethodGet, "/interests", nil, me.Token), http.StatusOK, "interests").decode(t, &catalog)
	if len(catalog.Interests) < 20 {
		t.Fatalf("expected the interest catalog, got %d", len(catalog.Interests))
	}
	list := e.discover(t, me)
	if len(list) < 2 || list[0].ID != shared.ID || list[1].ID != none.ID {
		t.Fatalf("candidate sharing 3 interests must rank first: %+v", list)
	}
	if len(list[0].Interests) != 3 {
		t.Fatalf("expected interests on candidate: %+v", list[0])
	}
	_ = none
}

func TestWebSocketRealtimeDelivery(t *testing.T) {
	e := newTestEnv(t)
	ar := newArea()
	a := e.onboard(t, ar, userOpts{Name: "WsA", Gender: "woman", InterestedIn: []string{"man"}})
	b := e.onboard(t, ar, userOpts{Name: "WsB", Gender: "man", InterestedIn: []string{"woman"}})

	srv := newTestServer(e)
	defer srv.Close()

	// invalid tickets and access tokens are refused
	if status := dialStatus(t, srv.URL, "garbage"); status != http.StatusUnauthorized {
		t.Fatalf("garbage ticket: %d", status)
	}
	if status := dialStatus(t, srv.URL, a.Token); status != http.StatusUnauthorized {
		t.Fatalf("an access token must not work as a ws ticket: %d", status)
	}
	if status := dialStatus(t, srv.URL, ""); status != http.StatusUnauthorized {
		t.Fatalf("missing ticket: %d", status)
	}

	wsB := dialWS(t, srv.URL, e.wsTicket(t, b))
	defer wsB.Close()
	wsA := dialWS(t, srv.URL, e.wsTicket(t, a))
	defer wsA.Close()

	// matching pushes a "match" event to the person who liked first
	e.swipe(t, a, b, "like")
	convo := e.swipe(t, b, a, "like").Match.ConversationID
	if ev := readEvent(t, wsA, "match"); ev.Data["conversationId"] != convo {
		t.Fatalf("unexpected match event %+v", ev)
	}
	_ = readEvent(t, wsB, "notification")

	// a message is pushed live to the recipient
	e.expect(t, e.do(t, http.MethodPost, "/conversations/"+convo+"/messages", map[string]string{"body": "live!"}, a.Token), http.StatusCreated, "send")
	got := readEventRaw(t, wsB, "message")
	if !strings.Contains(got, `"body":"live!"`) || !strings.Contains(got, a.ID) {
		t.Fatalf("unexpected message event %s", got)
	}
	// and the read receipt is pushed back to the sender
	e.expect(t, e.do(t, http.MethodPost, "/conversations/"+convo+"/read", nil, b.Token), http.StatusOK, "read")
	if ev := readEvent(t, wsA, "read"); ev.Data["readerId"] != b.ID {
		t.Fatalf("unexpected read event %+v", ev)
	}
}
