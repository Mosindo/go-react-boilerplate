package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	MaxUploadBytes = 10 << 20
	maxSourceSide  = 8000
	maxSourcePix   = 40_000_000
	minSourceSide  = 200
	maxOutputSide  = 1600
	jpegQuality    = 82
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrInvalidImage    = errors.New("invalid image")
	ErrImageTooSmall   = errors.New("image too small")
	ErrImageTooLarge   = errors.New("image too large")
)

var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates an uploaded image by content (never by file name or
// client-declared type), guards against decompression bombs, applies the
// EXIF orientation, downsizes, and re-encodes to JPEG. Re-encoding drops
// every metadata block (EXIF GPS position, device info, embedded payloads).
func Process(raw []byte) (Processed, error) {
	if len(raw) == 0 {
		return Processed{}, ErrInvalidImage
	}
	if len(raw) > MaxUploadBytes {
		return Processed{}, ErrImageTooLarge
	}
	if !allowedTypes[http.DetectContentType(raw)] {
		return Processed{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidImage
	}
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePix {
		return Processed{}, ErrImageTooLarge
	}
	if cfg.Width < minSourceSide || cfg.Height < minSourceSide {
		return Processed{}, ErrImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrInvalidImage
	}
	src = applyOrientation(src, jpegOrientation(raw))

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if longest := max(w, h); longest > maxOutputSide {
		scale := float64(maxOutputSide) / float64(longest)
		w = max(1, int(float64(w)*scale+0.5))
		h = max(1, int(float64(h)*scale+0.5))
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: out.Bytes(), Width: w, Height: h}, nil
}

// jpegOrientation extracts the EXIF orientation tag (1-8) from a JPEG, or 1.
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
		segment := data[i+4 : i+2+size]
		if marker == 0xE1 && len(segment) > 14 && string(segment[:6]) == "Exif\x00\x00" {
			return exifOrientation(segment[6:])
		}
		i += 2 + size
	}
	return 1
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd+2 > len(tiff) {
		return 1
	}
	entries := int(order.Uint16(tiff[ifd : ifd+2]))
	for e := 0; e < entries; e++ {
		off := ifd + 2 + e*12
		if off+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[off:off+2]) == 0x0112 {
			v := int(order.Uint16(tiff[off+8 : off+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation rotates/flips src according to an EXIF orientation value.
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := orientation >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
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
