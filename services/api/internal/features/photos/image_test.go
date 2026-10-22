package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 10, G: 200, B: 90, A: 128}) // semi transparent
		}
	}
	return img
}

func encodePNG(img image.Image) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := ProcessImage(bytes.NewReader(encodePNG(solid(2400, 1200))))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 540 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	full, err := jpeg.Decode(bytes.NewReader(out.Full))
	if err != nil || full.Bounds().Dx() != 1080 {
		t.Fatalf("full variant is not a valid JPEG: %v", err)
	}
	thumb, err := jpeg.Decode(bytes.NewReader(out.Thumb))
	if err != nil || thumb.Bounds().Dx() != 320 {
		t.Fatalf("thumb variant invalid: %v", err)
	}
}

func TestProcessImageDoesNotUpscale(t *testing.T) {
	out, err := ProcessImage(bytes.NewReader(encodePNG(solid(300, 400))))
	if err != nil || out.Width != 300 || out.Height != 400 {
		t.Fatalf("got %+v %v", out, err)
	}
}

func TestProcessImageRejectsUntrustedInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"script":       {[]byte("<script>alert(1)</script>"), ErrUnsupportedImage},
		"html as png":  {append([]byte("\x89PNG\r\n\x1a\n"), []byte("<html>")...), ErrInvalidImage},
		"svg":          {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), ErrUnsupportedImage},
		"gif":          {[]byte("GIF89a\x01\x00\x01\x00"), ErrUnsupportedImage},
		"truncated":    {encodePNG(solid(300, 300))[:80], ErrInvalidImage},
		"tiny":         {encodePNG(solid(40, 40)), ErrImageTooSmall},
		"empty":        {nil, ErrUnsupportedImage},
		"too large":    {bytes.Repeat([]byte{0xff}, MaxUploadBytes+10), ErrImageTooLarge},
		"php polyglot": {append(encodePNG(solid(300, 300))[:100], []byte("<?php system($_GET[0]); ?>")...), ErrInvalidImage},
	}
	for name, tc := range cases {
		if _, err := ProcessImage(bytes.NewReader(tc.data)); err != tc.want {
			t.Fatalf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func TestProcessedOutputStripsPayloads(t *testing.T) {
	// A valid PNG with a trailing payload must not survive re-encoding.
	data := append(encodePNG(solid(300, 300)), []byte("<?php evil() ?>")...)
	out, err := ProcessImage(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out.Full), "evil") || strings.Contains(string(out.Thumb), "evil") {
		t.Fatal("payload survived re-encoding")
	}
}

func TestStorageRejectsPathTraversal(t *testing.T) {
	s, err := NewStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../escape.jpg", "/etc/passwd", "a/../../b.jpg", ""} {
		if err := s.Save(key, []byte("x"), []byte("y")); err == nil {
			t.Fatalf("key %q must be rejected", key)
		}
	}
	if err := s.Save("ab/abcdef.jpg", []byte("x"), []byte("y")); err != nil {
		t.Fatal(err)
	}
	s.Remove("ab/abcdef.jpg", "../nothing.jpg")
	if f, err := s.Open("ab/abcdef.jpg", "full"); err == nil {
		f.Close()
		t.Fatal("file should be removed")
	}
}
