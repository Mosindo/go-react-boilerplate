package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHealth(t *testing.T) {
	pool := testutil.Pool(t)
	router := newTestRouter(t, pool)
	resp := testutil.Do(t, router, http.MethodGet, "/health", nil, "")
	if resp.Status != http.StatusOK || !strings.Contains(string(resp.Body), `"status":"ok"`) {
		t.Fatalf("unexpected health: %d %s", resp.Status, resp.Body)
	}
}

type journeyUser struct {
	ID, Token, Email string
}

// TestDatingJourney drives the real router through the critical path:
// register -> profile -> location -> photo -> preferences -> discover -> like -> match -> chat -> block -> delete account.
func TestDatingJourney(t *testing.T) {
	pool := testutil.Pool(t)
	router := newTestRouter(t, pool)
	suffix := time.Now().UTC().Format("150405.000000000")

	a := register(t, router, "journey_a_"+suffix+"@journey.test")
	b := register(t, router, "journey_b_"+suffix+"@journey.test")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = ANY($1)`, []string{a.ID, b.ID})
	})

	// A minor cannot register.
	resp := testutil.Do(t, router, http.MethodPost, "/auth/register", map[string]any{
		"email": "kid_" + suffix + "@journey.test", "password": "password123", "birthDate": time.Now().AddDate(-16, 0, 0).Format("2006-01-02"),
	}, "")
	if resp.Status != http.StatusUnprocessableEntity {
		t.Fatalf("under-18 register: expected 422, got %d %s", resp.Status, resp.Body)
	}

	// Before onboarding: profile incomplete, discovery refused.
	var me struct {
		ProfileComplete bool `json:"profileComplete"`
	}
	testutil.Do(t, router, http.MethodGet, "/me", nil, a.Token).JSON(t, &me)
	if me.ProfileComplete {
		t.Fatal("new account must not be profileComplete")
	}
	if r := testutil.Do(t, router, http.MethodGet, "/discover", nil, a.Token); r.Status != http.StatusUnprocessableEntity {
		t.Fatalf("discover without profile: expected 422, got %d %s", r.Status, r.Body)
	}

	onboard(t, router, a, "Ada", "woman", []string{"man"}, 45.76, 4.84)
	onboard(t, router, b, "Ben", "man", []string{"woman"}, 45.77, 4.85)

	testutil.Do(t, router, http.MethodGet, "/me", nil, a.Token).JSON(t, &me)
	if !me.ProfileComplete {
		t.Fatal("expected profileComplete after onboarding")
	}

	// Discovery: A sees B, never herself.
	var disc struct {
		Profiles []struct {
			UserID string `json:"userId"`
			Photos []struct {
				URL string `json:"url"`
			} `json:"photos"`
		} `json:"profiles"`
	}
	testutil.Do(t, router, http.MethodGet, "/discover?limit=10", nil, a.Token).JSON(t, &disc)
	found := false
	for _, p := range disc.Profiles {
		if p.UserID == a.ID {
			t.Fatal("discovery returned the viewer")
		}
		if p.UserID == b.ID {
			found = true
			if len(p.Photos) == 0 {
				t.Fatal("candidate has no signed photo url")
			}
			img := testutil.Do(t, router, http.MethodGet, p.Photos[0].URL, nil, "")
			if img.Status != http.StatusOK {
				t.Fatalf("signed photo url should load without a token, got %d", img.Status)
			}
			bad := testutil.Do(t, router, http.MethodGet, strings.Replace(p.Photos[0].URL, "sig=", "sig=00", 1), nil, "")
			if bad.Status != http.StatusNotFound {
				t.Fatalf("tampered signature must 404, got %d", bad.Status)
			}
		}
	}
	if !found {
		t.Fatalf("B not discoverable by A: %s", "empty result")
	}

	// A likes B: no match yet. Duplicate swipe is a conflict.
	var swipe struct {
		Matched      bool `json:"matched"`
		Conversation *struct {
			ID string `json:"id"`
		} `json:"conversation"`
	}
	testutil.Do(t, router, http.MethodPost, "/swipes", map[string]any{"userId": b.ID, "action": "like"}, a.Token).JSON(t, &swipe)
	if swipe.Matched {
		t.Fatal("one-sided like must not match")
	}
	if r := testutil.Do(t, router, http.MethodPost, "/swipes", map[string]any{"userId": b.ID, "action": "like"}, a.Token); r.Status != http.StatusConflict {
		t.Fatalf("duplicate swipe: expected 409, got %d", r.Status)
	}

	// B likes A: match + conversation.
	testutil.Do(t, router, http.MethodPost, "/swipes", map[string]any{"userId": a.ID, "action": "like"}, b.Token).JSON(t, &swipe)
	if !swipe.Matched || swipe.Conversation == nil {
		t.Fatalf("mutual like must match: %+v", swipe)
	}
	convID := swipe.Conversation.ID

	// Both got a match notification.
	for _, u := range []journeyUser{a, b} {
		var n struct {
			UnreadCount int `json:"unreadCount"`
		}
		testutil.Do(t, router, http.MethodGet, "/notifications", nil, u.Token).JSON(t, &n)
		if n.UnreadCount < 1 {
			t.Fatalf("user %s expected a match notification", u.ID)
		}
	}

	// Chat.
	if r := testutil.Do(t, router, http.MethodPost, "/conversations/"+convID+"/messages", map[string]any{"body": "  Hello Ada  "}, b.Token); r.Status != http.StatusCreated {
		t.Fatalf("send: %d %s", r.Status, r.Body)
	}
	var convs struct {
		Conversations []struct {
			ID          string `json:"id"`
			UnreadCount int    `json:"unreadCount"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	testutil.Do(t, router, http.MethodGet, "/conversations", nil, a.Token).JSON(t, &convs)
	if len(convs.Conversations) != 1 || convs.Conversations[0].UnreadCount != 1 || convs.Conversations[0].LastMessage.Body != "Hello Ada" {
		t.Fatalf("unexpected conversations for A: %s", mustJSON(convs))
	}

	// An outsider cannot read the conversation.
	c := register(t, router, "journey_c_"+suffix+"@journey.test")
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, c.ID) })
	if r := testutil.Do(t, router, http.MethodGet, "/conversations/"+convID+"/messages", nil, c.Token); r.Status != http.StatusNotFound {
		t.Fatalf("outsider read: expected 404, got %d", r.Status)
	}

	if r := testutil.Do(t, router, http.MethodPost, "/conversations/"+convID+"/read", nil, a.Token); r.Status != http.StatusNoContent {
		t.Fatalf("mark read: %d", r.Status)
	}
	testutil.Do(t, router, http.MethodGet, "/conversations", nil, a.Token).JSON(t, &convs)
	if convs.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread should be zero after read")
	}

	// Block: conversation disappears and B leaves A's discovery / cannot be messaged.
	if r := testutil.Do(t, router, http.MethodPost, "/blocks", map[string]any{"userId": b.ID}, a.Token); r.Status != http.StatusNoContent {
		t.Fatalf("block: %d %s", r.Status, r.Body)
	}
	testutil.Do(t, router, http.MethodGet, "/conversations", nil, a.Token).JSON(t, &convs)
	if len(convs.Conversations) != 0 {
		t.Fatal("blocked conversation must disappear")
	}
	if r := testutil.Do(t, router, http.MethodPost, "/conversations/"+convID+"/messages", map[string]any{"body": "hi"}, b.Token); r.Status == http.StatusCreated {
		t.Fatal("blocked user must not be able to send")
	}

	// Report.
	if r := testutil.Do(t, router, http.MethodPost, "/reports", map[string]any{"userId": b.ID, "reason": "spam"}, a.Token); r.Status != http.StatusCreated {
		t.Fatalf("report: %d %s", r.Status, r.Body)
	}

	// Account deletion erases the user and their data.
	if r := testutil.Do(t, router, http.MethodDelete, "/me", map[string]any{"password": "password123"}, b.Token); r.Status != http.StatusNoContent {
		t.Fatalf("delete account: %d %s", r.Status, r.Body)
	}
	var count int
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE id = $1`, b.ID).Scan(&count)
	if count != 0 {
		t.Fatal("user row should be gone")
	}
	_ = pool.QueryRow(context.Background(), `SELECT count(*) FROM photos WHERE user_id = $1`, b.ID).Scan(&count)
	if count != 0 {
		t.Fatal("photos should be gone")
	}
}

func newTestRouter(t *testing.T, pool *pgxpool.Pool) *gin.Engine {
	t.Helper()
	store, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	return setupRouter(config.Config{Env: "test", JWTSecret: testutil.Secret}, pool, store)
}

func register(t *testing.T, router http.Handler, email string) journeyUser {
	t.Helper()
	var out struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	resp := testutil.Do(t, router, http.MethodPost, "/auth/register", map[string]any{
		"email": email, "password": "password123", "birthDate": time.Now().AddDate(-30, 0, 0).Format("2006-01-02"),
	}, "")
	if resp.Status != http.StatusCreated {
		t.Fatalf("register %s: %d %s", email, resp.Status, resp.Body)
	}
	resp.JSON(t, &out)
	return journeyUser{ID: out.User.ID, Token: out.AccessToken, Email: email}
}

func onboard(t *testing.T, router http.Handler, u journeyUser, name, gender string, interestedIn []string, lat, lng float64) {
	t.Helper()
	steps := []struct {
		method, path string
		body         map[string]any
	}{
		{http.MethodPut, "/me/profile", map[string]any{
			"firstName": name, "gender": gender, "bio": "Hello", "city": "Lyon", "interests": []string{"travel", "music"},
			"showDistance": true, "showAge": true, "discoverable": true,
		}},
		{http.MethodPut, "/me/preferences", map[string]any{"interestedIn": interestedIn, "ageMin": 18, "ageMax": 99, "maxDistanceKm": 100}},
		{http.MethodPut, "/me/location", map[string]any{"latitude": lat, "longitude": lng}},
	}
	for _, s := range steps {
		if r := testutil.Do(t, router, s.method, s.path, s.body, u.Token); r.Status >= 300 {
			t.Fatalf("%s %s: %d %s", s.method, s.path, r.Status, r.Body)
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 120, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", "me.png")
	_, _ = part.Write(pngBuf.Bytes())
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/me/photos", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+u.Token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload photo: %d %s", rec.Code, rec.Body.String())
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
