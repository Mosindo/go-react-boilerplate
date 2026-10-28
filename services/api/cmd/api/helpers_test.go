package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/testutil"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testSecret = "0123456789abcdef0123456789abcdef-integration"

type captureMailer struct {
	mu   sync.Mutex
	sent []string
}

func (m *captureMailer) Send(to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, to+"\n"+body)
	return nil
}

// lastCode waits for the async mail goroutine and extracts the recovery code.
func (m *captureMailer) lastCode(t *testing.T, to string) string {
	t.Helper()
	for i := 0; i < 100; i++ {
		m.mu.Lock()
		for j := len(m.sent) - 1; j >= 0; j-- {
			if strings.HasPrefix(m.sent[j], to+"\n") {
				body := m.sent[j]
				m.mu.Unlock()
				idx := strings.Index(body, "code is: ")
				return body[idx+len("code is: ") : idx+len("code is: ")+8]
			}
		}
		m.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no mail sent to %s", to)
	return ""
}

type env struct {
	t    *testing.T
	srv  *httptest.Server
	mail *captureMailer
	hub  *realtime.Hub
	pool *pgxpool.Pool
}

func newEnv(t *testing.T) *env {
	t.Helper()
	pool := testutil.NewPool(t)
	blobs, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := &captureMailer{}
	hub := realtime.NewHub()
	srv := httptest.NewServer(setupRouter(&app{dbPool: pool, jwtSecret: []byte(testSecret), mailer: m, blobs: blobs, hub: hub}))
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, mail: m, hub: hub, pool: pool}
}

type result struct {
	Status int
	Body   map[string]any
	Raw    []byte
	Header http.Header
}

func (r result) str(path ...string) string {
	var cur any = r.Body
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[p]
	}
	s, _ := cur.(string)
	return s
}

func (r result) list(key string) []any {
	l, _ := r.Body[key].([]any)
	return l
}

func (e *env) call(method, path, token string, body any) result {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return e.do(req)
}

func (e *env) do(req *http.Request) result {
	e.t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := result{Status: resp.StatusCode, Raw: raw, Header: resp.Header}
	_ = json.Unmarshal(raw, &out.Body)
	return out
}

func (e *env) expect(r result, status int) result {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("expected status %d, got %d: %s", status, r.Status, string(r.Raw))
	}
	return r
}

func (e *env) upload(token, field, filename, contentType string, data []byte) result {
	e.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, filename))
	h.Set("Content-Type", contentType)
	part, _ := w.CreatePart(h)
	_, _ = part.Write(data)
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/photos", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	return e.do(req)
}

func testJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func testPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 200, G: uint8(x % 256), B: 10, A: 120})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

type user struct {
	ID, Email, Token, Refresh, Password string
}

type userSpec struct {
	Name         string
	Gender       string
	Age          int
	InterestedIn []string
	MinAge       int
	MaxAge       int
	MaxDistance  int
	Lat, Lon     float64
	HasLocation  bool
	Interests    []int
	NoPhoto      bool
}

func birthDateForAge(age int) string {
	return time.Now().AddDate(-age, 0, -2).Format("2006-01-02")
}

// onboard registers a user and completes profile, location, preferences and a photo.
func (e *env) onboard(spec userSpec) user {
	e.t.Helper()
	u := user{Email: strings.ToLower(spec.Name) + "-" + fmt.Sprint(time.Now().UnixNano()) + "@test.local", Password: "Sup3rSecret!"}
	reg := e.expect(e.call("POST", "/auth/register", "", map[string]any{"email": u.Email, "password": u.Password}), 201)
	u.ID, u.Token, u.Refresh = reg.str("user", "id"), reg.str("accessToken"), reg.str("refreshToken")

	e.expect(e.call("PUT", "/profile", u.Token, map[string]any{
		"firstName": spec.Name, "birthDate": birthDateForAge(spec.Age), "gender": spec.Gender,
		"bio": "Hello from " + spec.Name, "city": "Paris", "interestIds": spec.Interests,
		"showDistance": true, "discoverable": true,
	}), 200)
	if spec.HasLocation {
		e.expect(e.call("PUT", "/location", u.Token, map[string]any{"latitude": spec.Lat, "longitude": spec.Lon}), 200)
	}
	minAge, maxAge, dist := spec.MinAge, spec.MaxAge, spec.MaxDistance
	if minAge == 0 {
		minAge = 18
	}
	if maxAge == 0 {
		maxAge = 99
	}
	if dist == 0 {
		dist = 100
	}
	e.expect(e.call("PUT", "/preferences", u.Token, map[string]any{
		"interestedIn": spec.InterestedIn, "minAge": minAge, "maxAge": maxAge, "maxDistanceKm": dist,
	}), 200)
	if !spec.NoPhoto {
		e.expect(e.upload(u.Token, "photo", "me.jpg", "image/jpeg", testJPEG(640, 800)), 201)
	}
	return u
}

// wsClient connects and authenticates; events arrive on the returned channel.
func (e *env) ws(token string) (*websocket.Conn, <-chan map[string]any) {
	e.t.Helper()
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		e.t.Fatalf("dial ws: %v", err)
	}
	e.t.Cleanup(func() { _ = conn.Close() })
	if err := conn.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
		e.t.Fatal(err)
	}
	events := make(chan map[string]any, 32)
	go func() {
		defer close(events)
		for {
			var ev map[string]any
			if err := conn.ReadJSON(&ev); err != nil {
				return
			}
			events <- ev
		}
	}()
	select {
	case ev := <-events:
		if ev["type"] != "ready" {
			e.t.Fatalf("expected ready, got %v", ev)
		}
	case <-time.After(3 * time.Second):
		e.t.Fatal("ws did not become ready")
	}
	return conn, events
}

func waitEvent(t *testing.T, events <-chan map[string]any, kind string) map[string]any {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("ws closed while waiting for %q", kind)
			}
			if ev["type"] == kind {
				return ev
			}
		case <-deadline:
			t.Fatalf("timed out waiting for ws event %q", kind)
		}
	}
}

func websocketDial(url string) (*websocket.Conn, *http.Response, error) {
	return websocket.DefaultDialer.Dial(url, nil)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
