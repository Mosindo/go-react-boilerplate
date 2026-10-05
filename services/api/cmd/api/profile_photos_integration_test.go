package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type myProfile struct {
	Profile *struct {
		FirstName      string `json:"firstName"`
		Age            int    `json:"age"`
		City           string `json:"city"`
		IsDiscoverable bool   `json:"isDiscoverable"`
		Interests      []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	} `json:"profile"`
	Completion struct {
		Profile, Preferences, Location, Photos, Complete bool
	} `json:"completion"`
}

func TestProfileOnboardingCompletion(t *testing.T) {
	env := newEnv(t)
	s := env.register("onboard")

	var mine myProfile
	env.do(http.MethodGet, "/profile/me", s.Token, nil).expect(t, http.StatusOK).json(t, &mine)
	if mine.Profile != nil || mine.Completion.Complete {
		t.Fatalf("fresh account must have no profile: %+v", mine)
	}

	// Discovery is locked until onboarding is complete.
	env.do(http.MethodGet, "/discover", s.Token, nil).expect(t, http.StatusConflict)

	env.do(http.MethodPut, "/profile/me", s.Token, map[string]any{
		"firstName": "Léa-Marie", "birthDate": "1995-03-02", "gender": "woman", "bio": "Hello", "interests": []string{"travel", "yoga"},
	}).expect(t, http.StatusOK).json(t, &mine)
	if !mine.Completion.Profile || mine.Completion.Complete || mine.Profile.FirstName != "Léa-Marie" || len(mine.Profile.Interests) != 2 {
		t.Fatalf("unexpected profile state: %+v", mine)
	}
	env.do(http.MethodPut, "/profile/location", s.Token, map[string]any{"latitude": parisLat, "longitude": parisLng, "city": "Paris"}).expect(t, http.StatusOK)
	env.do(http.MethodPut, "/profile/preferences", s.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 30}).expect(t, http.StatusOK)
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "a.jpg", solidJPEG(300, 300)).expect(t, http.StatusCreated)

	env.do(http.MethodGet, "/profile/me", s.Token, nil).json(t, &mine)
	if !mine.Completion.Complete {
		t.Fatalf("expected a complete profile: %+v", mine.Completion)
	}
	env.do(http.MethodGet, "/discover", s.Token, nil).expect(t, http.StatusOK)
}

func TestProfileValidation(t *testing.T) {
	env := newEnv(t)
	s := env.register("validate")
	put := func(body map[string]any) resp { return env.do(http.MethodPut, "/profile/me", s.Token, body) }
	base := func() map[string]any {
		return map[string]any{"firstName": "Sam", "birthDate": "1990-01-01", "gender": "man", "bio": "", "interests": []string{}}
	}

	cases := map[string]struct {
		mutate func(map[string]any)
		status int
	}{
		"underage":          {func(m map[string]any) { m["birthDate"] = "2015-01-01" }, http.StatusForbidden},
		"turns 18 tomorrow": {func(m map[string]any) { m["birthDate"] = futureEighteenthBirthday() }, http.StatusForbidden},
		"bad date":          {func(m map[string]any) { m["birthDate"] = "yesterday" }, http.StatusBadRequest},
		"impossible age":    {func(m map[string]any) { m["birthDate"] = "1800-01-01" }, http.StatusBadRequest},
		"unknown gender":    {func(m map[string]any) { m["gender"] = "robot" }, http.StatusBadRequest},
		"digits in name":    {func(m map[string]any) { m["firstName"] = "R2D2" }, http.StatusBadRequest},
		"html in name":      {func(m map[string]any) { m["firstName"] = "<b>x</b>" }, http.StatusBadRequest},
		"bio too long":      {func(m map[string]any) { m["bio"] = strings.Repeat("a", 501) }, http.StatusBadRequest},
		"unknown interest":  {func(m map[string]any) { m["interests"] = []string{"does-not-exist"} }, http.StatusBadRequest},
		"too many interests": {func(m map[string]any) {
			m["interests"] = []string{"travel", "cooking", "hiking", "running", "fitness", "yoga", "cycling", "swimming", "football", "tennis", "climbing"}
		}, http.StatusBadRequest},
	}
	for name, tc := range cases {
		body := base()
		tc.mutate(body)
		if r := put(body); r.Status != tc.status {
			t.Errorf("%s: expected %d, got %d (%s)", name, tc.status, r.Status, r.Body)
		}
	}
	put(base()).expect(t, http.StatusOK)

	// The birth date is the age gate: it cannot be changed once set.
	changed := base()
	changed["birthDate"] = "1980-01-01"
	put(changed).expect(t, http.StatusConflict)

	prefs := func(m map[string]any) resp { return env.do(http.MethodPut, "/profile/preferences", s.Token, m) }
	prefs(map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 30, "maxDistanceKm": 10}).expect(t, http.StatusBadRequest)
	prefs(map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10}).expect(t, http.StatusBadRequest)
	prefs(map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 100000}).expect(t, http.StatusBadRequest)
	prefs(map[string]any{"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}).expect(t, http.StatusBadRequest)

	loc := func(lat, lng float64) resp {
		return env.do(http.MethodPut, "/profile/location", s.Token, map[string]any{"latitude": lat, "longitude": lng})
	}
	loc(91, 0).expect(t, http.StatusBadRequest)
	loc(0, 181).expect(t, http.StatusBadRequest)
}

func TestLocationIsCoarseAndNeverExposed(t *testing.T) {
	env := newEnv(t)
	a, b := env.register("loc-a"), env.register("loc-b")
	env.onboard(a, profileSpec{Name: "Ana", Gender: "woman", Interested: []string{"man"}, Lat: 48.856614, Lng: 2.352222})
	env.onboard(b, profileSpec{Name: "Ben", Gender: "man", Interested: []string{"woman"}, Lat: 48.90, Lng: 2.40})

	var lat, lng float64
	if err := env.pool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be rounded to 2 decimals, stored %v,%v", lat, lng)
	}

	r := env.do(http.MethodGet, "/profiles/"+b.ID, a.Token, nil).expect(t, http.StatusOK)
	body := string(r.Body)
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "48.9", "email"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("public profile leaks %q: %s", forbidden, body)
		}
	}
	var c card
	r.json(t, &c)
	if c.DistanceKm == nil || *c.DistanceKm%5 != 0 {
		t.Fatalf("distance must be rounded to a 5 km step, got %v", c.DistanceKm)
	}

	// Hiding the distance also hides the city.
	env.do(http.MethodPut, "/profile/privacy", b.Token, map[string]any{"isDiscoverable": true, "showDistance": false}).expect(t, http.StatusOK)
	c = card{}
	env.do(http.MethodGet, "/profiles/"+b.ID, a.Token, nil).json(t, &c)
	if c.DistanceKm != nil || c.City != "" {
		t.Fatalf("distance and city must be hidden: %+v", c)
	}
}

func futureEighteenthBirthday() string {
	// Someone born exactly 18 years minus one day ago is still 17.
	return timeNow().AddDate(-18, 0, 1).Format("2006-01-02")
}

func TestPhotoLifecycle(t *testing.T) {
	env := newEnv(t)
	s := env.register("photo")
	other := env.register("photo-other")

	type ph struct {
		ID       string `json:"id"`
		URL      string `json:"url"`
		Position int    `json:"position"`
		Width    int    `json:"width"`
	}
	var ids []string
	for i := 0; i < 3; i++ {
		var p ph
		env.upload2(http.MethodPost, "/photos", s.Token, "photo", "p.jpg", solidJPEG(800, 600)).expect(t, http.StatusCreated).json(t, &p)
		if p.Position != i {
			t.Fatalf("photo %d got position %d", i, p.Position)
		}
		ids = append(ids, p.ID)
	}

	// PNG with transparency is accepted and normalised to JPEG.
	var png ph
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "t.png", solidPNG(300, 300)).expect(t, http.StatusCreated).json(t, &png)
	ids = append(ids, png.ID)

	// Large images are downscaled.
	var big ph
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "big.jpg", solidJPEG(3000, 2000)).expect(t, http.StatusCreated).json(t, &big)
	if big.Width != 1440 {
		t.Errorf("expected long edge capped at 1440, got width %d", big.Width)
	}
	ids = append(ids, big.ID)

	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "six.jpg", solidJPEG(300, 300)).expect(t, http.StatusCreated)
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "seven.jpg", solidJPEG(300, 300)).expect(t, http.StatusConflict)

	// Reorder: must list every photo exactly once.
	var list struct{ Photos []ph }
	env.do(http.MethodGet, "/photos", s.Token, nil).json(t, &list)
	all := make([]string, 0, len(list.Photos))
	for _, p := range list.Photos {
		all = append(all, p.ID)
	}
	env.do(http.MethodPut, "/photos/order", s.Token, map[string]any{"photoIds": all[:3]}).expect(t, http.StatusBadRequest)
	reversed := make([]string, len(all))
	for i, id := range all {
		reversed[len(all)-1-i] = id
	}
	env.do(http.MethodPut, "/photos/order", s.Token, map[string]any{"photoIds": reversed}).expect(t, http.StatusOK).json(t, &list)
	if list.Photos[0].ID != all[len(all)-1] {
		t.Fatalf("first photo (primary) must follow the new order")
	}

	// Someone else cannot touch my photos.
	env.do(http.MethodDelete, "/photos/"+all[0], other.Token, nil).expect(t, http.StatusNotFound)
	env.upload2(http.MethodPut, "/photos/"+all[0], other.Token, "photo", "x.jpg", solidJPEG(300, 300)).expect(t, http.StatusNotFound)
	env.do(http.MethodPut, "/photos/order", other.Token, map[string]any{"photoIds": reversed}).expect(t, http.StatusBadRequest)

	// Replace keeps the position and swaps the file.
	before, _ := os.ReadDir(env.upload)
	var replaced ph
	env.upload2(http.MethodPut, "/photos/"+reversed[1], s.Token, "photo", "r.jpg", solidJPEG(500, 500)).expect(t, http.StatusOK).json(t, &replaced)
	after, _ := os.ReadDir(env.upload)
	if replaced.Position != 1 || len(before) != len(after) {
		t.Fatalf("replace must keep position 1 and not leak files: pos=%d files %d→%d", replaced.Position, len(before), len(after))
	}

	// Delete compacts positions.
	env.do(http.MethodDelete, "/photos/"+reversed[0], s.Token, nil).expect(t, http.StatusNoContent)
	env.do(http.MethodGet, "/photos", s.Token, nil).json(t, &list)
	for i, p := range list.Photos {
		if p.Position != i {
			t.Fatalf("positions must be compacted after delete: %+v", list.Photos)
		}
	}
	if len(list.Photos) != 5 {
		t.Fatalf("expected 5 photos left, got %d", len(list.Photos))
	}
	if files, _ := os.ReadDir(env.upload); len(files) != 5 {
		t.Fatalf("deleted photo must leave storage, %d files", len(files))
	}

	// Signed image URL: works without a bearer token, is a clean JPEG, and cannot be forged or reused for another id.
	url := list.Photos[0].URL
	img := env.do(http.MethodGet, url, "", nil).expect(t, http.StatusOK)
	if ct := img.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("content type %q", ct)
	}
	if _, err := jpeg.Decode(bytes.NewReader(img.Body)); err != nil {
		t.Errorf("served bytes are not a JPEG: %v", err)
	}
	env.do(http.MethodGet, strings.Replace(url, "sig=", "sig=00", 1), "", nil).expect(t, http.StatusNotFound)
	env.do(http.MethodGet, "/photos/"+list.Photos[1].ID+"/image"+url[strings.Index(url, "?"):], "", nil).expect(t, http.StatusNotFound)
	env.do(http.MethodGet, "/photos/"+list.Photos[0].ID+"/image", "", nil).expect(t, http.StatusNotFound)
}

func TestPhotoUploadRejectsDangerousFiles(t *testing.T) {
	env := newEnv(t)
	s := env.register("evil")

	script := []byte("<?php system($_GET['c']); ?>")
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "shell.jpg", script).expect(t, http.StatusUnsupportedMediaType)
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "page.html", []byte("<html><script>alert(1)</script></html>")).expect(t, http.StatusUnsupportedMediaType)
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "anim.gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")).expect(t, http.StatusUnsupportedMediaType)

	// A JPEG header followed by garbage.
	broken := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte("A"), 2000)...)
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "broken.jpg", broken).expect(t, http.StatusUnprocessableEntity)

	// A valid JPEG with a script appended must come out clean (re-encoded).
	polyglot := append(solidJPEG(300, 300), []byte("<script>alert(1)</script>")...)
	var p struct{ ID string }
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "poly.jpg", polyglot).expect(t, http.StatusCreated).json(t, &p)
	files, _ := os.ReadDir(env.upload)
	data, _ := os.ReadFile(filepath.Join(env.upload, files[0].Name()))
	if bytes.Contains(data, []byte("<script>")) {
		t.Error("appended payload survived re-encoding")
	}

	// Too small, too large, missing part, no auth.
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "tiny.jpg", solidJPEG(50, 50)).expect(t, http.StatusUnprocessableEntity)
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "huge.jpg", append(solidJPEG(300, 300), bytes.Repeat([]byte{0}, 9<<20)...)).expect(t, http.StatusRequestEntityTooLarge)
	env.upload2(http.MethodPost, "/photos", s.Token, "wrongfield", "a.jpg", solidJPEG(300, 300)).expect(t, http.StatusBadRequest)
	env.upload2(http.MethodPost, "/photos", "", "photo", "a.jpg", solidJPEG(300, 300)).expect(t, http.StatusUnauthorized)

	// Decompression bomb: tiny file declaring enormous dimensions.
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "bomb.jpg", bombJPEG()).expect(t, http.StatusUnprocessableEntity)
}

// bombJPEG returns a small JPEG whose SOF header declares 60000x60000 pixels.
func bombJPEG() []byte {
	data := solidJPEG(300, 300)
	for i := 0; i+9 < len(data); i++ {
		if data[i] == 0xFF && (data[i+1] == 0xC0 || data[i+1] == 0xC2) {
			binary.BigEndian.PutUint16(data[i+5:], 60000)
			binary.BigEndian.PutUint16(data[i+7:], 60000)
			break
		}
	}
	return data
}

func TestStoredPhotosHaveNoMetadata(t *testing.T) {
	env := newEnv(t)
	s := env.register("exif")
	withExif := injectExif(solidJPEG(400, 300), []byte("GPS-COORDINATES-SECRET"))
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "geo.jpg", withExif).expect(t, http.StatusCreated)

	files, _ := os.ReadDir(env.upload)
	data, _ := os.ReadFile(filepath.Join(env.upload, files[0].Name()))
	if bytes.Contains(data, []byte("GPS-COORDINATES-SECRET")) || bytes.Contains(data, []byte("Exif")) {
		t.Fatal("EXIF metadata must be stripped from stored photos")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width != 400 {
		t.Fatalf("unexpected stored image: %v %+v", err, cfg)
	}
}

// injectExif inserts an APP1 Exif segment (no orientation tag) carrying marker bytes after the SOI marker.
func injectExif(jpg, marker []byte) []byte {
	seg := append([]byte("Exif\x00\x00"), marker...)
	size := len(seg) + 2
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(size >> 8), byte(size)}
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func TestImageEndpointHeaders(t *testing.T) {
	env := newEnv(t)
	s := env.register("headers")
	var p struct{ URL string }
	env.upload2(http.MethodPost, "/photos", s.Token, "photo", "a.jpg", solidJPEG(300, 300)).expect(t, http.StatusCreated).json(t, &p)

	req := httptest.NewRequest(http.MethodGet, p.URL, nil)
	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(rec.Header().Get("Cache-Control"), "private") {
		t.Errorf("image response must be nosniff and private-cached: %v", rec.Header())
	}
}
