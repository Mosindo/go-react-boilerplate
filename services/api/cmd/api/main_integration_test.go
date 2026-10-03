package main

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCriticalJourney walks the path every user takes:
// register → profile → photo → discover → like → match → conversation.
func TestCriticalJourney(t *testing.T) {
	e := newEnv(t, false)
	lat, lng := scenarioOrigin()

	alice := e.register("alice")
	bob := e.register("bob")

	// Before the profile exists, discovery is refused with a clear code.
	r := e.expect(e.do(http.MethodGet, "/discover", alice.Token, nil), http.StatusConflict)
	if !strings.Contains(string(r.Body), "profile_incomplete") {
		t.Fatalf("expected profile_incomplete, got %s", r.Body)
	}

	e.makeProfile(alice, profileSpec{Name: "Alice", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	e.makeProfile(bob, profileSpec{Name: "Bob", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat + 0.05, Lng: lng})
	// A profile without a photo is not yet discoverable and cannot discover.
	e.expect(e.do(http.MethodGet, "/discover", alice.Token, nil), http.StatusConflict)
	e.addPhoto(alice)
	e.addPhoto(bob)

	if ids := idsOf(e.expect(e.do(http.MethodGet, "/discover", alice.Token, nil), http.StatusOK), t); len(ids) != 1 || ids[0] != bob.ID {
		t.Fatalf("alice should see exactly bob, got %v", ids)
	}

	var first struct {
		Matched bool `json:"matched"`
	}
	e.expect(e.do(http.MethodPost, "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), http.StatusOK).json(t, &first)
	if first.Matched {
		t.Fatal("a one-sided like must not match")
	}
	// Repeating the like is idempotent.
	e.expect(e.do(http.MethodPost, "/swipes", alice.Token, map[string]string{"targetId": bob.ID, "action": "like"}), http.StatusOK).json(t, &first)
	if first.Matched {
		t.Fatal("a repeated like must not create a match")
	}
	// Bob sees Alice, who already liked him; Alice no longer sees Bob.
	if ids := idsOf(e.do(http.MethodGet, "/discover", bob.Token, nil), t); len(ids) != 1 || ids[0] != alice.ID {
		t.Fatalf("bob should see alice, got %v", ids)
	}
	if ids := idsOf(e.do(http.MethodGet, "/discover", alice.Token, nil), t); len(ids) != 0 {
		t.Fatalf("alice already swiped bob, got %v", ids)
	}

	var second struct {
		Matched bool   `json:"matched"`
		MatchID string `json:"matchId"`
		Profile struct {
			FirstName string `json:"firstName"`
		} `json:"profile"`
	}
	e.expect(e.do(http.MethodPost, "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), http.StatusOK).json(t, &second)
	if !second.Matched || second.MatchID == "" || second.Profile.FirstName != "Alice" {
		t.Fatalf("expected a match with Alice, got %+v", second)
	}
	if ids := idsOf(e.do(http.MethodGet, "/discover", bob.Token, nil), t); len(ids) != 0 {
		t.Fatalf("bob already swiped alice, got %v", ids)
	}
	// Replaying the swipe returns the same match instead of a new one.
	var replay struct {
		MatchID string `json:"matchId"`
		Matched bool   `json:"matched"`
	}
	e.expect(e.do(http.MethodPost, "/swipes", bob.Token, map[string]string{"targetId": alice.ID, "action": "like"}), http.StatusOK).json(t, &replay)
	if !replay.Matched || replay.MatchID != second.MatchID {
		t.Fatalf("replay should return the existing match: %+v", replay)
	}
	var matchRows int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, alice.ID, bob.ID).Scan(&matchRows)
	if matchRows != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchRows)
	}

	// Both users were notified.
	for _, u := range []testUser{alice, bob} {
		var n struct {
			Notifications []struct {
				Type string `json:"type"`
			} `json:"notifications"`
			UnreadCount int `json:"unreadCount"`
		}
		e.expect(e.do(http.MethodGet, "/notifications", u.Token, nil), http.StatusOK).json(t, &n)
		if len(n.Notifications) != 1 || n.Notifications[0].Type != "match" || n.UnreadCount != 1 {
			t.Fatalf("expected one unread match notification, got %+v", n)
		}
	}

	// Conversation.
	e.expect(e.do(http.MethodPost, "/conversations/"+second.MatchID+"/messages", alice.Token, map[string]string{"body": "  Hi Bob!  "}), http.StatusCreated)
	var list struct {
		Conversations []struct {
			MatchID     string `json:"matchId"`
			UnreadCount int    `json:"unreadCount"`
			User        struct {
				ID        string `json:"id"`
				FirstName string `json:"firstName"`
			} `json:"user"`
			LastMessage *struct {
				Body string `json:"body"`
			} `json:"lastMessage"`
		} `json:"conversations"`
	}
	e.expect(e.do(http.MethodGet, "/conversations", bob.Token, nil), http.StatusOK).json(t, &list)
	if len(list.Conversations) != 1 || list.Conversations[0].UnreadCount != 1 || list.Conversations[0].User.ID != alice.ID ||
		list.Conversations[0].LastMessage == nil || list.Conversations[0].LastMessage.Body != "Hi Bob!" {
		t.Fatalf("unexpected bob conversation list: %+v", list)
	}

	var msgs struct {
		Messages []struct {
			Body   string  `json:"body"`
			ReadAt *string `json:"readAt"`
		} `json:"messages"`
	}
	e.expect(e.do(http.MethodGet, "/conversations/"+second.MatchID+"/messages", bob.Token, nil), http.StatusOK).json(t, &msgs)
	if len(msgs.Messages) != 1 || msgs.Messages[0].ReadAt != nil {
		t.Fatalf("expected one unread message, got %+v", msgs)
	}
	e.expect(e.do(http.MethodPost, "/conversations/"+second.MatchID+"/read", bob.Token, nil), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, "/conversations/"+second.MatchID+"/messages", alice.Token, nil), http.StatusOK).json(t, &msgs)
	if msgs.Messages[0].ReadAt == nil {
		t.Fatal("alice should see the message as read")
	}
	e.expect(e.do(http.MethodGet, "/conversations", bob.Token, nil), http.StatusOK).json(t, &list)
	if list.Conversations[0].UnreadCount != 0 {
		t.Fatal("unread counter should be cleared")
	}
}

func TestProfileValidationAndPrivacy(t *testing.T) {
	e := newEnv(t, false)
	lat, lng := scenarioOrigin()
	u := e.register("prof")
	valid := func() map[string]any {
		return map[string]any{
			"firstName": "Sam", "birthDate": time.Now().AddDate(-25, 0, 0).Format("2006-01-02"), "gender": "nonbinary",
			"bio": "Climber", "city": "Grenoble", "interests": []string{"hiking"}, "latitude": lat + 0.123456, "longitude": lng + 0.654321,
		}
	}
	mutate := func(k string, v any) map[string]any { m := valid(); m[k] = v; return m }

	for name, body := range map[string]map[string]any{
		"underage":        mutate("birthDate", time.Now().AddDate(-17, 0, 0).Format("2006-01-02")),
		"future":          mutate("birthDate", time.Now().AddDate(1, 0, 0).Format("2006-01-02")),
		"ancient":         mutate("birthDate", "1900-01-01"),
		"bad date":        mutate("birthDate", "31/12/1990"),
		"bad gender":      mutate("gender", "robot"),
		"empty name":      mutate("firstName", "   "),
		"long name":       mutate("firstName", strings.Repeat("a", 41)),
		"control chars":   mutate("firstName", "Sam\x00"),
		"long bio":        mutate("bio", strings.Repeat("b", 501)),
		"unknown":         mutate("interests", []string{"quidditch"}),
		"too many":        mutate("interests", []string{"travel", "cooking", "hiking", "music", "cinema", "reading", "sports", "yoga", "photography", "gaming", "art"}),
		"bad coordinates": mutate("latitude", 123.0),
	} {
		if r := e.do(http.MethodPut, "/me/profile", u.Token, body); r.Status != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d (%s)", name, r.Status, r.Body)
		}
	}
	e.expect(e.do(http.MethodGet, "/me/profile", u.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodPut, "/me/profile", u.Token, valid()), http.StatusOK)

	var own struct {
		BirthDate    string `json:"birthDate"`
		HasLocation  bool   `json:"hasLocation"`
		Discoverable bool   `json:"discoverable"`
	}
	e.expect(e.do(http.MethodGet, "/me/profile", u.Token, nil), http.StatusOK).json(t, &own)
	if own.BirthDate == "" || !own.HasLocation || own.Discoverable {
		t.Fatalf("unexpected own profile: %+v", own)
	}

	// Coordinates are stored at ~1 km precision only.
	var storedLat float64
	_ = e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id = $1`, u.ID).Scan(&storedLat)
	if diff := storedLat*100 - float64(int64(storedLat*100+0.5*sign(storedLat))); diff > 1e-6 || diff < -1e-6 {
		t.Fatalf("latitude was not rounded: %v", storedLat)
	}

	// Preferences validation.
	for name, body := range map[string]map[string]any{
		"min>max":     {"interestedIn": []string{"man"}, "minAge": 40, "maxAge": 30},
		"under 18":    {"interestedIn": []string{"man"}, "minAge": 16, "maxAge": 30},
		"bad gender":  {"interestedIn": []string{"alien"}, "minAge": 20, "maxAge": 30},
		"bad dist":    {"interestedIn": []string{"man"}, "minAge": 20, "maxAge": 30, "maxDistanceKm": 9999},
		"empty array": {"interestedIn": []string{}, "minAge": 20, "maxAge": 30},
	} {
		if r := e.do(http.MethodPut, "/me/preferences", u.Token, body); r.Status != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d (%s)", name, r.Status, r.Body)
		}
	}

	// Another user sees a public view with no birth date, email or coordinates.
	viewer := e.completeUser("viewer", profileSpec{Name: "Vic", Gender: "man", InterestedIn: []string{"nonbinary"}, Lat: lat, Lng: lng})
	e.addPhoto(u)
	r := e.expect(e.do(http.MethodGet, "/profiles/"+u.ID, viewer.Token, nil), http.StatusOK)
	for _, forbidden := range []string{"birthDate", "email", "latitude", "longitude", "lat", "lng", "isVisible"} {
		if strings.Contains(string(r.Body), `"`+forbidden+`"`) {
			t.Fatalf("public profile leaks %q: %s", forbidden, r.Body)
		}
	}
	var pub struct {
		Age        int  `json:"age"`
		DistanceKm *int `json:"distanceKm"`
	}
	r.json(t, &pub)
	if pub.Age != 25 || pub.DistanceKm == nil || *pub.DistanceKm < 1 {
		t.Fatalf("unexpected public profile: %s", r.Body)
	}

	// Hiding the location from others removes the distance; going invisible removes the profile.
	e.expect(e.do(http.MethodPut, "/me/privacy", u.Token, map[string]any{"isVisible": true, "showDistance": false}), http.StatusNoContent)
	r = e.expect(e.do(http.MethodGet, "/profiles/"+u.ID, viewer.Token, nil), http.StatusOK)
	if strings.Contains(string(r.Body), "distanceKm") {
		t.Fatalf("distance must be hidden: %s", r.Body)
	}
	e.expect(e.do(http.MethodPut, "/me/privacy", u.Token, map[string]any{"isVisible": false, "showDistance": false}), http.StatusNoContent)
	e.expect(e.do(http.MethodGet, "/profiles/"+u.ID, viewer.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/not-a-uuid", viewer.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/profiles/"+u.ID, "", nil), http.StatusUnauthorized)
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

func TestDiscoveryFiltersAndPass(t *testing.T) {
	e := newEnv(t, false)
	lat, lng := scenarioOrigin()
	me := e.completeUser("me", profileSpec{Name: "Me", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng, Age: 30})

	match := e.completeUser("ok", profileSpec{Name: "Ok", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat + 0.1, Lng: lng, Age: 28})
	wrongGender := e.completeUser("gender", profileSpec{Name: "Gender", Gender: "man", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	notInterested := e.completeUser("notint", profileSpec{Name: "NotInt", Gender: "woman", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})
	tooFar := e.completeUser("far", profileSpec{Name: "Far", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat + 3, Lng: lng})
	tooYoung := e.completeUser("young", profileSpec{Name: "Young", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng, Age: 19})
	noPhoto := e.register("nophoto")
	e.makeProfile(noPhoto, profileSpec{Name: "NoPhoto", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	hidden := e.completeUser("hidden", profileSpec{Name: "Hidden", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
	e.expect(e.do(http.MethodPut, "/me/privacy", hidden.Token, map[string]any{"isVisible": false, "showDistance": true}), http.StatusNoContent)

	// Narrow my age window so the 19-year-old drops out.
	e.expect(e.do(http.MethodPut, "/me/preferences", me.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": 50}), http.StatusOK)

	ids := idsOf(e.expect(e.do(http.MethodGet, "/discover?limit=20", me.Token, nil), http.StatusOK), t)
	if len(ids) != 1 || ids[0] != match.ID {
		t.Fatalf("expected only the eligible profile, got %v (wrongGender=%s notInterested=%s tooFar=%s tooYoung=%s hidden=%s)",
			ids, wrongGender.ID, notInterested.ID, tooFar.ID, tooYoung.ID, hidden.ID)
	}

	// "Anywhere" brings in the far profile but still not the others.
	e.expect(e.do(http.MethodPut, "/me/preferences", me.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 25, "maxAge": 40, "maxDistanceKm": nil}), http.StatusOK)
	ids = idsOf(e.do(http.MethodGet, "/discover?limit=20", me.Token, nil), t)
	if len(ids) < 2 || !contains(ids, tooFar.ID) || contains(ids, tooYoung.ID) || contains(ids, noPhoto.ID) {
		t.Fatalf("unexpected anywhere result: %v", ids)
	}
	// Closest first.
	if ids[0] != match.ID {
		t.Fatalf("closest profile should rank first: %v", ids)
	}

	// The server rejects swipes the client should never have been able to make.
	e.expect(e.do(http.MethodPost, "/swipes", me.Token, map[string]string{"targetId": wrongGender.ID, "action": "like"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/swipes", me.Token, map[string]string{"targetId": hidden.ID, "action": "like"}), http.StatusNotFound)
	e.expect(e.do(http.MethodPost, "/swipes", me.Token, map[string]string{"targetId": me.ID, "action": "like"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", me.Token, map[string]string{"targetId": match.ID, "action": "super"}), http.StatusBadRequest)
	e.expect(e.do(http.MethodPost, "/swipes", me.Token, map[string]string{"targetId": "nope", "action": "like"}), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/discover?limit=999", me.Token, nil), http.StatusBadRequest)

	// A pass hides the profile for good and never produces a match.
	var res struct {
		Matched bool `json:"matched"`
	}
	e.expect(e.do(http.MethodPost, "/swipes", me.Token, map[string]string{"targetId": match.ID, "action": "pass"}), http.StatusOK).json(t, &res)
	if res.Matched {
		t.Fatal("pass cannot match")
	}
	if ids := idsOf(e.do(http.MethodGet, "/discover?limit=20", me.Token, nil), t); contains(ids, match.ID) {
		t.Fatal("passed profile reappeared")
	}
	// Even if she likes me afterwards, my earlier pass is final: no match.
	e.expect(e.do(http.MethodPost, "/swipes", match.Token, map[string]string{"targetId": me.ID, "action": "like"}), http.StatusOK).json(t, &res)
	if res.Matched {
		t.Fatal("a pass must not turn into a match")
	}
}

func TestSimultaneousLikesProduceOneMatch(t *testing.T) {
	e := newEnv(t, false)
	for round := 0; round < 5; round++ {
		lat, lng := scenarioOrigin()
		a := e.completeUser("race-a", profileSpec{Name: "A", Gender: "woman", InterestedIn: []string{"man"}, Lat: lat, Lng: lng})
		b := e.completeUser("race-b", profileSpec{Name: "B", Gender: "man", InterestedIn: []string{"woman"}, Lat: lat, Lng: lng})

		var wg sync.WaitGroup
		matched := make([]bool, 2)
		for i, pair := range [][2]testUser{{a, b}, {b, a}} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var out struct {
					Matched bool `json:"matched"`
				}
				r := e.do(http.MethodPost, "/swipes", pair[0].Token, map[string]string{"targetId": pair[1].ID, "action": "like"})
				if r.Status == http.StatusOK {
					r.json(t, &out)
				}
				matched[i] = out.Matched
			}()
		}
		wg.Wait()
		if !matched[0] && !matched[1] {
			t.Fatalf("round %d: mutual likes were lost", round)
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM matches WHERE user_a IN ($1,$2) AND user_b IN ($1,$2)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: expected exactly one match, got %d", round, n)
		}
	}
}
