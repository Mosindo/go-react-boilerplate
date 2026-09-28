// Package imaging validates uploaded photos by content and re-encodes them to
// JPEG using only the standard library. Re-encoding drops every metadata
// segment (EXIF, GPS, ICC, comments) because only pixels survive a decode.
package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder; GIF/WebP/etc. are intentionally not registered.
	"net/http"
)

const (
	// MaxSide is the longest edge of a stored photo.
	MaxSide = 1280
	// MaxSourcePixels bounds decode memory (decompression bombs).
	MaxSourcePixels = 16_000_000
	// JPEGQuality of the stored image.
	JPEGQuality = 85
)

var (
	ErrUnsupported = errors.New("only JPEG and PNG images are accepted")
	ErrCorrupt     = errors.New("image could not be decoded")
	ErrTooLarge    = errors.New("image dimensions are too large")
)

// Result is a sanitised JPEG.
type Result struct {
	JPEG          []byte
	Width, Height int
}

// Process sniffs data by content, decodes it, applies the EXIF orientation of
// JPEGs, downsizes it to MaxSide and re-encodes it as JPEG. The declared
// filename / Content-Type of the upload are never consulted.
func Process(data []byte) (Result, error) {
	if len(data) == 0 {
		return Result{}, ErrUnsupported
	}
	sniffed := http.DetectContentType(data)
	if sniffed != "image/jpeg" && sniffed != "image/png" {
		return Result{}, ErrUnsupported
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Result{}, ErrCorrupt
	}
	if (format == "jpeg") != (sniffed == "image/jpeg") {
		return Result{}, ErrUnsupported
	}
	if cfg.Width < 1 || cfg.Height < 1 {
		return Result{}, ErrCorrupt
	}
	if cfg.Width > 20000 || cfg.Height > 20000 || cfg.Width*cfg.Height > MaxSourcePixels {
		return Result{}, ErrTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Result{}, ErrCorrupt
	}

	rgba := Downscale(img, MaxSide)
	if format == "jpeg" {
		if o := JPEGOrientation(data); o > 1 {
			rgba = applyOrientation(rgba, o)
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return Result{}, err
	}
	b := rgba.Bounds()
	return Result{JPEG: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// Downscale shrinks img so its longest side is at most maxSide using an
// area-averaging (box) filter, compositing transparency over white because
// JPEG has no alpha channel. Images already small enough are only flattened.
func Downscale(src image.Image, maxSide int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dw, dh := sw, sh
	if sw > maxSide || sh > maxSide {
		if sw >= sh {
			dw = maxSide
			dh = int(float64(sh) * float64(maxSide) / float64(sw))
		} else {
			dh = maxSide
			dw = int(float64(sw) * float64(maxSide) / float64(sh))
		}
		if dw < 1 {
			dw = 1
		}
		if dh < 1 {
			dh = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for dy := 0; dy < dh; dy++ {
		y0 := dy * sh / dh
		y1 := (dy + 1) * sh / dh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for dx := 0; dx < dw; dx++ {
			x0 := dx * sw / dw
			x1 := (dx + 1) * sw / dw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, n uint64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					cr, cg, cb, ca := src.At(sb.Min.X+x, sb.Min.Y+y).RGBA()
					// premultiplied over white
					r += uint64(cr + (0xffff - ca))
					g += uint64(cg + (0xffff - ca))
					b += uint64(cb + (0xffff - ca))
					n++
				}
			}
			dst.SetRGBA(dx, dy, color.RGBA{
				R: uint8((r / n) >> 8),
				G: uint8((g / n) >> 8),
				B: uint8((b / n) >> 8),
				A: 255,
			})
		}
	}
	return dst
}

// JPEGOrientation extracts the EXIF orientation (1..8) from a JPEG, or 1 when
// absent or malformed. It reads only APP1 segments before the image data.
func JPEGOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		if segLen < 2 || i+2+segLen > len(data) {
			return 1
		}
		if marker == 0xE1 {
			seg := data[i+4 : i+2+segLen]
			if o := exifOrientation(seg); o != 0 {
				return o
			}
		}
		i += 2 + segLen
	}
	return 1
}

func exifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	t := seg[6:]
	var u16 func([]byte) int
	var u32 func([]byte) int
	switch string(t[:2]) {
	case "II":
		u16 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 }
		u32 = func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) int { return int(b[0])<<8 | int(b[1]) }
		u32 = func(b []byte) int { return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3]) }
	default:
		return 0
	}
	if u16(t[2:4]) != 42 {
		return 0
	}
	off := u32(t[4:8])
	if off < 8 || off+2 > len(t) {
		return 0
	}
	count := u16(t[off : off+2])
	for n := 0; n < count; n++ {
		e := off + 2 + n*12
		if e+12 > len(t) {
			return 0
		}
		if u16(t[e:e+2]) == 0x0112 {
			v := u16(t[e+8 : e+10])
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// applyOrientation transforms pixels so the image displays upright.
func applyOrientation(src *image.RGBA, o int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	swap := o >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			default:
				nx, ny = x, y
			}
			dst.SetRGBA(nx, ny, src.RGBAAt(x, y))
		}
	}
	return dst
}
