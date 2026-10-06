package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder (input only)
)

const (
	maxUploadBytes = 8 << 20 // 8 MiB accepted from the client
	maxSourceSide  = 8000    // reject absurd dimensions before decoding pixels
	maxSourcePixel = 40_000_000
	minSide        = 200
	maxOutputSide  = 1600
	jpegQuality    = 85
)

var (
	ErrUnsupportedType = errors.New("unsupported image type (JPEG, PNG or WebP only)")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrImageTooSmall   = errors.New("image too small (min 200px per side)")
	ErrCorruptImage    = errors.New("image could not be decoded")
)

type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates an untrusted upload and returns a clean JPEG: the type is
// sniffed from content (never from the client-declared name or MIME), pixel
// dimensions are checked before decoding, EXIF orientation is applied and then
// every piece of metadata (EXIF, GPS, comments) is dropped by re-encoding.
func Process(raw []byte) (Processed, error) {
	if len(raw) == 0 {
		return Processed{}, ErrCorruptImage
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return Processed{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePixel {
		return Processed{}, ErrImageTooLarge
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return Processed{}, ErrImageTooSmall
	}

	src, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if format == "jpeg" {
		src = applyOrientation(src, jpegOrientation(raw))
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxOutputSide || h > maxOutputSide {
		scale := float64(maxOutputSide) / float64(max(w, h))
		w, h = max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	}

	// Flatten onto white: JPEG has no alpha channel.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, fmt.Errorf("encode jpeg: %w", err)
	}
	return Processed{Data: out.Bytes(), Width: w, Height: h}, nil
}

// jpegOrientation reads the EXIF Orientation tag (1-8); 1 when absent.
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
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[i+4 : i+2+size]); o != 0 {
				return o
			}
		}
		i += 2 + size
	}
	return 1
}

func exifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := seg[6:]
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	for n := 0; n < count; n++ {
		off := ifd + 2 + n*12
		if off+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[off:off+2]) == 0x0112 {
			v := int(order.Uint16(tiff[off+8 : off+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// applyOrientation rotates/flips src so it displays upright.
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	outW, outH := w, h
	if orientation >= 5 {
		outW, outH = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, outW, outH))
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

var _ = png.Decode // register the PNG decoder explicitly
