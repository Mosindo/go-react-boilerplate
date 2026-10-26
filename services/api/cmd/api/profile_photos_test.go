package main

import (
	"context"
	"image/color"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProfileValidationAndPrivacyRules(t *testing.T) {
	e := newEnv(t)
	u := e.register(uniqueEmail("prof"))

	e.expect(e.do("GET", "/me/profile", u.Token, nil), 404, "no profile yet")
	e.expect(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 30}), 409, "prefs need profile")

	adult := time.Now().AddDate(-30, 0, 0).Format("2006-01-02")
	minor := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")
	justAdult := time.Now().AddDate(-18, 0, -1).Format("2006-01-02")
	base := func(m map[string]any) map[string]any {
		out := map[string]any{"firstName": "Léa-Marie", "birthDate": adult, "gender": "woman", "bio": "Hello", "interests": []string{"travel"}}
		for k, v := range m {
			out[k] = v
		}
		return out
	}

	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"birthDate": minor})), 403, "minor rejected")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"birthDate": "31-12-1990"})), 400, "bad date format")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"birthDate": "1900-01-01"})), 400, "too old")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"firstName": ""})), 400, "empty name")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"firstName": "<script>"})), 400, "name with markup")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"gender": "robot"})), 400, "bad gender")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"bio": strings.Repeat("a", 501)})), 400, "bio too long")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"interests": []string{"does-not-exist"}})), 400, "unknown interest")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"interests": []string{"travel", "wine", "music", "art", "yoga", "cooking", "cinema", "series", "reading", "hiking", "nature"}})), 400, "too many interests")

	created := e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"birthDate": justAdult})), 200, "18 year old accepted")
	if created.str("firstName") != "Léa-Marie" {
		t.Fatalf("unexpected profile %s", created.Raw)
	}
	if mi, _ := created.JSON["missing"].([]any); len(mi) != 3 {
		t.Fatalf("new profile must miss preferences, location and photos: %s", created.Raw)
	}
	// the stored birth date is fixed
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"birthDate": adult})), 400, "birth date immutable")
	e.expect(e.do("PUT", "/me/profile", u.Token, base(map[string]any{"birthDate": justAdult, "bio": "Updated bio", "interests": []string{"wine"}})), 200, "update")

	// preferences validation
	pref := func(m map[string]any) map[string]any {
		out := map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 40, "maxDistanceKm": 25}
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	e.expect(e.do("PUT", "/me/preferences", u.Token, pref(map[string]any{"interestedIn": []string{}})), 400, "no gender")
	e.expect(e.do("PUT", "/me/preferences", u.Token, pref(map[string]any{"minAge": 17})), 400, "min age < 18")
	e.expect(e.do("PUT", "/me/preferences", u.Token, pref(map[string]any{"minAge": 50, "maxAge": 40})), 400, "inverted range")
	e.expect(e.do("PUT", "/me/preferences", u.Token, pref(map[string]any{"maxDistanceKm": 0})), 400, "zero distance")
	e.expect(e.do("PUT", "/me/preferences", u.Token, pref(nil)), 200, "valid preferences")

	// location is rounded server side and validated
	e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 91, "longitude": 0}), 400, "latitude out of range")
	e.expect(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 48.856614, "longitude": 2.352222, "city": "Paris"}), 200, "location")
	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be stored rounded to ~1km, got %v,%v", lat, lng)
	}

	// the owner never sees raw coordinates in the API
	own := e.expect(e.do("GET", "/me/profile", u.Token, nil), 200, "own profile")
	if strings.Contains(string(own.Raw), "latitude") || strings.Contains(string(own.Raw), "48.86") {
		t.Fatalf("coordinates must not be exposed: %s", own.Raw)
	}
	e.expect(e.do("PUT", "/me/privacy", u.Token, map[string]bool{"isVisible": false, "showDistance": false}), 200, "privacy")
}

func TestPhotoLifecycleAndSecurity(t *testing.T) {
	e := newEnv(t)
	u := e.newUser(userSpec{Name: "Anna", Gender: "woman", InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng, AgeYears: 28, NoPhoto: true})
	other := e.newUser(userSpec{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}, Lat: parisLat, Lng: parisLng, AgeYears: 30, NoPhoto: true})

	// dangerous / invalid uploads
	e.expect(e.upload("POST", "/me/photos", u.Token, []byte("<?php echo 1; ?>"), "shell.php.jpg"), 415, "php payload")
	e.expect(e.upload("POST", "/me/photos", u.Token, []byte("<html><script>alert(1)</script>"), "x.jpg"), 415, "html payload")
	e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(50, 50, color.RGBA{1, 2, 3, 255}), "tiny.jpg"), 422, "too small")
	e.expect(e.upload("POST", "/me/photos", u.Token, make([]byte, 9<<20), "big.jpg"), 413, "too large")
	e.expect(e.do("POST", "/me/photos", "", nil), 401, "unauthenticated upload")

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(1600, 1200, color.RGBA{uint8(40 * i), 90, 150, 255}), "p.jpg"), 201, "upload")
		ids = nil
		for _, p := range r.list("photos") {
			ids = append(ids, p["id"].(string))
		}
	}
	if len(ids) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(ids))
	}
	e.expect(e.upload("POST", "/me/photos", u.Token, testJPEG(400, 400, color.RGBA{9, 9, 9, 255}), "7.jpg"), 422, "seventh photo refused")

	// reorder: make the last photo the main one
	order := append([]string{ids[5]}, ids[:5]...)
	r := e.expect(e.do("PUT", "/me/photos/order", u.Token, map[string]any{"ids": order}), 200, "reorder")
	if r.list("photos")[0]["id"] != ids[5] {
		t.Fatalf("main photo not updated: %s", r.Raw)
	}
	e.expect(e.do("PUT", "/me/photos/order", u.Token, map[string]any{"ids": ids[:5]}), 422, "partial order")
	e.expect(e.do("PUT", "/me/photos/order", u.Token, map[string]any{"ids": []string{ids[0], ids[0], ids[1], ids[2], ids[3], ids[4]}}), 422, "duplicate id")

	// other users cannot touch my photos
	e.expect(e.do("DELETE", "/me/photos/"+ids[0], other.Token, nil), 404, "delete foreign photo")
	e.expect(e.upload("PUT", "/me/photos/"+ids[0], other.Token, testJPEG(400, 400, color.RGBA{9, 9, 9, 255}), "r.jpg"), 404, "replace foreign photo")

	// output is a re-encoded, resized JPEG served through a signed URL
	url := r.list("photos")[0]["url"].(string)
	img, err := http.Get(e.srv.URL + url)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(img.Body)
	img.Body.Close()
	if img.StatusCode != 200 || img.Header.Get("Content-Type") != "image/jpeg" || len(body) < 100 {
		t.Fatalf("photo fetch failed: %d %s", img.StatusCode, img.Header.Get("Content-Type"))
	}
	if img.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("nosniff header missing")
	}
	var w, h int
	_ = e.pool.QueryRow(context.Background(), `SELECT width, height FROM photos WHERE id = $1`, ids[5]).Scan(&w, &h)
	if w != 1080 || h != 810 {
		t.Fatalf("photo should be downscaled to 1080x810, got %dx%d", w, h)
	}
	tampered := strings.Replace(url, "sig=", "sig=00", 1)
	if res, _ := http.Get(e.srv.URL + tampered); res.StatusCode != 404 {
		t.Fatalf("tampered signature must be refused, got %d", res.StatusCode)
	}
	if res, _ := http.Get(e.srv.URL + "/photos/" + ids[5] + "/file"); res.StatusCode != 404 {
		t.Fatalf("unsigned access must be refused, got %d", res.StatusCode)
	}

	// replace keeps the slot, delete compacts positions
	rep := e.expect(e.upload("PUT", "/me/photos/"+ids[5], u.Token, testJPEG(500, 500, color.RGBA{250, 250, 0, 255}), "r.jpg"), 200, "replace")
	if rep.list("photos")[0]["id"] == ids[5] {
		t.Fatal("replacement must get a new id/url")
	}
	del := e.expect(e.do("DELETE", "/me/photos/"+ids[2], u.Token, nil), 200, "delete")
	if len(del.list("photos")) != 5 {
		t.Fatalf("expected 5 photos after delete: %s", del.Raw)
	}
	if n := e.count(`SELECT count(*) FROM photos WHERE user_id = $1 AND position >= 5`, u.ID); n != 0 {
		t.Fatal("positions must be compacted")
	}
	e.expect(e.do("DELETE", "/me/photos/not-a-uuid", u.Token, nil), 404, "invalid id")
}
