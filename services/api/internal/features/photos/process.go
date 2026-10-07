package photos

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"golang.org/x/image/draw"
)

const (
	maxUploadBytes = 5 << 20
	maxPixels      = 40_000_000 // refuse decompression bombs before decoding
	minSide        = 200
	maxOutputSide  = 1440
	jpegQuality    = 82
)

var (
	errBadImage = platformerrors.Validation("the file must be a JPEG or PNG image")
	errTooSmall = platformerrors.Validation("the image must be at least 200x200 pixels")
	errTooBig   = platformerrors.Validation("the image dimensions are too large")
)

type processed struct {
	Data          []byte
	Width, Height int
}

// processImage validates the real content type (never trusting client headers or file
// names), bounds the pixel count, strips all metadata by re-encoding, and downsizes.
func processImage(raw []byte) (processed, error) {
	if len(raw) == 0 {
		return processed{}, errBadImage
	}
	if len(raw) > maxUploadBytes {
		return processed{}, platformerrors.Validation("the image must be at most 5 MB")
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return processed{}, errBadImage
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processed{}, errBadImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return processed{}, errTooBig
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return processed{}, errTooSmall
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return processed{}, errBadImage
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if longest := max(w, h); longest > maxOutputSide {
		scale := float64(maxOutputSide) / float64(longest)
		nw, nh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
		img, w, h = dst, nw, nh
	}

	// Flatten onto white so PNG transparency becomes predictable in JPEG.
	flat := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(flat, flat.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, img.Bounds().Min, draw.Over)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, flat, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return processed{}, errors.New("encode jpeg: " + err.Error())
	}
	return processed{Data: out.Bytes(), Width: w, Height: h}, nil
}
