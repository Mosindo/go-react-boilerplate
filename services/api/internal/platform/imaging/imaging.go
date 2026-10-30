// Package imaging validates and normalizes user-uploaded pictures. Everything
// is decoded and re-encoded as JPEG, which drops metadata (GPS EXIF included)
// and any payload hidden in the original file.
package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
)

const (
	MaxInputBytes = 8 << 20
	MaxSide       = 1600
	MaxPixels     = 40_000_000
	jpegQuality   = 85
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrTooLarge        = errors.New("image too large")
	ErrCorrupt         = errors.New("corrupt image")
)

type Result struct {
	Data   []byte
	Width  int
	Height int
}

// Process reads at most MaxInputBytes from r, accepts only JPEG and PNG
// (verified by content, never by client-declared type) and returns a
// normalized JPEG no larger than MaxSide on its longest edge.
func Process(r io.Reader) (Result, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxInputBytes+1))
	if err != nil {
		return Result{}, err
	}
	if len(raw) > MaxInputBytes {
		return Result{}, ErrTooLarge
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrUnsupportedType
	}
	if format != "jpeg" && format != "png" {
		return Result{}, ErrUnsupportedType
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxPixels {
		return Result{}, ErrTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrCorrupt
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(raw))
	}
	img = flattenOnWhite(downscale(img, MaxSide))

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Result{}, err
	}
	b := img.Bounds()
	return Result{Data: out.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// downscale shrinks img with a box filter so its longest side is at most max.
func downscale(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}
	scale := float64(max) / float64(w)
	if h > w {
		scale = float64(max) / float64(h)
	}
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
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
			dst.SetRGBA(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(bl / n >> 8), uint8(a / n >> 8)})
		}
	}
	return dst
}

// flattenOnWhite removes transparency, which JPEG cannot represent.
func flattenOnWhite(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

// jpegOrientation reads the EXIF orientation tag (1-8); 1 means upright.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 1
		}
		marker := data[pos+1]
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		size := int(data[pos+2])<<8 | int(data[pos+3])
		if size < 2 || pos+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 && size >= 16 && string(data[pos+4:pos+10]) == "Exif\x00\x00" {
			return exifOrientation(data[pos+10 : pos+2+size])
		}
		pos += 2 + size
	}
	return 1
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var u16 func([]byte) int
	var u32 func([]byte) int
	switch string(tiff[:2]) {
	case "II":
		u16 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 }
		u32 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) int { return int(b[0])<<8 | int(b[1]) }
		u32 = func(b []byte) int { return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3]) }
	default:
		return 1
	}
	ifd := u32(tiff[4:8])
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	count := u16(tiff[ifd : ifd+2])
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12
		if entry+12 > len(tiff) {
			return 1
		}
		if u16(tiff[entry:entry+2]) == 0x0112 {
			if v := u16(tiff[entry+8 : entry+10]); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation bakes an EXIF orientation into the pixels.
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := w, h
	if orientation >= 5 {
		nw, nh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
