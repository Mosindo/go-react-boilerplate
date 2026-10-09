package main

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
)

func photoIDs(t *testing.T, r resp) []string {
	t.Helper()
	var out struct {
		Photos []struct {
			ID       string `json:"id"`
			Position int    `json:"position"`
		} `json:"photos"`
	}
	r.json(t, &out)
	ids := make([]string, len(out.Photos))
	for i, p := range out.Photos {
		if p.Position != i {
			t.Fatalf("positions must be contiguous, got %+v", out.Photos)
		}
		ids[i] = p.ID
	}
	return ids
}

func TestPhotoLifecycle(t *testing.T) {
	e := newEnv(t)
	u := e.register("")

	// Accepts PNG (re-encoded to JPEG) and JPEG.
	first := e.expect(e.upload("POST", "/me/photos", u.Token, "a.png", testPNG(500, 400)), http.StatusCreated).m(t)
	e.expect(e.upload("POST", "/me/photos", u.Token, "b.jpg", testJPEG(2000, 1500, 3)), http.StatusCreated)
	e.expect(e.upload("POST", "/me/photos", u.Token, "c.jpg", testJPEG(300, 300, 9)), http.StatusCreated)

	img := e.expect(e.do("GET", first["url"].(string), nil, u.Token), http.StatusOK)
	if img.Header.Get("Content-Type") != "image/jpeg" || !bytes.HasPrefix(img.Body, []byte{0xFF, 0xD8}) {
		t.Fatal("photos must be served as re-encoded JPEG")
	}
	if img.Header.Get("Cache-Control") == "" || strings.Contains(img.Header.Get("Cache-Control"), "public") {
		t.Fatalf("photos must be privately cached: %q", img.Header.Get("Cache-Control"))
	}

	// Rejected uploads.
	e.expect(e.upload("POST", "/me/photos", u.Token, "x.jpg", []byte("<?php system($_GET['c']); ?>")), http.StatusBadRequest)
	e.expect(e.upload("POST", "/me/photos", u.Token, "x.jpg", append([]byte("GIF89a"), make([]byte, 100)...)), http.StatusBadRequest)
	e.expect(e.upload("POST", "/me/photos", u.Token, "tiny.jpg", testJPEG(50, 50, 1)), http.StatusBadRequest)
	e.expect(e.upload("POST", "/me/photos", u.Token, "huge.bin", make([]byte, 9<<20)), http.StatusRequestEntityTooLarge)
	if r := e.do("POST", "/me/photos", nil, u.Token); r.Status != http.StatusBadRequest {
		t.Fatalf("missing file should be 400, got %d", r.Status)
	}
	e.expect(e.do("GET", "/me/photos", nil, ""), http.StatusUnauthorized)

	// Reorder & main photo.
	ids := photoIDs(t, e.expect(e.do("GET", "/me/photos", nil, u.Token), http.StatusOK))
	if len(ids) != 3 {
		t.Fatalf("expected 3 photos, got %d", len(ids))
	}
	reordered := photoIDs(t, e.expect(e.do("PUT", "/me/photos/order", map[string]any{"ids": []string{ids[2], ids[0], ids[1]}}, u.Token), http.StatusOK))
	if reordered[0] != ids[2] || reordered[1] != ids[0] {
		t.Fatalf("order not applied: %v", reordered)
	}
	e.expect(e.do("PUT", "/me/photos/order", map[string]any{"ids": []string{ids[0], ids[0], ids[1]}}, u.Token), http.StatusBadRequest)
	e.expect(e.do("PUT", "/me/photos/order", map[string]any{"ids": []string{ids[0], ids[1]}}, u.Token), http.StatusBadRequest)
	main := photoIDs(t, e.expect(e.do("POST", "/me/photos/"+ids[1]+"/main", nil, u.Token), http.StatusOK))
	if main[0] != ids[1] {
		t.Fatalf("main photo not applied: %v", main)
	}

	// Replace keeps position and swaps the file.
	var oldKey, newKey string
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[1]).Scan(&oldKey)
	rep := e.expect(e.upload("PUT", "/me/photos/"+ids[1], u.Token, "new.jpg", testJPEG(320, 320, 77)), http.StatusOK).m(t)
	if rep["position"].(float64) != 0 {
		t.Fatalf("replace changed position: %v", rep)
	}
	_ = e.pool.QueryRow(context.Background(), `SELECT storage_key FROM photos WHERE id = $1`, ids[1]).Scan(&newKey)
	if oldKey == newKey {
		t.Fatal("replace must store a new file")
	}
	if _, err := e.store.Open(context.Background(), oldKey); err == nil {
		t.Fatal("old file must be deleted after replace")
	}

	// Delete closes the gap and removes the file.
	e.expect(e.do("DELETE", "/me/photos/"+ids[1], nil, u.Token), http.StatusNoContent)
	if _, err := e.store.Open(context.Background(), newKey); err == nil {
		t.Fatal("deleted photo file still on disk")
	}
	left := photoIDs(t, e.expect(e.do("GET", "/me/photos", nil, u.Token), http.StatusOK))
	if len(left) != 2 {
		t.Fatalf("expected 2 photos left, got %v", left)
	}
	e.expect(e.do("DELETE", "/me/photos/"+ids[1], nil, u.Token), http.StatusNotFound)
}

func TestPhotoLimitAndOwnership(t *testing.T) {
	e := newEnv(t)
	u, other := e.register(""), e.register("")
	for i := 0; i < 6; i++ {
		e.expect(e.upload("POST", "/me/photos", u.Token, "p.jpg", testJPEG(260, 260, uint8(i))), http.StatusCreated)
	}
	r := e.expect(e.upload("POST", "/me/photos", u.Token, "p.jpg", testJPEG(260, 260, 99)), http.StatusConflict)
	if r.m(t)["code"] != "photo_limit" {
		t.Fatalf("unexpected body %s", r.Body)
	}

	ids := photoIDs(t, e.expect(e.do("GET", "/me/photos", nil, u.Token), http.StatusOK))
	// Other users cannot modify or delete my photos.
	e.expect(e.do("DELETE", "/me/photos/"+ids[0], nil, other.Token), http.StatusNotFound)
	e.expect(e.upload("PUT", "/me/photos/"+ids[0], other.Token, "p.jpg", testJPEG(260, 260, 5)), http.StatusNotFound)
	e.expect(e.do("POST", "/me/photos/"+ids[0]+"/main", nil, other.Token), http.StatusNotFound)
	// A user without a profile cannot see a non-discoverable (profile-less) user's photos.
	e.expect(e.do("GET", "/photos/"+ids[0]+"/file", nil, other.Token), http.StatusNotFound)
	e.expect(e.do("GET", "/photos/"+ids[0]+"/file", nil, u.Token), http.StatusOK)
	e.expect(e.do("GET", "/photos/"+ids[0]+"/file", nil, ""), http.StatusUnauthorized)
}

func TestPhotoVisibilityFollowsPrivacyAndBlocks(t *testing.T) {
	e := newEnv(t)
	a, b, _ := e.pair()
	bPhoto := photoIDs(t, e.expect(e.do("GET", "/me/photos", nil, b.Token), http.StatusOK))[0]
	url := "/photos/" + bPhoto + "/file"

	e.expect(e.do("GET", url, nil, a.Token), http.StatusOK) // discoverable
	e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"discoverable": false, "gender": "man", "firstName": "Bob"}), b.Token), http.StatusOK)
	e.expect(e.do("GET", url, nil, a.Token), http.StatusNotFound) // hidden
	e.expect(e.do("PUT", "/me/profile", profilePayload(map[string]any{"discoverable": true, "gender": "man", "firstName": "Bob"}), b.Token), http.StatusOK)
	e.expect(e.do("GET", url, nil, a.Token), http.StatusOK)
	e.expect(e.do("POST", "/blocks", map[string]any{"userId": b.ID}, a.Token), http.StatusNoContent)
	e.expect(e.do("GET", url, nil, a.Token), http.StatusNotFound) // blocked
}
