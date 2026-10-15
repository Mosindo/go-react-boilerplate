package main

import (
	"bytes"
	"context"
	"image"
	_ "image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnboardingProgressAndProfileValidation(t *testing.T) {
	e := newEnv(t)
	p := e.register("onboard")

	state := e.mustStatus(e.do(http.MethodGet, "/me/profile", p.Token, nil), http.StatusOK, "empty profile").json(t)
	if state["profile"] != nil || state["onboarding"].(map[string]any)["complete"] != false {
		t.Fatalf("a new account has no profile yet: %v", state)
	}

	valid := map[string]any{"firstName": "  Alex  ", "birthDate": birthDateFor(30), "gender": "nonbinary", "bio": "Hi", "interests": []string{"cinema", "voyages", "cinema"}}
	invalid := func(name string, change map[string]any, wantCode string) {
		t.Helper()
		body := map[string]any{}
		for k, v := range valid {
			body[k] = v
		}
		for k, v := range change {
			body[k] = v
		}
		r := e.do(http.MethodPut, "/me/profile", p.Token, body)
		if r.Status != http.StatusBadRequest || r.code(t) != wantCode {
			t.Errorf("%s: expected 400 %s, got %d %s", name, wantCode, r.Status, r.Body)
		}
	}
	invalid("underage", map[string]any{"birthDate": birthDateFor(17)}, "underage")
	invalid("far future", map[string]any{"birthDate": "2999-01-01"}, "invalid_birth_date")
	invalid("too old", map[string]any{"birthDate": birthDateFor(130)}, "invalid_birth_date")
	invalid("malformed date", map[string]any{"birthDate": "12/05/1990"}, "invalid_birth_date")
	invalid("gender", map[string]any{"gender": "robot"}, "invalid_gender")
	invalid("blank name", map[string]any{"firstName": "   "}, "invalid_first_name")
	invalid("long name", map[string]any{"firstName": strings.Repeat("a", 41)}, "invalid_first_name")
	invalid("long bio", map[string]any{"bio": strings.Repeat("b", 501)}, "invalid_bio")
	invalid("unknown interest", map[string]any{"interests": []string{"does-not-exist"}}, "invalid_interests")
	invalid("too many interests", map[string]any{"interests": []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}}, "invalid_interests")

	saved := e.mustStatus(e.do(http.MethodPut, "/me/profile", p.Token, valid), http.StatusOK, "save profile").json(t)
	profile := saved["profile"].(map[string]any)
	if profile["firstName"] != "Alex" || int(profile["age"].(float64)) != 30 {
		t.Fatalf("profile was not normalized: %v", profile)
	}
	if got := len(profile["interests"].([]any)); got != 2 {
		t.Fatalf("duplicate interests must be merged, got %d", got)
	}

	// The birth date is fixed once set.
	r := e.do(http.MethodPut, "/me/profile", p.Token, map[string]any{"firstName": "Alex", "birthDate": birthDateFor(35), "gender": "nonbinary", "bio": "", "interests": []string{}})
	if r.Status != http.StatusConflict || r.code(t) != "birth_date_locked" {
		t.Fatalf("birth date must be locked, got %d %s", r.Status, r.Body)
	}

	prefsBad := func(body map[string]any) {
		t.Helper()
		if r := e.do(http.MethodPut, "/me/preferences", p.Token, body); r.Status != http.StatusBadRequest {
			t.Errorf("preferences %v should be rejected, got %d %s", body, r.Status, r.Body)
		}
	}
	prefsBad(map[string]any{"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 20})
	prefsBad(map[string]any{"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 20})
	prefsBad(map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 30, "maxDistanceKm": 20})
	prefsBad(map[string]any{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 20})
	prefsBad(map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 0})
	prefsBad(map[string]any{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 501})
	e.mustStatus(e.do(http.MethodPut, "/me/preferences", p.Token, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 20, "maxAge": 40, "maxDistanceKm": 30}), http.StatusOK, "preferences")

	// Discovery stays closed until a photo exists.
	blocked := e.do(http.MethodGet, "/discover", p.Token, nil)
	if blocked.Status != http.StatusConflict || blocked.code(t) != "profile_incomplete" {
		t.Fatalf("discover must require a complete profile, got %d %s", blocked.Status, blocked.Body)
	}
	e.mustStatus(e.upload(http.MethodPost, "/photos", p.Token, "a.png", testPNG(400, 400)), http.StatusCreated, "photo")
	done := e.mustStatus(e.do(http.MethodGet, "/me/profile", p.Token, nil), http.StatusOK, "profile").json(t)
	if done["onboarding"].(map[string]any)["complete"] != true {
		t.Fatalf("onboarding should be complete: %v", done["onboarding"])
	}
	e.mustStatus(e.do(http.MethodGet, "/discover", p.Token, nil), http.StatusOK, "discover once complete")
}

func TestLocationIsStoredCoarselyAndNeverExposed(t *testing.T) {
	e := newEnv(t)
	viewer := e.signup(profileSpec{Name: "Vera", Gender: "woman", Seeks: []string{"man"}, Lat: 45.0, Lng: 5.0})
	target := e.signup(profileSpec{Name: "Tom", Gender: "man", Seeks: []string{"woman"}, Lat: 45.01234567, Lng: 5.07654321})

	var lat, lng float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, target.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 45.01 || lng != 5.08 {
		t.Fatalf("coordinates must be rounded to 2 decimals, stored %v,%v", lat, lng)
	}

	for _, path := range []string{"/discover", "/profiles/" + target.ID} {
		r := e.mustStatus(e.do(http.MethodGet, path, viewer.Token, nil), http.StatusOK, path)
		raw := string(r.Body)
		for _, secret := range []string{"45.01", "5.07", "5.08", "latitude", "longitude", "birthDate"} {
			if strings.Contains(raw, secret) {
				t.Fatalf("%s leaks %q: %s", path, secret, raw)
			}
		}
	}

	card := e.mustStatus(e.do(http.MethodGet, "/profiles/"+target.ID, viewer.Token, nil), http.StatusOK, "profile").json(t)
	km := int(card["distanceKm"].(float64))
	if km%5 != 0 || km < 5 {
		t.Fatalf("distance must be a multiple of 5 km, got %d", km)
	}

	// show_distance=false hides it entirely.
	e.mustStatus(e.do(http.MethodPatch, "/me/privacy", target.Token, map[string]any{"showDistance": false}), http.StatusOK, "privacy")
	card = e.mustStatus(e.do(http.MethodGet, "/profiles/"+target.ID, viewer.Token, nil), http.StatusOK, "profile").json(t)
	if _, shown := card["distanceKm"]; shown {
		t.Fatal("distance must be hidden when the user turned it off")
	}

	for _, bad := range []map[string]any{{"latitude": 91.0, "longitude": 0.0}, {"latitude": 0.0, "longitude": 181.0}} {
		e.mustStatus(e.do(http.MethodPut, "/me/location", viewer.Token, bad), http.StatusBadRequest, "bad location")
	}
	e.mustStatus(e.do(http.MethodDelete, "/me/location", viewer.Token, nil), http.StatusOK, "clear location")
	var stored *float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, viewer.ID).Scan(&stored); err != nil || stored != nil {
		t.Fatalf("location should be erased: %v %v", stored, err)
	}
}

func TestPhotoLifecycleAndValidation(t *testing.T) {
	e := newEnv(t)
	p := e.register("photos")

	reject := func(name string, data []byte, status int, code string) {
		t.Helper()
		r := e.upload(http.MethodPost, "/photos", p.Token, name, data)
		if r.Status != status || r.code(t) != code {
			t.Errorf("%s: expected %d %s, got %d %s", name, status, code, r.Status, r.Body)
		}
	}
	reject("notes.txt", []byte("this is not an image"), http.StatusUnsupportedMediaType, "unsupported_image")
	reject("script.png", []byte("<?php system($_GET['c']); ?>"), http.StatusUnsupportedMediaType, "unsupported_image")
	reject("tiny.png", testPNG(50, 50), http.StatusBadRequest, "image_too_small")
	reject("truncated.png", testPNG(600, 600)[:200], http.StatusUnsupportedMediaType, "unsupported_image")
	if r := e.do(http.MethodPost, "/photos", p.Token, nil); r.Status != http.StatusBadRequest {
		t.Errorf("missing file should be 400, got %d", r.Status)
	}

	// A polyglot: a valid image followed by a script. Re-encoding must drop the tail.
	polyglot := append(testJPEG(500, 500), []byte("<script>alert(1)</script>")...)
	first := e.mustStatus(e.upload(http.MethodPost, "/photos", p.Token, "poly.jpg", polyglot), http.StatusCreated, "polyglot upload").json(t)
	photos := first["photos"].([]any)
	photo := photos[0].(map[string]any)

	file := e.do(http.MethodGet, photo["url"].(string), "", nil)
	e.mustStatus(file, http.StatusOK, "serve photo")
	if file.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("unexpected content type %q", file.Header.Get("Content-Type"))
	}
	if bytes.Contains(file.Body, []byte("<script>")) {
		t.Fatal("trailing payload survived re-encoding")
	}
	if _, format, err := image.Decode(bytes.NewReader(file.Body)); err != nil || format != "jpeg" {
		t.Fatalf("served file is not a valid jpeg: %v %s", err, format)
	}
	e.mustStatus(e.do(http.MethodGet, photo["thumbUrl"].(string), "", nil), http.StatusOK, "serve thumbnail")

	// Tampered or expired signatures are refused.
	url := photo["url"].(string)
	e.mustStatus(e.do(http.MethodGet, strings.Replace(url, "sig=", "sig=0", 1), "", nil), http.StatusNotFound, "tampered signature")
	e.mustStatus(e.do(http.MethodGet, strings.Replace(url, "size=full", "size=thumb", 1), "", nil), http.StatusNotFound, "size swap")
	e.mustStatus(e.do(http.MethodGet, "/photos/"+photo["id"].(string)+"/file?size=full&exp=1&sig=abc", "", nil), http.StatusNotFound, "expired")
	e.mustStatus(e.do(http.MethodGet, "/photos/not-a-uuid/file", "", nil), http.StatusNotFound, "bad id")

	// Fill to the limit.
	for i := 0; i < 5; i++ {
		e.mustStatus(e.upload(http.MethodPost, "/photos", p.Token, "x.png", testPNG(300, 400)), http.StatusCreated, "upload")
	}
	r := e.upload(http.MethodPost, "/photos", p.Token, "x.png", testPNG(300, 400))
	if r.Status != http.StatusConflict || r.code(t) != "photo_limit" {
		t.Fatalf("7th photo must be refused, got %d %s", r.Status, r.Body)
	}

	list := func() []string {
		r := e.mustStatus(e.do(http.MethodGet, "/photos", p.Token, nil), http.StatusOK, "list").json(t)
		var ids []string
		for i, raw := range r["photos"].([]any) {
			m := raw.(map[string]any)
			if int(m["position"].(float64)) != i {
				t.Fatalf("positions must be dense and ordered: %v", r["photos"])
			}
			ids = append(ids, m["id"].(string))
		}
		return ids
	}
	ids := list()
	if len(ids) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(ids))
	}

	// Main photo and reorder.
	e.mustStatus(e.do(http.MethodPost, "/photos/"+ids[3]+"/main", p.Token, nil), http.StatusOK, "set main")
	after := list()
	if after[0] != ids[3] || after[1] != ids[0] {
		t.Fatalf("set main should move the photo first and keep the rest in order: %v", after)
	}
	reordered := []string{after[5], after[4], after[3], after[2], after[1], after[0]}
	e.mustStatus(e.do(http.MethodPut, "/photos/order", p.Token, map[string]any{"ids": reordered}), http.StatusOK, "reorder")
	if got := list(); got[0] != reordered[0] || got[5] != reordered[5] {
		t.Fatalf("reorder was not applied: %v", got)
	}
	bad := e.do(http.MethodPut, "/photos/order", p.Token, map[string]any{"ids": reordered[:3]})
	if bad.Status != http.StatusBadRequest || bad.code(t) != "invalid_order" {
		t.Fatalf("partial order must be rejected, got %d %s", bad.Status, bad.Body)
	}

	// Replace keeps the position.
	target := list()[2]
	e.mustStatus(e.upload(http.MethodPut, "/photos/"+target, p.Token, "new.jpg", testJPEG(400, 500)), http.StatusOK, "replace")
	if list()[2] != target {
		t.Fatal("replace must keep the photo position")
	}

	// Another user cannot touch my photos.
	other := e.register("photos-other")
	e.mustStatus(e.do(http.MethodDelete, "/photos/"+target, other.Token, nil), http.StatusNotFound, "foreign delete")
	e.mustStatus(e.upload(http.MethodPut, "/photos/"+target, other.Token, "evil.png", testPNG(300, 300)), http.StatusNotFound, "foreign replace")
	e.mustStatus(e.do(http.MethodPost, "/photos/"+target+"/main", other.Token, nil), http.StatusNotFound, "foreign main")

	// Delete compacts positions and removes the files from disk.
	var key string
	if err := e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, target).Scan(&key); err != nil {
		t.Fatal(err)
	}
	e.mustStatus(e.do(http.MethodDelete, "/photos/"+target, p.Token, nil), http.StatusOK, "delete")
	if got := list(); len(got) != 5 || contains(got, target) {
		t.Fatalf("photo was not deleted: %v", got)
	}
	if _, err := os.Stat(filepath.Join(e.uploadDir, key[:2], key)); !os.IsNotExist(err) {
		t.Fatalf("blob should be removed from disk, stat err=%v", err)
	}
}

func TestPhotoUploadSizeLimit(t *testing.T) {
	e := newEnv(t)
	p := e.register("bigupload")
	huge := make([]byte, 11<<20)
	r := e.upload(http.MethodPost, "/photos", p.Token, "huge.jpg", huge)
	if r.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d %s", r.Status, r.Body)
	}
}

func TestInterestsCatalog(t *testing.T) {
	e := newEnv(t)
	p := e.register("catalog")
	r := e.mustStatus(e.do(http.MethodGet, "/interests", p.Token, nil), http.StatusOK, "interests").json(t)
	if len(r["interests"].([]any)) < 10 {
		t.Fatalf("catalog looks empty: %v", r)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/interests", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatal("catalog requires authentication")
	}
}
