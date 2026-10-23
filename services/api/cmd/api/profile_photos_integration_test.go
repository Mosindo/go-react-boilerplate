package main

import (
	"net/http"
	"testing"
)

func TestProfileLifecycleAndValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("profile")

	e.expect(e.do(http.MethodGet, "/me/profile", nil, u.Token), http.StatusNotFound)

	valid := map[string]any{"firstName": "Léa", "birthDate": birthDateForAge(28), "gender": "woman", "bio": "Hello", "city": "Paris", "interests": []string{"cinema", "music"}}
	underage := map[string]any{"firstName": "Kid", "birthDate": birthDateForAge(17), "gender": "man", "bio": "", "city": "Paris", "interests": []string{}}
	badInterest := map[string]any{"firstName": "Léa", "birthDate": birthDateForAge(28), "gender": "woman", "bio": "", "city": "Paris", "interests": []string{"does-not-exist"}}
	badGender := map[string]any{"firstName": "Léa", "birthDate": birthDateForAge(28), "gender": "x", "bio": "", "city": "Paris", "interests": []string{}}

	r := e.expect(e.do(http.MethodPut, "/me/profile", underage, u.Token), http.StatusBadRequest).JSON(t)
	if r["fields"].(map[string]any)["birthDate"] == nil {
		t.Fatalf("expected a birthDate field error: %v", r)
	}
	e.expect(e.do(http.MethodPut, "/me/profile", badInterest, u.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPut, "/me/profile", badGender, u.Token), http.StatusBadRequest)

	p := e.expect(e.do(http.MethodPut, "/me/profile", valid, u.Token), http.StatusOK).JSON(t)
	if p["age"].(float64) != 28 || len(p["interests"].([]any)) != 2 || p["hasLocation"] != false {
		t.Fatalf("unexpected profile %v", p)
	}

	// Coordinates are coarsened and omitted coordinates keep the stored location.
	valid["latitude"], valid["longitude"] = 45.76412, 4.83566
	p = e.expect(e.do(http.MethodPut, "/me/profile", valid, u.Token), http.StatusOK).JSON(t)
	if p["hasLocation"] != true {
		t.Fatal("location should be stored")
	}
	var lat float64
	_ = e.pool.QueryRow(t.Context(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat)
	if lat != 45.76 {
		t.Fatalf("latitude must be rounded to the 2km grid, stored %v", lat)
	}
	delete(valid, "latitude")
	delete(valid, "longitude")
	p = e.expect(e.do(http.MethodPut, "/me/profile", valid, u.Token), http.StatusOK).JSON(t)
	if p["hasLocation"] != true {
		t.Fatal("omitting coordinates must keep the stored location")
	}
	valid["clearLocation"] = true
	p = e.expect(e.do(http.MethodPut, "/me/profile", valid, u.Token), http.StatusOK).JSON(t)
	if p["hasLocation"] != false {
		t.Fatal("clearLocation must remove coordinates")
	}

	interests := e.expect(e.do(http.MethodGet, "/interests", nil, u.Token), http.StatusOK).JSON(t)
	if len(interests["interests"].([]any)) < 20 {
		t.Fatal("interest catalogue missing")
	}
}

func TestPreferencesValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("prefs")
	defaults := e.expect(e.do(http.MethodGet, "/me/preferences", nil, u.Token), http.StatusOK).JSON(t)
	if defaults["discoverable"] != true || defaults["minAge"].(float64) != 18 {
		t.Fatalf("unexpected defaults %v", defaults)
	}
	bad := []map[string]any{
		{"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10, "discoverable": true, "showDistance": true},
		{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30, "maxDistanceKm": 10, "discoverable": true, "showDistance": true},
		{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 10, "discoverable": true, "showDistance": true},
		{"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 0, "discoverable": true, "showDistance": true},
	}
	for i, b := range bad {
		if r := e.do(http.MethodPut, "/me/preferences", b, u.Token); r.Status != http.StatusBadRequest {
			t.Errorf("case %d: expected 400, got %d", i, r.Status)
		}
	}
	good := map[string]any{"interestedIn": []string{"woman", "non_binary"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 20, "discoverable": false, "showDistance": false}
	e.expect(e.do(http.MethodPut, "/me/preferences", good, u.Token), http.StatusOK)
	got := e.expect(e.do(http.MethodGet, "/me/preferences", nil, u.Token), http.StatusOK).JSON(t)
	if got["discoverable"] != false || got["maxDistanceKm"].(float64) != 20 {
		t.Fatalf("preferences not persisted: %v", got)
	}
}

func TestPhotoUploadValidationAndLimits(t *testing.T) {
	e := newEnv(t)
	u := e.register("photos")

	e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "x.png", []byte("just text"), u.Token), http.StatusUnsupportedMediaType)
	e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "x.jpg", []byte("<svg onload=alert(1)></svg>"), u.Token), http.StatusUnsupportedMediaType)
	e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "x.png", pngBytes(20, 20), u.Token), http.StatusBadRequest)
	e.expect(e.upload(http.MethodPost, "/me/photos", "wrongfield", "x.png", pngBytes(200, 200), u.Token), http.StatusBadRequest)
	big := append(pngBytes(200, 200), make([]byte, 9<<20)...)
	e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "big.png", big, u.Token), http.StatusRequestEntityTooLarge)

	var ids []string
	for i := 0; i < 6; i++ {
		r := e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "p.png", pngBytes(200+i, 200), u.Token), http.StatusCreated).JSON(t)
		if int(r["position"].(float64)) != i {
			t.Fatalf("photo %d got position %v", i, r["position"])
		}
		ids = append(ids, r["id"].(string))
	}
	e.expect(e.upload(http.MethodPost, "/me/photos", "photo", "p.png", pngBytes(200, 200), u.Token), http.StatusConflict)

	// Reorder: last photo becomes main. Wrong/duplicate/partial lists are rejected.
	order := append([]string{ids[5]}, ids[:5]...)
	r := e.expect(e.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": order}, u.Token), http.StatusOK).JSON(t)
	first := r["photos"].([]any)[0].(map[string]any)
	if first["id"] != ids[5] || first["position"].(float64) != 0 {
		t.Fatalf("reorder failed: %v", first)
	}
	e.expect(e.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": ids[:3]}, u.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodPut, "/me/photos/order", map[string]any{"photoIds": append(ids[:5:5], ids[0])}, u.Token), http.StatusBadRequest)

	// Delete re-numbers the rest; replace keeps the position.
	e.expect(e.do(http.MethodDelete, "/me/photos/"+ids[5], nil, u.Token), http.StatusNoContent)
	list := e.expect(e.do(http.MethodGet, "/me/photos", nil, u.Token), http.StatusOK).JSON(t)["photos"].([]any)
	if len(list) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(list))
	}
	for i, p := range list {
		if int(p.(map[string]any)["position"].(float64)) != i {
			t.Fatalf("positions not contiguous: %v", list)
		}
	}
	replaced := e.expect(e.upload(http.MethodPut, "/me/photos/"+ids[0], "photo", "n.png", pngBytes(300, 150), u.Token), http.StatusOK).JSON(t)
	if replaced["id"] != ids[0] || replaced["width"].(float64) != 300 {
		t.Fatalf("replace failed: %v", replaced)
	}
	e.expect(e.do(http.MethodDelete, "/me/photos/"+ids[5], nil, u.Token), http.StatusNotFound)
}

func TestPhotoAccessControl(t *testing.T) {
	e := newEnv(t)
	a, b := e.pair()
	stranger := e.register("stranger")

	photoID := e.expect(e.do(http.MethodGet, "/me/photos", nil, a.Token), http.StatusOK).JSON(t)["photos"].([]any)[0].(map[string]any)["id"].(string)
	path := "/photos/" + photoID

	r := e.expect(e.do(http.MethodGet, path, nil, a.Token), http.StatusOK)
	if r.Header.Get("Content-Type") != "image/jpeg" || r.Header.Get("Cache-Control") != "private, max-age=86400" {
		t.Fatalf("unexpected headers %v", r.Header)
	}
	thumb := e.expect(e.do(http.MethodGet, path+"?size=thumb", nil, b.Token), http.StatusOK)
	if len(thumb.Body) >= len(r.Body) && len(thumb.Body) > 0 && thumb.Body[0] != 0xFF {
		t.Fatal("thumb should be a JPEG")
	}
	e.expect(e.do(http.MethodGet, path+"?size=huge", nil, a.Token), http.StatusBadRequest)
	e.expect(e.do(http.MethodGet, path, nil, ""), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, "/photos/not-a-uuid", nil, a.Token), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/photos/123e4567-e89b-12d3-a456-426614174000", nil, a.Token), http.StatusNotFound)
	// Stranger with a valid token can load it only while no block exists.
	e.expect(e.do(http.MethodGet, path, nil, stranger.Token), http.StatusOK)

	// Conditional requests.
	req := httpRequest(path, b.Token)
	req.Header.Set("If-None-Match", r.Header.Get("ETag"))
	if cond := e.serve(req); cond.Status != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", cond.Status)
	}

	// A block cuts access in both directions.
	e.expect(e.do(http.MethodPost, "/blocks", map[string]string{"userId": b.ID}, a.Token), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, path, nil, b.Token), http.StatusNotFound)
	bPhoto := e.expect(e.do(http.MethodGet, "/me/photos", nil, b.Token), http.StatusOK).JSON(t)["photos"].([]any)[0].(map[string]any)["id"].(string)
	e.expect(e.do(http.MethodGet, "/photos/"+bPhoto, nil, a.Token), http.StatusNotFound)
	// The owner always keeps access.
	e.expect(e.do(http.MethodGet, path, nil, a.Token), http.StatusOK)
}
