package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	MaxUploadBytes = 8 << 20
	fullMaxEdge    = 1080
	thumbMaxEdge   = 320
	minEdge        = 200
	maxPixels      = 40_000_000
	jpegQuality    = 85
)

var (
	ErrUnsupportedImage = errors.New("unsupported image type")
	ErrImageTooLarge    = errors.New("image too large")
	ErrImageTooSmall    = errors.New("image too small")
	ErrInvalidImage     = errors.New("invalid image")
)

// ProcessedImage is the sanitised output stored on disk.
type ProcessedImage struct {
	Full, Thumb   []byte
	Width, Height int
}

// ProcessImage validates an untrusted upload and re-encodes it. Re-encoding as JPEG
// guarantees the stored bytes are an image we produced (no polyglot payloads, no
// metadata such as EXIF GPS), and bounds the size of what we serve.
func ProcessImage(r io.Reader) (ProcessedImage, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if len(data) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return ProcessedImage{}, ErrUnsupportedImage
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if cfg.Width < minEdge || cfg.Height < minEdge {
		return ProcessedImage{}, ErrImageTooSmall
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}

	full := fit(img, fullMaxEdge)
	thumb := fit(full, thumbMaxEdge)
	fullBytes, err := encodeJPEG(full)
	if err != nil {
		return ProcessedImage{}, err
	}
	thumbBytes, err := encodeJPEG(thumb)
	if err != nil {
		return ProcessedImage{}, err
	}
	b := full.Bounds()
	return ProcessedImage{Full: fullBytes, Thumb: thumbBytes, Width: b.Dx(), Height: b.Dy()}, nil
}

// fit scales img down so its longest edge is at most maxEdge, flattening transparency onto white.
func fit(img image.Image, maxEdge int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxEdge || h > maxEdge {
		if w >= h {
			h = h * maxEdge / w
			w = maxEdge
		} else {
			w = w * maxEdge / h
			h = maxEdge
		}
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
