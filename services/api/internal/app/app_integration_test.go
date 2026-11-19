package app_test

import (
	"context"
	"encoding/json"
	"image/color"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestHealth(t *testing.T) {
	e := setup(t)
	r := e.do(t, http.MethodGet, "/health", "", nil)
	expect(t, r, http.StatusOK)
	if !strings.Contains(string(r.Body), `"ok"`) {
		t.Fatalf("unexpected body %s", r.Body)
	}
}

func TestAuthLifecycle(t *testing.T) {
	e := setup(t)
	u := e.register(t, "auth")

	// Email normalisation + duplicate detection.
	dup := e.do(t, http.MethodPost, "/auth/register", "", map[string]string{"email": "  " + strings.ToUpper(u.Email) + " ", "password": "an0ther-password"})
	expect(t, dup, http.StatusConflict)

	expect(t, e.do(t, http.MethodPost, "/auth/register", "", map[string]string{"email": uniqueEmail("weak"), "password": "password"}), http.StatusBadRequest)
	expect(t, e.do(t, http.MethodPost, "/auth/register", "", map[string]string{"email": "not-an-email", "password": "s3cret-passw0rd"}), http.StatusBadRequest)

	expect(t, e.do(t, http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "wrong-passw0rd"}), http.StatusUnauthorized)
	expect(t, e.do(t, http.MethodPost, "/auth/login", "", map[string]string{"email": uniqueEmail("ghost"), "password": "wrong-passw0rd"}), http.StatusUnauthorized)

	login := e.do(t, http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": u.Password})
	expect(t, login, http.StatusOK)
	var s session
	login.decode(t, &s)

	me := e.do(t, http.MethodGet, "/me", s.AccessToken, nil)
	expect(t, me, http.StatusOK)
	if !strings.Contains(string(me.Body), u.User.ID) {
		t.Fatalf("/me returned %s", me.Body)
	}
	expect(t, e.do(t, http.MethodGet, "/me", "", nil), http.StatusUnauthorized)
	expect(t, e.do(t, http.MethodGet, "/me", "garbage", nil), http.StatusUnauthorized)

	// Refresh rotates the token: the previous refresh token stops working.
	ref := e.do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": s.RefreshToken})
	expect(t, ref, http.StatusOK)
	var rotated session
	ref.decode(t, &rotated)
	expect(t, e.do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refreshToken": s.RefreshToken}), http.StatusUnauthorized)

	// Logout revokes the session: its access token is rejected immediately.
	expect(t, e.do(t, http.MethodPost, "/auth/logout", "", map[string]string{"refreshToken": rotated.RefreshToken}), http.StatusNoContent)
	expect(t, e.do(t, http.MethodGet, "/me", rotated.AccessToken, nil), http.StatusUnauthorized)
	// Other sessions are unaffected.
	expect(t, e.do(t, http.MethodGet, "/me", u.AccessToken, nil), http.StatusOK)
}

func TestPasswordRecovery(t *testing.T) {
	e := setup(t)
	u := e.register(t, "recovery")

	// Unknown emails get the same answer.
	expect(t, e.do(t, http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": uniqueEmail("nobody")}), http.StatusAccepted)
	expect(t, e.do(t, http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": u.Email}), http.StatusAccepted)
	code := e.mail.waitCode(t, u.Email)

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	expect(t, e.do(t, http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "n3w-password"}), http.StatusBadRequest)
	expect(t, e.do(t, http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "n3w-password"}), http.StatusNoContent)
	// Codes are single use.
	expect(t, e.do(t, http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "an0ther-pass"}), http.StatusBadRequest)

	// Existing sessions were revoked, the old password no longer works.
	expect(t, e.do(t, http.MethodGet, "/me", u.AccessToken, nil), http.StatusUnauthorized)
	expect(t, e.do(t, http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": u.Password}), http.StatusUnauthorized)
	expect(t, e.do(t, http.MethodPost, "/auth/login", "", map[string]string{"email": u.Email, "password": "n3w-password"}), http.StatusOK)
}

func TestPasswordRecoveryLocksAfterAttempts(t *testing.T) {
	e := setup(t)
	u := e.register(t, "bruteforce")
	expect(t, e.do(t, http.MethodPost, "/auth/password/forgot", "", map[string]string{"email": u.Email}), http.StatusAccepted)
	code := e.mail.waitCode(t, u.Email)
	wrong := "222222"
	if code == wrong {
		wrong = "333333"
	}
	// The recovery limiter is per IP (5 per 15 min) and the forgot call consumed one token,
	// so the remaining attempts reach the per-code counter.
	for i := 0; i < 4; i++ {
		r := e.do(t, http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": wrong, "newPassword": "n3w-password"})
		if r.Status != http.StatusBadRequest && r.Status != http.StatusTooManyRequests {
			t.Fatalf("unexpected status %d", r.Status)
		}
	}
	r := e.do(t, http.MethodPost, "/auth/password/reset", "", map[string]string{"email": u.Email, "code": code, "newPassword": "n3w-password"})
	if r.Status != http.StatusTooManyRequests {
		t.Fatalf("expected rate limiting after repeated attempts, got %d", r.Status)
	}
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := setup(t)
	u := e.register(t, "profile")

	expect(t, e.do(t, http.MethodGet, "/profile", u.AccessToken, nil), http.StatusNotFound)

	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	expect(t, e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{"firstName": "Léa", "birthdate": minor, "gender": "woman"}), http.StatusUnprocessableEntity)
	expect(t, e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{"firstName": "Léa", "birthdate": "1993-02-01", "gender": "robot"}), http.StatusBadRequest)
	expect(t, e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{"firstName": "Léa", "birthdate": "1993-02-01", "gender": "woman", "interestIds": []int{9999}}), http.StatusBadRequest)

	ok := e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{
		"firstName": "  Léa ", "birthdate": "1993-02-01", "gender": "woman", "bio": "Hello", "lookingFor": "relationship", "interestIds": []int{3, 4},
	})
	expect(t, ok, http.StatusOK)
	var own struct {
		FirstName   string `json:"firstName"`
		Age         int    `json:"age"`
		IsComplete  bool   `json:"isComplete"`
		Interests   []any  `json:"interests"`
		Preferences struct {
			InterestedIn []string `json:"interestedIn"`
		} `json:"preferences"`
	}
	ok.decode(t, &own)
	if own.FirstName != "Léa" || own.IsComplete || len(own.Interests) != 2 || len(own.Preferences.InterestedIn) != 3 {
		t.Fatalf("unexpected profile %s", ok.Body)
	}

	// Birthdate cannot be changed after creation.
	expect(t, e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{"firstName": "Léa", "birthdate": "2000-02-01", "gender": "woman"}), http.StatusUnprocessableEntity)

	// Discovery requires a photo.
	expect(t, e.do(t, http.MethodGet, "/discovery", u.AccessToken, nil), http.StatusConflict)

	// Coordinates are stored rounded (~1 km).
	expect(t, e.do(t, http.MethodPut, "/profile/location", u.AccessToken, map[string]any{"latitude": 48.856613, "longitude": 2.352222, "city": "Paris"}), http.StatusNoContent)
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.User.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be rounded, got %v,%v", lat, lng)
	}
	expect(t, e.do(t, http.MethodPut, "/profile/location", u.AccessToken, map[string]any{"latitude": 123, "longitude": 2}), http.StatusBadRequest)

	expect(t, e.do(t, http.MethodPut, "/profile/preferences", u.AccessToken, map[string]any{
		"interestedIn": []string{"man"}, "ageMin": 30, "ageMax": 20, "maxDistanceKm": 10, "isDiscoverable": true, "showDistance": true,
	}), http.StatusBadRequest)

	// Other users never see exact coordinates or birthdate.
	viewer := e.onboard(t, "viewer", profileSpec{Name: "Vic", Gender: "man", InterestedIn: []string{"woman"}, Lat: 48.9, Lng: 2.4})
	expect(t, e.upload(t, http.MethodPost, "/photos", u.AccessToken, pngImage(t, 300, 300, color.RGBA{1, 2, 3, 255})), http.StatusCreated)
	view := e.do(t, http.MethodGet, "/profiles/"+u.User.ID, viewer.AccessToken, nil)
	expect(t, view, http.StatusOK)
	body := string(view.Body)
	for _, leak := range []string{"latitude", "longitude", "birthdate", "48.86", "email"} {
		if strings.Contains(body, leak) {
			t.Fatalf("public profile leaks %q: %s", leak, body)
		}
	}
	expect(t, e.do(t, http.MethodGet, "/profiles/not-a-uuid", viewer.AccessToken, nil), http.StatusBadRequest)
}

func TestPhotos(t *testing.T) {
	e := setup(t)
	u := e.register(t, "photos")
	expect(t, e.do(t, http.MethodPut, "/profile", u.AccessToken, map[string]any{"firstName": "Inès", "birthdate": "1990-01-01", "gender": "woman"}), http.StatusOK)

	expect(t, e.upload(t, http.MethodPost, "/photos", u.AccessToken, []byte("<svg onload=alert(1)>")), http.StatusUnsupportedMediaType)
	expect(t, e.upload(t, http.MethodPost, "/photos", u.AccessToken, pngImage(t, 50, 50, color.RGBA{0, 0, 0, 255})), http.StatusBadRequest)

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.upload(t, http.MethodPost, "/photos", u.AccessToken, pngImage(t, 300, 400, color.RGBA{uint8(i * 30), 50, 50, 255}))
		expect(t, r, http.StatusCreated)
		var p struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
			URL      string `json:"url"`
		}
		r.decode(t, &p)
		if p.Position != i {
			t.Fatalf("photo %d got position %d", i, p.Position)
		}
		ids = append(ids, p.ID)
	}
	expect(t, e.upload(t, http.MethodPost, "/photos", u.AccessToken, pngImage(t, 300, 400, color.RGBA{1, 1, 1, 255})), http.StatusConflict)

	// Reorder: last becomes main photo.
	order := append([]string{ids[5]}, ids[:5]...)
	r := e.do(t, http.MethodPut, "/photos/order", u.AccessToken, map[string]any{"photoIds": order})
	expect(t, r, http.StatusOK)
	var list struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
			URL      string `json:"url"`
		} `json:"photos"`
	}
	r.decode(t, &list)
	if list.Photos[0].ID != ids[5] || list.Photos[0].Position != 0 {
		t.Fatalf("reorder failed: %s", r.Body)
	}
	expect(t, e.do(t, http.MethodPut, "/photos/order", u.AccessToken, map[string]any{"photoIds": ids[:3]}), http.StatusBadRequest)

	// Another user cannot touch these photos.
	other := e.register(t, "intruder")
	expect(t, e.do(t, http.MethodDelete, "/photos/"+ids[0], other.AccessToken, nil), http.StatusNotFound)
	expect(t, e.upload(t, http.MethodPut, "/photos/"+ids[0], other.AccessToken, pngImage(t, 300, 300, color.RGBA{9, 9, 9, 255})), http.StatusNotFound)

	// Signed media URL works; tampering does not.
	mediaURL := list.Photos[0].URL
	res, err := http.Get(e.server.URL + mediaURL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("media fetch: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	tampered := strings.Replace(mediaURL, "sig=", "sig=00", 1)
	res, _ = http.Get(e.server.URL + tampered)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("tampered media must be rejected, got %d", res.StatusCode)
	}
	res, _ = http.Get(e.server.URL + "/media/..%2F..%2Fetc%2Fpasswd?exp=9999999999&sig=x")
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("path traversal must be rejected, got %d", res.StatusCode)
	}

	// Replace keeps the position, delete compacts positions and removes the file.
	r = e.upload(t, http.MethodPut, "/photos/"+ids[1], u.AccessToken, pngImage(t, 300, 300, color.RGBA{9, 200, 9, 255}))
	expect(t, r, http.StatusOK)
	files, _ := os.ReadDir(e.uploadDir)
	before := len(files)
	r = e.do(t, http.MethodDelete, "/photos/"+ids[5], u.AccessToken, nil)
	expect(t, r, http.StatusOK)
	r.decode(t, &list)
	if len(list.Photos) != 5 || list.Photos[0].Position != 0 || list.Photos[4].Position != 4 {
		t.Fatalf("positions not compacted: %s", r.Body)
	}
	files, _ = os.ReadDir(e.uploadDir)
	if len(files) != before-1 {
		t.Fatalf("deleted photo file must be removed (%d -> %d)", before, len(files))
	}
	res, _ = http.Get(e.server.URL + mediaURL)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted photo must not be served, got %d", res.StatusCode)
	}

	// The last photo of a discoverable profile cannot be removed.
	for _, p := range list.Photos[1:] {
		expect(t, e.do(t, http.MethodDelete, "/photos/"+p.ID, u.AccessToken, nil), http.StatusOK)
	}
	expect(t, e.do(t, http.MethodDelete, "/photos/"+list.Photos[0].ID, u.AccessToken, nil), http.StatusConflict)
}

func TestDiscoveryFilters(t *testing.T) {
	e := setup(t)
	lat, lng := 10.0, 10.0
	viewer := e.onboard(t, "disc-viewer", profileSpec{Name: "Hugo", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	match := e.onboard(t, "disc-match", profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat + 0.05, Lng: lng})
	notIntoMen := e.onboard(t, "disc-women", profileSpec{Name: "Chloé", Gender: "woman", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	wrongGender := e.onboard(t, "disc-man", profileSpec{Name: "Marc", Gender: "man", InterestedIn: []string{"man", "woman"}, Lat: lat, Lng: lng})
	far := e.onboard(t, "disc-far", profileSpec{Name: "Zoé", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat + 3, Lng: lng})
	tooOld := e.onboard(t, "disc-old", profileSpec{Name: "Odette", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng, Birthdate: "1960-01-01"})
	hidden := e.onboard(t, "disc-hidden", profileSpec{Name: "Hélène", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	expect(t, e.do(t, http.MethodPut, "/profile/preferences", hidden.AccessToken, map[string]any{
		"interestedIn": []string{"man"}, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 50, "isDiscoverable": false, "showDistance": true,
	}), http.StatusOK)
	expect(t, e.do(t, http.MethodPut, "/profile/preferences", viewer.AccessToken, map[string]any{
		"interestedIn": []string{"woman"}, "ageMin": 18, "ageMax": 45, "maxDistanceKm": 50, "isDiscoverable": true, "showDistance": true,
	}), http.StatusOK)

	cards := e.discover(t, viewer)
	if !containsUser(cards, match.User.ID) {
		t.Fatal("compatible profile must be discoverable")
	}
	for name, u := range map[string]user{"not into men": notIntoMen, "wrong gender": wrongGender, "far": far, "too old": tooOld, "hidden": hidden, "self": viewer} {
		if containsUser(cards, u.User.ID) {
			t.Fatalf("%s profile must not be discoverable", name)
		}
	}
	for _, c := range cards {
		if c.UserID == match.User.ID {
			if c.DistanceKm == nil || *c.DistanceKm < 4 || *c.DistanceKm > 7 {
				t.Fatalf("approximate distance expected ~6km, got %v", c.DistanceKm)
			}
			if len(c.Photos) != 1 || len(c.Interests) != 2 {
				t.Fatalf("card must include photos and interests: %+v", c)
			}
		}
	}

	// Passing removes the profile for good.
	expect(t, e.do(t, http.MethodPost, "/discovery/"+match.User.ID+"/pass", viewer.AccessToken, nil), http.StatusOK)
	if containsUser(e.discover(t, viewer), match.User.ID) {
		t.Fatal("passed profile must not reappear")
	}
	expect(t, e.do(t, http.MethodPost, "/discovery/"+match.User.ID+"/like", viewer.AccessToken, nil), http.StatusConflict)
	expect(t, e.do(t, http.MethodPost, "/discovery/"+viewer.User.ID+"/like", viewer.AccessToken, nil), http.StatusBadRequest)
	expect(t, e.do(t, http.MethodPost, "/discovery/"+hidden.User.ID+"/like", viewer.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodGet, "/discovery?limit=500", viewer.AccessToken, nil), http.StatusBadRequest)
}

func TestLikeMatchAndNoDuplicates(t *testing.T) {
	e := setup(t)
	a := e.onboard(t, "like-a", profileSpec{Name: "Anaïs", Gender: "woman", InterestedIn: []string{"man"}, Lat: 20, Lng: 20})
	b := e.onboard(t, "like-b", profileSpec{Name: "Bruno", Gender: "man", InterestedIn: []string{"woman"}, Lat: 20, Lng: 20})

	if res := e.like(t, a, b); res.Matched {
		t.Fatal("one-sided like must not match")
	}
	expect(t, e.do(t, http.MethodPost, "/discovery/"+b.User.ID+"/like", a.AccessToken, nil), http.StatusConflict)

	// "Who liked me" is free.
	r := e.do(t, http.MethodGet, "/likes/received", b.AccessToken, nil)
	expect(t, r, http.StatusOK)
	if !strings.Contains(string(r.Body), a.User.ID) {
		t.Fatalf("likes received must contain the liker: %s", r.Body)
	}

	res := e.like(t, b, a)
	if !res.Matched || res.Match == nil || res.Match.ConversationID == "" || res.Match.User.UserID != a.User.ID {
		t.Fatalf("mutual like must create a match: %+v", res)
	}
	r = e.do(t, http.MethodGet, "/likes/received", b.AccessToken, nil)
	if strings.Contains(string(r.Body), a.User.ID) {
		t.Fatal("answered likes must leave the list")
	}

	var matches, conversations int
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.User.ID, b.User.ID).Scan(&matches)
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM conversations c JOIN matches m ON m.id = c.match_id WHERE m.user_a_id = LEAST($1::uuid,$2::uuid) AND m.user_b_id = GREATEST($1::uuid,$2::uuid)`, a.User.ID, b.User.ID).Scan(&conversations)
	if matches != 1 || conversations != 1 {
		t.Fatalf("expected exactly one match and conversation, got %d/%d", matches, conversations)
	}

	// Both receive a match notification.
	for _, u := range []user{a, b} {
		r := e.do(t, http.MethodGet, "/notifications", u.AccessToken, nil)
		expect(t, r, http.StatusOK)
		if !strings.Contains(string(r.Body), `"type":"match"`) {
			t.Fatalf("missing match notification: %s", r.Body)
		}
	}
}

func TestConcurrentMutualLikesCreateOneMatch(t *testing.T) {
	e := setup(t)
	for i := 0; i < 5; i++ {
		a := e.onboard(t, "race-a", profileSpec{Name: "Rose", Gender: "woman", InterestedIn: []string{"man"}, Lat: 30, Lng: 30})
		b := e.onboard(t, "race-b", profileSpec{Name: "Rémi", Gender: "man", InterestedIn: []string{"woman"}, Lat: 30, Lng: 30})
		var wg sync.WaitGroup
		results := make([]swipeResult, 2)
		for j, pair := range [][2]user{{a, b}, {b, a}} {
			wg.Add(1)
			go func(j int, from, to user) {
				defer wg.Done()
				r := e.do(t, http.MethodPost, "/discovery/"+to.User.ID+"/like", from.AccessToken, nil)
				json.Unmarshal(r.Body, &results[j])
			}(j, pair[0], pair[1])
		}
		wg.Wait()
		if results[0].Matched == results[1].Matched {
			t.Fatalf("exactly one of two simultaneous likes must report the match: %+v", results)
		}
		var n int
		e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a_id = LEAST($1::uuid,$2::uuid) AND user_b_id = GREATEST($1::uuid,$2::uuid)`, a.User.ID, b.User.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("expected one match, got %d", n)
		}
	}
}

type wsEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (e *env) connect(t *testing.T, u user) (*websocket.Conn, func(string) wsEvent) {
	t.Helper()
	r := e.do(t, http.MethodPost, "/realtime/ticket", u.AccessToken, nil)
	expect(t, r, http.StatusOK)
	var tk struct {
		Ticket string `json:"ticket"`
	}
	r.decode(t, &tk)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/realtime?ticket=" + tk.Ticket
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, "") })

	next := func(eventType string) wsEvent {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("waiting for %s: %v", eventType, err)
			}
			var ev wsEvent
			json.Unmarshal(data, &ev)
			if ev.Type == eventType {
				return ev
			}
		}
	}
	next("ready")

	// Tickets are single use.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	if _, _, err := websocket.Dial(ctx2, wsURL, nil); err == nil {
		t.Fatal("ticket reuse must be rejected")
	}
	return conn, next
}

type message struct {
	ID       string `json:"id"`
	SenderID string `json:"senderId"`
	Body     string `json:"body"`
	IsMine   bool   `json:"isMine"`
	IsRead   bool   `json:"isRead"`
}

func TestChatRealtimeAndPermissions(t *testing.T) {
	e := setup(t)
	a, b, _, conversationID := e.matchPair(t, 40, 40)
	_, nextB := e.connect(t, b)
	_, nextA := e.connect(t, a)

	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", a.AccessToken, map[string]string{"body": "   "}), http.StatusBadRequest)
	sent := e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", a.AccessToken, map[string]string{"body": " Salut Basile ! "})
	expect(t, sent, http.StatusCreated)
	var m message
	sent.decode(t, &m)
	if m.Body != "Salut Basile !" || !m.IsMine || m.IsRead {
		t.Fatalf("unexpected message %s", sent.Body)
	}

	// Real-time delivery to the recipient.
	ev := nextB("message.new")
	var received message
	json.Unmarshal(ev.Payload, &received)
	if received.ID != m.ID || received.IsMine {
		t.Fatalf("recipient got %s", ev.Payload)
	}
	nextB("notifications.unread")

	// Conversation list with unread count and partner info.
	r := e.do(t, http.MethodGet, "/conversations", b.AccessToken, nil)
	expect(t, r, http.StatusOK)
	var convs struct {
		Conversations []struct {
			ID          string   `json:"id"`
			UnreadCount int      `json:"unreadCount"`
			LastMessage *message `json:"lastMessage"`
			Partner     struct {
				UserID    string  `json:"userId"`
				FirstName string  `json:"firstName"`
				PhotoURL  *string `json:"photoUrl"`
			} `json:"partner"`
		} `json:"conversations"`
	}
	r.decode(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].Partner.UserID != a.User.ID || convs.Conversations[0].Partner.PhotoURL == nil {
		t.Fatalf("unexpected conversations %s", r.Body)
	}

	// Outsiders can neither read nor write.
	outsider := e.onboard(t, "outsider", profileSpec{Name: "Oscar", Gender: "man", InterestedIn: []string{"woman"}, Lat: 40, Lng: 40})
	expect(t, e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages", outsider.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodGet, "/conversations/"+conversationID, outsider.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", outsider.AccessToken, map[string]string{"body": "hi"}), http.StatusNotFound)
	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/read", outsider.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodDelete, "/conversations/"+conversationID, outsider.AccessToken, nil), http.StatusNotFound)

	// Read receipts reach the sender in real time.
	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/read", b.AccessToken, nil), http.StatusNoContent)
	nextA("conversation.read")
	r = e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages", a.AccessToken, nil)
	expect(t, r, http.StatusOK)
	var msgs struct {
		Messages []message `json:"messages"`
	}
	r.decode(t, &msgs)
	if len(msgs.Messages) != 1 || !msgs.Messages[0].IsRead {
		t.Fatalf("message must be read: %s", r.Body)
	}
	r = e.do(t, http.MethodGet, "/notifications/unread-count", b.AccessToken, nil)
	if !strings.Contains(string(r.Body), `"unreadCount":1`) {
		t.Fatalf("message notification must be cleared once read (only the match remains): %s", r.Body)
	}

	// Pagination.
	for i := 0; i < 4; i++ {
		expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", b.AccessToken, map[string]string{"body": "msg"}), http.StatusCreated)
	}
	r = e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages?limit=3", a.AccessToken, nil)
	var page struct {
		Messages   []message `json:"messages"`
		NextCursor *string   `json:"nextCursor"`
	}
	r.decode(t, &page)
	if len(page.Messages) != 3 || page.NextCursor == nil {
		t.Fatalf("unexpected page %s", r.Body)
	}
	r = e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages?limit=3&cursor="+*page.NextCursor, a.AccessToken, nil)
	r.decode(t, &page)
	if len(page.Messages) != 2 || page.NextCursor != nil {
		t.Fatalf("unexpected second page %s", r.Body)
	}

	// Local deletion hides history for one side only.
	expect(t, e.do(t, http.MethodDelete, "/conversations/"+conversationID, a.AccessToken, nil), http.StatusNoContent)
	r = e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages", a.AccessToken, nil)
	r.decode(t, &msgs)
	if len(msgs.Messages) != 0 {
		t.Fatal("cleared conversation must be empty for its owner")
	}
	r = e.do(t, http.MethodGet, "/conversations", a.AccessToken, nil)
	if strings.Contains(string(r.Body), conversationID) {
		t.Fatal("cleared conversation must leave the list until a new message")
	}
	r = e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages", b.AccessToken, nil)
	r.decode(t, &msgs)
	if len(msgs.Messages) != 5 {
		t.Fatalf("partner keeps the history, got %d", len(msgs.Messages))
	}
}

func TestUnmatchClosesConversation(t *testing.T) {
	e := setup(t)
	a, b, matchID, conversationID := e.matchPair(t, 45, 45)
	_, nextB := e.connect(t, b)
	expect(t, e.do(t, http.MethodDelete, "/matches/"+matchID, b.AccessToken, nil), http.StatusNoContent)
	nextB("conversation.closed")
	expect(t, e.do(t, http.MethodDelete, "/matches/"+matchID, a.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", a.AccessToken, map[string]string{"body": "hello?"}), http.StatusForbidden)
	r := e.do(t, http.MethodGet, "/conversations", a.AccessToken, nil)
	if strings.Contains(string(r.Body), conversationID) {
		t.Fatal("unmatched conversation must disappear")
	}
	expect(t, e.do(t, http.MethodGet, "/profiles/"+b.User.ID, a.AccessToken, nil), http.StatusOK) // still discoverable profile
}

func TestBlockAndReport(t *testing.T) {
	e := setup(t)
	a, b, _, conversationID := e.matchPair(t, 50, 50)

	expect(t, e.do(t, http.MethodPost, "/blocks", a.AccessToken, map[string]string{"userId": a.User.ID}), http.StatusBadRequest)
	expect(t, e.do(t, http.MethodPost, "/blocks", a.AccessToken, map[string]string{"userId": b.User.ID}), http.StatusNoContent)
	expect(t, e.do(t, http.MethodPost, "/blocks", a.AccessToken, map[string]string{"userId": b.User.ID}), http.StatusNoContent) // idempotent

	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", b.AccessToken, map[string]string{"body": "still there?"}), http.StatusForbidden)
	expect(t, e.do(t, http.MethodGet, "/conversations/"+conversationID+"/messages", b.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodGet, "/profiles/"+a.User.ID, b.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodGet, "/profiles/"+b.User.ID, a.AccessToken, nil), http.StatusNotFound)
	r := e.do(t, http.MethodGet, "/notifications", b.AccessToken, nil)
	if strings.Contains(string(r.Body), a.User.ID) {
		t.Fatal("notifications involving a blocker must be removed")
	}
	r = e.do(t, http.MethodGet, "/blocks", a.AccessToken, nil)
	if !strings.Contains(string(r.Body), b.User.ID) {
		t.Fatalf("block list must contain the blocked user: %s", r.Body)
	}

	// A blocked user disappears from discovery both ways.
	c := e.onboard(t, "block-c", profileSpec{Name: "Clara", Gender: "woman", InterestedIn: []string{"man"}, Lat: 50, Lng: 50})
	d := e.onboard(t, "block-d", profileSpec{Name: "David", Gender: "man", InterestedIn: []string{"woman"}, Lat: 50, Lng: 50})
	if !containsUser(e.discover(t, c), d.User.ID) {
		t.Fatal("precondition: d must be discoverable by c")
	}
	expect(t, e.do(t, http.MethodPost, "/reports", c.AccessToken, map[string]string{"userId": d.User.ID, "reason": "nonsense"}), http.StatusBadRequest)
	expect(t, e.do(t, http.MethodPost, "/reports", c.AccessToken, map[string]string{"userId": d.User.ID, "reason": "harassment", "details": "messages insistants"}), http.StatusCreated)
	if containsUser(e.discover(t, c), d.User.ID) || containsUser(e.discover(t, d), c.User.ID) {
		t.Fatal("reported users are blocked in both directions")
	}
	expect(t, e.do(t, http.MethodPost, "/discovery/"+c.User.ID+"/like", d.AccessToken, nil), http.StatusNotFound)
	var reports int
	e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_id = $1 AND reported_id = $2 AND status = 'open'`, c.User.ID, d.User.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected a stored report, got %d", reports)
	}

	// Unblocking does not restore the match.
	expect(t, e.do(t, http.MethodDelete, "/blocks/"+b.User.ID, a.AccessToken, nil), http.StatusNoContent)
	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", a.AccessToken, map[string]string{"body": "x"}), http.StatusForbidden)
}

func TestNotificationsReadFlow(t *testing.T) {
	e := setup(t)
	a, _, _, _ := e.matchPair(t, 55, 55)
	r := e.do(t, http.MethodGet, "/notifications?limit=1", a.AccessToken, nil)
	expect(t, r, http.StatusOK)
	var list struct {
		Notifications []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Actor *struct {
				FirstName string `json:"firstName"`
			} `json:"actor"`
		} `json:"notifications"`
		UnreadCount int `json:"unreadCount"`
	}
	r.decode(t, &list)
	if len(list.Notifications) != 1 || list.UnreadCount != 1 || list.Notifications[0].Actor == nil || list.Notifications[0].Actor.FirstName != "Basile" {
		t.Fatalf("unexpected notifications %s", r.Body)
	}
	other := e.register(t, "notif-other")
	expect(t, e.do(t, http.MethodPost, "/notifications/"+list.Notifications[0].ID+"/read", other.AccessToken, nil), http.StatusNotFound)
	expect(t, e.do(t, http.MethodPost, "/notifications/"+list.Notifications[0].ID+"/read", a.AccessToken, nil), http.StatusNoContent)
	r = e.do(t, http.MethodGet, "/notifications/unread-count", a.AccessToken, nil)
	if !strings.Contains(string(r.Body), `"unreadCount":0`) {
		t.Fatalf("expected no unread notification: %s", r.Body)
	}
	expect(t, e.do(t, http.MethodPost, "/notifications/read", a.AccessToken, nil), http.StatusNoContent)
}

func TestAccountDeletionErasesData(t *testing.T) {
	e := setup(t)
	a, b, _, conversationID := e.matchPair(t, 60, 60)
	expect(t, e.do(t, http.MethodPost, "/conversations/"+conversationID+"/messages", a.AccessToken, map[string]string{"body": "bonjour"}), http.StatusCreated)

	var keys []string
	rows, _ := e.pool.Query(context.Background(), `SELECT storage_key FROM photos WHERE user_id = $1`, a.User.ID)
	for rows.Next() {
		var k string
		rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) == 0 {
		t.Fatal("precondition: user has photos")
	}

	expect(t, e.do(t, http.MethodDelete, "/me", a.AccessToken, map[string]string{"password": "wrong-passw0rd"}), http.StatusForbidden)
	expect(t, e.do(t, http.MethodDelete, "/me", a.AccessToken, map[string]string{"password": a.Password}), http.StatusNoContent)

	expect(t, e.do(t, http.MethodGet, "/me", a.AccessToken, nil), http.StatusUnauthorized)
	expect(t, e.do(t, http.MethodPost, "/auth/login", "", map[string]string{"email": a.Email, "password": a.Password}), http.StatusUnauthorized)
	for _, k := range keys {
		if _, err := os.Stat(filepath.Join(e.uploadDir, k)); !os.IsNotExist(err) {
			t.Fatalf("photo file %s must be deleted", k)
		}
	}
	var leftovers int
	e.pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM profiles WHERE user_id = $1) + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM swipes WHERE swiper_id = $1 OR target_id = $1)
		     + (SELECT COUNT(*) FROM messages WHERE sender_id = $1)
		     + (SELECT COUNT(*) FROM sessions WHERE user_id = $1)
	`, a.User.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("personal data left after deletion: %d rows", leftovers)
	}
	r := e.do(t, http.MethodGet, "/conversations", b.AccessToken, nil)
	if strings.Contains(string(r.Body), conversationID) {
		t.Fatal("partner must not keep a conversation with a deleted account")
	}
}
