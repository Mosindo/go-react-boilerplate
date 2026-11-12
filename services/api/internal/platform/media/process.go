package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooSmall   = errors.New("image too small")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrCorruptImage    = errors.New("corrupt image")
)

const (
	MaxUploadBytes    = 10 << 20
	minImageSide      = 200
	maxImageSide      = 12000
	maxImagePixels    = 50_000_000
	outputLongSide    = 1440
	outputJPEGQuality = 82
)

// ProcessedImage is a sanitized JPEG ready to store.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates and normalizes an uploaded image:
//   - the real type is sniffed from bytes (client-declared MIME is ignored);
//   - dimensions are checked before decoding to block decompression bombs;
//   - EXIF orientation is applied, then the image is resized and re-encoded,
//     which strips every metadata block (GPS coordinates included) and any
//     payload smuggled after the image data.
func Process(raw []byte) (ProcessedImage, error) {
	if len(raw) == 0 {
		return ProcessedImage{}, ErrCorruptImage
	}
	sniffed := http.DetectContentType(raw)

	var decodeConfig func([]byte) (image.Config, error)
	var decode func([]byte) (image.Image, error)
	switch sniffed {
	case "image/jpeg":
		decodeConfig = func(b []byte) (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return jpeg.Decode(bytes.NewReader(b)) }
	case "image/png":
		decodeConfig = func(b []byte) (image.Config, error) { return png.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }
	case "image/webp":
		decodeConfig = func(b []byte) (image.Config, error) { return webp.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return webp.Decode(bytes.NewReader(b)) }
	default:
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width < minImageSide || cfg.Height < minImageSide {
		return ProcessedImage{}, ErrImageTooSmall
	}
	if cfg.Width > maxImageSide || cfg.Height > maxImageSide || cfg.Width*cfg.Height > maxImagePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	src, err := decode(raw)
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}

	orientation := 1
	if sniffed == "image/jpeg" {
		orientation = jpegOrientation(raw)
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if orientation >= 5 {
		w, h = h, w
	}
	tw, th := fit(w, h, outputLongSide)

	// Draw onto an opaque white canvas so transparent PNG/WebP areas don't turn black.
	oriented := applyOrientation(src, orientation)
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, xdraw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), oriented, oriented.Bounds(), xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: outputJPEGQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: out.Bytes(), Width: tw, Height: th}, nil
}

func fit(w, h, longSide int) (int, int) {
	if w <= longSide && h <= longSide {
		return w, h
	}
	if w >= h {
		return longSide, max(1, h*longSide/w)
	}
	return max(1, w*longSide/h), longSide
}

// applyOrientation returns src transformed according to EXIF orientation 1-8.
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
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
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// jpegOrientation extracts the EXIF orientation tag (0x0112) from a JPEG, or 1.
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
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		segment := data[i+4 : i+2+size]
		if marker == 0xE1 && len(segment) > 14 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
		i += 2 + size
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
