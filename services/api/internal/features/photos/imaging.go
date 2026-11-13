package photos

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

var (
	ErrUnsupportedType = errors.New("only JPEG, PNG and WebP images are accepted")
	ErrImageTooSmall   = errors.New("image must be at least 320x320 pixels")
	ErrImageTooLarge   = errors.New("image dimensions are too large")
	ErrFileTooLarge    = errors.New("image must be at most 10 MB")
	ErrCorruptImage    = errors.New("image could not be decoded")
)

const (
	MaxUploadBytes    = 10 << 20
	minDimension      = 320
	maxSourcePixels   = 40_000_000 // reject decompression bombs before decoding
	maxOutputEdge     = 1600
	outputJPEGQuality = 82
)

// ProcessedImage is a sanitized JPEG ready to be stored.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// sniff detects the real content type from magic bytes; the client-provided
// filename and Content-Type are never trusted.
func sniff(data []byte) string {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	return http.DetectContentType(head)
}

// ProcessUpload validates and re-encodes an image. Re-encoding strips every
// metadata block (EXIF GPS position, camera serials, embedded payloads) and
// guarantees the stored file is a plain JPEG whatever was uploaded.
func ProcessUpload(r io.Reader) (ProcessedImage, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return ProcessedImage{}, err
	}
	if len(data) > MaxUploadBytes {
		return ProcessedImage{}, ErrFileTooLarge
	}
	if len(data) == 0 {
		return ProcessedImage{}, ErrCorruptImage
	}

	var decodeConfig func(io.Reader) (image.Config, error)
	var decode func(io.Reader) (image.Image, error)
	switch sniff(data) {
	case "image/jpeg":
		decodeConfig, decode = jpeg.DecodeConfig, jpeg.Decode
	case "image/png":
		decodeConfig, decode = png.DecodeConfig, png.Decode
	case "image/webp":
		decodeConfig, decode = webp.DecodeConfig, webp.Decode
	default:
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, err := decodeConfig(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width < minDimension || cfg.Height < minDimension {
		return ProcessedImage{}, ErrImageTooSmall
	}
	if cfg.Width*cfg.Height > maxSourcePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	src, err := decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}

	w, h := fitWithin(src.Bounds().Dx(), src.Bounds().Dy(), maxOutputEdge)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// Flatten transparency onto white before JPEG encoding.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: outputJPEGQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: out.Bytes(), Width: w, Height: h}, nil
}

func fitWithin(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		return maxEdge, max(1, h*maxEdge/w)
	}
	return max(1, w*maxEdge/h), maxEdge
}
