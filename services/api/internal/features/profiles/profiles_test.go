package profiles

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) }

func validRequest() UpsertProfileRequest {
	r := UpsertProfileRequest{FirstName: " Léa ", BirthDate: "1995-06-15", Gender: "woman", Bio: "hi", City: "Lyon", Interests: []string{"travel", "travel", "music"}}
	r.Preferences.InterestedIn = []string{"man", "man"}
	r.Preferences.AgeMin, r.Preferences.AgeMax, r.Preferences.MaxDistanceKm = 25, 40, 30
	return r
}

func TestAgeOnBoundaries(t *testing.T) {
	now := fixedNow()
	cases := map[string]int{"2008-06-15": 18, "2008-06-16": 17, "1995-06-15": 31, "1995-06-14": 31, "1995-06-16": 30}
	for birth, want := range cases {
		b, _ := time.Parse("2006-01-02", birth)
		if got := AgeOn(b, now); got != want {
			t.Errorf("AgeOn(%s) = %d, want %d", birth, got, want)
		}
	}
}

func TestValidateNormalisesAndRejects(t *testing.T) {
	s := &Service{now: fixedNow}
	in, err := s.validate(validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if in.FirstName != "Léa" || len(in.InterestSlugs) != 2 || len(in.Preferences.InterestedIn) != 1 || !in.ShowDistance || !in.Discoverable {
		t.Fatalf("unexpected normalisation: %+v", in)
	}

	mutations := map[string]func(*UpsertProfileRequest){
		"minor":        func(r *UpsertProfileRequest) { r.BirthDate = "2008-06-16" },
		"too old":      func(r *UpsertProfileRequest) { r.BirthDate = "1900-01-01" },
		"bad date":     func(r *UpsertProfileRequest) { r.BirthDate = "15/06/1995" },
		"empty name":   func(r *UpsertProfileRequest) { r.FirstName = "  " },
		"long name":    func(r *UpsertProfileRequest) { r.FirstName = strings.Repeat("a", 41) },
		"ctrl name":    func(r *UpsertProfileRequest) { r.FirstName = "a\x00b" },
		"gender":       func(r *UpsertProfileRequest) { r.Gender = "x" },
		"long bio":     func(r *UpsertProfileRequest) { r.Bio = strings.Repeat("a", 501) },
		"half coords":  func(r *UpsertProfileRequest) { v := 1.0; r.Latitude = &v },
		"bad coords":   func(r *UpsertProfileRequest) { a, b := 91.0, 0.0; r.Latitude, r.Longitude = &a, &b },
		"no interest":  func(r *UpsertProfileRequest) { r.Preferences.InterestedIn = nil },
		"bad interest": func(r *UpsertProfileRequest) { r.Preferences.InterestedIn = []string{"alien"} },
		"age < 18":     func(r *UpsertProfileRequest) { r.Preferences.AgeMin = 16 },
		"age range":    func(r *UpsertProfileRequest) { r.Preferences.AgeMin, r.Preferences.AgeMax = 40, 30 },
		"distance":     func(r *UpsertProfileRequest) { r.Preferences.MaxDistanceKm = 501 },
		"too many":     func(r *UpsertProfileRequest) { r.Interests = strings.Split("a,b,c,d,e,f,g,h,i,j,k", ",") },
	}
	for name, mutate := range mutations {
		req := validRequest()
		mutate(&req)
		if _, err := s.validate(req); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: expected validation error, got %v", name, err)
		}
	}
	req := validRequest()
	req.Preferences.MaxDistanceKm = AnyDistance
	if _, err := s.validate(req); err != nil {
		t.Errorf("any-distance sentinel must be accepted: %v", err)
	}
}

func TestCoordinatesAreCoarsened(t *testing.T) {
	s := &Service{now: fixedNow}
	req := validRequest()
	la, lo := 48.856614, 2.352222
	req.Latitude, req.Longitude = &la, &lo
	in, _ := s.validate(req)
	if *in.Latitude != 48.86 || *in.Longitude != 2.35 {
		t.Fatalf("coordinates not rounded: %v,%v", *in.Latitude, *in.Longitude)
	}
}

func TestApproxDistanceAndHaversine(t *testing.T) {
	cases := map[float64]int{0: 5, 0.4: 5, 5: 5, 5.1: 10, 9.1: 10, 12: 15, 47: 50}
	for km, want := range cases {
		if got := ApproxDistance(km); got != want {
			t.Errorf("ApproxDistance(%v) = %d, want %d", km, got, want)
		}
	}
	d := DistanceKm(48.8566, 2.3522, 51.5074, -0.1278) // Paris - London
	if d < 340 || d > 348 {
		t.Fatalf("Paris-London distance off: %v", d)
	}
}

func TestMediaSigner(t *testing.T) {
	m := NewMediaSigner([]byte("secret-secret-secret-secret-1234"))
	now := time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)
	m.now = func() time.Time { return now }
	url := m.URL("p1")
	if m.URL("p1") != url {
		t.Fatal("URL must be stable within an hour bucket")
	}
	var exp, sig string
	parts := strings.SplitN(url, "?", 2)
	for _, kv := range strings.Split(parts[1], "&") {
		switch {
		case strings.HasPrefix(kv, "e="):
			exp = kv[2:]
		case strings.HasPrefix(kv, "s="):
			sig = kv[2:]
		}
	}
	if !m.Verify("p1", exp, sig) || m.Verify("p2", exp, sig) || m.Verify("p1", exp, sig+"0") || m.Verify("p1", "1", sig) {
		t.Fatal("signature verification wrong")
	}
	m.now = func() time.Time { return now.Add(3 * time.Hour) }
	if m.Verify("p1", exp, sig) {
		t.Fatal("expired URL must be rejected")
	}
}

func encodePNG(w, h int, withAlpha bool) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	a := uint8(255)
	if withAlpha {
		a = 0
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{200, 10, 10, a})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestProcessImage(t *testing.T) {
	out, err := ProcessImage(encodePNG(3000, 1500, false))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1280 || out.Height != 640 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a valid JPEG: %v", err)
	}
	small, err := ProcessImage(encodePNG(300, 300, true))
	if err != nil || small.Width != 300 {
		t.Fatalf("small image must be kept as is: %v", err)
	}
	if _, err := ProcessImage(encodePNG(100, 100, false)); !errors.Is(err, ErrImageTooSmall) {
		t.Fatalf("expected too small, got %v", err)
	}
	for name, data := range map[string][]byte{
		"text":    []byte("hello world"),
		"html":    []byte("<html><script>alert(1)</script></html>"),
		"gif":     []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"empty":   nil,
		"truncat": encodePNG(400, 400, false)[:100],
	} {
		if _, err := ProcessImage(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}
