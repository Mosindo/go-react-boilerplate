package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

const (
	MaxUploadBytes = 8 << 20
	fullMaxEdge    = 1080
	thumbMaxEdge   = 360
	// maxPixels caps decoded size so a tiny "decompression bomb" file cannot exhaust memory.
	maxPixels = 40_000_000
	maxEdge   = 10_000
	minEdge   = 200
)

var (
	ErrUnsupportedType = errors.New("unsupported image type (use JPEG, PNG or WebP)")
	ErrInvalidImage    = errors.New("invalid or corrupt image")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrImageTooSmall   = errors.New("image too small (minimum 200px)")
)

// Processed holds the re-encoded variants of an upload.
type Processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// Process validates raw upload bytes by content (never by filename or client MIME type),
// decodes them and re-encodes sanitised JPEGs: metadata (EXIF/GPS), embedded payloads and
// polyglot tricks are dropped because only decoded pixels survive.
func Process(raw []byte) (Processed, error) {
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return Processed{}, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidImage
	}
	if cfg.Width > maxEdge || cfg.Height > maxEdge || cfg.Width*cfg.Height > maxPixels {
		return Processed{}, ErrImageTooLarge
	}
	if cfg.Width < minEdge || cfg.Height < minEdge {
		return Processed{}, ErrImageTooSmall
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidImage
	}
	flat := flatten(src)
	full := fit(flat, fullMaxEdge)
	thumb := fit(flat, thumbMaxEdge)
	fullBytes, err := encode(full, 82)
	if err != nil {
		return Processed{}, err
	}
	thumbBytes, err := encode(thumb, 76)
	if err != nil {
		return Processed{}, err
	}
	b := full.Bounds()
	return Processed{Full: fullBytes, Thumb: thumbBytes, Width: b.Dx(), Height: b.Dy()}, nil
}

// flatten composites transparency over white so the JPEG output has no alpha artefacts.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func fit(src *image.RGBA, edge int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= edge && h <= edge {
		return src
	}
	nw, nh := edge, edge*h/w
	if h > w {
		nw, nh = edge*w/h, edge
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func encode(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
