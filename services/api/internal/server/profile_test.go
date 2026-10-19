package server_test

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("Eve")

	if r := u.do("GET", "/profile", nil); r.Status != 404 || r.code() != "profile_not_found" {
		t.Fatalf("no profile yet: %d %s", r.Status, r.Body)
	}
	base := func() map[string]any {
		return map[string]any{"firstName": "Eve", "birthDate": "1995-01-02", "gender": "woman", "bio": "hi", "city": "Paris"}
	}
	bad := map[string]func(m map[string]any){
		"underage":      func(m map[string]any) { m["birthDate"] = "2099-01-01" },
		"bad date":      func(m map[string]any) { m["birthDate"] = "02/01/1995" },
		"ancient":       func(m map[string]any) { m["birthDate"] = "1800-01-01" },
		"bad gender":    func(m map[string]any) { m["gender"] = "robot" },
		"empty name":    func(m map[string]any) { m["firstName"] = "   " },
		"long name":     func(m map[string]any) { m["firstName"] = strings.Repeat("a", 41) },
		"long bio":      func(m map[string]any) { m["bio"] = strings.Repeat("a", 501) },
		"control chars": func(m map[string]any) { m["firstName"] = "Ev\x00e" },
		"unknown int":   func(m map[string]any) { m["interests"] = []int{9999} },
		"bad lat":       func(m map[string]any) { m["location"] = map[string]float64{"lat": 123, "lng": 0} },
		"too many int":  func(m map[string]any) { m["interests"] = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11} },
	}
	for name, mutate := range bad {
		m := base()
		mutate(m)
		if r := u.do("PUT", "/profile", m); r.Status < 400 || r.Status >= 500 {
			t.Errorf("%s: expected 4xx, got %d %s", name, r.Status, r.Body)
		}
	}
	m := base()
	m["birthDate"] = "2099-01-01"
	if r := u.do("PUT", "/profile", m); r.code() != "underage" {
		t.Errorf("minor must be refused with code underage, got %s", r.Body)
	}

	m = base()
	m["firstName"] = "  Eve  "
	m["interests"] = []int{1, 2, 2, 3}
	m["location"] = map[string]float64{"lat": 45.76412345, "lng": 4.83567891}
	u.expect(200, "PUT", "/profile", m)

	r := u.expect(200, "GET", "/profile", nil)
	var own struct {
		FirstName   string `json:"firstName"`
		HasLocation bool   `json:"hasLocation"`
		Interests   []struct {
			ID int `json:"id"`
		} `json:"interests"`
	}
	r.decode(t, &own)
	if own.FirstName != "Eve" || len(own.Interests) != 3 || !own.HasLocation {
		t.Fatalf("unexpected profile: %s", r.Body)
	}
	if strings.Contains(string(r.Body), "45.76") || strings.Contains(string(r.Body), "4.83") {
		t.Fatalf("coordinates must never be returned: %s", r.Body)
	}

	var lat, lng float64
	if err := e.pool.QueryRow(t.Context(), `SELECT lat, lng FROM profiles WHERE user_id=$1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 45.76 || lng != 4.84 {
		t.Fatalf("stored location must be snapped to ~1km, got %v,%v", lat, lng)
	}

	// Preferences validation.
	for name, p := range map[string]map[string]any{
		"empty":       {"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "showDistance": true},
		"bad gender":  {"interestedIn": []string{"x"}, "minAge": 18, "maxAge": 40},
		"minor range": {"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 40},
		"inverted":    {"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30},
		"distance":    {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 30, "maxDistanceKm": 0},
	} {
		if r := u.do("PUT", "/preferences", p); r.Status != 400 {
			t.Errorf("preferences %s: expected 400, got %d", name, r.Status)
		}
	}
	u.expect(200, "PUT", "/preferences", map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 30, "showDistance": false})

	if r := u.expect(200, "GET", "/interests", nil); !strings.Contains(string(r.Body), "Voyages") {
		t.Fatalf("interests catalogue missing: %s", r.Body)
	}
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	u := e.register("Fay")
	u.expect(200, "PUT", "/profile", map[string]any{"firstName": "Fay", "birthDate": "1995-01-02", "gender": "woman"})

	var txt bytes.Buffer
	txt.WriteString("<?php system($_GET['c']); ?>")
	u.upload(txt.Bytes(), nil, 415)
	u.upload(append([]byte("GIF89a"), make([]byte, 600)...), nil, 415)
	u.upload(testJPEG(50, 50), nil, 415)    // below minimum size
	u.upload(make([]byte, 9<<20), nil, 413) // over 8 MB
	// A script smuggled behind a valid JPEG header must not survive re-encoding.
	polyglot := append(testJPEG(300, 300), []byte("<script>alert(1)</script>")...)
	r := u.upload(polyglot, nil, 201)

	var list struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	r.decode(t, &list)
	first := list.Photos[0].ID

	served := u.expect(200, "GET", "/photos/"+first, nil)
	if served.Header.Get("Content-Type") != "image/jpeg" || bytes.Contains(served.Body, []byte("<script>")) {
		t.Fatalf("served photo must be sanitised JPEG, got %s", served.Header.Get("Content-Type"))
	}
	if _, err := os.Stat(e.uploads); err != nil {
		t.Fatal(err)
	}

	// Large photos are downscaled and EXIF-free by re-encoding.
	big := u.upload(testJPEG(3000, 2000), nil, 201)
	big.decode(t, &list)
	var w, h int
	if err := e.pool.QueryRow(t.Context(), `SELECT width, height FROM photos WHERE id=$1`, list.Photos[1].ID).Scan(&w, &h); err != nil || w != 1280 || h != 853 {
		t.Fatalf("expected 1280x853, got %dx%d (%v)", w, h, err)
	}

	// PNG with alpha accepted.
	var pngBuf bytes.Buffer
	_ = png.Encode(&pngBuf, image.NewNRGBA(image.Rect(0, 0, 300, 300)))
	u.upload(pngBuf.Bytes(), nil, 201)

	// Photo limit is 6.
	for i := 0; i < 3; i++ {
		u.upload(testJPEG(300, 300), nil, 201)
	}
	u.upload(testJPEG(300, 300), nil, 409)

	// Reorder: reverse the list; the last photo becomes primary.
	r = u.expect(200, "GET", "/profile", nil)
	var own struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	r.decode(t, &own)
	ids := make([]string, 0, len(own.Photos))
	for i := len(own.Photos) - 1; i >= 0; i-- {
		ids = append(ids, own.Photos[i].ID)
	}
	u.expect(200, "PUT", "/profile/photos/order", map[string]any{"photoIds": ids})
	r = u.expect(200, "GET", "/profile", nil)
	r.decode(t, &own)
	if own.Photos[0].ID != ids[0] {
		t.Fatalf("reorder did not apply: %s", r.Body)
	}
	u.expect(400, "PUT", "/profile/photos/order", map[string]any{"photoIds": ids[:2]})
	u.expect(400, "PUT", "/profile/photos/order", map[string]any{"photoIds": []string{"nope"}})

	// Replace position 0: old file disappears, new id appears.
	oldKey := photoKey(t, e, ids[0])
	pos := 0
	u.upload(testJPEG(320, 320), &pos, 201)
	if _, err := os.Stat(filepath.Join(e.uploads, oldKey[:2], oldKey)); !os.IsNotExist(err) {
		t.Fatalf("replaced photo file must be deleted from disk (err=%v)", err)
	}
	u.expect(404, "GET", "/photos/"+ids[0], nil)

	// Delete compacts positions and removes the file.
	r = u.expect(200, "GET", "/profile", nil)
	r.decode(t, &own)
	victim := own.Photos[1].ID
	key := photoKey(t, e, victim)
	u.expect(200, "DELETE", "/profile/photos/"+victim, nil)
	if _, err := os.Stat(filepath.Join(e.uploads, key[:2], key)); !os.IsNotExist(err) {
		t.Fatal("deleted photo file must be removed")
	}
	var maxPos, n int
	_ = e.pool.QueryRow(t.Context(), `SELECT max(position), count(*) FROM photos WHERE user_id=$1`, u.ID).Scan(&maxPos, &n)
	if maxPos != n-1 {
		t.Fatalf("positions must stay contiguous, max=%d count=%d", maxPos, n)
	}

	// Another user cannot delete or reorder someone else's photo.
	other := e.register("Gus")
	other.expect(404, "DELETE", "/profile/photos/"+own.Photos[0].ID, nil)
	other.expect(200, "GET", "/photos/"+own.Photos[0].ID, nil) // visible: not blocked
	if r := e.do("GET", "/photos/"+own.Photos[0].ID, "", nil); r.Status != 401 {
		t.Fatalf("photos must require authentication, got %d", r.Status)
	}
}

func photoKey(t *testing.T, e *env, id string) string {
	t.Helper()
	var key string
	if err := e.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE id=$1`, id).Scan(&key); err != nil {
		t.Fatal(err)
	}
	return key
}
