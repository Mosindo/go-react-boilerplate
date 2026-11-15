package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), 80, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessImageReencodesAndDownscales(t *testing.T) {
	out, err := ProcessImage(encodePNG(t, 2400, 1200))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != OutputMaxSide || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a JPEG: %v", err)
	}
}

func TestProcessImageStripsMetadata(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	var src bytes.Buffer
	_ = jpeg.Encode(&src, img, nil)
	// Inject a fake APP1/EXIF segment right after SOI.
	payload := []byte("Exif\x00\x00GPS-SECRET-COORDINATES")
	segment := append([]byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}, payload...)
	withExif := append(append([]byte{}, src.Bytes()[:2]...), append(segment, src.Bytes()[2:]...)...)

	out, err := ProcessImage(withExif)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Data, []byte("GPS-SECRET")) {
		t.Fatal("metadata must be stripped")
	}
}

func TestProcessImageRejections(t *testing.T) {
	gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
	cases := map[string]struct {
		data []byte
		want error
	}{
		"empty":     {nil, ErrCorruptImage},
		"text":      {[]byte("hello world, not an image"), ErrUnsupportedType},
		"script":    {[]byte("<svg onload=alert(1)>"), ErrUnsupportedType},
		"gif":       {gif, ErrUnsupportedType},
		"too small": {encodePNG(t, 100, 300), ErrImageDimensions},
		"truncated": {encodePNG(t, 400, 400)[:200], ErrCorruptImage},
		"oversized": {make([]byte, MaxUploadBytes+1), ErrImageTooLarge},
	}
	for name, c := range cases {
		if _, err := ProcessImage(c.data); err != c.want {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
}

func TestFitWithinAndSameSet(t *testing.T) {
	if w, h := FitWithin(800, 600, 1600); w != 800 || h != 600 {
		t.Fatal("small images must not be upscaled")
	}
	if w, h := FitWithin(1000, 4000, 1600); w != 400 || h != 1600 {
		t.Fatalf("portrait scaling wrong: %dx%d", w, h)
	}
	photos := []Photo{{ID: "a"}, {ID: "b"}}
	if !SameSet(photos, []string{"b", "a"}) || SameSet(photos, []string{"a", "a"}) || SameSet(photos, []string{"a"}) {
		t.Fatal("SameSet is wrong")
	}
}
