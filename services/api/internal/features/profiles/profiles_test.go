package profiles

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
	"time"
)

func TestAgeAt(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		birth string
		want  int
	}{
		{"2008-06-15", 18}, // birthday today
		{"2008-06-16", 17}, // turns 18 tomorrow
		{"2008-06-14", 18},
		{"1990-12-31", 35},
		{"2000-02-29", 26},
	}
	for _, c := range cases {
		b, _ := time.Parse("2006-01-02", c.birth)
		if got := AgeAt(b, now); got != c.want {
			t.Errorf("AgeAt(%s) = %d, want %d", c.birth, got, c.want)
		}
	}
}

func TestValidateUpsertAgeRule(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) }}
	req := UpsertProfileRequest{FirstName: "Ana", Gender: GenderWoman, BirthDate: "2008-06-16"}
	if _, err := s.validateUpsert(t.Context(), req); !errors.Is(err, ErrUnderage) {
		t.Fatalf("17y364d must be underage, got %v", err)
	}
	req.BirthDate = "2008-06-15"
	if _, err := s.validateUpsert(t.Context(), req); err != nil {
		t.Fatalf("exactly 18 must pass: %v", err)
	}
}

func TestValidatePreferences(t *testing.T) {
	ok := UpdatePreferencesRequest{InterestedIn: []string{"man", "man", "woman"}, MinAge: 20, MaxAge: 30}
	p, err := ValidatePreferences(ok)
	if err != nil || len(p.InterestedIn) != 2 {
		t.Fatalf("expected dedup to 2 genders, got %+v %v", p, err)
	}
	for name, req := range map[string]UpdatePreferencesRequest{
		"no genders": {MinAge: 20, MaxAge: 30},
		"minor":      {InterestedIn: []string{"man"}, MinAge: 17, MaxAge: 30},
		"over 99":    {InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 100},
		"inverted":   {InterestedIn: []string{"man"}, MinAge: 31, MaxAge: 30},
		"distance":   {InterestedIn: []string{"man"}, MinAge: 20, MaxAge: 30, MaxDistanceKm: ptr(501)},
	} {
		if _, err := ValidatePreferences(req); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: expected ErrInvalidInput, got %v", name, err)
		}
	}
}

func ptr[T any](v T) *T { return &v }

func TestApproxDistanceKm(t *testing.T) {
	cases := map[float64]int{0: 1, 0.4: 1, 1: 1, 1.01: 5, 4.9: 5, 5: 5, 5.01: 10, 17.3: 20, 389: 390}
	for in, want := range cases {
		if got := ApproxDistanceKm(in); got != want {
			t.Errorf("ApproxDistanceKm(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestSnapCoordinate(t *testing.T) {
	cases := map[float64]float64{48.85661: 48.86, 2.3522: 2.35, -73.9857: -73.99, 0.004: 0, -0.004: 0}
	for in, want := range cases {
		if got := SnapCoordinate(in); got != want && !(got == 0 && want == 0) {
			t.Errorf("SnapCoordinate(%v) = %v, want %v", in, got, want)
		}
	}
}

func gradient(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), 90, 255})
		}
	}
	return img
}

func TestProcessImage(t *testing.T) {
	var jpg, pngData, transparent bytes.Buffer
	_ = jpeg.Encode(&jpg, gradient(2400, 1600), nil)
	_ = png.Encode(&pngData, gradient(400, 400))
	_ = png.Encode(&transparent, image.NewNRGBA(image.Rect(0, 0, 300, 300)))

	out, err := ProcessImage(jpg.Bytes())
	if err != nil || out.Width != 1280 || out.Height != 853 {
		t.Fatalf("expected downscale to 1280x853, got %+v %v", out, err)
	}
	if _, _, err := image.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a decodable image: %v", err)
	}
	if out.Data[0] != 0xFF || out.Data[1] != 0xD8 {
		t.Fatal("output must be JPEG")
	}
	if out, err := ProcessImage(pngData.Bytes()); err != nil || out.Width != 400 {
		t.Fatalf("png should pass through at size: %+v %v", out, err)
	}
	flat, err := ProcessImage(transparent.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	img, _, _ := image.Decode(bytes.NewReader(flat.Data))
	if r, g, b, _ := img.At(10, 10).RGBA(); r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Fatal("transparent pixels must be flattened onto white")
	}

	bad := map[string][]byte{
		"empty":     nil,
		"text":      []byte("hello world, definitely not an image"),
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"gif":       append([]byte("GIF89a"), make([]byte, 100)...),
		"truncated": jpg.Bytes()[:200],
		"tiny":      func() []byte { var b bytes.Buffer; _ = jpeg.Encode(&b, gradient(50, 50), nil); return b.Bytes() }(),
	}
	for name, data := range bad {
		if _, err := ProcessImage(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := ProcessImage(make([]byte, MaxUploadBytes+1)); !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("oversize: got %v", err)
	}
}
