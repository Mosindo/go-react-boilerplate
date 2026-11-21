package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- helpers ---------------------------------------------------------------

type member struct {
	ID       string
	Email    string
	Token    string
	Refresh  string
	Password string
}

type photoPayload struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
}

type cardPayload struct {
	UserID     string         `json:"userId"`
	FirstName  string         `json:"firstName"`
	Age        int            `json:"age"`
	DistanceKm *int           `json:"distanceKm"`
	Photos     []photoPayload `json:"photos"`
	LikedYou   bool           `json:"likedYou"`
}

type feedPayload struct {
	Profiles []cardPayload `json:"profiles"`
}

type swipePayload struct {
	Matched bool `json:"matched"`
	Match   *struct {
		ID   string `json:"id"`
		User struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
		} `json:"user"`
	} `json:"match"`
}

type chatListPayload struct {
	Chats []struct {
		MatchID string `json:"matchId"`
		User    struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
		} `json:"user"`
		UnreadCount int `json:"unreadCount"`
		LastMessage *struct {
			Content string `json:"content"`
		} `json:"lastMessage"`
	} `json:"chats"`
}

type messagesPayload struct {
	Messages []struct {
		ID           string `json:"id"`
		SenderUserID string `json:"senderUserId"`
		Content      string `json:"content"`
		IsRead       bool   `json:"isRead"`
	} `json:"messages"`
	HasMore bool `json:"hasMore"`
}

type meProfilePayload struct {
	IsComplete bool           `json:"isComplete"`
	Missing    []string       `json:"missing"`
	Photos     []photoPayload `json:"photos"`
	Profile    *struct {
		FirstName   string `json:"firstName"`
		Age         int    `json:"age"`
		HasLocation bool   `json:"hasLocation"`
	} `json:"profile"`
	Interests []struct {
		ID int `json:"id"`
	} `json:"interests"`
}

func decode[T any](t *testing.T, res testResponse) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(res.Body, &out); err != nil {
		t.Fatalf("decode %T: %v body=%s", out, err, string(res.Body))
	}
	return out
}

func expectStatus(t *testing.T, res testResponse, want int, what string) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("%s: expected %d, got %d body=%s", what, want, res.Status, string(res.Body))
	}
}

var memberSeq int
var memberMu sync.Mutex

func registerMember(t *testing.T, router http.Handler, prefix string) member {
	t.Helper()
	memberMu.Lock()
	memberSeq++
	email := fmt.Sprintf("%s_%d_%s", prefix, memberSeq, uniqueEmail("dating"))
	memberMu.Unlock()
	res := doRequest(t, router, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": "Password123"}, "")
	expectStatus(t, res, http.StatusCreated, "register "+prefix)
	auth := decode[authResponse](t, res)
	return member{ID: auth.User.ID, Email: email, Token: auth.AccessToken, Refresh: auth.RefreshToken, Password: "Password123"}
}

func testPNG(t *testing.T, seed uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 320, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{seed, uint8(x), uint8(y), 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png: %v", err)
	}
	return buf.Bytes()
}

func uploadPhoto(t *testing.T, router http.Handler, token string, data []byte, method, path string) testResponse {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "photo.png")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	_, _ = part.Write(data)
	_ = w.Close()
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return testResponse{Status: rec.Code, Body: rec.Body.Bytes()}
}

type profileSpec struct {
	FirstName    string
	Birthdate    string
	Gender       string
	InterestedIn []string
	Lat, Lng     float64
	Interests    []int
}

func completeProfile(t *testing.T, router http.Handler, m member, spec profileSpec) {
	t.Helper()
	res := doRequest(t, router, http.MethodPut, "/me/profile", map[string]any{
		"firstName": spec.FirstName,
		"birthdate": spec.Birthdate,
		"gender":    spec.Gender,
		"bio":       "Bonjour, je suis " + spec.FirstName,
		"city":      "Paris",
	}, m.Token)
	expectStatus(t, res, http.StatusOK, "upsert profile "+spec.FirstName)
	res = doRequest(t, router, http.MethodPut, "/me/preferences", map[string]any{
		"interestedIn": spec.InterestedIn, "minAge": 18, "maxAge": 60, "maxDistanceKm": 50,
	}, m.Token)
	expectStatus(t, res, http.StatusOK, "preferences "+spec.FirstName)
	if len(spec.Interests) > 0 {
		res = doRequest(t, router, http.MethodPut, "/me/interests", map[string]any{"interestIds": spec.Interests}, m.Token)
		expectStatus(t, res, http.StatusOK, "interests "+spec.FirstName)
	}
	res = doRequest(t, router, http.MethodPut, "/me/location", map[string]any{"latitude": spec.Lat, "longitude": spec.Lng}, m.Token)
	expectStatus(t, res, http.StatusNoContent, "location "+spec.FirstName)
	res = uploadPhoto(t, router, m.Token, testPNG(t, 10), http.MethodPost, "/me/photos")
	expectStatus(t, res, http.StatusCreated, "upload photo "+spec.FirstName)
}

func feed(t *testing.T, router http.Handler, m member) feedPayload {
	t.Helper()
	res := doRequest(t, router, http.MethodGet, "/discovery?limit=20", nil, m.Token)
	expectStatus(t, res, http.StatusOK, "discovery")
	return decode[feedPayload](t, res)
}

func feedHas(f feedPayload, userID string) (cardPayload, bool) {
	for _, p := range f.Profiles {
		if p.UserID == userID {
			return p, true
		}
	}
	return cardPayload{}, false
}

func swipe(t *testing.T, router http.Handler, m member, target, action string) testResponse {
	t.Helper()
	return doRequest(t, router, http.MethodPost, "/swipes", map[string]string{"targetUserId": target, "action": action}, m.Token)
}

func cleanupMembers(t *testing.T, pool *pgxpool.Pool, members ...member) {
	t.Helper()
	emails := make([]string, 0, len(members))
	for _, m := range members {
		emails = append(emails, m.Email)
	}
	cleanupUsers(t, pool, emails)
}

// isolatedTenant moves members to a fresh organization so discovery results
// are not polluted by data from other tests or a seeded database.
func isolatedTenant(t *testing.T, router http.Handler, pool *pgxpool.Pool, members ...*member) {
	t.Helper()
	orgID := createOrganization(t, pool, "t-"+strings.ReplaceAll(uniqueEmail("org"), "@", "-"), "Test tenant")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE organization_id = $1`, orgID)
		cleanupOrganizations(t, pool, []string{orgID})
	})
	for _, m := range members {
		moveUserToOrganization(t, pool, m.ID, orgID)
		res := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": m.Password}, "")
		expectStatus(t, res, http.StatusOK, "relogin")
		auth := decode[authResponse](t, res)
		m.Token, m.Refresh = auth.AccessToken, auth.RefreshToken
	}
}

// --- tests -----------------------------------------------------------------

func TestDatingEndToEndFlow(t *testing.T) {
	router, pool := setupTestRouter(t)

	alice := registerMember(t, router, "alice")
	bob := registerMember(t, router, "bob")
	chloe := registerMember(t, router, "chloe")
	defer cleanupMembers(t, pool, alice, bob, chloe)
	isolatedTenant(t, router, pool, &alice, &bob, &chloe)

	// Discovery requires a complete profile.
	expectStatus(t, doRequest(t, router, http.MethodGet, "/discovery", nil, alice.Token), http.StatusConflict, "discovery without profile")

	// Minors are rejected server-side.
	minorBirth := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	res := doRequest(t, router, http.MethodPut, "/me/profile", map[string]any{"firstName": "Alice", "birthdate": minorBirth, "gender": "woman"}, alice.Token)
	expectStatus(t, res, http.StatusUnprocessableEntity, "underage profile")

	// Invalid preferences are rejected.
	completeProfile(t, router, alice, profileSpec{FirstName: "Alice", Birthdate: "1994-04-02", Gender: "woman", InterestedIn: []string{"man"}, Lat: 48.8566, Lng: 2.3522, Interests: []int{1, 2, 3}})
	res = doRequest(t, router, http.MethodPut, "/me/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10}, alice.Token)
	expectStatus(t, res, http.StatusBadRequest, "invalid age range")

	completeProfile(t, router, bob, profileSpec{FirstName: "Bob", Birthdate: "1992-09-12", Gender: "man", InterestedIn: []string{"woman"}, Lat: 48.8666, Lng: 2.3333, Interests: []int{2, 3, 4}})
	completeProfile(t, router, chloe, profileSpec{FirstName: "Chloé", Birthdate: "1996-01-20", Gender: "woman", InterestedIn: []string{"woman"}, Lat: 48.85, Lng: 2.35})

	me := decode[meProfilePayload](t, doRequest(t, router, http.MethodGet, "/me/profile", nil, alice.Token))
	if !me.IsComplete || len(me.Photos) != 1 || me.Profile == nil || !me.Profile.HasLocation || len(me.Interests) != 3 {
		t.Fatalf("unexpected own profile: %+v", me)
	}

	// Mutual preferences: Alice sees Bob, not Chloé (Chloé wants women but Alice wants men).
	aliceFeed := feed(t, router, alice)
	bobCard, ok := feedHas(aliceFeed, bob.ID)
	if !ok {
		t.Fatalf("alice should discover bob: %+v", aliceFeed)
	}
	if _, ok := feedHas(aliceFeed, chloe.ID); ok {
		t.Fatal("alice must not discover chloé (incompatible preferences)")
	}
	if bobCard.DistanceKm == nil || *bobCard.DistanceKm < 1 || *bobCard.DistanceKm > 5 {
		t.Fatalf("expected an approximate distance, got %v", bobCard.DistanceKm)
	}
	if len(bobCard.Photos) != 1 {
		t.Fatalf("expected bob's photo in card")
	}

	// Cards never leak private data.
	rawFeed := doRequest(t, router, http.MethodGet, "/discovery", nil, alice.Token)
	for _, forbidden := range []string{bob.Email, "latitude", "longitude", "birthdate", "password"} {
		if strings.Contains(string(rawFeed.Body), forbidden) {
			t.Fatalf("discovery payload leaks %q", forbidden)
		}
	}

	// Signed media URL works, tampering is rejected.
	photoURL := bobCard.Photos[0].URL
	media := doRequest(t, router, http.MethodGet, photoURL, nil, "")
	expectStatus(t, media, http.StatusOK, "media download")
	if !bytes.HasPrefix(media.Body, []byte{0xFF, 0xD8}) {
		t.Fatal("photos must be served as re-encoded JPEG")
	}
	tampered := regexp.MustCompile(`sig=[^&]+`).ReplaceAllString(photoURL, "sig=forged")
	expectStatus(t, doRequest(t, router, http.MethodGet, tampered, nil, ""), http.StatusForbidden, "tampered media url")

	// Like -> no match yet; repeated like is refused.
	first := decode[swipePayload](t, swipe(t, router, alice, bob.ID, "like"))
	if first.Matched {
		t.Fatal("one-sided like must not match")
	}
	expectStatus(t, swipe(t, router, alice, bob.ID, "like"), http.StatusConflict, "duplicate like")
	expectStatus(t, swipe(t, router, alice, alice.ID, "like"), http.StatusBadRequest, "self like")
	expectStatus(t, swipe(t, router, alice, bob.ID, "superlike"), http.StatusBadRequest, "invalid action")

	// Processed profiles never resurface.
	if _, ok := feedHas(feed(t, router, alice), bob.ID); ok {
		t.Fatal("liked profile must not reappear in discovery")
	}

	// Bob sees that Alice liked him, likes back -> match.
	bobFeed := feed(t, router, bob)
	aliceCard, ok := feedHas(bobFeed, alice.ID)
	if !ok || !aliceCard.LikedYou {
		t.Fatalf("bob should see alice with likedYou=true: %+v", bobFeed)
	}
	if bobFeed.Profiles[0].UserID != alice.ID {
		t.Fatal("members who already liked you are ranked first")
	}
	matchRes := swipe(t, router, bob, alice.ID, "like")
	expectStatus(t, matchRes, http.StatusCreated, "reciprocal like")
	matched := decode[swipePayload](t, matchRes)
	if !matched.Matched || matched.Match == nil || matched.Match.User.ID != alice.ID {
		t.Fatalf("expected match with alice, got %+v", matched)
	}
	var matchCount int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid,$2::uuid) AND user_high_id = GREATEST($1::uuid,$2::uuid)`, alice.ID, bob.ID).Scan(&matchCount); err != nil || matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d (%v)", matchCount, err)
	}

	// Both get a match notification.
	notifs := decode[struct {
		Notifications []notificationResponse `json:"notifications"`
		UnreadCount   int                    `json:"unreadCount"`
	}](t, doRequest(t, router, http.MethodGet, "/notifications", nil, alice.Token))
	if notifs.UnreadCount < 1 || notifs.Notifications[0].Type != "match.new" {
		t.Fatalf("expected a match notification for alice: %+v", notifs)
	}

	// Matched profile is viewable; a stranger's hidden profile is not.
	expectStatus(t, doRequest(t, router, http.MethodGet, "/profiles/"+bob.ID, nil, alice.Token), http.StatusOK, "view matched profile")

	// Chat: send, list with unread count, read receipts.
	sent := doRequest(t, router, http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": "  Salut Bob !  "}, alice.Token)
	expectStatus(t, sent, http.StatusCreated, "send message")
	bobChats := decode[chatListPayload](t, doRequest(t, router, http.MethodGet, "/chats", nil, bob.Token))
	if len(bobChats.Chats) != 1 || bobChats.Chats[0].User.ID != alice.ID || bobChats.Chats[0].UnreadCount != 1 {
		t.Fatalf("unexpected bob chats: %+v", bobChats)
	}
	if bobChats.Chats[0].LastMessage == nil || bobChats.Chats[0].LastMessage.Content != "Salut Bob !" {
		t.Fatalf("unexpected last message: %+v", bobChats.Chats[0].LastMessage)
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+alice.ID+"/read", nil, bob.Token), http.StatusOK, "mark read")
	aliceMsgs := decode[messagesPayload](t, doRequest(t, router, http.MethodGet, "/chats/"+bob.ID+"/messages", nil, alice.Token))
	if len(aliceMsgs.Messages) != 1 || !aliceMsgs.Messages[0].IsRead {
		t.Fatalf("expected message to be read by bob: %+v", aliceMsgs)
	}
	bobChats = decode[chatListPayload](t, doRequest(t, router, http.MethodGet, "/chats", nil, bob.Token))
	if bobChats.Chats[0].UnreadCount != 0 {
		t.Fatal("unread count must reset after reading")
	}

	// Permissions: a third member can neither read nor write this conversation.
	expectStatus(t, doRequest(t, router, http.MethodGet, "/chats/"+alice.ID+"/messages", nil, chloe.Token), http.StatusNotFound, "outsider read")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+alice.ID+"/messages", map[string]string{"content": "hi"}, chloe.Token), http.StatusNotFound, "outsider send")
	expectStatus(t, swipe(t, router, chloe, alice.ID, "like"), http.StatusCreated, "chloé can still like alice")

	// Local clear hides history for the caller only.
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/chats/"+alice.ID+"/messages", nil, bob.Token), http.StatusNoContent, "clear chat")
	if msgs := decode[messagesPayload](t, doRequest(t, router, http.MethodGet, "/chats/"+alice.ID+"/messages", nil, bob.Token)); len(msgs.Messages) != 0 {
		t.Fatal("cleared conversation must be empty for bob")
	}
	if msgs := decode[messagesPayload](t, doRequest(t, router, http.MethodGet, "/chats/"+bob.ID+"/messages", nil, alice.Token)); len(msgs.Messages) != 1 {
		t.Fatal("clearing must not affect alice")
	}

	// Block: conversation disappears, chat and profile become unreachable.
	expectStatus(t, doRequest(t, router, http.MethodPost, "/blocks", map[string]string{"userId": alice.ID}, bob.Token), http.StatusNoContent, "block")
	if chats := decode[chatListPayload](t, doRequest(t, router, http.MethodGet, "/chats", nil, alice.Token)); len(chats.Chats) != 0 {
		t.Fatal("blocked conversation must disappear")
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+bob.ID+"/messages", map[string]string{"content": "hey?"}, alice.Token), http.StatusNotFound, "send after block")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/profiles/"+bob.ID, nil, alice.Token), http.StatusNotFound, "view blocker profile")
	blocks := decode[struct {
		Blocks []struct {
			User struct{ ID string } `json:"user"`
		} `json:"blocks"`
	}](t, doRequest(t, router, http.MethodGet, "/blocks", nil, bob.Token))
	if len(blocks.Blocks) != 1 || blocks.Blocks[0].User.ID != alice.ID {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}

	// Report: stored with a valid reason, and it blocks automatically.
	expectStatus(t, doRequest(t, router, http.MethodPost, "/reports", map[string]string{"userId": alice.ID, "reason": "nope"}, chloe.Token), http.StatusBadRequest, "invalid reason")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/reports", map[string]string{"userId": alice.ID, "reason": "spam", "details": "spam links"}, chloe.Token), http.StatusCreated, "report")
	var reports int
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_user_id = $1 AND reported_user_id = $2`, chloe.ID, alice.ID).Scan(&reports)
	if reports != 1 {
		t.Fatalf("expected one report, got %d", reports)
	}
	var blocked bool
	_ = pool.QueryRow(context.Background(), `SELECT EXISTS (SELECT 1 FROM blocks WHERE blocker_user_id = $1 AND blocked_user_id = $2)`, chloe.ID, alice.ID).Scan(&blocked)
	if !blocked {
		t.Fatal("report must block the reported member")
	}

	// Account deletion: password required, data and files removed, tokens revoked.
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/me", map[string]string{"password": "WrongPass1"}, alice.Token), http.StatusForbidden, "delete with wrong password")
	alicePhotoURL := me.Photos[0].URL
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/me", map[string]string{"password": alice.Password}, alice.Token), http.StatusNoContent, "delete account")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/me", nil, alice.Token), http.StatusUnauthorized, "token after deletion")
	expectStatus(t, doRequest(t, router, http.MethodGet, alicePhotoURL, nil, ""), http.StatusNotFound, "photo after deletion")
	var leftovers int
	_ = pool.QueryRow(context.Background(), `
		SELECT (SELECT COUNT(*) FROM profiles WHERE user_id = $1) + (SELECT COUNT(*) FROM photos WHERE user_id = $1)
		     + (SELECT COUNT(*) FROM swipes WHERE actor_user_id = $1 OR target_user_id = $1)
		     + (SELECT COUNT(*) FROM messages WHERE sender_user_id = $1)
	`, alice.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("expected all personal data removed, %d rows left", leftovers)
	}
	_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM reports WHERE reporter_user_id = $1 AND reported_user_id IS NULL`, chloe.ID).Scan(&reports)
	if reports != 1 {
		t.Fatal("moderation records are kept with the reference anonymized")
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	router, pool := setupTestRouter(t)

	for i := 0; i < 5; i++ {
		a := registerMember(t, router, "race_a")
		b := registerMember(t, router, "race_b")
		isolatedTenant(t, router, pool, &a, &b)
		completeProfile(t, router, a, profileSpec{FirstName: "Ana", Birthdate: "1990-01-01", Gender: "woman", InterestedIn: []string{"man"}, Lat: 45.76, Lng: 4.83})
		completeProfile(t, router, b, profileSpec{FirstName: "Ben", Birthdate: "1990-01-01", Gender: "man", InterestedIn: []string{"woman"}, Lat: 45.76, Lng: 4.84})

		var wg sync.WaitGroup
		results := make([]swipePayload, 2)
		for idx, pair := range [][2]member{{a, b}, {b, a}} {
			wg.Add(1)
			go func(idx int, actor, target member) {
				defer wg.Done()
				res := swipe(t, router, actor, target.ID, "like")
				if res.Status != http.StatusCreated {
					t.Errorf("concurrent like: %d %s", res.Status, string(res.Body))
					return
				}
				_ = json.Unmarshal(res.Body, &results[idx])
			}(idx, pair[0], pair[1])
		}
		wg.Wait()

		matchedResponses := 0
		for _, r := range results {
			if r.Matched {
				matchedResponses++
			}
		}
		var count int
		_ = pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_low_id = LEAST($1::uuid,$2::uuid) AND user_high_id = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&count)
		if count != 1 || matchedResponses != 1 {
			t.Fatalf("iteration %d: expected exactly one match (rows=%d, matched responses=%d)", i, count, matchedResponses)
		}
		cleanupMembers(t, pool, a, b)
	}
}

func TestPhotoManagement(t *testing.T) {
	router, pool := setupTestRouter(t)
	m := registerMember(t, router, "photos")
	defer cleanupMembers(t, pool, m)

	expectStatus(t, uploadPhoto(t, router, m.Token, []byte("<script>alert(1)</script>"), http.MethodPost, "/me/photos"), http.StatusUnprocessableEntity, "html upload")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/me/photos", nil, m.Token), http.StatusBadRequest, "missing file")

	var photos []photoPayload
	for i := 0; i < 6; i++ {
		res := uploadPhoto(t, router, m.Token, testPNG(t, uint8(i*30)), http.MethodPost, "/me/photos")
		expectStatus(t, res, http.StatusCreated, "upload")
		photos = decode[struct {
			Photos []photoPayload `json:"photos"`
		}](t, res).Photos
	}
	if len(photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(photos))
	}
	expectStatus(t, uploadPhoto(t, router, m.Token, testPNG(t, 1), http.MethodPost, "/me/photos"), http.StatusConflict, "7th photo")

	// Reorder: reverse the list, the last photo becomes the main one.
	ids := make([]string, len(photos))
	for i, p := range photos {
		ids[len(photos)-1-i] = p.ID
	}
	res := doRequest(t, router, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": ids}, m.Token)
	expectStatus(t, res, http.StatusOK, "reorder")
	reordered := decode[struct {
		Photos []photoPayload `json:"photos"`
	}](t, res).Photos
	if reordered[0].ID != photos[5].ID || reordered[0].Position != 0 {
		t.Fatalf("unexpected order: %+v", reordered)
	}
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/photos/order", map[string]any{"photoIds": ids[:2]}, m.Token), http.StatusBadRequest, "partial order")

	// Replace keeps the position, delete compacts positions.
	res = uploadPhoto(t, router, m.Token, testPNG(t, 99), http.MethodPut, "/me/photos/"+reordered[0].ID)
	expectStatus(t, res, http.StatusOK, "replace")
	res = doRequest(t, router, http.MethodDelete, "/me/photos/"+reordered[1].ID, nil, m.Token)
	expectStatus(t, res, http.StatusOK, "delete")
	after := decode[struct {
		Photos []photoPayload `json:"photos"`
	}](t, res).Photos
	if len(after) != 5 || after[0].ID != reordered[0].ID {
		t.Fatalf("unexpected photos after delete: %+v", after)
	}
	for i, p := range after {
		if p.Position != i {
			t.Fatalf("positions must stay compact: %+v", after)
		}
	}

	// Someone else cannot touch these photos.
	other := registerMember(t, router, "photos_other")
	defer cleanupMembers(t, pool, other)
	expectStatus(t, doRequest(t, router, http.MethodDelete, "/me/photos/"+after[0].ID, nil, other.Token), http.StatusNotFound, "delete foreign photo")
}

type captureMailer struct {
	mu   sync.Mutex
	last mailer.Message
}

func (c *captureMailer) Send(_ context.Context, msg mailer.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last = msg
	return nil
}

func TestPasswordRecoveryFlow(t *testing.T) {
	router, pool, mail := setupRouterWithMailer(t)
	m := registerMember(t, router, "recovery")
	defer cleanupMembers(t, pool, m)

	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/forgot", map[string]string{"email": "unknown_" + m.Email}, ""), http.StatusAccepted, "forgot unknown")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/forgot", map[string]string{"email": strings.ToUpper(m.Email)}, ""), http.StatusAccepted, "forgot")
	code := regexp.MustCompile(`\b\d{6}\b`).FindString(mail.last.Body)
	if code == "" || mail.last.To != m.Email {
		t.Fatalf("expected a code to be mailed, got %+v", mail.last)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": wrong, "newPassword": "NewPassword42"}, ""), http.StatusBadRequest, "wrong code")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "weak"}, ""), http.StatusBadRequest, "weak password")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "NewPassword42"}, ""), http.StatusNoContent, "reset")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "OtherPassword42"}, ""), http.StatusBadRequest, "code reuse")

	// Old sessions are revoked, the old password no longer works.
	expectStatus(t, doRequest(t, router, http.MethodGet, "/me", nil, m.Token), http.StatusUnauthorized, "old access token")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/refresh", map[string]string{"refreshToken": m.Refresh}, ""), http.StatusUnauthorized, "old refresh token")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": "Password123"}, ""), http.StatusUnauthorized, "old password")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": m.Email, "password": "NewPassword42"}, ""), http.StatusOK, "new password")

	// Brute force protection: the code is invalidated after 5 wrong attempts.
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/forgot", map[string]string{"email": m.Email}, ""), http.StatusAccepted, "forgot again")
	code = regexp.MustCompile(`\b\d{6}\b`).FindString(mail.last.Body)
	wrong = "222222"
	if code == wrong {
		wrong = "333333"
	}
	for i := 0; i < 5; i++ {
		doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": wrong, "newPassword": "Another42pass"}, "")
	}
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/password/reset", map[string]string{"email": m.Email, "code": code, "newPassword": "Another42pass"}, ""), http.StatusBadRequest, "code after too many attempts")
}

func TestLogoutRevokesAccessTokenImmediately(t *testing.T) {
	router, pool := setupTestRouter(t)
	m := registerMember(t, router, "logout")
	defer cleanupMembers(t, pool, m)
	expectStatus(t, doRequest(t, router, http.MethodGet, "/me", nil, m.Token), http.StatusOK, "me before logout")
	expectStatus(t, doRequest(t, router, http.MethodPost, "/auth/logout", map[string]string{"refreshToken": m.Refresh}, ""), http.StatusNoContent, "logout")
	expectStatus(t, doRequest(t, router, http.MethodGet, "/me", nil, m.Token), http.StatusUnauthorized, "me after logout")
}

func TestRealtimeDeliversMessagesAndMatches(t *testing.T) {
	router, pool := setupTestRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()

	a := registerMember(t, router, "rt_a")
	b := registerMember(t, router, "rt_b")
	defer cleanupMembers(t, pool, a, b)
	isolatedTenant(t, router, pool, &a, &b)
	completeProfile(t, router, a, profileSpec{FirstName: "Lina", Birthdate: "1993-05-05", Gender: "woman", InterestedIn: []string{"man"}, Lat: 43.6, Lng: 1.44})
	completeProfile(t, router, b, profileSpec{FirstName: "Marc", Birthdate: "1991-05-05", Gender: "man", InterestedIn: []string{"woman"}, Lat: 43.6, Lng: 1.45})

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/realtime"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "auth", "token": b.Token})
	next := func() map[string]any {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev map[string]any
		if err := conn.ReadJSON(&ev); err != nil {
			t.Fatalf("read event: %v", err)
		}
		return ev
	}
	if ev := next(); ev["type"] != "ready" {
		t.Fatalf("expected ready, got %v", ev)
	}

	expectStatus(t, swipe(t, router, b, a.ID, "like"), http.StatusCreated, "b likes a")
	expectStatus(t, swipe(t, router, a, b.ID, "like"), http.StatusCreated, "a likes b")
	seen := map[string]bool{}
	for len(seen) < 2 {
		seen[fmt.Sprint(next()["type"])] = true
	}
	if !seen["match.new"] || !seen["notification.new"] {
		t.Fatalf("expected match and notification events, got %v", seen)
	}

	expectStatus(t, doRequest(t, router, http.MethodPost, "/chats/"+b.ID+"/messages", map[string]string{"content": "coucou"}, a.Token), http.StatusCreated, "send")
	for {
		ev := next()
		if ev["type"] != "message.new" {
			continue
		}
		data := ev["data"].(map[string]any)
		msg := data["message"].(map[string]any)
		if msg["content"] != "coucou" || data["fromUserId"] != a.ID {
			t.Fatalf("unexpected message event %v", ev)
		}
		break
	}

	// An invalid token is refused.
	bad, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer bad.Close()
	_ = bad.WriteJSON(map[string]string{"type": "auth", "token": "forged"})
	_ = bad.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("forged token must be rejected")
	}
}

func TestRateLimitProtectsLogin(t *testing.T) {
	_, pool := setupTestRouter(t)
	store, _ := storage.NewLocal(t.TempDir())
	router := setupRouter(&app{dbPool: pool, jwtSecret: []byte("integration-test-secret"), store: store, rateLimits: true})

	limited := false
	for i := 0; i < 25; i++ {
		res := doRequest(t, router, http.MethodPost, "/auth/login", map[string]string{"email": "nobody@ratelimit.test", "password": "Password123"}, "")
		if res.Status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected login to be rate limited")
	}
}

func TestProfileValidationAndPrivacySettings(t *testing.T) {
	router, pool := setupTestRouter(t)
	a := registerMember(t, router, "privacy_a")
	b := registerMember(t, router, "privacy_b")
	defer cleanupMembers(t, pool, a, b)
	isolatedTenant(t, router, pool, &a, &b)

	bad := doRequest(t, router, http.MethodPut, "/me/profile", map[string]any{"firstName": "R2D2", "birthdate": "1990-01-01", "gender": "woman"}, a.Token)
	expectStatus(t, bad, http.StatusBadRequest, "invalid first name")
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/location", map[string]any{"latitude": 44.8, "longitude": -0.5}, a.Token), http.StatusNotFound, "location without profile")

	completeProfile(t, router, a, profileSpec{FirstName: "Inès", Birthdate: "1990-01-01", Gender: "woman", InterestedIn: []string{"man"}, Lat: 44.84, Lng: -0.58})
	completeProfile(t, router, b, profileSpec{FirstName: "Hugo", Birthdate: "1990-01-01", Gender: "man", InterestedIn: []string{"woman"}, Lat: 44.84, Lng: -0.57})
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/location", map[string]any{"latitude": 123, "longitude": 2}, a.Token), http.StatusBadRequest, "invalid coordinates")

	// Exact coordinates are never stored.
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/location", map[string]any{"latitude": 44.837789, "longitude": -0.57918}, a.Token), http.StatusNoContent, "precise location")
	var lat, lng float64
	_ = pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat, &lng)
	if fmt.Sprintf("%.6f", lat) != "44.840000" || fmt.Sprintf("%.6f", lng) != "-0.580000" {
		t.Fatalf("location must be coarsened, got %v,%v", lat, lng)
	}

	// Hiding distance.
	expectStatus(t, doRequest(t, router, http.MethodPatch, "/me/settings", map[string]any{"showDistance": false}, a.Token), http.StatusOK, "hide distance")
	card, ok := feedHas(feed(t, router, b), a.ID)
	if !ok || card.DistanceKm != nil {
		t.Fatalf("distance must be hidden: %+v", card)
	}

	// Pausing the profile removes it from discovery.
	expectStatus(t, doRequest(t, router, http.MethodPatch, "/me/settings", map[string]any{"isVisible": false}, a.Token), http.StatusOK, "pause profile")
	if _, ok := feedHas(feed(t, router, b), a.ID); ok {
		t.Fatal("paused profile must not be discoverable")
	}
	expectStatus(t, doRequest(t, router, http.MethodGet, "/profiles/"+a.ID, nil, b.Token), http.StatusNotFound, "paused profile not viewable by strangers")

	// Interests catalogue.
	interests := decode[struct {
		Interests []struct{ ID int } `json:"interests"`
	}](t, doRequest(t, router, http.MethodGet, "/interests", nil, a.Token))
	if len(interests.Interests) < 20 {
		t.Fatal("expected interests catalogue")
	}
	expectStatus(t, doRequest(t, router, http.MethodPut, "/me/interests", map[string]any{"interestIds": []int{9999}}, a.Token), http.StatusBadRequest, "unknown interest")
}

func TestDiscoveryExcludesAndPaginates(t *testing.T) {
	router, pool := setupTestRouter(t)
	viewer := registerMember(t, router, "disc_viewer")
	others := make([]member, 4)
	ptrs := []*member{&viewer}
	for i := range others {
		others[i] = registerMember(t, router, "disc_other")
		ptrs = append(ptrs, &others[i])
	}
	defer cleanupMembers(t, pool, append(others, viewer)...)
	isolatedTenant(t, router, pool, ptrs...)

	completeProfile(t, router, viewer, profileSpec{FirstName: "Zoé", Birthdate: "1990-01-01", Gender: "woman", InterestedIn: []string{"man"}, Lat: 50.63, Lng: 3.06})
	for i, o := range others {
		lat := 50.63
		if i == 3 {
			lat = 48.85 // ~200 km away, outside the 50 km radius
		}
		completeProfile(t, router, o, profileSpec{FirstName: "Paul", Birthdate: "1990-01-01", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: 3.06})
	}

	res := doRequest(t, router, http.MethodGet, "/discovery?limit=2", nil, viewer.Token)
	expectStatus(t, res, http.StatusOK, "discovery page")
	page := decode[feedPayload](t, res)
	if len(page.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(page.Profiles))
	}
	exclude := url.QueryEscape(page.Profiles[0].UserID + "," + page.Profiles[1].UserID)
	rest := decode[feedPayload](t, doRequest(t, router, http.MethodGet, "/discovery?limit=10&exclude="+exclude, nil, viewer.Token))
	if len(rest.Profiles) != 1 {
		t.Fatalf("expected only the remaining nearby profile, got %d", len(rest.Profiles))
	}
	if _, ok := feedHas(rest, others[3].ID); ok {
		t.Fatal("profiles outside the distance radius must be excluded")
	}
	expectStatus(t, doRequest(t, router, http.MethodGet, "/discovery?exclude=not-a-uuid", nil, viewer.Token), http.StatusBadRequest, "invalid exclude")

	// A pass is final too.
	expectStatus(t, swipe(t, router, viewer, rest.Profiles[0].UserID, "pass"), http.StatusCreated, "pass")
	again := decode[feedPayload](t, doRequest(t, router, http.MethodGet, "/discovery?limit=10&exclude="+exclude, nil, viewer.Token))
	if len(again.Profiles) != 0 {
		t.Fatal("passed profile must not reappear")
	}
}

func setupRouterWithMailer(t *testing.T) (*gin.Engine, *pgxpool.Pool, *captureMailer) {
	t.Helper()
	_, pool := setupTestRouter(t)
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	mail := &captureMailer{}
	router := setupRouter(&app{dbPool: pool, jwtSecret: []byte("integration-test-secret"), store: store, mailer: mail})
	return router, pool, mail
}
