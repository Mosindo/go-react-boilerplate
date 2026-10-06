package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func photoIDs(t *testing.T, r response) []string {
	t.Helper()
	var out struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	r.decode(t, &out)
	ids := make([]string, len(out.Photos))
	for i, p := range out.Photos {
		if p.Position != i {
			t.Fatalf("positions must be contiguous from 0, got %+v", out.Photos)
		}
		ids[i] = p.ID
	}
	return ids
}

func TestPhotoUploadValidationAndLimits(t *testing.T) {
	e := newEnv(t)
	u := e.register("pierre")

	// Dangerous or invalid content, even when named and typed like an image.
	for name, data := range map[string][]byte{
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"php":       []byte("<?php system('id'); ?>"),
		"svg":       []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"empty":     {},
		"truncated": testJPEG(t, 1)[:64],
	} {
		r := e.upload(u.Token, "/photos", "evil.jpg", data)
		if r.Status != http.StatusUnsupportedMediaType && r.Status != http.StatusUnprocessableEntity && r.Status != http.StatusBadRequest {
			t.Errorf("%s must be rejected, got %d %s", name, r.Status, r.Body)
		}
	}
	// Oversized body is cut off.
	big := bytes.Repeat([]byte{0xFF}, 10<<20)
	e.expect(e.upload(u.Token, "/photos", "big.jpg", big), http.StatusRequestEntityTooLarge)
	// No file part.
	e.expect(e.raw(http.MethodPost, "/photos", u.Token, "", []byte("x"), "multipart/form-data; boundary=x"), http.StatusBadRequest)
	// Unauthenticated.
	e.expect(e.upload("", "/photos", "a.jpg", testJPEG(t, 1)), http.StatusUnauthorized)

	for i := 0; i < 6; i++ {
		e.expect(e.upload(u.Token, "/photos", "ok.jpg", testJPEG(t, i)), http.StatusCreated)
	}
	e.expect(e.upload(u.Token, "/photos", "seventh.jpg", testJPEG(t, 9)), http.StatusConflict)

	// Stored files are clean JPEGs under random server-chosen names.
	var key string
	if err := e.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE user_id=$1 LIMIT 1`, u.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(e.dir, filepath.FromSlash(key)))
	if err != nil || !bytes.HasPrefix(stored, []byte{0xFF, 0xD8}) || strings.Contains(key, "ok.jpg") {
		t.Fatalf("unexpected stored object %q err=%v", key, err)
	}
}

func TestPhotoOrderPrimaryDeleteAndReplace(t *testing.T) {
	e := newEnv(t)
	u := e.register("sophie")
	for i := 0; i < 3; i++ {
		e.expect(e.upload(u.Token, "/photos", "p.jpg", testJPEG(t, i)), http.StatusCreated)
	}
	ids := photoIDs(t, e.expect(e.do(http.MethodGet, "/photos", u.Token, nil), http.StatusOK))

	// Reorder: the first id becomes the primary photo.
	reordered := []string{ids[2], ids[0], ids[1]}
	got := photoIDs(t, e.expect(e.do(http.MethodPut, "/photos/order", u.Token, map[string]any{"photoIds": reordered}), http.StatusOK))
	if fmt.Sprint(got) != fmt.Sprint(reordered) {
		t.Fatalf("order not applied: %v", got)
	}
	// Incomplete, duplicated or foreign ids are refused.
	e.expect(e.do(http.MethodPut, "/photos/order", u.Token, map[string]any{"photoIds": reordered[:2]}), http.StatusUnprocessableEntity)
	e.expect(e.do(http.MethodPut, "/photos/order", u.Token, map[string]any{"photoIds": []string{ids[0], ids[0], ids[1]}}), http.StatusUnprocessableEntity)
	e.expect(e.do(http.MethodPut, "/photos/order", u.Token, map[string]any{"photoIds": []string{"nope"}}), http.StatusBadRequest)

	// Deleting the primary promotes the next one and keeps positions contiguous.
	e.expect(e.do(http.MethodDelete, "/photos/"+reordered[0], u.Token, nil), http.StatusNoContent)
	got = photoIDs(t, e.expect(e.do(http.MethodGet, "/photos", u.Token, nil), http.StatusOK))
	if fmt.Sprint(got) != fmt.Sprint(reordered[1:]) {
		t.Fatalf("after delete got %v", got)
	}
	e.expect(e.do(http.MethodDelete, "/photos/"+reordered[0], u.Token, nil), http.StatusNotFound)

	// Replace keeps the slot and removes the old file from disk.
	var oldKey string
	_ = e.pool.QueryRow(t.Context(), `SELECT storage_key FROM photos WHERE id=$1`, got[0]).Scan(&oldKey)
	r := e.expect(e.upload(u.Token, "/photos?replace="+got[0], "new.jpg", testJPEG(t, 7)), http.StatusCreated)
	if r.json(t)["position"].(float64) != 0 {
		t.Fatalf("replacement must keep position 0: %s", r.Body)
	}
	if _, err := os.Stat(filepath.Join(e.dir, filepath.FromSlash(oldKey))); !os.IsNotExist(err) {
		t.Fatal("replaced file must be removed from storage")
	}
	e.expect(e.upload(u.Token, "/photos?replace=00000000-0000-0000-0000-000000000000", "x.jpg", testJPEG(t, 1)), http.StatusNotFound)
}

func TestPhotoContentIsAccessControlled(t *testing.T) {
	e := newEnv(t)
	owner := e.newPerson(persona{Name: "Owner", Gender: "woman", InterestedIn: []string{"man"}})
	stranger := e.register("stranger") // no profile: not part of the app yet
	peer := e.newPerson(persona{Name: "Peer", Gender: "man", InterestedIn: []string{"woman"}})

	photo := photoIDs(t, e.expect(e.do(http.MethodGet, "/photos", owner.Token, nil), http.StatusOK))[0]
	path := "/photos/" + photo + "/content"

	e.expect(e.do(http.MethodGet, path, "", nil), http.StatusUnauthorized)
	e.expect(e.do(http.MethodGet, path, owner.Token, nil), http.StatusOK)
	e.expect(e.do(http.MethodGet, path, peer.Token, nil), http.StatusOK) // discoverable owner

	// Owner hides from discovery: strangers lose access, existence is not disclosed.
	e.expect(e.do(http.MethodPut, "/profile/privacy", owner.Token, map[string]any{"showAge": true, "showDistance": true, "discoverable": false}), http.StatusOK)
	e.expect(e.do(http.MethodGet, path, peer.Token, nil), http.StatusNotFound)
	_ = stranger

	// ...but a match can still see them.
	e.makeMatchIgnoringDiscovery(owner, peer)
	e.expect(e.do(http.MethodGet, path, peer.Token, nil), http.StatusOK)

	// Nobody can delete or reorder somebody else's photo.
	e.expect(e.do(http.MethodDelete, "/photos/"+photo, peer.Token, nil), http.StatusNotFound)
	e.expect(e.do(http.MethodGet, "/photos/not-a-uuid/content", peer.Token, nil), http.StatusNotFound)
}

// makeMatchIgnoringDiscovery matches two users whose discovery visibility
// does not matter (the peer likes first, the hidden owner likes back).
func (e *env) makeMatchIgnoringDiscovery(hidden, other user) string {
	e.t.Helper()
	e.swipe(other, hidden, "like", http.StatusCreated)
	out := e.swipe(hidden, other, "like", http.StatusCreated)
	if !out.Matched {
		e.t.Fatalf("expected match, got %+v", out)
	}
	return out.MatchID
}
