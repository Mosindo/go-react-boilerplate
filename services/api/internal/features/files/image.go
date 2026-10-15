package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

var (
	errUnsupportedImage = errors.New("unsupported image")
	errImageTooLarge    = errors.New("image dimensions too large")
	errImageTooSmall    = errors.New("image too small")
)

type processed struct {
	Full   []byte
	Thumb  []byte
	Width  int
	Height int
}

// processImage validates untrusted bytes as an image and returns clean JPEGs.
// Everything is re-encoded, which strips EXIF (GPS!), ICC profiles, comments
// and any payload appended after the image data.
func processImage(data []byte) (processed, error) {
	sniffLen := len(data)
	if sniffLen > 512 {
		sniffLen = 512
	}
	switch http.DetectContentType(data[:sniffLen]) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return processed{}, errUnsupportedImage
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return processed{}, errUnsupportedImage
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension || cfg.Width*cfg.Height > maxPixels {
		return processed{}, errImageTooLarge
	}
	if cfg.Width < minDimension || cfg.Height < minDimension {
		return processed{}, errImageTooSmall
	}

	img, err := decode(data)
	if err != nil {
		return processed{}, errUnsupportedImage
	}
	img = applyOrientation(img, exifOrientation(data))
	img = flatten(img)

	full, err := encodeScaled(img, fullSize)
	if err != nil {
		return processed{}, err
	}
	thumb, err := encodeScaled(img, thumbSize)
	if err != nil {
		return processed{}, err
	}
	scaled := fit(img.Bounds(), fullSize)
	return processed{Full: full, Thumb: thumb, Width: scaled.Dx(), Height: scaled.Dy()}, nil
}

func decode(data []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// flatten composites transparency onto white so JPEG output has no black halos.
func flatten(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func fit(b image.Rectangle, longSide int) image.Rectangle {
	w, h := b.Dx(), b.Dy()
	if w <= longSide && h <= longSide {
		return image.Rect(0, 0, w, h)
	}
	if w >= h {
		return image.Rect(0, 0, longSide, max(1, h*longSide/w))
	}
	return image.Rect(0, 0, max(1, w*longSide/h), longSide)
}

func encodeScaled(src image.Image, longSide int) ([]byte, error) {
	target := fit(src.Bounds(), longSide)
	out := src
	if target.Dx() != src.Bounds().Dx() || target.Dy() != src.Bounds().Dy() {
		dst := image.NewRGBA(target)
		xdraw.CatmullRom.Scale(dst, target, src, src.Bounds(), xdraw.Over, nil)
		out = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// exifOrientation reads the EXIF orientation tag (1-8) of a JPEG; 1 when absent.
func exifOrientation(data []byte) int {
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
		segLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if segLen < 2 || pos+2+segLen > len(data) {
			return 1
		}
		if marker == 0xE1 {
			seg := data[pos+4 : pos+2+segLen]
			if len(seg) >= 14 && string(seg[:6]) == "Exif\x00\x00" {
				return parseTIFFOrientation(seg[6:])
			}
		}
		pos += 2 + segLen
	}
	return 1
}

func parseTIFFOrientation(tiff []byte) int {
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
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12
		if entry+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 {
			v := int(order.Uint16(tiff[entry+8 : entry+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation rotates/mirrors the pixels so the image is upright without EXIF.
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
			var nx, ny int
			switch orientation {
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
