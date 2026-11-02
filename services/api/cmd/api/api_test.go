package main

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"net/http/httptest"
	"strings"
)

type deck struct {
	Items []struct {
		UserID     string `json:"userId"`
		FirstName  string
		DistanceKm *int `json:"distanceKm"`
		Photos     []struct{ URL string }
	}
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	if r := e.do("GET", "/health", "", nil); r.Status != 200 {
		t.Fatalf("health %d %s", r.Status, r.Body)
	}
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	u := e.register("auth")

	if r := e.do("POST", "/auth/register", "", map[string]string{"email": u.Email, "password": "Password123"}); r.Status != 409 {
		t.Fatalf("duplicate register: %d", r.Status)
	}
	if r := e.do("POST", "/auth/register", "", map[string]string{"email": "weak@tandem.test", "password": "abcdefgh"}); r.Status != 400 {
		t.Fatalf("weak password should be 400, got %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "WrongPass1"}); r.Status != 401 {
		t.Fatalf("bad login: %d", r.Status)
	}
	r := e.do("POST", "/auth/login", "", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": "Password123"})
	if r.Status != 200 {
		t.Fatalf("login normalizes email: %d %s", r.Status, r.Body)
	}
	if r := e.do("GET", "/me", u.Token, nil); r.Status != 200 {
		t.Fatalf("me: %d", r.Status)
	}
	if r := e.do("GET", "/me", "", nil); r.Status != 401 {
		t.Fatalf("me without token: %d", r.Status)
	}
	// Tokens signed with another algorithm / unsigned are rejected.
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": u.ID, "sid": "x", "exp": time.Now().Add(time.Hour).Unix()})
	s, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if r := e.do("GET", "/me", s, nil); r.Status != 401 {
		t.Fatalf("alg none must be rejected: %d", r.Status)
	}
	// Refresh rotates, old refresh token becomes invalid.
	var tok struct{ AccessToken, RefreshToken string }
	rr := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh})
	if rr.Status != 200 {
		t.Fatalf("refresh: %d", rr.Status)
	}
	rr.JSON(&tok)
	if r := e.do("POST", "/auth/refresh", "", map[string]string{"refreshToken": u.Refresh}); r.Status != 401 {
		t.Fatalf("reused refresh token must fail: %d", r.Status)
	}
	// Logout revokes the session immediately, even for the unexpired access token.
	if r := e.do("POST", "/auth/logout", "", map[string]string{"refreshToken": tok.RefreshToken}); r.Status != 204 {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := e.do("GET", "/me", tok.AccessToken, nil); r.Status != 401 {
		t.Fatalf("access token must die with its session: %d", r.Status)
	}
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	u := e.register("reset")
	if r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": "nobody@tandem.test"}); r.Status != 202 {
		t.Fatalf("forgot unknown email must look identical: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/forgot", "", map[string]string{"email": u.Email}); r.Status != 202 {
		t.Fatalf("forgot: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"token": "nope", "password": "NewPassword1"}); r.Status != 400 {
		t.Fatalf("bad token: %d", r.Status)
	}
	// Plant a known token the way the service stores it (hash only).
	if _, err := e.pool.Exec(t.Context(), `UPDATE password_resets SET token_hash = encode(sha256('known-token'::bytea), 'hex') WHERE user_id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"token": "known-token", "password": "NewPassword1"}); r.Status != 204 {
		t.Fatalf("reset: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]string{"token": "known-token", "password": "NewPassword2"}); r.Status != 400 {
		t.Fatalf("token is single use: %d", r.Status)
	}
	if r := e.do("GET", "/me", u.Token, nil); r.Status != 401 {
		t.Fatalf("sessions are revoked after reset: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": u.Email, "password": "NewPassword1"}); r.Status != 200 {
		t.Fatalf("login with new password: %d", r.Status)
	}
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("val")
	base := func() map[string]any {
		return map[string]any{"firstName": "Zoé", "birthDate": birth(25), "gender": "woman", "latitude": 1.234, "longitude": 2.345}
	}
	cases := map[string]func(m map[string]any){
		"underage":     func(m map[string]any) { m["birthDate"] = birth(17) },
		"future date":  func(m map[string]any) { m["birthDate"] = "2999-01-01" },
		"bad gender":   func(m map[string]any) { m["gender"] = "robot" },
		"empty name":   func(m map[string]any) { m["firstName"] = "   " },
		"bad lat":      func(m map[string]any) { m["latitude"] = 123.0 },
		"lat only":     func(m map[string]any) { delete(m, "longitude") },
		"long bio":     func(m map[string]any) { m["bio"] = strings.Repeat("a", 501) },
		"dup interest": func(m map[string]any) { m["interestIds"] = []int{1, 1} },
		"bad interest": func(m map[string]any) { m["interestIds"] = []int{9999} },
	}
	for name, mutate := range cases {
		m := base()
		mutate(m)
		if r := e.do("PUT", "/me/profile", u.Token, m); r.Status != 400 {
			t.Errorf("%s: expected 400, got %d %s", name, r.Status, r.Body)
		}
	}
	r := e.do("PUT", "/me/profile", u.Token, base())
	if r.Status != 200 {
		t.Fatalf("valid profile: %d %s", r.Status, r.Body)
	}
	var b struct {
		Profile  struct{ Latitude, Longitude float64 }
		Complete bool
	}
	r.JSON(&b)
	if b.Profile.Latitude != 1.23 || b.Profile.Longitude != 2.35 {
		t.Fatalf("coordinates must be rounded to ~1km, got %v,%v", b.Profile.Latitude, b.Profile.Longitude)
	}
	if b.Complete {
		t.Fatal("profile without preferences and photo is not complete")
	}
	if r := e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10}); r.Status != 400 {
		t.Fatalf("inverted age range: %d", r.Status)
	}
	if r := e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 40, "maxDistanceKm": 10}); r.Status != 400 {
		t.Fatalf("min age under 18: %d", r.Status)
	}
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()

	// Non-images and tiny images are refused.
	if r := e.upload("POST", "/me/photos", a.Token, []byte("<?php echo 1; ?>")); r.Status != 400 {
		t.Fatalf("non-image: %d", r.Status)
	}
	if r := e.upload("POST", "/me/photos", a.Token, testJPEG(50, 50)); r.Status != 400 {
		t.Fatalf("tiny image: %d", r.Status)
	}
	// Add until the limit.
	var ids []string
	var first struct{ Items []struct{ ID string } }
	e.do("GET", "/me/photos", a.Token, nil).JSON(&first)
	ids = append(ids, first.Items[0].ID)
	for i := 0; i < 5; i++ {
		r := e.upload("POST", "/me/photos", a.Token, testJPEG(300, 300))
		if r.Status != 201 {
			t.Fatalf("upload %d: %d %s", i, r.Status, r.Body)
		}
		var p struct{ ID string }
		r.JSON(&p)
		ids = append(ids, p.ID)
	}
	if r := e.upload("POST", "/me/photos", a.Token, testJPEG(300, 300)); r.Status != 409 {
		t.Fatalf("7th photo must be refused: %d", r.Status)
	}
	// Make the last one main; order becomes [last, first, ...].
	if r := e.do("POST", "/me/photos/"+ids[5]+"/main", a.Token, nil); r.Status != 200 {
		t.Fatalf("set main: %d", r.Status)
	}
	var list struct {
		Items []struct {
			ID       string
			Position int
		}
	}
	e.do("GET", "/me/photos", a.Token, nil).JSON(&list)
	if list.Items[0].ID != ids[5] || list.Items[1].ID != ids[0] {
		t.Fatalf("unexpected order %+v", list.Items)
	}
	// Invalid reorder (missing id) is refused.
	if r := e.do("PUT", "/me/photos/order", a.Token, map[string]any{"photoIds": ids[:2]}); r.Status != 400 {
		t.Fatalf("partial order: %d", r.Status)
	}
	// Another user cannot delete or replace my photo.
	if r := e.do("DELETE", "/me/photos/"+ids[0], b.Token, nil); r.Status != 404 {
		t.Fatalf("foreign delete: %d", r.Status)
	}
	if r := e.upload("PUT", "/me/photos/"+ids[0], b.Token, testJPEG(300, 300)); r.Status != 404 {
		t.Fatalf("foreign replace: %d", r.Status)
	}
	// Replace keeps position but changes id.
	r := e.upload("PUT", "/me/photos/"+ids[5], a.Token, testJPEG(640, 480))
	if r.Status != 200 {
		t.Fatalf("replace: %d %s", r.Status, r.Body)
	}
	var rep struct {
		ID       string
		Position int
	}
	r.JSON(&rep)
	if rep.Position != 0 || rep.ID == ids[5] {
		t.Fatalf("replace result %+v", rep)
	}
	// Served photo is a re-encoded JPEG.
	img := e.do("GET", "/photos/"+rep.ID, b.Token, nil) // b sees discoverable a
	if img.Status != 200 || len(img.Body) < 100 || img.Body[0] != 0xFF {
		t.Fatalf("photo fetch: %d", img.Status)
	}
	if r := e.do("GET", "/photos/"+rep.ID, "", nil); r.Status != 401 {
		t.Fatalf("photo needs auth: %d", r.Status)
	}
	// Blocking hides photos too.
	e.do("POST", "/users/"+a.ID+"/block", b.Token, nil)
	if r := e.do("GET", "/photos/"+rep.ID, b.Token, nil); r.Status != 404 {
		t.Fatalf("blocked photo: %d", r.Status)
	}
	// Delete everything but the last photo.
	for _, it := range list.Items[:5] {
		id := it.ID
		if id == ids[5] {
			id = rep.ID
		}
		if r := e.do("DELETE", "/me/photos/"+id, a.Token, nil); r.Status != 204 {
			t.Fatalf("delete: %d %s", r.Status, r.Body)
		}
	}
	e.do("GET", "/me/photos", a.Token, nil).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].Position != 0 {
		t.Fatalf("positions must be compacted: %+v", list.Items)
	}
	if r := e.do("DELETE", "/me/photos/"+list.Items[0].ID, a.Token, nil); r.Status != 409 {
		t.Fatalf("last photo cannot be deleted: %d", r.Status)
	}
}

func TestDiscoveryFilters(t *testing.T) {
	e := newEnv(t)
	lat, lng := region()
	me := e.register("me")
	e.onboard(me, spec{"Moi", "woman", birth(28), []string{"man"}, lat, lng})
	near := e.register("near")
	e.onboard(near, spec{"Near", "man", birth(30), []string{"woman"}, lat + 0.05, lng})
	wrongGender := e.register("wg")
	e.onboard(wrongGender, spec{"Wrong", "woman", birth(30), []string{"woman"}, lat, lng})
	notInterested := e.register("ni")
	e.onboard(notInterested, spec{"NotInterested", "man", birth(30), []string{"man"}, lat, lng})
	far := e.register("far")
	e.onboard(far, spec{"Far", "man", birth(30), []string{"woman"}, lat + 1.5, lng}) // ~166 km
	e.register("incomplete")

	var d deck
	r := e.do("GET", "/discover", me.Token, nil)
	if r.Status != 200 {
		t.Fatalf("deck: %d %s", r.Status, r.Body)
	}
	r.JSON(&d)
	if len(d.Items) != 1 || d.Items[0].UserID != near.ID {
		t.Fatalf("expected only the compatible nearby profile, got %+v", d.Items)
	}
	if d.Items[0].DistanceKm == nil || *d.Items[0].DistanceKm != 10 {
		t.Fatalf("distance must be bucketed (5 km steps), got %v", d.Items[0].DistanceKm)
	}
	if strings.Contains(string(r.Body), "latitude") || strings.Contains(string(r.Body), "birthDate") || strings.Contains(string(r.Body), "email") {
		t.Fatalf("deck leaks private fields: %s", r.Body)
	}
	if r := e.do("GET", "/discover?limit=500", me.Token, nil); r.Status != 400 {
		t.Fatalf("limit validation: %d", r.Status)
	}
	// Cannot interact with a profile that is not eligible, nor with oneself.
	if r := e.do("POST", "/discover/"+far.ID+"/like", me.Token, nil); r.Status != 404 {
		t.Fatalf("like out-of-range: %d", r.Status)
	}
	if r := e.do("POST", "/discover/"+me.ID+"/like", me.Token, nil); r.Status != 400 {
		t.Fatalf("self like: %d", r.Status)
	}
	if r := e.do("GET", "/users/"+wrongGender.ID, me.Token, nil); r.Status != 200 {
		// Viewing a discoverable profile is allowed even outside preferences? No: only via discovery rules.
		t.Logf("profile view of non-matching preference: %d", r.Status)
	}
	// Hiding my profile removes me from others' decks.
	var nd deck
	e.do("GET", "/discover", near.Token, nil).JSON(&nd)
	if len(nd.Items) != 1 {
		t.Fatalf("near should see me: %+v", nd.Items)
	}
	e.do("PUT", "/me/profile", me.Token, map[string]any{"firstName": "Moi", "birthDate": birth(28), "gender": "woman", "latitude": lat, "longitude": lng, "discoverable": false})
	e.do("GET", "/discover", near.Token, nil).JSON(&nd)
	if len(nd.Items) != 0 {
		t.Fatalf("paused profile must disappear: %+v", nd.Items)
	}
	// Incomplete user cannot like.
	inc := e.register("inc")
	if r := e.do("POST", "/discover/"+near.ID+"/like", inc.Token, nil); r.Status != 409 {
		t.Fatalf("incomplete like: %d %s", r.Status, r.Body)
	}
}

type swipeRes struct {
	Matched bool
	MatchID string
}

func TestMatchingAndChat(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	lat, lng := region()
	c := e.register("carol")
	e.onboard(c, spec{"Carol", "woman", birth(27), []string{"man"}, lat, lng})

	var res swipeRes
	r := e.do("POST", "/discover/"+b.ID+"/like", a.Token, nil)
	if r.Status != 200 {
		t.Fatalf("like: %d %s", r.Status, r.Body)
	}
	r.JSON(&res)
	if res.Matched {
		t.Fatal("one-sided like must not match")
	}
	// Repeating is idempotent; contradicting is refused.
	e.do("POST", "/discover/"+b.ID+"/like", a.Token, nil).JSON(&res)
	if res.Matched {
		t.Fatal("repeat like must not match")
	}
	if r := e.do("POST", "/discover/"+b.ID+"/pass", a.Token, nil); r.Status != 409 {
		t.Fatalf("pass after like: %d", r.Status)
	}
	// Already-liked profile no longer shows up for a.
	var d deck
	e.do("GET", "/discover", a.Token, nil).JSON(&d)
	for _, it := range d.Items {
		if it.UserID == b.ID {
			t.Fatal("processed profile reappeared in deck")
		}
	}
	// No conversation before the match.
	var inbox struct{ Items []struct{ MatchID string } }
	e.do("GET", "/matches", a.Token, nil).JSON(&inbox)
	if len(inbox.Items) != 0 {
		t.Fatal("no match yet")
	}

	// Reciprocal like creates exactly one match visible to both.
	e.do("POST", "/discover/"+a.ID+"/like", b.Token, nil).JSON(&res)
	if !res.Matched || res.MatchID == "" {
		t.Fatalf("expected a match: %+v", res)
	}
	matchID := res.MatchID
	for _, tok := range []string{a.Token, b.Token} {
		e.do("GET", "/matches", tok, nil).JSON(&inbox)
		if len(inbox.Items) != 1 || inbox.Items[0].MatchID != matchID {
			t.Fatalf("inbox: %+v", inbox.Items)
		}
		var n struct {
			Items       []struct{ Type string }
			UnreadCount int
		}
		e.do("GET", "/notifications", tok, nil).JSON(&n)
		if n.UnreadCount != 1 || n.Items[0].Type != "match" {
			t.Fatalf("match notification missing: %+v", n)
		}
	}
	// Match is stable after repeated likes.
	var again swipeRes
	e.do("POST", "/discover/"+a.ID+"/like", b.Token, nil).JSON(&again)
	if again.MatchID != matchID {
		t.Fatal("match id must be stable")
	}

	// Chat.
	path := "/matches/" + matchID + "/messages"
	if r := e.do("POST", path, a.Token, map[string]string{"body": "   "}); r.Status != 400 {
		t.Fatalf("empty message: %d", r.Status)
	}
	if r := e.do("POST", path, a.Token, map[string]string{"body": strings.Repeat("x", 2001)}); r.Status != 400 {
		t.Fatalf("long message: %d", r.Status)
	}
	if r := e.do("POST", path, a.Token, map[string]string{"body": "  Salut !  "}); r.Status != 201 {
		t.Fatalf("send: %d %s", r.Status, r.Body)
	}
	e.do("POST", path, a.Token, map[string]string{"body": "Ça va ?"})

	var conv struct {
		Items []struct {
			UnreadCount int
			LastMessage *struct{ Body string }
		}
	}
	e.do("GET", "/matches", b.Token, nil).JSON(&conv)
	if conv.Items[0].UnreadCount != 2 || conv.Items[0].LastMessage.Body != "Ça va ?" {
		t.Fatalf("unread state: %+v", conv.Items[0])
	}
	e.do("GET", "/matches", a.Token, nil).JSON(&conv)
	if conv.Items[0].UnreadCount != 0 {
		t.Fatal("sender has nothing unread")
	}
	var msgs struct {
		Items []struct {
			Body   string
			ReadAt *time.Time
		}
		NextCursor *int64
	}
	e.do("GET", path, b.Token, nil).JSON(&msgs)
	if len(msgs.Items) != 2 || msgs.Items[0].Body != "Ça va ?" || msgs.Items[1].Body != "Salut !" || msgs.Items[0].ReadAt != nil {
		t.Fatalf("messages newest-first & trimmed: %+v", msgs.Items)
	}
	if r := e.do("POST", "/matches/"+matchID+"/read", b.Token, nil); r.Status != 204 {
		t.Fatalf("read: %d", r.Status)
	}
	e.do("GET", path, a.Token, nil).JSON(&msgs)
	if msgs.Items[0].ReadAt == nil {
		t.Fatal("read receipt expected")
	}
	e.do("GET", "/matches", b.Token, nil).JSON(&conv)
	if conv.Items[0].UnreadCount != 0 {
		t.Fatal("unread count must drop to 0")
	}
	// Pagination: limit 1 yields a cursor.
	e.do("GET", path+"?limit=1", b.Token, nil).JSON(&msgs)
	if len(msgs.Items) != 1 || msgs.NextCursor == nil {
		t.Fatalf("pagination: %+v", msgs)
	}

	// Permissions: an outsider can neither read, write, mark read, hide nor unmatch.
	for _, call := range [][2]string{{"GET", path}, {"POST", path}, {"POST", "/matches/" + matchID + "/read"}, {"POST", "/matches/" + matchID + "/hide"}, {"DELETE", "/matches/" + matchID}} {
		if r := e.do(call[0], call[1], c.Token, map[string]string{"body": "intrus"}); r.Status != 404 {
			t.Fatalf("outsider %s %s must get 404, got %d", call[0], call[1], r.Status)
		}
	}
	// Invalid ids never reach SQL.
	if r := e.do("GET", "/matches/not-a-uuid/messages", a.Token, nil); r.Status != 404 {
		t.Fatalf("bad id: %d", r.Status)
	}

	// Local deletion hides the conversation for a only, until b writes again.
	if r := e.do("POST", "/matches/"+matchID+"/hide", a.Token, nil); r.Status != 204 {
		t.Fatalf("hide: %d", r.Status)
	}
	e.do("GET", "/matches", a.Token, nil).JSON(&inbox)
	if len(inbox.Items) != 0 {
		t.Fatal("hidden conversation should leave a's inbox")
	}
	e.do("GET", "/matches", b.Token, nil).JSON(&inbox)
	if len(inbox.Items) != 1 {
		t.Fatal("b keeps the conversation")
	}
	e.do("POST", path, b.Token, map[string]string{"body": "Tu es là ?"})
	e.do("GET", "/matches", a.Token, nil).JSON(&inbox)
	if len(inbox.Items) != 1 {
		t.Fatal("new message revives the conversation")
	}
	e.do("GET", path, a.Token, nil).JSON(&msgs)
	if len(msgs.Items) != 1 || msgs.Items[0].Body != "Tu es là ?" {
		t.Fatalf("old messages stay hidden for a: %+v", msgs.Items)
	}

	// Block: removes the match everywhere, and the blocked user cannot write.
	if r := e.do("POST", "/users/"+b.ID+"/block", a.Token, nil); r.Status != 204 {
		t.Fatalf("block: %d", r.Status)
	}
	if r := e.do("POST", path, b.Token, map[string]string{"body": "hey"}); r.Status != 404 {
		t.Fatalf("blocked user cannot write: %d", r.Status)
	}
	e.do("GET", "/matches", b.Token, nil).JSON(&inbox)
	if len(inbox.Items) != 0 {
		t.Fatal("match must disappear after block")
	}
	if r := e.do("GET", "/users/"+a.ID, b.Token, nil); r.Status != 404 {
		t.Fatalf("blocked profile view: %d", r.Status)
	}
	e.do("GET", "/discover", b.Token, nil).JSON(&d)
	for _, it := range d.Items {
		if it.UserID == a.ID {
			t.Fatal("blocker visible in blocked user's deck")
		}
	}
	var bl struct{ Items []struct{ UserID string } }
	e.do("GET", "/blocks", a.Token, nil).JSON(&bl)
	if len(bl.Items) != 1 || bl.Items[0].UserID != b.ID {
		t.Fatalf("blocks list: %+v", bl)
	}
	if r := e.do("DELETE", "/users/"+b.ID+"/block", a.Token, nil); r.Status != 204 {
		t.Fatalf("unblock: %d", r.Status)
	}

	// Reports.
	if r := e.do("POST", "/users/"+b.ID+"/report", a.Token, map[string]string{"reason": "bogus"}); r.Status != 400 {
		t.Fatalf("bad reason: %d", r.Status)
	}
	if r := e.do("POST", "/users/"+b.ID+"/report", a.Token, map[string]string{"reason": "harassment", "details": "x"}); r.Status != 201 {
		t.Fatalf("report: %d", r.Status)
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2`, a.ID, b.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("report not stored: %d", n)
	}
	if r := e.do("POST", "/users/"+a.ID+"/report", a.Token, map[string]string{"reason": "spam"}); r.Status != 400 {
		t.Fatalf("self report: %d", r.Status)
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 5; i++ {
		a, b := e.pair()
		var wg sync.WaitGroup
		results := make([]swipeRes, 2)
		wg.Add(2)
		go func() { defer wg.Done(); e.do("POST", "/discover/"+b.ID+"/like", a.Token, nil).JSON(&results[0]) }()
		go func() { defer wg.Done(); e.do("POST", "/discover/"+a.ID+"/like", b.Token, nil).JSON(&results[1]) }()
		wg.Wait()
		var n int
		_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE user_a_id IN ($1,$2) AND user_b_id IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("iteration %d: expected exactly one match, got %d", i, n)
		}
		if !results[0].Matched && !results[1].Matched {
			t.Fatalf("iteration %d: at least one response must report the match", i)
		}
	}
}

func TestPassNeverMatches(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	e.do("POST", "/discover/"+b.ID+"/pass", a.Token, nil)
	var res swipeRes
	e.do("POST", "/discover/"+a.ID+"/like", b.Token, nil).JSON(&res)
	if res.Matched {
		t.Fatal("a pass followed by a like must not match")
	}
}

func TestAccountDeletion(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	var res swipeRes
	e.do("POST", "/discover/"+b.ID+"/like", a.Token, nil)
	e.do("POST", "/discover/"+a.ID+"/like", b.Token, nil).JSON(&res)
	e.do("POST", "/matches/"+res.MatchID+"/messages", a.Token, map[string]string{"body": "hello"})

	if r := e.do("DELETE", "/me", a.Token, map[string]string{"password": "wrong"}); r.Status != 403 {
		t.Fatalf("wrong password: %d", r.Status)
	}
	if r := e.do("DELETE", "/me", a.Token, map[string]string{"password": "Password123"}); r.Status != 204 {
		t.Fatalf("delete account: %d", r.Status)
	}
	if r := e.do("GET", "/me", a.Token, nil); r.Status != 401 {
		t.Fatalf("deleted user's token must stop working: %d", r.Status)
	}
	for table, col := range map[string]string{"profiles": "user_id", "photos": "user_id", "preferences": "user_id", "user_interests": "user_id", "sessions": "user_id", "notifications": "user_id", "swipes": "from_user_id"} {
		var n int
		_ = e.pool.QueryRow(t.Context(), "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n)
		if n != 0 {
			t.Errorf("%s still has %d rows for the deleted user", table, n)
		}
	}
	var n int
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM matches WHERE id = $1`, res.MatchID).Scan(&n)
	if n != 0 {
		t.Fatal("match must be deleted with the account")
	}
	_ = e.pool.QueryRow(t.Context(), `SELECT COUNT(*) FROM messages WHERE match_id = $1`, res.MatchID).Scan(&n)
	if n != 0 {
		t.Fatal("messages must be deleted with the account")
	}
	if r := e.do("POST", "/auth/login", "", map[string]string{"email": a.Email, "password": "Password123"}); r.Status != 401 {
		t.Fatalf("login after deletion: %d", r.Status)
	}
	var d deck
	e.do("GET", "/discover", b.Token, nil).JSON(&d)
	for _, it := range d.Items {
		if it.UserID == a.ID {
			t.Fatal("deleted user still in deck")
		}
	}
}

func TestRealtimeWebSocket(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	var res swipeRes
	e.do("POST", "/discover/"+b.ID+"/like", a.Token, nil)

	srv := httptest.NewServer(e.router)
	defer srv.Close()

	connect := func(u user) *websocket.Conn {
		var tk struct{ Ticket string }
		e.do("POST", "/ws/ticket", u.Token, nil).JSON(&tk)
		c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket="+tk.Ticket, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		// A ticket is single use.
		if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket="+tk.Ticket, nil); err == nil {
			t.Fatal("ticket reuse must fail")
		}
		return c
	}
	if _, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws?ticket=bogus", nil); err == nil {
		t.Fatal("bogus ticket must fail")
	}
	ca, cb := connect(a), connect(b)
	defer ca.Close()
	defer cb.Close()

	expect := func(c *websocket.Conn, typ string) map[string]any {
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var ev struct {
				Type string
				Data map[string]any
			}
			if err := c.ReadJSON(&ev); err != nil {
				t.Fatalf("waiting for %s: %v", typ, err)
			}
			if ev.Type == typ {
				return ev.Data
			}
		}
	}

	e.do("POST", "/discover/"+a.ID+"/like", b.Token, nil).JSON(&res)
	expect(ca, "match.new")
	expect(cb, "match.new")

	e.do("POST", "/matches/"+res.MatchID+"/messages", a.Token, map[string]string{"body": "temps réel"})
	if got := expect(cb, "message.new"); got["body"] != "temps réel" {
		t.Fatalf("payload: %v", got)
	}
	e.do("POST", "/matches/"+res.MatchID+"/read", b.Token, nil)
	expect(ca, "message.read")
}

func TestRateLimitOnAuth(t *testing.T) {
	e := newEnv(t)
	limited := false
	for i := 0; i < 60; i++ {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"x@tandem.test","password":"nope"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.9:4000"
		w := httptest.NewRecorder()
		e.router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			limited = true
			if w.Header().Get("Retry-After") == "" {
				t.Fatal("Retry-After expected")
			}
			break
		}
	}
	if !limited {
		t.Fatal("login must be rate limited")
	}
}
