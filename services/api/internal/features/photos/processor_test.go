package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"testing"

	platformerrors "example.com/api/internal/platform/errors"
)

func solid(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	return img
}

func status(t *testing.T, err error) int {
	t.Helper()
	appErr, ok := platformerrors.As(err)
	if !ok {
		t.Fatalf("expected AppError, got %v", err)
	}
	return appErr.Status
}

func TestProcessResizesAndReencodesAsJPEG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(3000, 1500)); err != nil {
		t.Fatal(err)
	}
	out, err := Process(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != MaxDimension || out.Height != MaxDimension/2 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if http.DetectContentType(out.Data) != "image/jpeg" {
		t.Fatal("output must be JPEG")
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatal(err)
	}
}

func TestProcessRejectsUnsafeInput(t *testing.T) {
	var gifBuf bytes.Buffer
	_ = gif.Encode(&gifBuf, solid(300, 300), &gif.Options{NumColors: 2, Quantizer: nil, Drawer: nil})
	var small bytes.Buffer
	_ = png.Encode(&small, image.NewGray(image.Rect(0, 0, 50, 50)))
	// valid JPEG header followed by garbage: sniffed as JPEG but undecodable
	truncated := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 16, 'J', 'F', 'I', 'F', 0}, bytes.Repeat([]byte{1}, 64)...)

	cases := map[string]struct {
		data []byte
		want int
	}{
		"empty":     {nil, 400},
		"html":      {[]byte("<html><script>alert(1)</script></html>"), 415},
		"svg":       {[]byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), 415},
		"php":       {[]byte("<?php echo 1;"), 415},
		"gif":       {gifBuf.Bytes(), 415},
		"too small": {small.Bytes(), 400},
		"truncated": {truncated, 400},
		"oversized": {bytes.Repeat([]byte{0xFF}, MaxUploadSize+10), 413},
	}
	for name, tc := range cases {
		_, err := Process(bytes.NewReader(tc.data))
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if got := status(t, err); got != tc.want {
			t.Fatalf("%s: want %d got %d", name, tc.want, got)
		}
	}
	_ = color.White
}

func TestLocalStorageRejectsTraversal(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../x", "a/../../x", "/etc/passwd", ""} {
		if err := s.Save(key, []byte("x")); err == nil {
			t.Fatalf("key %q must be rejected", key)
		}
	}
	if err := s.Save("user/photo.jpg", []byte("x")); err != nil {
		t.Fatal(err)
	}
}
