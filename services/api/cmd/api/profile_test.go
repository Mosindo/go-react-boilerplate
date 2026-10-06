package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestProfileOnboardingProgressAndValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("lea")

	self := e.expect(e.do(http.MethodGet, "/profile", u.Token, nil), http.StatusOK).json(t)
	onboarding := self["onboarding"].(map[string]any)
	if onboarding["complete"] != false || self["profile"] != nil {
		t.Fatalf("fresh account must have incomplete onboarding: %v", self)
	}

	// Underage is refused server-side whatever the client says.
	e.expect(e.do(http.MethodPut, "/profile", u.Token, map[string]any{"firstName": "Lea", "birthdate": birthdateFor(17), "gender": "woman"}), http.StatusUnprocessableEntity)
	for name, body := range map[string]map[string]any{
		"bad gender":    {"firstName": "Lea", "birthdate": birthdateFor(30), "gender": "alien"},
		"bad name":      {"firstName": "<script>", "birthdate": birthdateFor(30), "gender": "woman"},
		"bad date":      {"firstName": "Lea", "birthdate": "31/12/1990", "gender": "woman"},
		"too old":       {"firstName": "Lea", "birthdate": "1900-01-01", "gender": "woman"},
		"bio too long":  {"firstName": "Lea", "birthdate": birthdateFor(30), "gender": "woman", "bio": strings.Repeat("a", 501)},
		"missing field": {"firstName": "Lea"},
	} {
		t.Run(name, func(t *testing.T) {
			if r := e.do(http.MethodPut, "/profile", u.Token, body); r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
				t.Fatalf("expected 4xx, got %d %s", r.Status, r.Body)
			}
		})
	}

	// Location/preferences before the profile exists is a conflict, not a crash.
	e.expect(e.do(http.MethodPut, "/profile/location", u.Token, map[string]any{"latitude": 48.8, "longitude": 2.3}), http.StatusConflict)

	e.expect(e.do(http.MethodPut, "/profile", u.Token, map[string]any{"firstName": "  Léa  ", "birthdate": birthdateFor(30), "gender": "woman", "bio": "Salut"}), http.StatusOK)
	self = e.expect(e.do(http.MethodGet, "/profile", u.Token, nil), http.StatusOK).json(t)
	if self["profile"].(map[string]any)["firstName"] != "Léa" {
		t.Fatalf("first name must be trimmed: %v", self["profile"])
	}
	missing := self["onboarding"].(map[string]any)["missing"].([]any)
	if len(missing) != 3 { // location, preferences, photos
		t.Fatalf("expected 3 missing steps, got %v", missing)
	}

	e.expect(e.do(http.MethodPut, "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40, "maxDistanceKm": 30}), http.StatusUnprocessableEntity)
	e.expect(e.do(http.MethodPut, "/preferences", u.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 30}), http.StatusUnprocessableEntity)
}

func TestLocationIsStoredCoarsely(t *testing.T) {
	e := newEnv(t)
	u := e.newPerson(persona{Name: "Noa", Gender: "woman", InterestedIn: []string{"man"}})
	e.expect(e.do(http.MethodPut, "/profile/location", u.Token, map[string]any{"latitude": 48.856614, "longitude": 2.352222, "city": "Paris"}), http.StatusOK)

	var lat, lng float64
	if err := e.pool.QueryRow(t.Context(), `SELECT latitude, longitude FROM profiles WHERE user_id=$1`, u.ID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 || lng != 2.35 {
		t.Fatalf("coordinates must be rounded before storage, got %v,%v", lat, lng)
	}
	e.expect(e.do(http.MethodPut, "/profile/location", u.Token, map[string]any{"latitude": 95, "longitude": 0}), http.StatusBadRequest)
	// Equator / prime meridian are valid coordinates.
	e.expect(e.do(http.MethodPut, "/profile/location", u.Token, map[string]any{"latitude": 0, "longitude": 0}), http.StatusOK)
}

func TestPublicProfileNeverLeaksPrivateData(t *testing.T) {
	e := newEnv(t)
	a := e.newPerson(persona{Name: "Ana", Gender: "woman", InterestedIn: []string{"man"}, Bio: "Hello"})
	b := e.newPerson(persona{Name: "Ben", Gender: "man", InterestedIn: []string{"woman"}})

	r := e.expect(e.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil), http.StatusOK)
	body := strings.ToLower(string(r.Body))
	for _, forbidden := range []string{"latitude", "longitude", "email", "birthdate", a.Email, "passwordhash"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public profile leaks %q: %s", forbidden, r.Body)
		}
	}
	dist := r.json(t)["distanceKm"].(float64)
	if int(dist)%5 != 0 {
		t.Fatalf("distance must be bucketed by 5 km, got %v", dist)
	}

	// Privacy switches: age and distance can be hidden, discovery can be turned off.
	e.expect(e.do(http.MethodPut, "/profile/privacy", a.Token, map[string]any{"showAge": false, "showDistance": false, "discoverable": false}), http.StatusOK)
	got := e.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil)
	// Not discoverable and no relationship: invisible.
	e.expect(got, http.StatusNotFound)
	if contains(e.discoverIDs(b), a.ID) {
		t.Fatal("a non-discoverable profile must not be shown")
	}
	e.expect(e.do(http.MethodGet, "/profiles/not-a-uuid", b.Token, nil), http.StatusNotFound)

	e.expect(e.do(http.MethodPut, "/profile/privacy", a.Token, map[string]any{"showAge": false, "showDistance": false, "discoverable": true}), http.StatusOK)
	vis := e.expect(e.do(http.MethodGet, "/profiles/"+a.ID, b.Token, nil), http.StatusOK).json(t)
	if _, hasAge := vis["age"]; hasAge {
		t.Fatal("age must be hidden when showAge=false")
	}
	if _, hasDist := vis["distanceKm"]; hasDist {
		t.Fatal("distance must be hidden when showDistance=false")
	}
}

func TestInterestsCatalogAndSelection(t *testing.T) {
	e := newEnv(t)
	u := e.newPerson(persona{Name: "Ivy", Gender: "woman", InterestedIn: []string{"man"}})
	cat := e.expect(e.do(http.MethodGet, "/interests", u.Token, nil), http.StatusOK).json(t)["interests"].([]any)
	if len(cat) < 12 {
		t.Fatalf("catalogue too small: %d", len(cat))
	}
	id := func(i int) int { return int(cat[i].(map[string]any)["id"].(float64)) }

	r := e.expect(e.do(http.MethodPut, "/profile/interests", u.Token, map[string]any{"interestIds": []int{id(0), id(1), id(1)}}), http.StatusOK).json(t)
	if n := len(r["interests"].([]any)); n != 2 {
		t.Fatalf("duplicates must collapse, got %d interests", n)
	}
	e.expect(e.do(http.MethodPut, "/profile/interests", u.Token, map[string]any{"interestIds": []int{30000}}), http.StatusUnprocessableEntity)
	tooMany := make([]int, 11)
	for i := range tooMany {
		tooMany[i] = id(i)
	}
	e.expect(e.do(http.MethodPut, "/profile/interests", u.Token, map[string]any{"interestIds": tooMany}), http.StatusUnprocessableEntity)
}
