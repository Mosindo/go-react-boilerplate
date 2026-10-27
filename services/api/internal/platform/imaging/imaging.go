// Package imaging validates and normalizes user uploaded pictures. Every upload
// is fully decoded and re-encoded as a bounded JPEG, which strips metadata (EXIF
// GPS included) and neutralizes polyglot or malformed payloads.
package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
)

var (
	ErrUnsupported = errors.New("unsupported image format")
	ErrTooLarge    = errors.New("image dimensions too large")
	ErrTooSmall    = errors.New("image too small")
	ErrCorrupt     = errors.New("invalid image")
)

const (
	MaxUploadBytes = 8 << 20
	MaxSourceSide  = 10000
	MinSide        = 200
	MaxOutputSide  = 1280
	jpegQuality    = 85
)

type Result struct {
	Data   []byte
	Width  int
	Height int
}

// Normalize accepts JPEG or PNG bytes and returns a metadata-free JPEG.
func Normalize(raw []byte) (Result, error) {
	if len(raw) == 0 || len(raw) > MaxUploadBytes {
		return Result{}, ErrCorrupt
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrUnsupported
	}
	if format != "jpeg" && format != "png" {
		return Result{}, ErrUnsupported
	}
	if cfg.Width > MaxSourceSide || cfg.Height > MaxSourceSide {
		return Result{}, ErrTooLarge
	}
	if cfg.Width < MinSide || cfg.Height < MinSide {
		return Result{}, ErrTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrCorrupt
	}

	out := fit(src, MaxOutputSide)
	// Flatten onto white so transparent PNGs do not turn black in JPEG.
	canvas := image.NewRGBA(out.Bounds())
	draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), out, out.Bounds().Min, draw.Over)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, canvas, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Result{}, err
	}
	b := canvas.Bounds()
	return Result{Data: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// fit downsizes with a box filter so the longest side is at most max.
func fit(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}
	var nw, nh int
	if w >= h {
		nw, nh = max, h*max/w
	} else {
		nw, nh = w*max/h, max
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy0 := b.Min.Y + y*h/nh
		sy1 := b.Min.Y + (y+1)*h/nh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < nw; x++ {
			sx0 := b.Min.X + x*w/nw
			sx1 := b.Min.X + (x+1)*w/nw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, bl, a, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r += uint64(cr)
					g += uint64(cg)
					bl += uint64(cb)
					a += uint64(ca)
					n++
				}
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i+0] = uint8(r / n >> 8)
			dst.Pix[i+1] = uint8(g / n >> 8)
			dst.Pix[i+2] = uint8(bl / n >> 8)
			dst.Pix[i+3] = uint8(a / n >> 8)
		}
	}
	return dst
}
