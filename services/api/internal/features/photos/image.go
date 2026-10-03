package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
)

const (
	maxUploadBytes    = 8 << 20
	maxSourcePixels   = 40_000_000
	maxOutputEdge     = 1280
	jpegQuality       = 82
	outputContentType = "image/jpeg"
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrInvalidImage    = errors.New("the file is not a valid image")
	ErrImageTooLarge   = errors.New("image dimensions are too large")
)

type processedImage struct {
	data          []byte
	width, height int
}

// processImage sniffs the real content type (never trusting the client), bounds the
// decoded size to defuse decompression bombs, downsizes, and re-encodes as JPEG. Re-encoding
// drops EXIF metadata (including GPS) and any payload appended to the original file.
func processImage(raw []byte) (processedImage, error) {
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return processedImage{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processedImage{}, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return processedImage{}, ErrImageTooLarge
	}

	var src image.Image
	if http.DetectContentType(raw) == "image/png" {
		src, err = png.Decode(bytes.NewReader(raw))
	} else {
		src, err = jpeg.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return processedImage{}, ErrInvalidImage
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if longest := max(w, h); longest > maxOutputEdge {
		w, h = w*maxOutputEdge/longest, h*maxOutputEdge/longest
		w, h = max(w, 1), max(h, 1)
	}

	// Flatten onto white so transparent PNGs do not turn black in JPEG.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return processedImage{}, err
	}
	return processedImage{data: out.Bytes(), width: w, height: h}, nil
}
