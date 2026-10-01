package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t)
	u := e.register("prof")
	valid := func() map[string]any {
		return map[string]any{"firstName": "Léa", "birthDate": time.Now().AddDate(-30, 0, 0).Format("2006-01-02"), "gender": "woman", "bio": "Salut"}
	}

	e.want(e.do("GET", "/me/profile", "", nil), 401)
	p := e.want(e.do("GET", "/me/profile", u.Token, nil), 200).json()
	if p["exists"] != false || p["complete"] != false {
		t.Fatalf("fresh account must have no profile: %v", p)
	}

	// 18+ is enforced server-side, to the day.
	under := valid()
	under["birthDate"] = time.Now().AddDate(-18, 0, 1).Format("2006-01-02")
	e.want(e.do("PUT", "/me/profile", u.Token, under), 422)
	exact := valid()
	exact["birthDate"] = time.Now().AddDate(-18, 0, 0).Format("2006-01-02")
	e.want(e.do("PUT", "/me/profile", u.Token, exact), 200)

	for name, mut := range map[string]func(m map[string]any){
		"bad gender":  func(m map[string]any) { m["gender"] = "robot" },
		"bad date":    func(m map[string]any) { m["birthDate"] = "31/12/1990" },
		"future date": func(m map[string]any) { m["birthDate"] = time.Now().AddDate(1, 0, 0).Format("2006-01-02") },
		"empty name":  func(m map[string]any) { m["firstName"] = "   " },
		"long name":   func(m map[string]any) { m["firstName"] = strings.Repeat("a", 41) },
		"long bio":    func(m map[string]any) { m["bio"] = strings.Repeat("é", 501) },
	} {
		m := valid()
		mut(m)
		if r := e.do("PUT", "/me/profile", u.Token, m); r.Status < 400 || r.Status >= 500 {
			t.Errorf("%s: expected 4xx, got %d %s", name, r.Status, r.Body)
		}
	}

	p = e.want(e.do("PUT", "/me/profile", u.Token, valid()), 200).json()
	if p["exists"] != true || p["complete"] != false {
		t.Fatalf("profile without a photo is incomplete: %v", p)
	}
	if m := p["missing"].([]any); len(m) != 1 || m[0] != "photos" {
		t.Fatalf("missing: %v", m)
	}

	e.want(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40, "maxDistanceKm": 30}), 400)
	e.want(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 25, "maxDistanceKm": 30}), 400)
	e.want(e.do("PUT", "/me/preferences", u.Token, map[string]any{"interestedIn": []string{"man", "woman"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 30}), 200)

	// Only a ~1 km snapped position is ever stored.
	e.want(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 91, "longitude": 0}), 400)
	e.want(e.do("PUT", "/me/location", u.Token, map[string]any{"latitude": 48.856614, "longitude": 2.352222, "city": "Paris"}), 200)
	var lat, lng float64
	if err := e.pool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be snapped, stored %v,%v", lat, lng)
	}

	ints := e.want(e.do("GET", "/interests", u.Token, nil), 200).json()["interests"].([]any)
	if len(ints) < 10 {
		t.Fatalf("interest catalog too small: %d", len(ints))
	}
	id1, id2 := ints[0].(map[string]any)["id"], ints[1].(map[string]any)["id"]
	r := e.want(e.do("PUT", "/me/interests", u.Token, map[string]any{"interestIds": []any{id1, id2}}), 200).json()
	if len(r["interests"].([]any)) != 2 {
		t.Fatalf("interests: %v", r["interests"])
	}
	e.want(e.do("PUT", "/me/interests", u.Token, map[string]any{"interestIds": []any{id1, id1}}), 400)
	e.want(e.do("PUT", "/me/interests", u.Token, map[string]any{"interestIds": []any{99999}}), 400)
	many := make([]int, 11)
	for i := range many {
		many[i] = i + 1
	}
	e.want(e.do("PUT", "/me/interests", u.Token, map[string]any{"interestIds": many}), 400)
}

func TestPhotos(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(spec{Name: "Ana", Gender: "woman", NoPhoto: true})
	b := e.newUser(spec{Name: "Ben", Gender: "man"})

	e.want(e.do("POST", "/me/photos", "", nil), 401)
	bad := e.upload("POST", "/me/photos", a.Token, []byte("<?php echo 1; ?>"))
	if bad.Status != 422 {
		t.Fatalf("non-image must be rejected: %d %s", bad.Status, bad.Body)
	}
	if n := count(e, `SELECT count(*) FROM photos WHERE user_id = $1`, a.ID); n != 0 {
		t.Fatalf("rejected upload left %d rows", n)
	}
	big := make([]byte, 9<<20)
	copy(big, jpegBytes(1))
	if r := e.upload("POST", "/me/photos", a.Token, big); r.Status != 413 && r.Status != 422 {
		t.Fatalf("oversized upload must be refused, got %d", r.Status)
	}

	ids := []string{}
	for i := 0; i < 6; i++ {
		r := e.want(e.upload("POST", "/me/photos", a.Token, jpegBytes(i)), 201).json()
		if int(r["position"].(float64)) != i {
			t.Fatalf("position %d: %v", i, r)
		}
		ids = append(ids, r["id"].(string))
	}
	e.want(e.upload("POST", "/me/photos", a.Token, jpegBytes(9)), 409)

	// Reorder: the first id becomes the primary photo; partial or foreign lists are refused.
	rev := []string{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
	r := e.want(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"photoIds": rev}), 200).json()
	if r["photos"].([]any)[0].(map[string]any)["id"] != ids[5] {
		t.Fatalf("primary photo not updated: %v", r)
	}
	e.want(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"photoIds": rev[:5]}), 400)
	e.want(e.do("PUT", "/me/photos/order", a.Token, map[string]any{"photoIds": []string{ids[0], ids[0], ids[1], ids[2], ids[3], ids[4]}}), 400)

	// Serving: authenticated, sanitised JPEG, never cached publicly.
	e.want(e.do("GET", "/photos/"+ids[0]+"/image", "", nil), 401)
	img := e.want(e.do("GET", "/photos/"+ids[0]+"/image", a.Token, nil), 200)
	if img.Header.Get("Content-Type") != "image/jpeg" || !strings.HasPrefix(img.Header.Get("Cache-Control"), "private") {
		t.Fatalf("headers: %v", img.Header)
	}
	e.want(e.do("GET", "/photos/"+ids[0]+"/thumb", a.Token, nil), 200)
	e.want(e.do("GET", "/photos/not-a-uuid/image", a.Token, nil), 404)
	// Another visible member can see it; a block hides it in both directions.
	e.want(e.do("GET", "/photos/"+ids[0]+"/image", b.Token, nil), 200)
	e.want(e.do("POST", "/blocks", a.Token, map[string]any{"userId": b.ID}), 204)
	e.want(e.do("GET", "/photos/"+ids[0]+"/image", b.Token, nil), 404)
	e.want(e.do("DELETE", "/blocks/"+b.ID, a.Token, nil), 204)

	// Others cannot modify or delete them.
	e.want(e.do("DELETE", "/me/photos/"+ids[0], b.Token, nil), 404)
	e.want(e.upload("PUT", "/me/photos/"+ids[0], b.Token, jpegBytes(3)), 404)

	// Replace creates a new id at the same slot and removes the old files.
	var oldKey string
	_ = e.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE id = $1`, ids[0]).Scan(&oldKey)
	rep := e.want(e.upload("PUT", "/me/photos/"+ids[0], a.Token, jpegBytes(30)), 200).json()
	if rep["id"] == ids[0] || int(rep["position"].(float64)) != 5 {
		t.Fatalf("replace: %v", rep)
	}
	if _, err := os.Stat(filepath.Join(e.uploads, oldKey)); !os.IsNotExist(err) {
		t.Fatalf("old file must be deleted, stat err=%v", err)
	}
	e.want(e.do("GET", "/photos/"+ids[0]+"/image", a.Token, nil), 404)

	// Delete closes the gap so positions stay contiguous.
	e.want(e.do("DELETE", "/me/photos/"+ids[5], a.Token, nil), 204)
	list := e.want(e.do("GET", "/me/photos", a.Token, nil), 200).json()["photos"].([]any)
	if len(list) != 5 {
		t.Fatalf("expected 5 photos, got %d", len(list))
	}
	for i, p := range list {
		if int(p.(map[string]any)["position"].(float64)) != i {
			t.Fatalf("positions not contiguous: %v", list)
		}
	}
}
