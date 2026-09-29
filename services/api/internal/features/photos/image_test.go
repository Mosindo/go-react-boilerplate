package photos

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func testImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 128, 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func decodeJPEG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
	return img
}

// pngWithSize forges a PNG whose header claims w x h (a decompression bomb candidate).
func pngWithSize(t *testing.T, w, h uint32) []byte {
	t.Helper()
	data := encodePNG(t, testImage(1, 1))
	// signature(8) + length(4) + "IHDR"(4) + data(13) + crc(4)
	binary.BigEndian.PutUint32(data[16:20], w)
	binary.BigEndian.PutUint32(data[20:24], h)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	return data
}

// 1x1 lossless WebP.
const tinyWebP = "UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA=="

func TestProcessImageFormats(t *testing.T) {
	webpData, err := base64.StdEncoding.DecodeString(tinyWebP)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"jpeg": encodeJPEG(t, testImage(300, 200)),
		"png":  encodePNG(t, testImage(300, 200)),
		"webp": webpData,
	}
	for name, in := range cases {
		out, w, h, err := ProcessImage(in)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		img := decodeJPEG(t, out)
		if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
			t.Errorf("%s: reported %dx%d, actual %v", name, w, h, img.Bounds())
		}
	}
}

func TestProcessImageResizesLongestSide(t *testing.T) {
	out, w, h, err := ProcessImage(encodeJPEG(t, testImage(3000, 1500)))
	if err != nil {
		t.Fatal(err)
	}
	if w != 1080 || h != 540 {
		t.Fatalf("got %dx%d", w, h)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 1080 || b.Dy() != 540 {
		t.Fatalf("bounds %v", b)
	}
	_, w, h, err = ProcessImage(encodePNG(t, testImage(800, 2400)))
	if err != nil || w != 360 || h != 1080 {
		t.Fatalf("portrait got %dx%d %v", w, h, err)
	}
	// Never upscale.
	if _, w, h, _ = ProcessImage(encodeJPEG(t, testImage(100, 50))); w != 100 || h != 50 {
		t.Fatalf("upscaled to %dx%d", w, h)
	}
}

func TestProcessImageRejects(t *testing.T) {
	gif := []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"text", []byte("hello world, definitely not an image"), ErrUnsupportedImage},
		{"empty", nil, ErrUnsupportedImage},
		{"html", []byte("<html><script>alert(1)</script></html>"), ErrUnsupportedImage},
		{"gif", gif, ErrUnsupportedImage},
		{"truncated jpeg", encodeJPEG(t, testImage(200, 200))[:200], ErrCorruptImage},
		{"truncated png", encodePNG(t, testImage(200, 200))[:60], ErrCorruptImage},
		{"png bomb 30000x30000", pngWithSize(t, 30000, 30000), ErrImageTooBig},
		{"png 7000x7000 (49 MP)", pngWithSize(t, 7000, 7000), ErrImageTooBig},
		{"png 8001 wide", pngWithSize(t, 8001, 10), ErrImageTooBig},
	}
	for _, c := range cases {
		if _, _, _, err := ProcessImage(c.data); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v want %v", c.name, err, c.want)
		}
	}
}

func TestProcessImageStripsMetadata(t *testing.T) {
	src := encodeJPEG(t, testImage(64, 64))
	secret := []byte("GPS-SECRET-LOCATION")
	// Inject a COM segment carrying data that must not survive re-encoding.
	seg := append([]byte{0xFF, 0xFE, 0, byte(len(secret) + 2)}, secret...)
	withMeta := append(append(append([]byte{}, src[:2]...), seg...), src[2:]...)
	if !bytes.Contains(withMeta, secret) {
		t.Fatal("fixture broken")
	}
	out, _, _, err := ProcessImage(withMeta)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, secret) || bytes.Contains(out, []byte("Exif")) {
		t.Fatal("metadata leaked into output")
	}
}

func exifJPEG(t *testing.T, w, h, orientation int) []byte {
	t.Helper()
	src := encodeJPEG(t, testImage(w, h))
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(orientation), 0, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	seg = append(seg, payload...)
	return append(append(append([]byte{}, src[:2]...), seg...), src[2:]...)
}

func TestProcessImageAppliesExifOrientation(t *testing.T) {
	if got := exifOrientation(exifJPEG(t, 40, 20, 6)); got != 6 {
		t.Fatalf("orientation parse = %d", got)
	}
	_, w, h, err := ProcessImage(exifJPEG(t, 40, 20, 6))
	if err != nil || w != 20 || h != 40 {
		t.Fatalf("rotated size %dx%d err=%v", w, h, err)
	}
	_, w, h, _ = ProcessImage(exifJPEG(t, 40, 20, 3))
	if w != 40 || h != 20 {
		t.Fatalf("180 rotation changed size %dx%d", w, h)
	}
	if exifOrientation([]byte("garbage")) != 0 {
		t.Fatal("garbage must give 0")
	}
}

func TestProcessImageFlattensAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8)) // fully transparent
	out, _, _, err := ProcessImage(encodePNG(t, img))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decodeJPEG(t, out).At(4, 4).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("transparent pixel not white: %d %d %d", r>>8, g>>8, b>>8)
	}
}

func TestFit(t *testing.T) {
	cases := [][5]int{{2000, 1000, 1080, 1080, 540}, {1000, 2000, 1080, 540, 1080}, {1080, 1080, 1080, 1080, 1080}, {5000, 1, 1080, 1080, 1}}
	for _, c := range cases {
		w, h := fit(c[0], c[1], c[2])
		if w != c[3] || h != c[4] {
			t.Errorf("fit(%d,%d)=%d,%d want %d,%d", c[0], c[1], w, h, c[3], c[4])
		}
	}
}
