package photos

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	maxUploadBytes = 8 << 20 // 8 MiB per upload
	maxPixels      = 40_000_000
	fullMaxSide    = 1600
	thumbMaxSide   = 400
	jpegQuality    = 82
)

var (
	ErrUnsupportedImage = errors.New("unsupported image: use JPEG, PNG or WebP")
	ErrImageTooLarge    = errors.New("image dimensions are too large")
)

type processed struct {
	full, thumb   []byte
	width, height int
}

// process validates untrusted bytes and returns sanitised JPEG renditions.
// The input is fully decoded and re-encoded, which strips EXIF/GPS metadata and
// neutralises polyglot or malformed files. The declared content type is never trusted.
func process(data []byte) (processed, error) {
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return processed{}, ErrUnsupportedImage
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return processed{}, ErrUnsupportedImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return processed{}, ErrImageTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return processed{}, ErrUnsupportedImage
	}
	if http.DetectContentType(data) == "image/jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}

	full := fit(img, fullMaxSide)
	thumb := fit(img, thumbMaxSide)
	fullBytes, err := encodeJPEG(full)
	if err != nil {
		return processed{}, err
	}
	thumbBytes, err := encodeJPEG(thumb)
	if err != nil {
		return processed{}, err
	}
	b := full.Bounds()
	return processed{full: fullBytes, thumb: thumbBytes, width: b.Dx(), height: b.Dy()}, nil
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fit downsizes so the longest side is at most maxSide; it never upscales.
// Transparency is flattened onto white since JPEG has no alpha.
func fit(src image.Image, maxSide int) *image.RGBA {
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
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// jpegOrientation extracts the EXIF orientation tag (1-8); 1 when absent.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		segLen := int(data[i+2])<<8 | int(data[i+3])
		if segLen < 2 || i+2+segLen > len(data) {
			return 1
		}
		if marker == 0xE1 && segLen >= 16 && string(data[i+4:i+10]) == "Exif\x00\x00" {
			return exifOrientation(data[i+10 : i+2+segLen])
		}
		i += 2 + segLen
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
	off := u32(tiff[4:8])
	if off < 8 || off+2 > len(tiff) {
		return 1
	}
	n := u16(tiff[off : off+2])
	for k := 0; k < n; k++ {
		e := off + 2 + k*12
		if e+12 > len(tiff) {
			return 1
		}
		if u16(tiff[e:e+2]) == 0x0112 {
			if v := u16(tiff[e+8 : e+10]); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation rotates/flips pixels so the image displays upright once EXIF is stripped.
func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
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
			}
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
