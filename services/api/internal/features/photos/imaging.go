package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register decoders
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	maxSidePx    = 8000
	maxPixels    = 25_000_000
	minSidePx    = 200
	fullMaxSide  = 1440
	thumbMaxSide = 360
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrBadImage        = errors.New("invalid image")
	ErrImageTooLarge   = errors.New("image dimensions too large")
)

type Processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// Process validates untrusted bytes and returns two clean JPEG renditions.
// Steps: content sniffing (never the client's filename or MIME), dimension check before decoding
// (decompression-bomb guard), full decode, resize, and JPEG re-encode which drops EXIF/metadata and
// any trailing payload.
func Process(data []byte) (Processed, error) {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	switch http.DetectContentType(head) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return Processed{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Processed{}, ErrBadImage
	}
	if cfg.Width > maxSidePx || cfg.Height > maxSidePx || cfg.Width*cfg.Height > maxPixels {
		return Processed{}, ErrImageTooLarge
	}
	if cfg.Width < minSidePx || cfg.Height < minSidePx {
		return Processed{}, ErrBadImage
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Processed{}, ErrBadImage
	}

	full, w, h, err := encodeFit(src, fullMaxSide, 85)
	if err != nil {
		return Processed{}, err
	}
	thumb, _, _, err := encodeFit(src, thumbMaxSide, 80)
	if err != nil {
		return Processed{}, err
	}
	return Processed{Full: full, Thumb: thumb, Width: w, Height: h}, nil
}

func encodeFit(src image.Image, maxSide, quality int) ([]byte, int, int, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxSide || h > maxSide {
		if w >= h {
			h = h * maxSide / w
			w = maxSide
		} else {
			w = w * maxSide / h
			h = maxSide
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	// Flatten onto white so transparent PNGs do not turn black in JPEG.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}
