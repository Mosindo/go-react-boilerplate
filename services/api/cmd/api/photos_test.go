package main

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhotoLifecycleAndSecurity(t *testing.T) {
	e := newEnv(t)
	owner, viewer, stranger := e.register("pown"), e.register("pview"), e.register("pstr")
	e.onboard(owner, profileSpec{Name: "Owen", Gender: "man", InterestedIn: []string{"woman"}, Age: 30})
	e.onboard(viewer, profileSpec{Name: "Vera", Gender: "woman", InterestedIn: []string{"man"}, Age: 30})

	var prof struct {
		Photos []struct {
			ID string `json:"id"`
		} `json:"photos"`
	}
	e.expect(e.do("GET", "/me/profile", owner.Token, nil), http.StatusOK).json(t, &prof)
	first := prof.Photos[0].ID

	// uploads are validated by content, not by name or extension
	e.expect(e.upload("POST", "/me/photos", owner.Token, []byte("<?php system($_GET['c']); ?>"), "shell.jpg"), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/me/photos", owner.Token, []byte("GIF89a....."), "a.gif"), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/me/photos", owner.Token, nil, "empty.jpg"), http.StatusUnsupportedMediaType)
	e.expect(e.upload("POST", "/me/photos", owner.Token, testPNG(10001, 1, color.White), "wide.png"), http.StatusRequestEntityTooLarge)
	e.expect(e.upload("POST", "/me/photos", owner.Token, bytes.Repeat([]byte{0xFF}, 9<<20), "huge.jpg"), http.StatusRequestEntityTooLarge)
	e.expect(e.upload("POST", "/me/photos", "", testPNG(10, 10, color.White), "a.png"), http.StatusUnauthorized)

	// big photos are shrunk and re-encoded as JPEG; a JPEG upload works too
	var jpg bytes.Buffer
	_ = jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 3000, 2000)), nil)
	second := e.expect(e.upload("POST", "/me/photos", owner.Token, jpg.Bytes(), "big.jpg"), http.StatusCreated)
	if second.str(t, "id") == "" || !strings.Contains(string(second.Body), `"width":1080`) || !strings.Contains(string(second.Body), `"height":720`) {
		t.Fatalf("expected 1080x720 after resize: %s", second.Body)
	}
	res := e.do("GET", "/photos/"+second.str(t, "id"), owner.Token, nil)
	if res.Status != http.StatusOK {
		t.Fatalf("serve own photo: %d", res.Status)
	}
	if img, err := jpeg.Decode(bytes.NewReader(res.Body)); err != nil || img.Bounds().Dx() != 1080 {
		t.Fatalf("served photo must be a 1080px JPEG: %v", err)
	}
	thumb := e.do("GET", "/photos/"+second.str(t, "id")+"?size=thumb", owner.Token, nil)
	if img, err := jpeg.Decode(bytes.NewReader(thumb.Body)); err != nil || img.Bounds().Dx() != 320 {
		t.Fatalf("thumbnail must be 320px wide: %v", err)
	}

	// fill up to the 6 photo limit
	for i := 0; i < 4; i++ {
		e.expect(e.upload("POST", "/me/photos", owner.Token, testPNG(20, 20, color.Black), "x.png"), http.StatusCreated)
	}
	e.expect(e.upload("POST", "/me/photos", owner.Token, testPNG(20, 20, color.Black), "x.png"), http.StatusConflict)

	// reorder: second photo becomes the main one
	e.expect(e.do("GET", "/me/profile", owner.Token, nil), http.StatusOK).json(t, &prof)
	ids := []string{}
	for _, p := range prof.Photos {
		ids = append(ids, p.ID)
	}
	reordered := append([]string{ids[1], ids[0]}, ids[2:]...)
	e.expect(e.do("PUT", "/me/photos/order", owner.Token, map[string]any{"photoIds": reordered[:5]}), http.StatusUnprocessableEntity) // missing one
	e.expect(e.do("PUT", "/me/photos/order", owner.Token, map[string]any{"photoIds": append([]string{ids[1], ids[1]}, ids[2:]...)}), http.StatusUnprocessableEntity)
	e.expect(e.do("PUT", "/me/photos/order", owner.Token, map[string]any{"photoIds": reordered}), http.StatusNoContent)
	e.expect(e.do("GET", "/me/profile", owner.Token, nil), http.StatusOK).json(t, &prof)
	if prof.Photos[0].ID != ids[1] {
		t.Fatalf("main photo should be %s, got %s", ids[1], prof.Photos[0].ID)
	}

	// replace keeps the slot, changes the id, and removes old files
	before, _ := filepath.Glob(filepath.Join(e.dir, "*.jpg"))
	rep := e.expect(e.upload("PUT", "/me/photos/"+ids[1], owner.Token, testPNG(30, 30, color.RGBA{0, 0, 255, 255}), "n.png"), http.StatusOK)
	if rep.str(t, "id") == ids[1] || !strings.Contains(string(rep.Body), `"position":0`) {
		t.Fatalf("replace should keep position 0 with a fresh id: %s", rep.Body)
	}
	after, _ := filepath.Glob(filepath.Join(e.dir, "*.jpg"))
	if len(after) != len(before) {
		t.Fatalf("replace must not leak files: %d -> %d", len(before), len(after))
	}
	e.expect(e.do("GET", "/photos/"+ids[1], owner.Token, nil), http.StatusNotFound)

	// only the owner can modify
	e.expect(e.do("DELETE", "/me/photos/"+rep.str(t, "id"), viewer.Token, nil), http.StatusNotFound)
	e.expect(e.upload("PUT", "/me/photos/"+rep.str(t, "id"), viewer.Token, testPNG(10, 10, color.White), "a.png"), http.StatusNotFound)

	// delete compacts positions
	e.expect(e.do("DELETE", "/me/photos/"+first, owner.Token, nil), http.StatusNoContent)
	e.expect(e.do("DELETE", "/me/photos/"+first, owner.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/me/profile", owner.Token, nil), http.StatusOK).json(t, &prof)
	if len(prof.Photos) != 5 {
		t.Fatalf("expected 5 photos left, got %d", len(prof.Photos))
	}

	// visibility: discoverable owners' photos are visible to other users, hidden owners' are not
	e.expect(e.do("GET", "/photos/"+rep.str(t, "id"), viewer.Token, nil), http.StatusOK)
	e.expect(e.do("GET", "/photos/not-a-uuid", viewer.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/photos/"+rep.str(t, "id"), "", nil), http.StatusUnauthorized)
	e.expect(e.do("PUT", "/me/profile", owner.Token, map[string]any{"firstName": "Owen", "birthDate": strings.Split(string(e.do("GET", "/me/profile", owner.Token, nil).Body), `"birthDate":"`)[1][:10], "gender": "man", "discoverable": false}), http.StatusOK)
	e.expect(e.do("GET", "/photos/"+rep.str(t, "id"), viewer.Token, nil), http.StatusNotFound)
	e.expect(e.do("GET", "/photos/"+rep.str(t, "id"), owner.Token, nil), http.StatusOK)
	_ = stranger
}
