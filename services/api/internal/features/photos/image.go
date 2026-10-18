package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
)

var (
	ErrUnsupportedImage = errors.New("only JPEG and PNG images are accepted")
	ErrImageTooSmall    = errors.New("image must be at least 200x200 pixels")
	ErrImageTooLarge    = errors.New("image dimensions are too large")
	ErrFileTooLarge     = errors.New("image must be smaller than 8 MB")
)

const (
	minSide      = 200
	maxSide      = 12000
	maxPixels    = 50_000_000
	fullMaxEdge  = 1280
	thumbMaxEdge = 360
	jpegQuality  = 85
)

type processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// processImage is the only path from untrusted bytes to stored files. It sniffs
// the real content type (never trusting the client), bounds dimensions before
// decoding to defuse decompression bombs, then re-encodes to JPEG. Re-encoding
// strips EXIF metadata (including GPS) and any polyglot payload.
func processImage(data []byte) (processed, error) {
	if len(data) > maxUploadBytes {
		return processed{}, ErrFileTooLarge
	}
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	switch http.DetectContentType(sniff) {
	case "image/jpeg", "image/png":
	default:
		return processed{}, ErrUnsupportedImage
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return processed{}, ErrUnsupportedImage
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return processed{}, ErrImageTooSmall
	}
	if cfg.Width > maxSide || cfg.Height > maxSide || cfg.Width*cfg.Height > maxPixels {
		return processed{}, ErrImageTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return processed{}, ErrUnsupportedImage
	}

	full := resizeWithin(flatten(src), fullMaxEdge)
	thumb := resizeWithin(full, thumbMaxEdge)

	fullBytes, err := encodeJPEG(full)
	if err != nil {
		return processed{}, err
	}
	thumbBytes, err := encodeJPEG(thumb)
	if err != nil {
		return processed{}, err
	}
	b := full.Bounds()
	return processed{Full: fullBytes, Thumb: thumbBytes, Width: b.Dx(), Height: b.Dy()}, nil
}

// flatten draws the image onto an opaque white canvas (JPEG has no alpha).
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func resizeWithin(src *image.RGBA, maxEdge int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	if w >= h {
		h = h * maxEdge / w
		w = maxEdge
	} else {
		w = w * maxEdge / h
		h = maxEdge
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
