package app_test

import (
	"encoding/json"
	"image/color"
	"strings"
	"testing"
)

func TestProfileOnboardingAndValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("onb")

	var state struct {
		Profile  *json.RawMessage `json:"profile"`
		Complete bool             `json:"complete"`
	}
	e.expect(u.do("GET", "/profile", nil), 200).decode(t, &state)
	if state.Profile != nil || state.Complete {
		t.Fatalf("fresh account must have no profile: %+v", state)
	}

	base := func() map[string]any {
		return map[string]any{"firstName": "Léa", "birthDate": birthDateForAge(28), "gender": "woman", "bio": "Salut", "interests": []string{"travel", "music"}}
	}
	underage := base()
	underage["birthDate"] = birthDateForAge(17)
	if r := e.expect(u.do("PUT", "/profile", underage), 400); r.errCode() != "underage" {
		t.Fatalf("expected underage, got %s", r.Body)
	}
	for name, mutate := range map[string]func(map[string]any){
		"empty name":        func(m map[string]any) { m["firstName"] = "   " },
		"long name":         func(m map[string]any) { m["firstName"] = strings.Repeat("a", 41) },
		"bad gender":        func(m map[string]any) { m["gender"] = "robot" },
		"bad date":          func(m map[string]any) { m["birthDate"] = "12/03/1990" },
		"future date":       func(m map[string]any) { m["birthDate"] = "2999-01-01" },
		"too old":           func(m map[string]any) { m["birthDate"] = "1900-01-01" },
		"long bio":          func(m map[string]any) { m["bio"] = strings.Repeat("x", 501) },
		"unknown interest":  func(m map[string]any) { m["interests"] = []string{"nope"} },
		"too many interest": func(m map[string]any) { m["interests"] = strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",") },
	} {
		body := base()
		mutate(body)
		if r := u.do("PUT", "/profile", body); r.Status != 400 {
			t.Fatalf("%s: expected 400, got %d %s", name, r.Status, r.Body)
		}
	}

	var profile struct {
		FirstName   string `json:"firstName"`
		Age         int    `json:"age"`
		HasLocation bool   `json:"hasLocation"`
		Interests   []struct {
			Slug string `json:"slug"`
		} `json:"interests"`
	}
	e.expect(u.do("PUT", "/profile", base()), 200).decode(t, &profile)
	if profile.FirstName != "Léa" || profile.Age != 28 || len(profile.Interests) != 2 || profile.HasLocation {
		t.Fatalf("unexpected profile %+v", profile)
	}

	// Still incomplete: no preferences and no photo.
	e.expect(u.do("GET", "/profile", nil), 200).decode(t, &state)
	if state.Complete {
		t.Fatal("profile without preferences/photo must not be complete")
	}
	for _, bad := range []map[string]any{
		{"interestedIn": []string{}, "minAge": 18, "maxAge": 40},
		{"interestedIn": []string{"alien"}, "minAge": 18, "maxAge": 40},
		{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40},
		{"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30},
		{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 100},
		{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 0},
		{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 501},
	} {
		if r := u.do("PUT", "/preferences", bad); r.Status != 400 {
			t.Fatalf("preferences %v: expected 400, got %d", bad, r.Status)
		}
	}
	e.expect(u.do("PUT", "/preferences", map[string]any{"interestedIn": []string{"man", "man"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 50}), 200)
	e.expect(e.upload("POST", "/photos", u.Token, pngBytes(30, 30, color.Black), "p.png"), 201)

	e.expect(u.do("GET", "/profile", nil), 200).decode(t, &state)
	if !state.Complete {
		t.Fatal("expected complete profile")
	}
	var me authBody
	e.expect(u.do("GET", "/me", nil), 200).decode(t, &me.User)
	if !me.User.ProfileComplete {
		t.Fatal("/me should report profileComplete")
	}

	var catalog struct {
		Items []struct{ Slug string } `json:"items"`
	}
	e.expect(u.do("GET", "/interests", nil), 200).decode(t, &catalog)
	if len(catalog.Items) < 20 {
		t.Fatalf("interest catalog too small: %d", len(catalog.Items))
	}
}

func TestLocationIsCoarseAndNeverExposed(t *testing.T) {
	e := newEnv(t)
	a := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}})
	b := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})

	e.expect(a.do("PUT", "/profile/location", map[string]any{"latitude": 48.85661, "longitude": 2.35222, "city": "Paris"}), 200)
	e.expect(b.do("PUT", "/profile/location", map[string]any{"latitude": 48.9, "longitude": 2.5, "city": "Saint-Denis"}), 200)
	for _, bad := range []map[string]any{
		{"latitude": 91, "longitude": 0}, {"latitude": 0, "longitude": 181}, {"longitude": 3}, {"latitude": 3},
	} {
		if r := a.do("PUT", "/profile/location", bad); r.Status != 400 {
			t.Fatalf("location %v: expected 400, got %d", bad, r.Status)
		}
	}

	// Stored coordinates are rounded to 2 decimals.
	var lat, lng float64
	if err := e.pool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, a.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be rounded, got %v,%v", lat, lng)
	}

	r := e.expect(a.do("GET", "/discovery", nil), 200)
	body := string(r.Body)
	if strings.Contains(body, "latitude") || strings.Contains(body, "longitude") || strings.Contains(body, "48.9") {
		t.Fatalf("coordinates leaked in discovery payload: %s", body)
	}
	var feed struct {
		Items []struct {
			DistanceKm *int `json:"distanceKm"`
		} `json:"items"`
	}
	r.decode(t, &feed)
	if len(feed.Items) != 1 || feed.Items[0].DistanceKm == nil || *feed.Items[0].DistanceKm%5 != 0 || *feed.Items[0].DistanceKm < 5 {
		t.Fatalf("distance must be approximate (multiple of 5): %s", body)
	}

	// Hiding the distance removes it from what others see.
	e.expect(b.do("PUT", "/profile", map[string]any{"firstName": "Bob", "birthDate": birthDateForAge(30), "gender": "man", "showDistance": false}), 200)
	e.expect(a.do("GET", "/discovery", nil), 200).decode(t, &feed)
	if len(feed.Items) != 1 || feed.Items[0].DistanceKm != nil {
		t.Fatalf("distance should be hidden: %+v", feed.Items)
	}

	e.expect(a.do("DELETE", "/profile/location", nil), 204)
}

func TestPhotoLifecycleAndSecurity(t *testing.T) {
	e := newEnv(t)
	a := e.signup("Alice", signupOpts{Gender: "woman", InterestedIn: []string{"man"}, NoPhoto: true})
	b := e.signup("Bob", signupOpts{Gender: "man", InterestedIn: []string{"woman"}})
	stranger := e.register("stranger")

	// Dangerous or invalid uploads.
	e.expect(e.upload("POST", "/photos", a.Token, []byte("<?php system($_GET['c']); ?>"), "shell.php.png"), 400)
	e.expect(e.upload("POST", "/photos", a.Token, []byte("GIF89a......"), "a.gif"), 400)
	e.expect(e.upload("POST", "/photos", a.Token, append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 50)...), "broken.png"), 400)
	e.expect(e.upload("POST", "/photos", a.Token, nil, "empty.png"), 400)
	e.expect(e.upload("POST", "/photos", a.Token, make([]byte, 9<<20), "huge.png"), 413)

	type photo struct {
		ID        string `json:"id"`
		Position  int    `json:"position"`
		IsPrimary bool   `json:"isPrimary"`
		URL       string `json:"url"`
	}
	var ids []string
	colors := []color.Color{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}}
	for i, c := range colors {
		var p photo
		e.expect(e.upload("POST", "/photos", a.Token, pngBytes(2000+i, 1500, c), "p.png"), 201).decode(t, &p)
		if p.Position != i || p.IsPrimary != (i == 0) {
			t.Fatalf("unexpected photo %+v", p)
		}
		ids = append(ids, p.ID)
	}

	// Served as a downsized JPEG, authentication required.
	img := e.expect(e.do("GET", "/photos/"+ids[0]+"/image", a.Token, nil), 200)
	if img.Header.Get("Content-Type") != "image/jpeg" || len(img.Body) < 100 || img.Body[0] != 0xFF || img.Body[1] != 0xD8 {
		t.Fatalf("expected a JPEG, got %s (%d bytes)", img.Header.Get("Content-Type"), len(img.Body))
	}
	thumb := e.expect(e.do("GET", "/photos/"+ids[0]+"/image?size=thumb", a.Token, nil), 200)
	if len(thumb.Body) >= len(img.Body) {
		t.Fatal("thumbnail should be smaller than the full image")
	}
	e.expect(e.do("GET", "/photos/"+ids[0]+"/image?size=huge", a.Token, nil), 400)
	e.expect(e.do("GET", "/photos/"+ids[0]+"/image", "", nil), 401)
	e.expect(e.do("GET", "/photos/not-a-uuid/image", a.Token, nil), 404)

	// Alice has no complete profile yet? She has photos + prefs + profile, so she is visible to Bob.
	e.expect(b.do("GET", "/photos/"+ids[0]+"/image", nil), 200)
	// Any member can see discoverable profiles photos; a blocked user cannot.
	e.expect(stranger.do("GET", "/photos/"+ids[0]+"/image", nil), 200) // discoverable & complete profile: allowed to any member
	e.expect(b.do("POST", "/blocks", map[string]string{"userId": a.ID}), 204)
	e.expect(b.do("GET", "/photos/"+ids[0]+"/image", nil), 404)
	e.expect(a.do("GET", "/photos/"+ids[0]+"/image", nil), 200) // owner unaffected

	// Hidden profiles (not discoverable, not matched) hide their photos too.
	e.expect(a.do("PUT", "/profile", map[string]any{"firstName": "Alice", "birthDate": birthDateForAge(30), "gender": "woman", "discoverable": false}), 200)
	e.expect(stranger.do("GET", "/photos/"+ids[1]+"/image", nil), 404)
	e.expect(a.do("PUT", "/profile", map[string]any{"firstName": "Alice", "birthDate": birthDateForAge(30), "gender": "woman", "discoverable": true}), 200)

	// Reorder: must be exactly the user's photos.
	e.expect(a.do("PUT", "/photos/order", map[string]any{"photoIds": []string{ids[0]}}), 400)
	e.expect(a.do("PUT", "/photos/order", map[string]any{"photoIds": []string{ids[0], ids[0], ids[1]}}), 400)
	e.expect(a.do("PUT", "/photos/order", map[string]any{"photoIds": []string{ids[0], ids[1], b.ID}}), 400)
	var reordered struct {
		Items []photo `json:"items"`
	}
	e.expect(a.do("PUT", "/photos/order", map[string]any{"photoIds": []string{ids[2], ids[0], ids[1]}}), 200).decode(t, &reordered)
	if reordered.Items[0].ID != ids[2] || !reordered.Items[0].IsPrimary || reordered.Items[2].ID != ids[1] {
		t.Fatalf("unexpected order %+v", reordered.Items)
	}

	// Another user cannot delete or replace it.
	e.expect(b.do("DELETE", "/photos/"+ids[2], nil), 404)
	e.expect(e.upload("PUT", "/photos/"+ids[2], b.Token, pngBytes(10, 10, color.Black), "p.png"), 404)

	// Replace keeps the position and returns a fresh id.
	var replaced photo
	e.expect(e.upload("PUT", "/photos/"+ids[0], a.Token, pngBytes(50, 50, color.White), "p.png"), 200).decode(t, &replaced)
	if replaced.Position != 1 || replaced.ID == ids[0] {
		t.Fatalf("unexpected replacement %+v", replaced)
	}
	e.expect(a.do("GET", "/photos/"+ids[0]+"/image", nil), 404)

	// Delete compacts positions; the limit is 6.
	e.expect(a.do("DELETE", "/photos/"+ids[2], nil), 204)
	var state struct {
		Profile struct {
			Photos []photo `json:"photos"`
		} `json:"profile"`
	}
	e.expect(a.do("GET", "/profile", nil), 200).decode(t, &state)
	if len(state.Profile.Photos) != 2 || state.Profile.Photos[0].Position != 0 || state.Profile.Photos[1].Position != 1 {
		t.Fatalf("positions should be compacted: %+v", state.Profile.Photos)
	}
	for i := 0; i < 4; i++ {
		e.expect(e.upload("POST", "/photos", a.Token, pngBytes(20, 20, color.Black), "p.png"), 201)
	}
	if r := e.expect(e.upload("POST", "/photos", a.Token, pngBytes(20, 20, color.Black), "p.png"), 409); r.errCode() != "photo_limit" {
		t.Fatalf("expected photo_limit, got %s", r.Body)
	}
}
