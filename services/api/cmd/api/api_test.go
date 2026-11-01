package main

import (
	"context"
	"image/color"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const (
	parisLat, parisLng = 48.8566, 2.3522
	lyonLat, lyonLng   = 45.7640, 4.8357
	marsLat, marsLng   = 43.2965, 5.3698
)

func TestHealth(t *testing.T) {
	e := newTestEnv(t)
	if r := e.do("GET", "/health", "", nil); r.Status != http.StatusOK {
		t.Fatalf("health: %d", r.Status)
	}
}

func TestAuthFlow(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("auth")

	if r := e.do("GET", "/me", "", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("me without token: %d", r.Status)
	}
	if r := e.do("GET", "/me", "not-a-jwt", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("me bad token: %d", r.Status)
	}
	me := e.do("GET", "/me", u.Token, nil)
	if me.Status != 200 || me.str("email") != u.Email || me.Body["profileComplete"] != false {
		t.Fatalf("me: %d %s", me.Status, me.Raw)
	}
	if r := e.do("POST", "/auth/register", "", map[string]any{"email": u.Email, "password": "Passw0rd!x"}); r.Status != http.StatusConflict {
		t.Fatalf("duplicate register: %d", r.Status)
	}
	if r := e.do("POST", "/auth/register", "", map[string]any{"email": "weak@test.example", "password": "alllettersnodigits"}); r.Status != http.StatusBadRequest {
		t.Fatalf("weak password: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": u.Email, "password": "WrongPass1"}); r.Status != http.StatusUnauthorized {
		t.Fatalf("bad login: %d", r.Status)
	}
	login := e.do("POST", "/auth/login", "", map[string]any{"email": "  " + u.Email + " ", "password": "Passw0rd!x"})
	if login.Status != 200 {
		t.Fatalf("login: %d %s", login.Status, login.Raw)
	}
	// Rotation du refresh token: l'ancien devient invalide.
	ref := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": login.str("refreshToken")})
	if ref.Status != 200 || ref.str("refreshToken") == login.str("refreshToken") {
		t.Fatalf("refresh: %d", ref.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": login.str("refreshToken")}); r.Status != http.StatusUnauthorized {
		t.Fatalf("reused refresh token must fail: %d", r.Status)
	}
	if r := e.do("POST", "/auth/logout", "", map[string]any{"refreshToken": ref.str("refreshToken")}); r.Status != http.StatusNoContent {
		t.Fatalf("logout: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": ref.str("refreshToken")}); r.Status != http.StatusUnauthorized {
		t.Fatalf("refresh after logout: %d", r.Status)
	}
}

func TestPasswordReset(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("reset")
	// Reponse identique pour un email inconnu (pas d'enumeration).
	if r := e.do("POST", "/auth/password/forgot", "", map[string]any{"email": "nobody@test.example"}); r.Status != http.StatusNoContent {
		t.Fatalf("forgot unknown: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/forgot", "", map[string]any{"email": u.Email}); r.Status != http.StatusNoContent {
		t.Fatalf("forgot: %d", r.Status)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]any{"token": "bogus", "password": "NewPassw0rd"}); r.Status != http.StatusBadRequest {
		t.Fatalf("bogus reset token: %d", r.Status)
	}
	// Insere un token connu (le vrai est envoye par email et seul son hash est stocke).
	known := "known-reset-token-for-test"
	if _, err := e.pool.Exec(context.Background(),
		`INSERT INTO password_resets (user_id, token_hash, expires_at) VALUES ($1, encode(digest($2,'sha256'),'hex'), NOW() + INTERVAL '1 hour')`,
		u.ID, known); err != nil {
		t.Fatal(err)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]any{"token": known, "password": "NewPassw0rd"}); r.Status != http.StatusNoContent {
		t.Fatalf("reset: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/auth/password/reset", "", map[string]any{"token": known, "password": "Another1Pass"}); r.Status != http.StatusBadRequest {
		t.Fatalf("token reuse must fail: %d", r.Status)
	}
	if r := e.do("POST", "/auth/refresh", "", map[string]any{"refreshToken": u.Refresh}); r.Status != http.StatusUnauthorized {
		t.Fatalf("sessions must be revoked after reset: %d", r.Status)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": u.Email, "password": "NewPassw0rd"}); r.Status != 200 {
		t.Fatalf("login with new password: %d", r.Status)
	}
}

func TestProfileValidation(t *testing.T) {
	e := newTestEnv(t)
	u := e.register("val")
	base := func() map[string]any {
		return map[string]any{"firstName": "Zoe", "birthDate": "1995-05-05", "gender": "woman", "bio": "", "city": "Paris",
			"latitude": parisLat, "longitude": parisLng, "interests": []string{"music"}}
	}
	cases := []struct {
		name   string
		mutate func(m map[string]any)
		want   int
	}{
		{"ok", func(m map[string]any) {}, 200},
		{"underage", func(m map[string]any) { m["birthDate"] = "2020-01-01" }, 403},
		{"bad date", func(m map[string]any) { m["birthDate"] = "05/05/1995" }, 400},
		{"bad gender", func(m map[string]any) { m["gender"] = "x" }, 400},
		{"empty name", func(m map[string]any) { m["firstName"] = "   " }, 400},
		{"bad latitude", func(m map[string]any) { m["latitude"] = 123.0 }, 400},
		{"unknown interest", func(m map[string]any) { m["interests"] = []string{"does-not-exist"} }, 400},
		{"bio too long", func(m map[string]any) { m["bio"] = string(make([]rune, 501)) }, 400},
	}
	for _, tc := range cases {
		m := base()
		tc.mutate(m)
		if tc.name == "bio too long" {
			long := make([]byte, 501)
			for i := range long {
				long[i] = 'a'
			}
			m["bio"] = string(long)
		}
		if r := e.do("PUT", "/me/profile", u.Token, m); r.Status != tc.want {
			t.Errorf("%s: got %d want %d (%s)", tc.name, r.Status, tc.want, r.Raw)
		}
	}
	// Les coordonnees sont arrondies (~1 km) avant stockage.
	var lat float64
	if err := e.pool.QueryRow(context.Background(), `SELECT latitude FROM profiles WHERE user_id=$1`, u.ID).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat != 48.86 {
		t.Fatalf("expected rounded latitude 48.86, got %v", lat)
	}
	// Reglages invalides.
	bad := map[string]any{"interestedIn": []string{"man"}, "minAge": 30, "maxAge": 20, "maxDistanceKm": 10, "discoverable": true, "showDistance": true}
	if r := e.do("PUT", "/me/settings", u.Token, bad); r.Status != 400 {
		t.Fatalf("settings min>max: %d", r.Status)
	}
	// Sans profil, les reglages sont refuses (409).
	u2 := e.register("noprof")
	if r := e.do("GET", "/me/settings", u2.Token, nil); r.Status != http.StatusConflict {
		t.Fatalf("settings without profile: %d", r.Status)
	}
}

func TestPhotos(t *testing.T) {
	e := newTestEnv(t)
	a := e.register("pa")
	b := e.register("pb")
	e.profile(a, "Alice", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
	e.profile(b, "Bob", "man", []string{"woman"}, "1992-02-02", parisLat, parisLng)

	if r := e.upload(a.Token, []byte("not an image at all"), ""); r.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("non-image upload: %d", r.Status)
	}
	if r := e.upload(a.Token, testPNG(50, 50, color.RGBA{1, 2, 3, 255}), ""); r.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("tiny image: %d", r.Status)
	}
	if r := e.upload(a.Token, append([]byte("<?php system($_GET['c']); ?>"), testPNG(300, 300, color.RGBA{1, 2, 3, 255})...), ""); r.Status != http.StatusUnsupportedMediaType {
		t.Fatalf("polyglot upload must be rejected: %d", r.Status)
	}

	// Jusqu'a 6 photos, la 7e est refusee.
	for i := 0; i < 5; i++ {
		if r := e.upload(a.Token, testPNG(1500, 900, color.RGBA{uint8(i * 40), 90, 200, 255}), ""); r.Status != http.StatusCreated {
			t.Fatalf("upload %d: %d %s", i, r.Status, r.Raw)
		}
	}
	if r := e.upload(a.Token, testPNG(300, 300, color.RGBA{9, 9, 9, 255}), ""); r.Status != http.StatusConflict {
		t.Fatalf("7th photo must be refused: %d", r.Status)
	}
	list := e.do("GET", "/me/photos", a.Token, nil)
	items, _ := list.Body["items"].([]any)
	if len(items) != 6 {
		t.Fatalf("expected 6 photos, got %d", len(items))
	}
	first := items[0].(map[string]any)
	if w := first["width"].(float64); w > 1080 {
		t.Fatalf("photo not resized: %v", w)
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.(map[string]any)["id"].(string)
	}

	// Photo principale + ordre.
	if r := e.do("PUT", "/me/photos/"+ids[3]+"/primary", a.Token, nil); r.Status != 200 {
		t.Fatalf("primary: %d %s", r.Status, r.Raw)
	}
	again := e.do("GET", "/me/photos", a.Token, nil).Body["items"].([]any)
	if again[0].(map[string]any)["id"] != ids[3] {
		t.Fatalf("primary not first")
	}
	if r := e.do("PUT", "/me/photos/order", a.Token, map[string]any{"ids": ids[:2]}); r.Status != 400 {
		t.Fatalf("partial order must be rejected: %d", r.Status)
	}
	bPhotos := e.do("GET", "/me/photos", b.Token, nil).Body["items"].([]any)
	bID := bPhotos[0].(map[string]any)["id"].(string)
	swapped := append([]string{bID}, ids[1:]...)
	if r := e.do("PUT", "/me/photos/order", a.Token, map[string]any{"ids": swapped}); r.Status != 400 {
		t.Fatalf("foreign photo id in order must be rejected: %d", r.Status)
	}

	// Remplacement conserve la position; suppression compacte.
	if r := e.upload(a.Token, testPNG(400, 400, color.RGBA{5, 5, 5, 255}), "?replace="+ids[2]); r.Status != http.StatusCreated {
		t.Fatalf("replace: %d %s", r.Status, r.Raw)
	}
	if r := e.do("DELETE", "/me/photos/"+bID, a.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("deleting someone else's photo must 404: %d", r.Status)
	}
	if r := e.do("DELETE", "/me/photos/"+ids[0], a.Token, nil); r.Status != http.StatusNoContent {
		t.Fatalf("delete: %d", r.Status)
	}
	// Acces: proprietaire OK, profil decouvrable OK, non authentifie KO.
	if r := e.do("GET", "/photos/"+ids[3], a.Token, nil); r.Status != 200 || r.Raw[0] != 0xFF {
		t.Fatalf("owner photo fetch: %d", r.Status)
	}
	if r := e.do("GET", "/photos/"+ids[3], "", nil); r.Status != http.StatusUnauthorized {
		t.Fatalf("anonymous photo fetch: %d", r.Status)
	}
	// Un utilisateur qui masque son profil n'expose plus ses photos aux non-matches.
	if r := e.do("PUT", "/me/settings", a.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 50, "discoverable": false, "showDistance": true}); r.Status != 200 {
		t.Fatalf("hide: %d", r.Status)
	}
	if r := e.do("GET", "/photos/"+ids[3], b.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("hidden profile photo must be 404 for others: %d", r.Status)
	}
	if r := e.do("GET", "/profiles/"+a.ID, b.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("hidden profile must be 404 for others: %d", r.Status)
	}
}

func photoIDs(t *testing.T, e *testEnv, tok string) []string {
	t.Helper()
	items := e.do("GET", "/me/photos", tok, nil).Body["items"].([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.(map[string]any)["id"].(string))
	}
	return out
}

func TestDiscoveryMatchingChat(t *testing.T) {
	e := newTestEnv(t)
	alice := e.register("alice")
	bob := e.register("bob")
	carol := e.register("carol") // femme, cherche des hommes: incompatible avec Alice
	dan := e.register("dan")     // trop loin
	eve := e.register("eve")     // intrus pour les tests de permissions

	e.profile(alice, "Alice", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng, "music", "travel")
	e.profile(bob, "Bob", "man", []string{"woman"}, "1990-06-06", lyonLat, lyonLng, "music", "cooking")
	e.profile(carol, "Carol", "woman", []string{"man"}, "1993-01-01", parisLat, parisLng)
	e.profile(dan, "Dan", "man", []string{"woman"}, "1991-01-01", marsLat, marsLng)
	e.profile(eve, "Eve", "woman", []string{"man"}, "1996-01-01", parisLat, parisLng)

	// Alice voit Bob (Lyon ~390 km: limite 100 km => non), regle la distance.
	if r := e.do("PUT", "/me/settings", alice.Token, map[string]any{"interestedIn": []string{"man"}, "minAge": 25, "maxAge": 45, "maxDistanceKm": 500, "discoverable": true, "showDistance": true}); r.Status != 200 {
		t.Fatalf("settings alice: %d", r.Status)
	}
	if r := e.do("PUT", "/me/settings", bob.Token, map[string]any{"interestedIn": []string{"woman"}, "minAge": 18, "maxAge": 99, "maxDistanceKm": 500, "discoverable": true, "showDistance": true}); r.Status != 200 {
		t.Fatalf("settings bob: %d", r.Status)
	}
	feed := e.do("GET", "/discover?limit=10", alice.Token, nil)
	items, _ := feed.Body["items"].([]any)
	if len(items) != 2 { // Bob (Lyon) et Dan (Marseille, 660 km > limite de Dan = 100 km => exclu)
		// Dan limite 100 km: exclu par la regle reciproque. Seul Bob reste.
		if len(items) != 1 {
			t.Fatalf("expected only Bob in Alice's feed, got %d: %s", len(items), feed.Raw)
		}
	}
	card := items[0].(map[string]any)
	if card["userId"] != bob.ID || card["firstName"] != "Bob" {
		t.Fatalf("unexpected first card: %v", card)
	}
	if card["sharedInterests"].(float64) != 1 {
		t.Fatalf("shared interests: %v", card["sharedInterests"])
	}
	// Aucune coordonnee/date de naissance ne fuit; distance arrondie a 5 km.
	for _, forbidden := range []string{"latitude", "longitude", "birthDate", "email"} {
		if _, found := card[forbidden]; found {
			t.Fatalf("card leaks %q", forbidden)
		}
	}
	if d := card["distanceKm"].(float64); int(d)%5 != 0 || d < 300 {
		t.Fatalf("distance should be bucketed: %v", d)
	}

	// Alice ne peut pas swiper une cible incompatible (Carol) ni elle-meme ni un inconnu.
	for _, target := range []string{carol.ID, alice.ID, "00000000-0000-0000-0000-000000000000"} {
		if r := e.do("POST", "/profiles/"+target+"/like", alice.Token, nil); r.Status != http.StatusNotFound {
			t.Fatalf("like of ineligible %s: %d", target, r.Status)
		}
	}
	if r := e.do("POST", "/profiles/not-a-uuid/like", alice.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("bad uuid: %d", r.Status)
	}

	// Like non reciproque: pas de match, pas de doublon.
	r := e.do("POST", "/profiles/"+bob.ID+"/like", alice.Token, nil)
	if r.Status != 200 || r.Body["matched"] != false {
		t.Fatalf("like: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/profiles/"+bob.ID+"/like", alice.Token, nil); r.Status != http.StatusConflict {
		t.Fatalf("repeated like must conflict: %d", r.Status)
	}
	if r := e.do("POST", "/profiles/"+bob.ID+"/pass", alice.Token, nil); r.Status != http.StatusConflict {
		t.Fatalf("pass after like must conflict: %d", r.Status)
	}
	// Bob n'apparait plus dans le flux d'Alice; Alice apparait en tete pour Bob (elle l'a like).
	if after := e.do("GET", "/discover", alice.Token, nil).Body["items"].([]any); len(after) != 0 {
		t.Fatalf("already swiped profile must not reappear")
	}
	bobFeed := e.do("GET", "/discover", bob.Token, nil).Body["items"].([]any)
	if len(bobFeed) != 1 || bobFeed[0].(map[string]any)["userId"] != alice.ID {
		t.Fatalf("bob feed: %v", bobFeed)
	}
	if n := e.do("GET", "/notifications", bob.Token, nil).Body["unreadCount"].(float64); n != 0 {
		t.Fatalf("no notification before match, got %v", n)
	}

	// Like reciproque => match.
	m := e.do("POST", "/profiles/"+alice.ID+"/like", bob.Token, nil)
	if m.Status != 200 || m.Body["matched"] != true || m.str("conversationId") == "" {
		t.Fatalf("expected match: %d %s", m.Status, m.Raw)
	}
	convID := m.str("conversationId")
	matchID := m.str("matchId")
	for _, u := range []user{alice, bob} {
		ml := e.do("GET", "/matches", u.Token, nil).Body["items"].([]any)
		if len(ml) != 1 {
			t.Fatalf("matches for %s: %d", u.Email, len(ml))
		}
		if n := e.do("GET", "/notifications", u.Token, nil).Body["unreadCount"].(float64); n != 1 {
			t.Fatalf("match notification for %s: %v", u.Email, n)
		}
	}
	var matchCount int
	_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches`).Scan(&matchCount)
	if matchCount != 1 {
		t.Fatalf("expected exactly one match row, got %d", matchCount)
	}
	// Le profil complet est accessible une fois matche.
	if r := e.do("GET", "/profiles/"+bob.ID, alice.Token, nil); r.Status != 200 {
		t.Fatalf("matched profile: %d", r.Status)
	}

	// Chat: envoi, non-lu, lu.
	if r := e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "   "}); r.Status != 400 {
		t.Fatalf("blank message: %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "Salut Bob !"}); r.Status != http.StatusCreated {
		t.Fatalf("send: %d %s", r.Status, r.Raw)
	}
	if r := e.do("POST", "/conversations/"+convID+"/messages", bob.Token, map[string]any{"body": "Salut Alice"}); r.Status != http.StatusCreated {
		t.Fatalf("reply: %d", r.Status)
	}
	e.do("POST", "/conversations/"+convID+"/messages", alice.Token, map[string]any{"body": "Un café ?"})
	convs := e.do("GET", "/conversations", bob.Token, nil).Body["items"].([]any)
	if len(convs) != 1 {
		t.Fatalf("conversations: %d", len(convs))
	}
	c0 := convs[0].(map[string]any)
	if c0["unreadCount"].(float64) != 1 || c0["lastMessage"].(map[string]any)["body"] != "Un café ?" {
		t.Fatalf("conversation summary: %v", c0)
	}
	if r := e.do("POST", "/conversations/"+convID+"/read", bob.Token, nil); r.Status != http.StatusNoContent {
		t.Fatalf("read: %d", r.Status)
	}
	if n := e.do("GET", "/conversations", bob.Token, nil).Body["items"].([]any)[0].(map[string]any)["unreadCount"].(float64); n != 0 {
		t.Fatalf("unread after read: %v", n)
	}
	msgs := e.do("GET", "/conversations/"+convID+"/messages?limit=2", bob.Token, nil)
	page := msgs.Body["items"].([]any)
	if len(page) != 2 || msgs.str("nextCursor") == "" || page[0].(map[string]any)["body"] != "Un café ?" {
		t.Fatalf("messages page: %s", msgs.Raw)
	}
	older := e.do("GET", "/conversations/"+convID+"/messages?limit=2&cursor="+msgs.str("nextCursor"), bob.Token, nil).Body["items"].([]any)
	if len(older) != 1 {
		t.Fatalf("older page: %d", len(older))
	}

	// Permissions: un tiers ne peut ni lire, ni ecrire, ni marquer comme lu, ni supprimer.
	for _, call := range []struct{ method, path string }{
		{"GET", "/conversations/" + convID + "/messages"},
		{"POST", "/conversations/" + convID + "/messages"},
		{"POST", "/conversations/" + convID + "/read"},
		{"DELETE", "/conversations/" + convID},
	} {
		if r := e.do(call.method, call.path, eve.Token, map[string]any{"body": "intrusion"}); r.Status != http.StatusNotFound {
			t.Fatalf("outsider %s %s: %d", call.method, call.path, r.Status)
		}
	}
	if r := e.do("DELETE", "/matches/"+matchID, eve.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("outsider unmatch: %d", r.Status)
	}
	if len(e.do("GET", "/conversations", eve.Token, nil).Body["items"].([]any)) != 0 {
		t.Fatalf("outsider sees conversations")
	}

	// Suppression locale: disparait pour Alice seulement; reapparait avec un nouveau message.
	if r := e.do("DELETE", "/conversations/"+convID, alice.Token, nil); r.Status != http.StatusNoContent {
		t.Fatalf("clear: %d", r.Status)
	}
	if len(e.do("GET", "/conversations", alice.Token, nil).Body["items"].([]any)) != 0 {
		t.Fatalf("cleared conversation should be hidden")
	}
	if len(e.do("GET", "/conversations", bob.Token, nil).Body["items"].([]any)) != 1 {
		t.Fatalf("bob must still see the conversation")
	}
	e.do("POST", "/conversations/"+convID+"/messages", bob.Token, map[string]any{"body": "Tu es là ?"})
	aliceMsgs := e.do("GET", "/conversations/"+convID+"/messages", alice.Token, nil).Body["items"].([]any)
	if len(aliceMsgs) != 1 {
		t.Fatalf("alice should only see messages after clearing, got %d", len(aliceMsgs))
	}

	// Blocage: le match disparait, plus de messages, plus de profil, dans les deux sens.
	if r := e.do("POST", "/users/"+alice.ID+"/block", alice.Token, nil); r.Status != http.StatusBadRequest {
		t.Fatalf("self block: %d", r.Status)
	}
	if r := e.do("POST", "/users/"+bob.ID+"/block", alice.Token, nil); r.Status != http.StatusNoContent {
		t.Fatalf("block: %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+convID+"/messages", bob.Token, map[string]any{"body": "hello?"}); r.Status == http.StatusCreated {
		t.Fatalf("blocked user must not message")
	}
	if len(e.do("GET", "/matches", bob.Token, nil).Body["items"].([]any)) != 0 {
		t.Fatalf("match must be removed after block")
	}
	if r := e.do("GET", "/profiles/"+alice.ID, bob.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("blocker profile visible to blocked: %d", r.Status)
	}
	if r := e.do("GET", "/profiles/"+bob.ID, alice.Token, nil); r.Status != http.StatusNotFound {
		t.Fatalf("blocked profile visible to blocker: %d", r.Status)
	}
	bl := e.do("GET", "/blocks", alice.Token, nil).Body["items"].([]any)
	if len(bl) != 1 {
		t.Fatalf("blocks list: %d", len(bl))
	}
	if r := e.do("DELETE", "/users/"+bob.ID+"/block", alice.Token, nil); r.Status != http.StatusNoContent {
		t.Fatalf("unblock: %d", r.Status)
	}

	// Signalement.
	if r := e.do("POST", "/users/"+bob.ID+"/report", alice.Token, map[string]any{"reason": "nope"}); r.Status != http.StatusBadRequest {
		t.Fatalf("bad reason: %d", r.Status)
	}
	if r := e.do("POST", "/users/"+bob.ID+"/report", alice.Token, map[string]any{"reason": "spam", "details": "test"}); r.Status != http.StatusCreated {
		t.Fatalf("report: %d", r.Status)
	}
	if r := e.do("POST", "/users/"+uuidZero+"/report", alice.Token, map[string]any{"reason": "spam"}); r.Status != http.StatusNotFound {
		t.Fatalf("report unknown user: %d", r.Status)
	}
}

const uuidZero = "00000000-0000-0000-0000-000000000000"

func TestNotificationsReadFlow(t *testing.T) {
	e := newTestEnv(t)
	a := e.register("na")
	b := e.register("nb")
	e.profile(a, "Ana", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
	e.profile(b, "Ben", "man", []string{"woman"}, "1994-03-03", parisLat, parisLng)
	e.do("POST", "/profiles/"+b.ID+"/like", a.Token, nil)
	m := e.do("POST", "/profiles/"+a.ID+"/like", b.Token, nil)
	conv := m.str("conversationId")
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "1"})
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "2"})

	list := e.do("GET", "/notifications", b.Token, nil)
	if list.Body["unreadCount"].(float64) != 2 { // 1 match + 1 message (dedoublonne)
		t.Fatalf("unread: %v %s", list.Body["unreadCount"], list.Raw)
	}
	items := list.Body["items"].([]any)
	if r := e.do("POST", "/notifications/read", b.Token, map[string]any{"ids": []string{items[0].(map[string]any)["id"].(string)}}); r.Status != http.StatusNoContent {
		t.Fatalf("mark one: %d", r.Status)
	}
	if n := e.do("GET", "/notifications", b.Token, nil).Body["unreadCount"].(float64); n != 1 {
		t.Fatalf("after one: %v", n)
	}
	// On ne peut pas marquer les notifications d'un autre.
	e.do("POST", "/notifications/read", a.Token, map[string]any{"ids": []string{items[1].(map[string]any)["id"].(string)}})
	if n := e.do("GET", "/notifications", b.Token, nil).Body["unreadCount"].(float64); n != 1 {
		t.Fatalf("foreign mark must not affect: %v", n)
	}
	if r := e.do("POST", "/notifications/read", b.Token, map[string]any{"ids": []string{}}); r.Status != http.StatusNoContent {
		t.Fatalf("mark all: %d", r.Status)
	}
	if n := e.do("GET", "/notifications", b.Token, nil).Body["unreadCount"].(float64); n != 0 {
		t.Fatalf("after all: %v", n)
	}
	// Pagination invalide.
	if r := e.do("GET", "/notifications?limit=0", b.Token, nil); r.Status != 400 {
		t.Fatalf("limit=0: %d", r.Status)
	}
	if r := e.do("GET", "/notifications?cursor=@@@", b.Token, nil); r.Status != 400 {
		t.Fatalf("bad cursor: %d", r.Status)
	}
}

func TestConcurrentMutualLikesCreateSingleMatch(t *testing.T) {
	e := newTestEnv(t)
	for round := 0; round < 5; round++ {
		a := e.register("ca")
		b := e.register("cb")
		e.profile(a, "A", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
		e.profile(b, "B", "man", []string{"woman"}, "1994-03-03", parisLat, parisLng)
		var wg sync.WaitGroup
		results := make([]resp, 2)
		wg.Add(2)
		go func() { defer wg.Done(); results[0] = e.do("POST", "/profiles/"+b.ID+"/like", a.Token, nil) }()
		go func() { defer wg.Done(); results[1] = e.do("POST", "/profiles/"+a.ID+"/like", b.Token, nil) }()
		wg.Wait()
		matched := 0
		for _, r := range results {
			if r.Status != 200 {
				t.Fatalf("round %d like: %d %s", round, r.Status, r.Raw)
			}
			if r.Body["matched"] == true {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("round %d: exactly one side must observe the match, got %d", round, matched)
		}
		var n int
		_ = e.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM matches WHERE user_a = LEAST($1::uuid,$2::uuid) AND user_b = GREATEST($1::uuid,$2::uuid)`, a.ID, b.ID).Scan(&n)
		if n != 1 {
			t.Fatalf("round %d: %d match rows", round, n)
		}
	}
}

func TestUnmatchAndDeleteAccount(t *testing.T) {
	e := newTestEnv(t)
	a := e.register("da")
	b := e.register("db")
	e.profile(a, "Ada", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
	e.profile(b, "Bo", "man", []string{"woman"}, "1994-03-03", parisLat, parisLng)
	e.do("POST", "/profiles/"+b.ID+"/like", a.Token, nil)
	m := e.do("POST", "/profiles/"+a.ID+"/like", b.Token, nil)
	conv := m.str("conversationId")
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "salut"})

	// Suppression de compte: mot de passe requis.
	if r := e.do("DELETE", "/me", a.Token, map[string]any{"password": "Wrong1234"}); r.Status != http.StatusForbidden {
		t.Fatalf("delete wrong password: %d", r.Status)
	}
	aPhoto := photoIDs(t, e, a.Token)[0]
	var key string
	if err := e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id=$1`, aPhoto).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if r := e.do("DELETE", "/me", a.Token, map[string]any{"password": "Passw0rd!x"}); r.Status != http.StatusNoContent {
		t.Fatalf("delete account: %d %s", r.Status, r.Raw)
	}
	ctx := context.Background()
	for table, col := range map[string]string{"users": "id", "profiles": "user_id", "photos": "user_id", "swipes": "swiper_id", "sessions": "user_id", "preferences": "user_id"} {
		var n int
		if err := e.pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE "+col+" = $1", a.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s still has %d rows for deleted user", table, n)
		}
	}
	var leftovers int
	_ = e.pool.QueryRow(ctx, `SELECT (SELECT COUNT(*) FROM matches)+(SELECT COUNT(*) FROM conversations)+(SELECT COUNT(*) FROM messages)+(SELECT COUNT(*) FROM swipes WHERE target_id=$1)`, a.ID).Scan(&leftovers)
	if leftovers != 0 {
		t.Fatalf("matches/conversations/messages/swipes must be cascaded, %d left", leftovers)
	}
	if r := e.do("POST", "/auth/login", "", map[string]any{"email": a.Email, "password": "Passw0rd!x"}); r.Status != http.StatusUnauthorized {
		t.Fatalf("login after deletion: %d", r.Status)
	}
	if len(e.do("GET", "/matches", b.Token, nil).Body["items"].([]any)) != 0 {
		t.Fatalf("deleted user's matches must vanish")
	}
}

func TestUnmatch(t *testing.T) {
	e := newTestEnv(t)
	a := e.register("ua")
	b := e.register("ub")
	e.profile(a, "Uma", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
	e.profile(b, "Ulf", "man", []string{"woman"}, "1994-03-03", parisLat, parisLng)
	e.do("POST", "/profiles/"+b.ID+"/like", a.Token, nil)
	m := e.do("POST", "/profiles/"+a.ID+"/like", b.Token, nil)
	if r := e.do("DELETE", "/matches/"+m.str("matchId"), a.Token, nil); r.Status != http.StatusNoContent {
		t.Fatalf("unmatch: %d", r.Status)
	}
	if r := e.do("POST", "/conversations/"+m.str("conversationId")+"/messages", b.Token, map[string]any{"body": "x"}); r.Status != http.StatusNotFound {
		t.Fatalf("message after unmatch: %d", r.Status)
	}
	// Pas de re-match: la decision de swipe reste definitive.
	if r := e.do("POST", "/profiles/"+b.ID+"/like", a.Token, nil); r.Status != http.StatusConflict {
		t.Fatalf("re-like after unmatch: %d", r.Status)
	}
}

func TestRealtimeWebSocket(t *testing.T) {
	e := newTestEnv(t)
	a := e.register("wa")
	b := e.register("wb")
	e.profile(a, "Wen", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
	e.profile(b, "Wim", "man", []string{"woman"}, "1994-03-03", parisLat, parisLng)

	srv := httptest.NewServer(e.router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	dial := func(token string) *websocket.Conn {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		if err := c.WriteJSON(map[string]string{"type": "auth", "token": token}); err != nil {
			t.Fatal(err)
		}
		return c
	}
	next := func(c *websocket.Conn) map[string]any {
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		var ev map[string]any
		if err := c.ReadJSON(&ev); err != nil {
			t.Fatalf("read ws: %v", err)
		}
		return ev
	}

	// Un token invalide est refuse.
	bad := dial("garbage")
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, _, err := bad.ReadMessage(); err == nil {
		t.Fatalf("invalid token must close the socket")
	}

	wa, wb := dial(a.Token), dial(b.Token)
	defer wa.Close()
	defer wb.Close()
	if next(wa)["type"] != "ready" || next(wb)["type"] != "ready" {
		t.Fatal("expected ready events")
	}

	e.do("POST", "/profiles/"+b.ID+"/like", a.Token, nil)
	m := e.do("POST", "/profiles/"+a.ID+"/like", b.Token, nil)
	if ev := next(wa); ev["type"] != "match" {
		t.Fatalf("alice should receive match event: %v", ev)
	}
	if ev := next(wb); ev["type"] != "match" {
		t.Fatalf("bob should receive match event: %v", ev)
	}

	conv := m.str("conversationId")
	e.do("POST", "/conversations/"+conv+"/messages", a.Token, map[string]any{"body": "temps réel"})
	ev := next(wb)
	payload, _ := ev["payload"].(map[string]any)
	if ev["type"] != "message" || payload["body"] != "temps réel" || payload["senderId"] != a.ID {
		t.Fatalf("bob should receive the message: %v", ev)
	}
	e.do("POST", "/conversations/"+conv+"/read", b.Token, nil)
	// alice recoit d'abord l'echo de son message (multi-appareils), puis l'accuse de lecture.
	got := []string{next(wa)["type"].(string), next(wa)["type"].(string)}
	if got[0] != "message" || got[1] != "read" {
		t.Fatalf("alice events: %v", got)
	}
}

func TestDeleteAccountRemovesPhotoFiles(t *testing.T) {
	e := newTestEnv(t)
	a := e.register("pf")
	e.profile(a, "Pia", "woman", []string{"man"}, "1994-03-03", parisLat, parisLng)
	var key string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE user_id=$1`, a.ID).Scan(&key)
	matches, _ := filepath.Glob(filepath.Join(e.uploadDir, key))
	if len(matches) != 1 {
		t.Fatalf("photo file should exist before deletion")
	}
	e.do("DELETE", "/me", a.Token, map[string]any{"password": "Passw0rd!x"})
	if matches, _ := filepath.Glob(filepath.Join(e.uploadDir, key)); len(matches) != 0 {
		t.Fatalf("photo file must be removed with the account")
	}
}
