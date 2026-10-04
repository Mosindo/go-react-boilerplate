package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	return img
}

func TestProcessResizesAndReencodes(t *testing.T) {
	var src bytes.Buffer
	if err := jpeg.Encode(&src, solid(3000, 2000), nil); err != nil {
		t.Fatal(err)
	}
	// Trailing garbage (polyglot attempt) must disappear after re-encoding.
	data := append(src.Bytes(), []byte("<?php system($_GET['c']); ?>")...)
	out, err := Process(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Full, []byte("<?php")) || bytes.Contains(out.Thumb, []byte("<?php")) {
		t.Fatal("trailing payload survived re-encoding")
	}
	full, _, err := image.DecodeConfig(bytes.NewReader(out.Full))
	if err != nil || full.Width != 1440 || full.Height != 960 {
		t.Fatalf("full: %+v %v", full, err)
	}
	thumb, _, _ := image.DecodeConfig(bytes.NewReader(out.Thumb))
	if thumb.Width != 360 || thumb.Height != 240 {
		t.Fatalf("thumb: %+v", thumb)
	}
	if out.Width != 1440 || out.Height != 960 {
		t.Fatalf("reported size %dx%d", out.Width, out.Height)
	}
}

func TestProcessRejectsBadInput(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 300, 300), color.Palette{color.Black, color.White}), nil)
	var tiny bytes.Buffer
	_ = png.Encode(&tiny, solid(40, 40))
	var huge bytes.Buffer
	_ = png.Encode(&huge, image.NewGray(image.Rect(0, 0, 8001, 10)))

	cases := map[string]struct {
		data []byte
		want error
	}{
		"text":      {[]byte("hello world, definitely not an image"), ErrUnsupportedType},
		"html":      {[]byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		"gif":       {g.Bytes(), ErrUnsupportedType},
		"empty":     {nil, ErrUnsupportedType},
		"truncated": {[]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), ErrBadImage},
		"tiny":      {tiny.Bytes(), ErrBadImage},
		"too wide":  {huge.Bytes(), ErrImageTooLarge},
	}
	for name, c := range cases {
		if _, err := Process(c.data); err != c.want {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
}

func TestProcessFlattensTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300)) // fully transparent
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	out, err := Process(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	dec, _ := jpeg.Decode(bytes.NewReader(out.Full))
	r, g, b, _ := dec.At(150, 150).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("transparent pixel not flattened to white: %d %d %d", r>>8, g>>8, b>>8)
	}
}
