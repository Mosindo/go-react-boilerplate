package app_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"example.com/api/internal/app"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

func TestAuthLifecycle(t *testing.T) {
	e := newEnv(t)
	email := "Mixed.Case@Example.org"
	res := e.do("POST", "/auth/register", "", map[string]any{"email": "  " + email + " ", "password": testPassword})
	if res.Status != 201 || res.str("user", "email") != strings.ToLower(email) || res.Body["user"].(map[string]any)["profileComplete"] != false {
		t.Fatalf("register: %d %s", res.Status, res.Raw)
	}
	if res.str("user", "passwordHash") != "" || contains(string(res.Raw), "password") {
		t.Fatalf("response leaks password data: %s", res.Raw)
	}
	for name, body := range map[string]map[string]any{
		"duplicate (case-insensitive)": {"email": strings.ToUpper(email), "password": testPassword},
		"short password":               {"email": "x@example.org", "password": "short"},
		"bad email":                    {"email": "nope", "password": testPassword},
		"72+ byte password":            {"email": "y@example.org", "password": strings.Repeat("é", 40)},
	} {
		if r := e.do("POST", "/auth/register", "", body); r.Status < 400 {
			t.Errorf("%s accepted", name)
		}
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": email, "password": "wrong-password"}); r.Status != 401 {
		t.Errorf("wrong password: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": "ghost@example.org", "password": testPassword}); r.Status != 401 || r.str("error") != "invalid credentials" {
		t.Errorf("unknown email must look identical: %d %s", r.Status, r.Raw)
	}

	login := e.do("POST", "/auth/login", "", map[string]any{"email": email, "password": testPassword})
	if login.Status != 200 {
		t.Fatalf("login: %d", login.Status)
	}
	refresh := login.str("refreshToken")
	next := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": refresh})
	if next.Status != 200 || next.str("refreshToken") == refresh {
		t.Fatalf("refresh must rotate: %d %s", next.Status, next.Raw)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": refresh}); r.Status != 401 {
		t.Errorf("replayed refresh token accepted: %d", r.Status)
	}
	if r := e.do("GET", "/me", next.str("accessToken"), nil); r.Status != 200 || r.str("email") != strings.ToLower(email) {
		t.Errorf("me: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": next.str("refreshToken")}); r.Status != 204 {
		t.Errorf("logout: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": next.str("refreshToken")}); r.Status != 401 {
		t.Errorf("refresh after logout: %d", r.Status)
	}
}

func TestAccessTokenValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("tok")
	secret := []byte("integration-secret-0123456789abcdef0123")

	sign := func(method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	parsed, _, err := jwt.NewParser().ParseUnverified(u.Token, jwt.MapClaims{})
	if err != nil {
		t.Fatal(err)
	}
	sid := parsed.Claims.(jwt.MapClaims)["sid"].(string)
	valid := jwt.MapClaims{"uid": u.ID, "sid": sid, "exp": time.Now().Add(time.Minute).Unix()}
	noExp := jwt.MapClaims{"uid": u.ID, "sid": sid}
	expired := jwt.MapClaims{"uid": u.ID, "sid": sid, "exp": time.Now().Add(-time.Minute).Unix()}
	noUID := jwt.MapClaims{"sid": "s", "exp": time.Now().Add(time.Minute).Unix()}

	cases := map[string]struct {
		token string
		want  int
	}{
		"valid":            {sign(jwt.SigningMethodHS256, valid, secret), 200},
		"no expiry":        {sign(jwt.SigningMethodHS256, noExp, secret), 401},
		"expired":          {sign(jwt.SigningMethodHS256, expired, secret), 401},
		"missing uid":      {sign(jwt.SigningMethodHS256, noUID, secret), 401},
		"wrong secret":     {sign(jwt.SigningMethodHS256, valid, []byte("another-secret-another-secret-123456")), 401},
		"HS512":            {sign(jwt.SigningMethodHS512, valid, secret), 401},
		"alg none":         {sign(jwt.SigningMethodNone, valid, jwt.UnsafeAllowNoneSignatureType), 401},
		"garbage":          {"not.a.jwt", 401},
		"tampered payload": {tamper(sign(jwt.SigningMethodHS256, valid, secret), u.ID), 401},
	}
	for name, c := range cases {
		if r := e.do("GET", "/me", c.token, nil); r.Status != c.want {
			t.Errorf("%s: want %d got %d", name, c.want, r.Status)
		}
	}
	// HMAC with attacker-chosen key over a forged payload.
	forged := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(`{"uid":"`+u.ID+`","sid":"s","exp":9999999999}`))
	mac := hmac.New(sha256.New, []byte("attacker"))
	mac.Write([]byte(forged))
	if r := e.do("GET", "/me", forged+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil); r.Status != 401 {
		t.Errorf("forged token accepted: %d", r.Status)
	}
}

func tamper(token, uid string) string {
	parts := strings.Split(token, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	changed := strings.Replace(string(payload), uid, "00000000-0000-0000-0000-000000000000", 1)
	parts[1] = base64.RawURLEncoding.EncodeToString([]byte(changed))
	return strings.Join(parts, ".")
}

func TestPasswordRecovery(t *testing.T) {
	e := newEnv(t)
	u := e.register("recover")
	if r := e.do("POST", "/auth/password-reset/request", "", map[string]any{"email": "ghost@example.org"}); r.Status != 202 {
		t.Fatalf("unknown email must also answer 202: %d", r.Status)
	}
	if e.mail.body("ghost@example.org") != "" {
		t.Fatal("mail sent to unknown address")
	}
	if r := e.do("POST", "/auth/password-reset/request", "", map[string]any{"email": strings.ToUpper(u.Email)}); r.Status != 202 {
		t.Fatalf("request: %d", r.Status)
	}
	code := regexp.MustCompile(`code is: ([A-Z0-9]{10})`).FindStringSubmatch(e.mail.body(u.Email))
	if code == nil {
		t.Fatalf("no code in mail: %q", e.mail.body(u.Email))
	}
	if r := e.do("POST", "/auth/password-reset/confirm", "", map[string]any{"token": "AAAAAAAAAA", "password": "NewPassw0rd!"}); r.Status != 400 {
		t.Errorf("wrong code accepted: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password-reset/confirm", "", map[string]any{"token": strings.ToLower(code[1]), "password": "NewPassw0rd!"}); r.Status != 204 {
		t.Fatalf("confirm: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password-reset/confirm", "", map[string]any{"token": code[1], "password": "Another1Passw0rd"}); r.Status != 400 {
		t.Errorf("code reuse accepted: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": u.Refresh}); r.Status != 401 {
		t.Errorf("old sessions must be revoked: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": u.Email, "password": testPassword}); r.Status != 401 {
		t.Errorf("old password still works")
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": u.Email, "password": "NewPassw0rd!"}); r.Status != 200 {
		t.Errorf("new password rejected: %d", r.Status)
	}
}

func TestRateLimiting(t *testing.T) {
	e := newEnv(t, func(d *app.Deps) { d.Limits.AuthPerMinute, d.Limits.AuthBurst = 1, 3 })
	var last int
	for i := 0; i < 6; i++ {
		last = e.do("POST", "/auth/login", "", map[string]any{"email": "a@example.org", "password": "whatever-pass"}).Status
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("login brute force not throttled: %d", last)
	}
}

func TestMessageRateLimit(t *testing.T) {
	e := newEnv(t, func(d *app.Deps) { d.Limits.SendPerMinute, d.Limits.SendBurst = 1, 2 })
	a := e.onboard("alice", woman("Alice"))
	b := e.onboard("bob", man("Bob"))
	e.do("POST", "/swipes", a.Token, map[string]any{"targetUserId": b.ID, "action": "like"})
	conv := e.do("POST", "/swipes", b.Token, map[string]any{"targetUserId": a.ID, "action": "like"}).str("match", "conversationId")
	var statuses []int
	for i := 0; i < 4; i++ {
		statuses = append(statuses, e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "spam"}).Status)
	}
	if statuses[0] != 201 || statuses[3] != http.StatusTooManyRequests {
		t.Fatalf("send limiter: %v", statuses)
	}
	// Limits are per user: the other side is unaffected.
	if r := e.do("POST", "/conversations/"+conv+"/messages", b.Token, map[string]any{"body": "hi"}); r.Status != 201 {
		t.Fatalf("limit leaked across users: %d", r.Status)
	}
}

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("val")
	underage := time.Now().AddDate(-17, 0, -1+1).Format("2006-01-02")
	base := func(mod func(m map[string]any)) map[string]any {
		m := map[string]any{"firstName": "Zoé", "birthDate": "1990-05-05", "gender": "woman", "bio": "ok", "city": "Lyon", "interests": []string{"music"}}
		mod(m)
		return m
	}
	cases := map[string]struct {
		body map[string]any
		want int
	}{
		"valid unicode name":  {base(func(m map[string]any) {}), 200},
		"underage":            {base(func(m map[string]any) { m["birthDate"] = underage }), 403},
		"exactly 18 tomorrow": {base(func(m map[string]any) { m["birthDate"] = time.Now().AddDate(-18, 0, 1).Format("2006-01-02") }), 403},
		"exactly 18 today":    {base(func(m map[string]any) { m["birthDate"] = time.Now().AddDate(-18, 0, 0).Format("2006-01-02") }), 200},
		"future":              {base(func(m map[string]any) { m["birthDate"] = "2999-01-01" }), 403},
		"implausible age":     {base(func(m map[string]any) { m["birthDate"] = "1900-01-01" }), 400},
		"bad date":            {base(func(m map[string]any) { m["birthDate"] = "05/05/1990" }), 400},
		"bad gender":          {base(func(m map[string]any) { m["gender"] = "robot" }), 400},
		"name with digits":    {base(func(m map[string]any) { m["firstName"] = "R2D2" }), 400},
		"name with link":      {base(func(m map[string]any) { m["firstName"] = "http://x.co" }), 400},
		"empty name":          {base(func(m map[string]any) { m["firstName"] = "  " }), 400},
		"long bio":            {base(func(m map[string]any) { m["bio"] = strings.Repeat("b", 501) }), 400},
		"unknown interest":    {base(func(m map[string]any) { m["interests"] = []string{"nope"} }), 400},
		"too many interests": {base(func(m map[string]any) {
			m["interests"] = []string{"hiking", "cooking", "travel", "music", "concerts", "movies", "reading", "gaming", "fitness", "yoga", "running"}
		}), 400},
	}
	for name, c := range cases {
		if r := e.do("PUT", "/profile/me", u.Token, c.body); r.Status != c.want {
			t.Errorf("%s: want %d got %d (%s)", name, c.want, r.Status, r.Raw)
		}
	}

	prefs := map[string]map[string]any{
		"min>max":         {"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10},
		"under 18":        {"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30, "maxDistanceKm": 10},
		"no gender":       {"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10},
		"bad gender":      {"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10},
		"distance bounds": {"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 100000},
	}
	for name, body := range prefs {
		if r := e.do("PUT", "/profile/preferences", u.Token, body); r.Status != 400 {
			t.Errorf("prefs %s: got %d", name, r.Status)
		}
	}
	for name, body := range map[string]map[string]any{
		"lat range": {"latitude": 91, "longitude": 0},
		"lng range": {"latitude": 0, "longitude": 181},
		"missing":   {"latitude": 10},
	} {
		if r := e.do("PUT", "/profile/location", u.Token, body); r.Status != 400 {
			t.Errorf("location %s: got %d", name, r.Status)
		}
	}
	// A location can only be stored once a profile exists, and is rounded to ~1 km.
	fresh := e.register("noprofile")
	if r := e.do("PUT", "/profile/location", fresh.Token, map[string]any{"latitude": 1.0, "longitude": 2.0}); r.Status != 409 {
		t.Errorf("location before profile: %d", r.Status)
	}
	var lat float64
	if err := e.dbPool().QueryRow(t.Context(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat); err == nil && lat != 0 {
		t.Errorf("unexpected location before it was set")
	}
	e.do("PUT", "/profile/location", u.Token, map[string]any{"latitude": 48.856614, "longitude": 2.352222})
	var lng float64
	if err := e.dbPool().QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil || lat != 48.86 || lng != 2.35 {
		t.Errorf("stored location must be rounded to 2 decimals: %v %v %v", lat, lng, err)
	}
}

func TestPhotoUploadRules(t *testing.T) {
	e := newEnv(t)
	u := e.register("ph")
	e.do("PUT", "/profile/me", u.Token, map[string]any{"firstName": "Pho", "birthDate": "1990-05-05", "gender": "woman"})

	png400 := func() []byte {
		var buf bytes.Buffer
		_ = png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 400, 400)))
		return buf.Bytes()
	}()
	bad := map[string]struct {
		data []byte
		want int
	}{
		"text file":            {[]byte("hello world, definitely not an image"), 415},
		"html disguised":       {[]byte("<html><script>alert(1)</script></html>"), 415},
		"gif header":           {[]byte("GIF89a" + strings.Repeat("\x00", 100)), 415},
		"truncated jpeg":       {testJPEG(400, 400)[:200], 400},
		"too small":            {testJPEG(50, 50), 400},
		"huge payload":         {append([]byte{0xFF, 0xD8, 0xFF}, make([]byte, 9<<20)...), 413},
		"png accepted":         {png400, 201},
		"large jpeg downsized": {testJPEG(3000, 2000), 201},
	}
	for name, c := range bad {
		if r := e.upload(u.Token, c.data, "x.jpg"); r.Status != c.want {
			t.Errorf("%s: want %d got %d (%s)", name, c.want, r.Status, r.Raw)
		}
	}
	if r := e.do("POST", "/photos", u.Token, map[string]any{"x": 1}); r.Status != 400 {
		t.Errorf("non-multipart: %d", r.Status)
	}
	// 2 stored so far; fill up to 6 then expect 409.
	for i := 0; i < 4; i++ {
		if r := e.upload(u.Token, testJPEG(300, 300), "x.jpg"); r.Status != 201 {
			t.Fatalf("upload %d: %d %s", i, r.Status, r.Raw)
		}
	}
	if r := e.upload(u.Token, testJPEG(300, 300), "x.jpg"); r.Status != 409 {
		t.Errorf("7th photo: %d", r.Status)
	}

	// The stored file is a sanitized JPEG, capped at 1280px.
	mine := e.do("GET", "/profile/me", u.Token, nil)
	photos := mine.Body["profile"].(map[string]any)["photos"].([]any)
	if len(photos) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(photos))
	}
	pics := ids(photos, "id")
	var largest image.Config
	for _, p := range photos {
		file := e.do("GET", p.(map[string]any)["url"].(string), u.Token, nil)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(file.Raw))
		if err != nil || format != "jpeg" || cfg.Width > 1280 || cfg.Height > 1280 {
			t.Fatalf("stored photo not sanitized: %v %s %v", err, format, cfg)
		}
		if cfg.Width > largest.Width {
			largest = cfg
		}
	}
	if largest.Width != 1280 {
		t.Errorf("3000px image should be downsized to 1280, got %d", largest.Width)
	}

	// Reorder: must be exactly the user's photos.
	reversed := []string{}
	for i := len(pics) - 1; i >= 0; i-- {
		reversed = append(reversed, pics[i])
	}
	r := e.do("PUT", "/photos/order", u.Token, map[string]any{"photoIds": reversed})
	if r.Status != 200 || ids(r.list("photos"), "id")[0] != reversed[0] {
		t.Fatalf("reorder: %d %s", r.Status, r.Raw)
	}
	if r := e.do("PUT", "/photos/order", u.Token, map[string]any{"photoIds": reversed[:5]}); r.Status != 400 {
		t.Errorf("partial reorder: %d", r.Status)
	}
	dupes := append([]string{reversed[0]}, reversed[:5]...)
	if r := e.do("PUT", "/photos/order", u.Token, map[string]any{"photoIds": dupes}); r.Status != 400 {
		t.Errorf("duplicate reorder: %d", r.Status)
	}
	// Other users cannot delete or reorder with foreign ids.
	other := e.register("intruder")
	if r := e.do("DELETE", "/photos/"+pics[0], other.Token, nil); r.Status != 404 {
		t.Errorf("foreign delete: %d", r.Status)
	}
	// Delete the primary: remaining positions compact to 0..4 and the next becomes primary.
	if r := e.do("DELETE", "/photos/"+reversed[0], u.Token, nil); r.Status != 204 {
		t.Fatalf("delete: %d", r.Status)
	}
	after := e.do("PUT", "/photos/order", u.Token, map[string]any{"photoIds": reversed[1:]})
	for i, p := range after.list("photos") {
		if p.(map[string]any)["position"] != float64(i) {
			t.Errorf("positions not compact: %s", after.Raw)
		}
	}
	if r := e.do("GET", "/photos/"+reversed[0]+"/file", u.Token, nil); r.Status != 404 {
		t.Errorf("deleted photo served: %d", r.Status)
	}
}

func TestWebSocketRealtime(t *testing.T) {
	e := newEnv(t)
	alice := e.onboard("alice", woman("Alice"))
	bob := e.onboard("bob", man("Bob"))
	srv := httptest.NewServer(e.router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	connect := func(token string) *websocket.Conn {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		_ = c.WriteJSON(map[string]string{"type": "auth", "token": token})
		return c
	}
	next := func(c *websocket.Conn) map[string]any {
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev map[string]any
		if err := c.ReadJSON(&ev); err != nil {
			t.Fatalf("read ws: %v", err)
		}
		return ev
	}

	// Bad token is rejected.
	bad := connect("garbage")
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatal("invalid token should close the socket")
	}
	// Skipping auth is rejected too.
	noAuth, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
	_ = noAuth.WriteJSON(map[string]string{"type": "ping"})
	_ = noAuth.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := noAuth.ReadMessage(); err == nil {
		t.Fatal("unauthenticated frames should close the socket")
	}

	wa, wb := connect(alice.Token), connect(bob.Token)
	defer wa.Close()
	defer wb.Close()
	if next(wa)["type"] != "ready" || next(wb)["type"] != "ready" {
		t.Fatal("expected ready frames")
	}
	_ = wa.WriteJSON(map[string]string{"type": "ping"})
	if next(wa)["type"] != "pong" {
		t.Fatal("expected pong")
	}

	e.do("POST", "/swipes", alice.Token, map[string]any{"targetUserId": bob.ID, "action": "like"})
	match := e.do("POST", "/swipes", bob.Token, map[string]any{"targetUserId": alice.ID, "action": "like"})
	conv := match.str("match", "conversationId")

	gotTypes := func(c *websocket.Conn, n int) []string {
		out := []string{}
		for i := 0; i < n; i++ {
			out = append(out, next(c)["type"].(string))
		}
		return out
	}
	// Each side hears about the match and its notification.
	for _, c := range []*websocket.Conn{wa, wb} {
		types := gotTypes(c, 2)
		if !has(types, "match.new") || !has(types, "notification.new") {
			t.Fatalf("match events: %v", types)
		}
	}

	e.do("POST", "/conversations/"+conv+"/messages", alice.Token, map[string]any{"body": "live!"})
	var gotMessage bool
	for i := 0; i < 2; i++ {
		ev := next(wb)
		if ev["type"] == "message.new" {
			msg, _ := json.Marshal(ev["message"])
			gotMessage = contains(string(msg), "live!")
		}
	}
	if !gotMessage {
		t.Fatal("bob did not receive the message in real time")
	}
	e.do("POST", "/conversations/"+conv+"/read", bob.Token, nil)
	for {
		ev := next(wa)
		if ev["type"] == "message.read" && ev["userId"] == bob.ID {
			break
		}
		if ev["type"] == "message.new" || ev["type"] == "notification.new" {
			continue
		}
		t.Fatalf("unexpected event %v", ev)
	}
}

func TestRevokedSessionsLoseAccessImmediately(t *testing.T) {
	e := newEnv(t)
	u := e.register("revoke")
	if r := e.do("GET", "/me", u.Token, nil); r.Status != 200 {
		t.Fatalf("me: %d", r.Status)
	}
	e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": u.Refresh})
	if r := e.do("GET", "/me", u.Token, nil); r.Status != 401 {
		t.Fatalf("access token must die with its session: %d", r.Status)
	}
	// Mutating endpoints are covered too (no 500 from a dangling user id).
	if r := e.do("PUT", "/profile/me", u.Token, map[string]any{"firstName": "X", "birthDate": "1990-01-01", "gender": "man"}); r.Status != 401 {
		t.Fatalf("revoked session wrote data: %d", r.Status)
	}
}

func TestForwardedHeadersCannotBypassRateLimit(t *testing.T) {
	e := newEnv(t, func(d *app.Deps) { d.Limits.AuthPerMinute, d.Limits.AuthBurst = 1, 2 })
	var last int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"a@example.org","password":"whatever-pass"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('1'+i)))
		last = e.serve(req).Status
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("spoofed X-Forwarded-For bypassed the limiter: %d", last)
	}
}
