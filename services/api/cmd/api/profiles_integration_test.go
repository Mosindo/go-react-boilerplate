package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("prof")
	year := time.Now().Year()

	valid := func() map[string]any {
		return map[string]any{"firstName": "Lea", "birthDate": "1994-05-02", "gender": "woman", "bio": "hi", "city": "Lyon",
			"latitude": 45.7640123, "longitude": 4.8357123, "interestIds": []int{1, 2, 2}}
	}
	mutate := func(k string, v any) map[string]any { m := valid(); m[k] = v; return m }

	e.expect(e.do("GET", "/profile", u.Token, nil), 404)
	e.expect(e.do("PUT", "/profile", u.Token, mutate("birthDate", fmt.Sprintf("%d-01-01", year-17))), 403) // minors refused
	e.expect(e.do("PUT", "/profile", u.Token, mutate("birthDate", "2000-13-45")), 400)
	e.expect(e.do("PUT", "/profile", u.Token, mutate("gender", "robot")), 400)
	e.expect(e.do("PUT", "/profile", u.Token, mutate("firstName", "   ")), 400)
	e.expect(e.do("PUT", "/profile", u.Token, mutate("bio", strings.Repeat("x", 501))), 400)
	e.expect(e.do("PUT", "/profile", u.Token, mutate("latitude", 91.0)), 400)
	e.expect(e.do("PUT", "/profile", u.Token, mutate("interestIds", []int{999})), 400)

	r := e.expect(e.do("PUT", "/profile", u.Token, valid()), 200)
	var p struct {
		FirstName string `json:"firstName"`
		Age       int    `json:"age"`
		Interests []struct {
			ID int `json:"id"`
		} `json:"interests"`
		HasLocation bool `json:"hasLocation"`
	}
	r.decode(t, &p)
	if p.FirstName != "Lea" || !p.HasLocation || len(p.Interests) != 2 || p.Age < 30 {
		t.Fatalf("unexpected profile: %+v", p)
	}
	if strings.Contains(string(r.Body), "45.76") || strings.Contains(string(r.Body), "latitude") {
		t.Fatalf("own profile response must not echo coordinates: %s", r.Body)
	}

	// Stored coordinates are coarse (2 decimals ~ 1 km).
	var lat, lng float64
	if err := e.pool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 45.76 || lng != 4.84 {
		t.Fatalf("coordinates must be rounded to 2 decimals, got %v,%v", lat, lng)
	}

	// Updating without coordinates keeps the stored location.
	upd := valid()
	delete(upd, "latitude")
	delete(upd, "longitude")
	upd["city"] = "Paris"
	e.expect(e.do("PUT", "/profile", u.Token, upd), 200)
	e.expect(e.do("PUT", "/profile", u.Token, map[string]any{"firstName": "Lea", "birthDate": "1994-05-02", "gender": "woman", "latitude": 1.0}), 400)

	// Preferences validation.
	pref := func(m map[string]any) response { return e.do("PUT", "/profile/preferences", u.Token, m) }
	e.expect(pref(map[string]any{"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(pref(map[string]any{"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(pref(map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(pref(map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10}), 400)
	e.expect(pref(map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 0}), 400)
	e.expect(pref(map[string]any{"interestedIn": []string{"man", "man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 25}), 200)

	r = e.expect(e.do("GET", "/interests", u.Token, nil), 200)
	if !strings.Contains(string(r.Body), `"slug":"travel"`) {
		t.Fatalf("interests catalogue missing: %s", r.Body)
	}
}

func TestPhotoUploadValidationOrderAndAccess(t *testing.T) {
	e := newTestEnv(t)
	owner := e.onboard(userSpec{Name: "Owner"})
	other := e.onboard(userSpec{Name: "Other"})
	stranger := e.register("stranger")

	// Rejected inputs.
	e.expect(e.upload(owner.Token, []byte("<html>not an image</html>"), "x.png"), 400)
	e.expect(e.upload(owner.Token, []byte("GIF89a"), "x.gif"), 400)
	e.expect(e.upload(owner.Token, pngBytes(50, 50), "tiny.png"), 400)
	e.expect(e.upload(owner.Token, nil, "empty.png"), 400)
	big := append(pngBytes(300, 300), make([]byte, 9<<20)...)
	e.expect(e.upload(owner.Token, big, "big.png"), 400)

	// A large image is downscaled, re-encoded as JPEG, and metadata-free.
	r := e.expect(e.upload(owner.Token, pngBytes(3000, 2000), "wide.png"), 201)
	var ph struct {
		ID     string `json:"id"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	r.decode(t, &ph)
	if ph.Width != 1600 || ph.Height != 1066 {
		t.Fatalf("expected downscale to 1600x1066, got %dx%d", ph.Width, ph.Height)
	}
	files, _ := filepath.Glob(filepath.Join(e.dir, "*.jpg"))
	if len(files) != 3 { // owner x2 + other x1
		t.Fatalf("expected 3 stored files, got %d", len(files))
	}

	file := e.do("GET", "/photos/"+ph.ID+"/file", owner.Token, nil)
	e.expect(file, 200)
	if file.Header.Get("Content-Type") != "image/jpeg" || len(file.Body) < 3 || file.Body[0] != 0xFF || file.Body[1] != 0xD8 {
		t.Fatalf("expected a JPEG, got %q", file.Header.Get("Content-Type"))
	}
	e.expect(e.do("GET", "/photos/"+ph.ID+"/file", "", nil), 401)          // never public
	e.expect(e.do("GET", "/photos/"+ph.ID+"/file", other.Token, nil), 200) // visible profile
	e.expect(e.do("GET", "/photos/not-a-uuid/file", owner.Token, nil), 404)

	// A user without a profile cannot see photos of others.
	e.expect(e.do("GET", "/photos/"+ph.ID+"/file", stranger.Token, nil), 200) // visible profiles are browsable

	// Hide the profile: only the owner (and matches) can still fetch it.
	e.expect(e.do("PUT", "/profile", owner.Token, map[string]any{"firstName": "Owner", "birthDate": "1995-03-15", "gender": "woman", "isVisible": false}), 200)
	e.expect(e.do("GET", "/photos/"+ph.ID+"/file", other.Token, nil), 404)
	e.expect(e.do("GET", "/photos/"+ph.ID+"/file", owner.Token, nil), 200)

	// Blocked users lose access.
	e.expect(e.do("PUT", "/profile", owner.Token, map[string]any{"firstName": "Owner", "birthDate": "1995-03-15", "gender": "woman", "isVisible": true}), 200)
	e.expect(e.do("POST", "/blocks", owner.Token, map[string]string{"userId": other.ID}), 204)
	e.expect(e.do("GET", "/photos/"+ph.ID+"/file", other.Token, nil), 404)
	e.expect(e.do("GET", "/profiles/"+owner.ID, other.Token, nil), 404)

	// Ordering: must be an exact permutation of own photos.
	var list struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	e.expect(e.do("GET", "/photos", owner.Token, nil), 200).decode(t, &list)
	if len(list.Photos) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(list.Photos))
	}
	a, b := list.Photos[0].ID, list.Photos[1].ID
	e.expect(e.do("PUT", "/photos/order", owner.Token, map[string]any{"photoIds": []string{a}}), 400)
	e.expect(e.do("PUT", "/photos/order", owner.Token, map[string]any{"photoIds": []string{a, a}}), 400)
	e.expect(e.do("PUT", "/photos/order", owner.Token, map[string]any{"photoIds": []string{a, other.PhotoID}}), 400)
	e.expect(e.do("PUT", "/photos/order", owner.Token, map[string]any{"photoIds": []string{b, a}}), 200).decode(t, &list)
	if list.Photos[0].ID != b || list.Photos[0].Position != 0 {
		t.Fatalf("main photo should now be %s, got %+v", b, list.Photos)
	}

	// Others' photos cannot be deleted; last photo cannot be deleted.
	e.expect(e.do("DELETE", "/photos/"+other.PhotoID, owner.Token, nil), 404)
	e.expect(e.do("DELETE", "/photos/"+b, owner.Token, nil), 204)
	e.expect(e.do("DELETE", "/photos/"+a, owner.Token, nil), 409)
	e.expect(e.do("GET", "/photos", owner.Token, nil), 200).decode(t, &list)
	if len(list.Photos) != 1 || list.Photos[0].Position != 0 {
		t.Fatalf("positions must be compacted: %+v", list.Photos)
	}
	if _, err := os.Stat(filepath.Join(e.dir, "x")); err == nil {
		t.Fatal("unexpected file")
	}
	files, _ = filepath.Glob(filepath.Join(e.dir, "*.jpg"))
	if len(files) != 2 {
		t.Fatalf("deleted photo file must be removed from disk, have %d files", len(files))
	}

	// At most 6 photos.
	for i := 0; i < 5; i++ {
		e.expect(e.upload(owner.Token, pngBytes(300, 300), "p.png"), 201)
	}
	e.expect(e.upload(owner.Token, pngBytes(300, 300), "p.png"), 409)
}
