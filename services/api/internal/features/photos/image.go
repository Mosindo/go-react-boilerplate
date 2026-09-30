package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	apperr "example.com/api/internal/platform/errors"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	MaxUploadBytes  = 10 << 20 // 10 MiB
	maxSourcePixels = 40_000_000
	minSourceSide   = 200
	maxOutputSide   = 1440
	jpegQuality     = 82
)

var (
	ErrUnsupportedType = apperr.New(http.StatusUnsupportedMediaType, "unsupported_media_type", "only JPEG, PNG and WebP images are accepted")
	ErrImageTooSmall   = apperr.Validation("image is too small (minimum 200px per side)")
	ErrImageTooLarge   = apperr.New(http.StatusRequestEntityTooLarge, "image_too_large", "image is too large")
	ErrImageCorrupted  = apperr.Validation("image could not be decoded")
)

var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// ProcessImage validates an upload by its actual bytes (never the declared
// filename or content type), guards against decompression bombs, applies the
// EXIF orientation, downsizes, and re-encodes to JPEG. Re-encoding drops all
// metadata, including EXIF GPS coordinates, and neutralizes polyglot files.
func ProcessImage(raw []byte) (ProcessedImage, error) {
	if len(raw) == 0 {
		return ProcessedImage{}, ErrImageCorrupted
	}
	if len(raw) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	contentType := http.DetectContentType(raw)
	if !allowedTypes[contentType] {
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrImageCorrupted
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if cfg.Width < minSourceSide || cfg.Height < minSourceSide {
		return ProcessedImage{}, ErrImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrImageCorrupted
	}
	if contentType == "image/jpeg" {
		src = applyOrientation(src, jpegOrientation(raw))
	}

	dst := resizeToFit(src, maxOutputSide)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, err
	}
	bounds := dst.Bounds()
	return ProcessedImage{Data: out.Bytes(), Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

func resizeToFit(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		// Still copy into RGBA so every output goes through the same encoder.
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
		return dst
	}
	if w >= h {
		h = h * maxSide / w
		w = maxSide
	} else {
		w = w * maxSide / h
		h = maxSide
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

// jpegOrientation extracts the EXIF orientation tag (1-8) from a JPEG, or 1.
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
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		segLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if segLen < 2 || pos+2+segLen > len(data) {
			return 1
		}
		if marker == 0xE1 {
			seg := data[pos+4 : pos+2+segLen]
			if len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
				return tiffOrientation(seg[6:])
			}
		}
		pos += 2 + segLen
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
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
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	entries := int(order.Uint16(tiff[ifd : ifd+2]))
	for i := 0; i < entries; i++ {
		off := ifd + 2 + i*12
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

// applyOrientation returns the image transformed per EXIF orientation.
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
