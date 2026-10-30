package main

import (
	"context"
	"image/color"
	"net/http"
	"strings"
	"testing"
)

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t)
	u := e.register("prof")
	at := uniqueGeo()

	e.expect(e.do("GET", "/profile", nil, u.Token), http.StatusNotFound)

	valid := map[string]any{
		"displayName": "Camille", "birthDate": "1994-02-03", "gender": "woman", "bio": "Salut",
		"city": "Lyon", "latitude": at.lat, "longitude": at.lon, "interests": []string{"voyage", "yoga"},
	}
	minor := cloneMap(valid)
	minor["birthDate"] = "2015-01-01"
	e.expect(e.do("PUT", "/profile", minor, u.Token), http.StatusBadRequest)

	noLocation := cloneMap(valid)
	delete(noLocation, "latitude")
	delete(noLocation, "longitude")
	e.expect(e.do("PUT", "/profile", noLocation, u.Token), http.StatusBadRequest)

	unknown := cloneMap(valid)
	unknown["interests"] = []string{"does-not-exist"}
	e.expect(e.do("PUT", "/profile", unknown, u.Token), http.StatusBadRequest)

	r := e.expect(e.do("PUT", "/profile", valid, u.Token), http.StatusOK)
	body := string(r.Body)
	if !strings.Contains(body, `"isComplete":false`) || !strings.Contains(body, `"photos"`) {
		t.Fatalf("profile without photo must be incomplete: %s", body)
	}
	if strings.Contains(body, "latitude") || strings.Contains(body, "longitude") {
		t.Fatalf("exact coordinates must never be returned: %s", body)
	}

	// Coordinates are stored rounded (~1 km), nothing more precise is kept.
	var lat float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if diff := lat*100 - float64(int64(lat*100+0.5*sign(lat))); diff > 1e-6 || diff < -1e-6 {
		t.Fatalf("stored latitude %v is not rounded to 2 decimals", lat)
	}

	// Updating without sending coordinates keeps the stored location.
	update := cloneMap(noLocation)
	update["bio"] = "Nouvelle bio"
	e.expect(e.do("PUT", "/profile", update, u.Token), http.StatusOK)

	e.expect(e.do("PUT", "/preferences", map[string]any{"interestedIn": []string{}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 10}, u.Token), http.StatusBadRequest)
	e.expect(e.do("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30, "maxDistanceKm": 10}, u.Token), http.StatusBadRequest)
	e.expect(e.do("PUT", "/preferences", map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 35, "maxDistanceKm": 25}, u.Token), http.StatusOK)

	cat := e.expect(e.do("GET", "/interests", nil, u.Token), http.StatusOK)
	if !strings.Contains(string(cat.Body), `"slug":"voyage"`) {
		t.Fatalf("interest catalog missing: %s", cat.Body)
	}
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func TestPhotoLifecycleAndProtection(t *testing.T) {
	e := newEnv(t)
	u := e.register("photo")
	other := e.register("photo-other")
	at := uniqueGeo()
	e.expect(e.do("PUT", "/profile", map[string]any{"displayName": "Pat", "birthDate": "1990-01-01", "gender": "man", "latitude": at.lat, "longitude": at.lon}, u.Token), http.StatusOK)

	type photosOut struct {
		Photos []struct {
			ID       string `json:"id"`
			URL      string `json:"url"`
			Position int    `json:"position"`
		} `json:"photos"`
	}

	img := pngBytes(80, 60, color.RGBA{10, 120, 200, 255})
	var list photosOut
	e.expect(e.upload("POST", "/photos", u.Token, "file", img), http.StatusCreated).decode(t, &list)
	e.expect(e.upload("POST", "/photos", u.Token, "file", img), http.StatusCreated).decode(t, &list)
	e.expect(e.upload("POST", "/photos", u.Token, "file", img), http.StatusCreated).decode(t, &list)
	if len(list.Photos) != 3 || list.Photos[2].Position != 2 {
		t.Fatalf("unexpected photos %+v", list.Photos)
	}

	// Rejected uploads: not an image, wrong field, script disguised as image.
	e.expect(e.upload("POST", "/photos", u.Token, "file", []byte("<script>alert(1)</script>")), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/photos", u.Token, "other", img), http.StatusBadRequest)
	e.expect(e.upload("POST", "/photos", u.Token, "file", append([]byte("GIF89a"), make([]byte, 50)...)), http.StatusUnsupportedMediaType)

	// Served file is a re-encoded JPEG; signature is mandatory and bound to the photo.
	first := list.Photos[0]
	file := e.expect(e.do("GET", first.URL, nil, ""), http.StatusOK)
	if file.Header.Get("Content-Type") != "image/jpeg" || len(file.Body) < 2 || file.Body[0] != 0xFF || file.Body[1] != 0xD8 {
		t.Fatalf("photo must be served as a JPEG, got %s", file.Header.Get("Content-Type"))
	}
	e.expect(e.do("GET", "/photos/"+first.ID+"/file", nil, ""), http.StatusNotFound)
	e.expect(e.do("GET", strings.Replace(first.URL, "sig=", "sig=00", 1), nil, ""), http.StatusNotFound)
	e.expect(e.do("GET", strings.Replace(first.URL, first.ID, list.Photos[1].ID, 1), nil, ""), http.StatusNotFound)

	// Only the owner can change or delete a photo.
	e.expect(e.do("DELETE", "/photos/"+first.ID, nil, other.Token), http.StatusNotFound)
	e.expect(e.upload("PUT", "/photos/"+first.ID, other.Token, "file", img), http.StatusNotFound)

	// Reordering must list exactly the owner's photos.
	e.expect(e.do("PUT", "/photos/order", map[string]any{"photoIds": []string{first.ID}}, u.Token), http.StatusBadRequest)
	e.expect(e.do("PUT", "/photos/order", map[string]any{"photoIds": []string{first.ID, list.Photos[1].ID, "not-a-uuid"}}, u.Token), http.StatusBadRequest)
	e.expect(e.do("PUT", "/photos/order", map[string]any{"photoIds": []string{list.Photos[2].ID, first.ID, list.Photos[1].ID}}, u.Token), http.StatusOK).decode(t, &list)
	if list.Photos[0].ID == first.ID || list.Photos[1].ID != first.ID {
		t.Fatalf("order not applied: %+v", list.Photos)
	}
	primary := list.Photos[0]

	// Replace keeps the slot, delete compacts positions and removes the file.
	e.expect(e.upload("PUT", "/photos/"+primary.ID, u.Token, "file", pngBytes(30, 30, color.RGBA{255, 0, 0, 255})), http.StatusOK).decode(t, &list)
	if list.Photos[0].ID != primary.ID {
		t.Fatal("replace must keep the position")
	}
	var key string
	if err := e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, first.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	e.expect(e.do("DELETE", "/photos/"+first.ID, nil, u.Token), http.StatusOK).decode(t, &list)
	if len(list.Photos) != 2 || list.Photos[1].Position != 1 {
		t.Fatalf("positions must be compacted: %+v", list.Photos)
	}
	if _, err := e.store.Open(key); err == nil {
		t.Fatal("deleted photo must be removed from storage")
	}

	// Upload limit.
	for i := 0; i < 4; i++ {
		e.expect(e.upload("POST", "/photos", u.Token, "file", img), http.StatusCreated)
	}
	e.expect(e.upload("POST", "/photos", u.Token, "file", img), http.StatusConflict)
}
