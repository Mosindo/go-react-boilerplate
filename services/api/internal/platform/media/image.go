package media

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"

	xdraw "golang.org/x/image/draw"
)

const (
	// MaxUploadBytes is the largest accepted upload.
	MaxUploadBytes = 8 << 20
	// MaxDimension is the longest side of a stored photo.
	MaxDimension = 1280
	minDimension = 200
	// maxDecodePixels protects against decompression bombs.
	maxDecodePixels = 40_000_000
)

var (
	ErrNotImage   = errors.New("file is not a supported image (jpeg or png)")
	ErrTooLarge   = errors.New("image is too large")
	ErrTooSmall   = errors.New("image is too small")
	ErrUnreadable = errors.New("image could not be read")
)

// Processed is a sanitised JPEG ready to be stored.
type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates the real content type (magic bytes, not the client header),
// rejects decompression bombs, downsizes, and re-encodes as JPEG. Re-encoding drops
// EXIF (including GPS) and any payload appended to the file.
func Process(r io.Reader) (Processed, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return Processed{}, ErrUnreadable
	}
	if len(data) > MaxUploadBytes {
		return Processed{}, ErrTooLarge
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return Processed{}, ErrNotImage
	}
	if cfg.Width*cfg.Height > maxDecodePixels {
		return Processed{}, ErrTooLarge
	}
	if cfg.Width < minDimension || cfg.Height < minDimension {
		return Processed{}, ErrTooSmall
	}

	var src image.Image
	if format == "jpeg" {
		src, err = jpeg.Decode(bytes.NewReader(data))
	} else {
		src, err = png.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return Processed{}, ErrUnreadable
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > MaxDimension || h > MaxDimension {
		if w >= h {
			h = h * MaxDimension / w
			w = MaxDimension
		} else {
			w = w * MaxDimension / h
			h = MaxDimension
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// Flatten transparency onto white so PNGs become valid JPEGs.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 82}); err != nil {
		return Processed{}, ErrUnreadable
	}
	return Processed{Data: out.Bytes(), Width: w, Height: h}, nil
}
