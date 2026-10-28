package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const (
	maxUploadBytes  = 8 << 20
	maxSourceSide   = 12000
	maxSourcePixels = 50_000_000
	minSide         = 200
	maxOutputSide   = 1600
	jpegQuality     = 82
)

var (
	errUnsupportedType = errors.New("unsupported image type")
	errBadImage        = errors.New("invalid image")
)

type processed struct {
	Data          []byte
	Width, Height int
}

// processImage validates untrusted bytes by decoding them, then produces a clean JPEG:
// correct orientation, bounded size, and no metadata (EXIF/GPS) or trailing payload carried over.
func processImage(raw []byte) (processed, error) {
	if len(raw) == 0 || len(raw) > maxUploadBytes {
		return processed{}, errBadImage
	}
	sniffed := http.DetectContentType(raw)
	switch sniffed {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return processed{}, errUnsupportedType
	}

	// Check dimensions from the header before allocating pixels (decompression-bomb guard).
	cfg, err := decodeConfig(sniffed, raw)
	if err != nil {
		return processed{}, errBadImage
	}
	if cfg.Width < minSide || cfg.Height < minSide ||
		cfg.Width > maxSourceSide || cfg.Height > maxSourceSide ||
		cfg.Width*cfg.Height > maxSourcePixels {
		return processed{}, errBadImage
	}

	img, err := decodeImage(sniffed, raw)
	if err != nil {
		return processed{}, errBadImage
	}
	if sniffed == "image/jpeg" {
		img = applyOrientation(img, jpegOrientation(raw))
	}
	flat := fit(flatten(img), maxOutputSide)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return processed{}, err
	}
	b := flat.Bounds()
	return processed{Data: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

func decodeConfig(kind string, raw []byte) (image.Config, error) {
	r := bytes.NewReader(raw)
	switch kind {
	case "image/jpeg":
		return jpeg.DecodeConfig(r)
	case "image/png":
		return png.DecodeConfig(r)
	default:
		return webp.DecodeConfig(r)
	}
}

func decodeImage(kind string, raw []byte) (image.Image, error) {
	r := bytes.NewReader(raw)
	switch kind {
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	default:
		return webp.Decode(r)
	}
}

// flatten draws the image over white so transparency survives the JPEG conversion predictably.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func fit(src *image.RGBA, maxSide int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	if w >= h {
		h = h * maxSide / w
		w = maxSide
	} else {
		w = w * maxSide / h
		h = maxSide
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

// jpegOrientation reads the EXIF orientation tag (1..8) from a JPEG; 1 when absent or malformed.
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
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image: no more headers
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if size < 2 || pos+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[pos+4 : pos+2+size]); o != 0 {
				return o
			}
		}
		pos += 2 + size
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
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 {
			v := int(order.Uint16(tiff[entry+8 : entry+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// applyOrientation rotates/flips pixels so the stored JPEG (which has no EXIF) looks upright.
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := orientation >= 5
	var dst *image.RGBA
	if swap {
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
	}
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
