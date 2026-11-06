package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func sample(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), 90, 128}) // semi-transparent
		}
	}
	return img
}

func TestProcessReencodesAndResizes(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, sample(2560, 1600)); err != nil {
		t.Fatal(err)
	}
	out, err := Process(buf.Bytes())
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1280 || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	img, format, err := image.Decode(bytes.NewReader(out.Data))
	if err != nil || format != "jpeg" || img.Bounds().Dx() != 1280 {
		t.Fatalf("output is not the expected jpeg: %v %s", err, format)
	}
}

func TestProcessStripsTrailingPayload(t *testing.T) {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, sample(300, 300), nil)
	polyglot := append(buf.Bytes(), []byte("<script>alert(1)</script>")...)
	out, err := Process(polyglot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Data, []byte("<script>")) {
		t.Fatal("hidden payload survived re-encoding")
	}
}

func TestProcessRejections(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 300, 300), color.Palette{color.Black, color.White}), nil)
	var small bytes.Buffer
	_ = png.Encode(&small, sample(100, 100))
	cases := map[string]struct {
		data []byte
		want error
	}{
		"text":    {[]byte("hello world, definitely not an image"), ErrUnsupportedType},
		"html":    {[]byte("<html><body>hi</body></html>"), ErrUnsupportedType},
		"gif":     {g.Bytes(), ErrUnsupportedType},
		"corrupt": {append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 40)...), ErrCorruptImage},
		"small":   {small.Bytes(), ErrImageTooSmall},
		"empty":   {nil, ErrUnsupportedType},
	}
	for name, c := range cases {
		if _, err := Process(c.data); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
}

func TestFit(t *testing.T) {
	if w, h := fit(4000, 3000, 1280); w != 1280 || h != 960 {
		t.Errorf("landscape: %dx%d", w, h)
	}
	if w, h := fit(3000, 4000, 1280); w != 960 || h != 1280 {
		t.Errorf("portrait: %dx%d", w, h)
	}
	if w, h := fit(800, 600, 1280); w != 800 || h != 600 {
		t.Errorf("no upscale: %dx%d", w, h)
	}
}
