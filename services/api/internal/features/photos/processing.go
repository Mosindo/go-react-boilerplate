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
	MaxUploadBytes = 10 << 20
	maxSourcePixel = 8000
	maxSourceArea  = 40_000_000
	minSourceSide  = 200
	maxOutputSide  = 1600
	jpegQuality    = 82
)

var (
	ErrUnsupportedType = apperr.BadRequest("unsupported_image", "only JPEG, PNG or WebP images are accepted")
	ErrImageTooLarge   = apperr.BadRequest("image_too_large", "image must be 10 MB or less")
	ErrImageDimensions = apperr.BadRequest("invalid_image_dimensions", "image must be between 200 and 8000 pixels per side")
	ErrCorruptImage    = apperr.BadRequest("corrupt_image", "the image could not be read")
)

var allowedFormats = map[string]string{
	"image/jpeg": "jpeg",
	"image/png":  "png",
	"image/webp": "webp",
}

// ProcessedImage is a sanitised JPEG ready to be stored.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// ProcessImage validates an upload by its magic bytes (never the client's
// declared type), bounds its dimensions to defeat decompression bombs, then
// re-encodes it as JPEG. Re-encoding strips every metadata block (EXIF GPS
// position included) and neutralises polyglot files.
func ProcessImage(data []byte) (ProcessedImage, error) {
	if len(data) == 0 {
		return ProcessedImage{}, ErrCorruptImage
	}
	if len(data) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	sniffLen := len(data)
	if sniffLen > 512 {
		sniffLen = 512
	}
	expectedFormat, ok := allowedFormats[http.DetectContentType(data[:sniffLen])]
	if !ok {
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != expectedFormat {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width < minSourceSide || cfg.Height < minSourceSide || cfg.Width > maxSourcePixel || cfg.Height > maxSourcePixel || cfg.Width*cfg.Height > maxSourceArea {
		return ProcessedImage{}, ErrImageDimensions
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}

	resized := resizeToFit(src, maxOutputSide)
	orientation := 1
	if format == "jpeg" {
		orientation = jpegOrientation(data)
	}
	final := applyOrientation(resized, orientation)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, final, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, err
	}
	b := final.Bounds()
	return ProcessedImage{Data: out.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

func resizeToFit(src image.Image, maxSide int) *image.RGBA {
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
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

// applyOrientation bakes the EXIF orientation into the pixels so photos are
// displayed upright once metadata is stripped.
func applyOrientation(src *image.RGBA, orientation int) *image.RGBA {
	if orientation < 2 || orientation > 8 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if orientation >= 5 {
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
			dst.SetRGBA(dx, dy, src.RGBAAt(x, y))
		}
	}
	return dst
}

// jpegOrientation reads the EXIF orientation tag (0x0112) from a JPEG.
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
		if marker == 0xD9 || marker == 0xDA {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 {
			segment := data[i+4 : i+2+size]
			if len(segment) >= 6 && string(segment[:6]) == "Exif\x00\x00" {
				return tiffOrientation(segment[6:])
			}
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	offset := int(order.Uint32(t[4:8]))
	if offset < 0 || offset+2 > len(t) {
		return 1
	}
	entries := int(order.Uint16(t[offset : offset+2]))
	for k := 0; k < entries; k++ {
		entry := offset + 2 + k*12
		if entry+12 > len(t) {
			return 1
		}
		if order.Uint16(t[entry:entry+2]) == 0x0112 {
			value := int(order.Uint16(t[entry+8 : entry+10]))
			if value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
	}
	return 1
}
