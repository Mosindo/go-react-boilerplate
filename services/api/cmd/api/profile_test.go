package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func profilePayload(overrides map[string]any) map[string]any {
	p := map[string]any{
		"firstName": "Camille", "birthDate": birthDateForAge(27), "gender": "woman",
		"bio": "Salut", "city": "Lyon", "latitude": 45.7640012, "longitude": 4.8356999,
		"interests": []string{"travel", "cooking"},
	}
	for k, v := range overrides {
		p[k] = v
	}
	return p
}

func TestOnboardingProgressAndProfileValidation(t *testing.T) {
	e := newEnv(t)
	u := e.register("")

	e.expect(e.do("GET", "/me/profile", nil, u.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/discover", nil, u.Token), http.StatusForbidden)

	invalid := map[string]map[string]any{
		"underage":        {"birthDate": time.Now().UTC().AddDate(-17, 0, 0).Format("2006-01-02")},
		"turning 18 soon": {"birthDate": time.Now().UTC().AddDate(-18, 0, 1).Format("2006-01-02")},
		"ancient":         {"birthDate": "1900-01-01"},
		"bad date":        {"birthDate": "12/03/1990"},
		"future":          {"birthDate": time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02")},
		"gender":          {"gender": "robot"},
		"no name":         {"firstName": "   "},
		"digits in name":  {"firstName": "R2D2"},
		"long bio":        {"bio": strings.Repeat("a", 501)},
		"bad latitude":    {"latitude": 91},
		"unknown tag":     {"interests": []string{"definitely-not-an-interest"}},
		"too many tags":   {"interests": []string{"travel", "hiking", "cooking", "coffee", "wine", "restaurants", "movies", "series", "reading", "music", "concerts"}},
	}
	for name, override := range invalid {
		if r := e.do("PUT", "/me/profile", profilePayload(override), u.Token); r.Status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d (%s)", name, r.Status, r.Body)
		}
	}
	missingCoords := profilePayload(nil)
	delete(missingCoords, "latitude")
	e.expect(e.do("PUT", "/me/profile", missingCoords, u.Token), http.StatusBadRequest)

	// Valid profile: coordinates are stored coarsely and the age is computed.
	p := e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"firstName": "  Anne-Laure  "}), u.Token), http.StatusOK).m(t)
	if p["firstName"] != "Anne-Laure" || p["age"].(float64) != 27 {
		t.Fatalf("unexpected profile %v", p)
	}
	if p["latitude"].(float64) != 45.76 || p["longitude"].(float64) != 4.84 {
		t.Fatalf("coordinates must be rounded to 0.01 degree, got %v,%v", p["latitude"], p["longitude"])
	}
	if p["discoverable"] != true || p["showDistance"] != true {
		t.Fatalf("privacy defaults: %v", p)
	}
	if got := len(p["interests"].([]any)); got != 2 {
		t.Fatalf("expected 2 interests, got %d", got)
	}
	// Updating replaces the interests and keeps unspecified privacy flags.
	e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"interests": []string{"music"}, "discoverable": false}), u.Token), http.StatusOK)
	p = e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"interests": []string{"music"}}), u.Token), http.StatusOK).m(t)
	if len(p["interests"].([]any)) != 1 || p["discoverable"] != false {
		t.Fatalf("update should replace interests and keep privacy flags: %v", p)
	}

	// Preferences.
	defaults := e.expect(e.do("GET", "/me/preferences", nil, u.Token), http.StatusOK).m(t)
	if len(defaults["interestedIn"].([]any)) != 0 {
		t.Fatalf("unexpected default preferences %v", defaults)
	}
	for name, bad := range map[string]map[string]any{
		"none":     {"interestedIn": []string{}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 20},
		"gender":   {"interestedIn": []string{"x"}, "minAge": 18, "maxAge": 40, "maxDistanceKm": 20},
		"minor":    {"interestedIn": []string{"man"}, "minAge": 17, "maxAge": 40, "maxDistanceKm": 20},
		"inverted": {"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30, "maxDistanceKm": 20},
		"distance": {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 30, "maxDistanceKm": 0},
		"far":      {"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 30, "maxDistanceKm": 501},
	} {
		if r := e.do("PUT", "/me/preferences", bad, u.Token); r.Status != http.StatusBadRequest {
			t.Errorf("preferences %s: expected 400, got %d", name, r.Status)
		}
	}
	e.expect(e.do("PUT", "/me/preferences", map[string]any{"interestedIn": []string{"man", "man", "woman"}, "minAge": 20, "maxAge": 35, "maxDistanceKm": 30}, u.Token), http.StatusOK)

	// Onboarding completes once the first photo is uploaded.
	me := e.expect(e.do("GET", "/me", nil, u.Token), http.StatusOK).m(t)
	if me["onboarding"].(map[string]any)["complete"] != false {
		t.Fatal("onboarding should still need a photo")
	}
	e.expect(e.upload("POST", "/me/photos", u.Token, "a.jpg", testJPEG(300, 300, 1)), http.StatusCreated)
	me = e.expect(e.do("GET", "/me", nil, u.Token), http.StatusOK).m(t)
	if me["onboarding"].(map[string]any)["complete"] != true {
		t.Fatalf("onboarding should be complete: %v", me)
	}
}

func TestInterestCatalogue(t *testing.T) {
	e := newEnv(t)
	u := e.register("")
	res := e.expect(e.do("GET", "/interests", nil, u.Token), http.StatusOK).m(t)
	if len(res["interests"].([]any)) < 20 {
		t.Fatalf("catalogue too small: %v", res)
	}
	e.expect(e.do("GET", "/interests", nil, ""), http.StatusUnauthorized)
}

func TestPublicProfileNeverLeaksPrivateData(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()
	r := e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), http.StatusOK)
	body := string(r.Body)
	for _, secret := range []string{"email", "birthDate", "latitude", "longitude", b.Email} {
		if strings.Contains(body, secret) {
			t.Fatalf("public profile leaks %q: %s", secret, body)
		}
	}
	p := r.m(t)
	if p["distanceKm"] == nil || p["distanceKm"].(float64) > 1 {
		t.Fatalf("expected the coarse 1 km bucket for neighbours, got %v", p["distanceKm"])
	}

	// Hidden distance and hidden profile.
	e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"showDistance": false, "gender": "man", "firstName": "Bob", "latitude": 0.0, "longitude": 0.0}), b.Token), http.StatusOK)
	if p := e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), http.StatusOK).m(t); p["distanceKm"] != nil {
		t.Fatalf("distance must be hidden: %v", p["distanceKm"])
	}
	e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"discoverable": false, "gender": "man", "firstName": "Bob"}), b.Token), http.StatusOK)
	e.expect(e.do("GET", "/profiles/"+b.ID, nil, a.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/profiles/"+a.ID, nil, a.Token), http.StatusOK) // own profile stays visible
	e.expect(e.do("GET", "/profiles/not-a-uuid", nil, a.Token), http.StatusNotFound)
	e.expect(e.do("GET", fmt.Sprintf("/profiles/%s", "00000000-0000-0000-0000-000000000000"), nil, a.Token), http.StatusNotFound)
}
