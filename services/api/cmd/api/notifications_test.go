package main

import "testing"

func TestNotificationsReadFlowAndPagination(t *testing.T) {
	e := newEnv(t)
	me := e.newUser(userSpec{Name: "Zoe", Gender: "woman", InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng, AgeYears: 29})
	var firstMatch string
	for i := 0; i < 3; i++ {
		other := e.newUser(userSpec{Name: "Man", Gender: "man", InterestedIn: []string{"woman"}, Lat: nearLat, Lng: nearLng, AgeYears: 31})
		id := match(e, other, me)
		if i == 0 {
			firstMatch = id
		}
	}

	page1 := e.expect(e.do("GET", "/notifications?limit=2", me.Token, nil), 200, "page 1")
	if len(page1.list("notifications")) != 2 || page1.str("nextCursor") == "" || page1.JSON["unreadCount"].(float64) != 3 {
		t.Fatalf("unexpected first page: %s", page1.Raw)
	}
	page2 := e.expect(e.do("GET", "/notifications?limit=2&cursor="+page1.str("nextCursor"), me.Token, nil), 200, "page 2")
	if len(page2.list("notifications")) != 1 || page2.str("nextCursor") != "" {
		t.Fatalf("unexpected second page: %s", page2.Raw)
	}
	if actor, _ := page1.list("notifications")[0]["actor"].(map[string]any); actor["firstName"] != "Man" || actor["photoUrl"] == nil {
		t.Fatalf("actor summary missing: %v", page1.list("notifications")[0])
	}
	e.expect(e.do("GET", "/notifications?cursor=garbage", me.Token, nil), 400, "bad cursor")
	e.expect(e.do("GET", "/notifications?limit=0", me.Token, nil), 400, "bad limit")

	first := page1.list("notifications")[0]["id"].(string)
	e.expect(e.do("POST", "/notifications/"+first+"/read", me.Token, nil), 204, "mark one")
	e.expect(e.do("POST", "/notifications/"+first+"/read", me.Token, nil), 204, "mark twice is fine")
	e.expect(e.do("POST", "/notifications/00000000-0000-0000-0000-000000000000/read", me.Token, nil), 404, "unknown")
	if got := e.expect(e.do("GET", "/notifications/unread-count", me.Token, nil), 200, "count").JSON["unreadCount"].(float64); got != 2 {
		t.Fatalf("expected 2 unread, got %v", got)
	}

	// users cannot mark someone else's notification
	stranger := e.register(uniqueEmail("stranger"))
	e.expect(e.do("POST", "/notifications/"+first+"/read", stranger.Token, nil), 404, "foreign notification")

	e.expect(e.do("POST", "/notifications/read-all", me.Token, nil), 204, "read all")
	if got := e.expect(e.do("GET", "/notifications/unread-count", me.Token, nil), 200, "count").JSON["unreadCount"].(float64); got != 0 {
		t.Fatalf("expected 0 unread, got %v", got)
	}

	// unmatching removes the related notification
	e.expect(e.do("DELETE", "/matches/"+firstMatch, me.Token, nil), 204, "unmatch")
	if n := len(e.expect(e.do("GET", "/notifications", me.Token, nil), 200, "list").list("notifications")); n != 2 {
		t.Fatalf("expected 2 notifications after unmatch, got %d", n)
	}
}

func TestConversationListPagination(t *testing.T) {
	e := newEnv(t)
	me := e.newUser(userSpec{Name: "Mia", Gender: "woman", InterestedIn: []string{"man"}, Lat: parisLat, Lng: parisLng, AgeYears: 29})
	var last string
	for i := 0; i < 3; i++ {
		other := e.newUser(userSpec{Name: "Lou", Gender: "man", InterestedIn: []string{"woman"}, Lat: nearLat, Lng: nearLng, AgeYears: 31})
		last = match(e, other, me)
	}
	p1 := e.expect(e.do("GET", "/conversations?limit=2", me.Token, nil), 200, "page1")
	if len(p1.list("conversations")) != 2 || p1.str("nextCursor") == "" || p1.list("conversations")[0]["matchId"] != last {
		t.Fatalf("unexpected page 1 (most recent first): %s", p1.Raw)
	}
	p2 := e.expect(e.do("GET", "/conversations?limit=2&cursor="+p1.str("nextCursor"), me.Token, nil), 200, "page2")
	if len(p2.list("conversations")) != 1 || p2.str("nextCursor") != "" {
		t.Fatalf("unexpected page 2: %s", p2.Raw)
	}
	e.expect(e.do("GET", "/conversations?cursor=garbage", me.Token, nil), 400, "bad cursor")
}
