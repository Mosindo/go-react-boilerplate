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
	MaxUploadBytes = 10 << 20 // 10 MiB
	minDimension   = 200
	maxDimension   = 12_000
	maxPixels      = 50_000_000
	fullMaxSide    = 1440
	thumbMaxSide   = 480
	jpegQuality    = 82
)

var (
	ErrUnsupportedType = errors.New("unsupported image type (use JPEG, PNG or WebP)")
	ErrImageTooLarge   = errors.New("image too large (max 10 MB)")
	ErrImageTooSmall   = errors.New("image too small (min 200x200)")
	ErrImageCorrupt    = errors.New("image could not be decoded")
	ErrImageDimensions = errors.New("image dimensions are too large")
)

var allowedTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

// ProcessedImage holds the re-encoded variants of an upload.
type ProcessedImage struct {
	Full   []byte
	Thumb  []byte
	Width  int
	Height int
}

// ProcessImage validates an uploaded file by its content (never by its name
// or client-declared MIME type), rejects decompression bombs, applies the
// EXIF orientation and re-encodes the picture as JPEG. Re-encoding drops all
// metadata, including GPS coordinates embedded by phones.
func ProcessImage(data []byte) (ProcessedImage, error) {
	if len(data) == 0 {
		return ProcessedImage{}, ErrImageCorrupt
	}
	if len(data) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	contentType := http.DetectContentType(sniff)
	if _, ok := allowedTypes[contentType]; !ok {
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrImageCorrupt
	}
	if "image/"+format != contentType {
		return ProcessedImage{}, ErrUnsupportedType
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension || cfg.Width*cfg.Height > maxPixels {
		return ProcessedImage{}, ErrImageDimensions
	}
	if cfg.Width < minDimension || cfg.Height < minDimension {
		return ProcessedImage{}, ErrImageTooSmall
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrImageCorrupt
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}

	full := resizeToFit(img, fullMaxSide)
	thumb := resizeToFit(img, thumbMaxSide)

	fullBytes, err := encodeJPEG(full)
	if err != nil {
		return ProcessedImage{}, err
	}
	thumbBytes, err := encodeJPEG(thumb)
	if err != nil {
		return ProcessedImage{}, err
	}
	b := full.Bounds()
	return ProcessedImage{Full: fullBytes, Thumb: thumbBytes, Width: b.Dx(), Height: b.Dy()}, nil
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func resizeToFit(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
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
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
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
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
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
	ifdOffset := int(order.Uint32(tiff[4:8]))
	if ifdOffset < 8 || ifdOffset+2 > len(tiff) {
		return 0
	}
	entries := int(order.Uint16(tiff[ifdOffset : ifdOffset+2]))
	for e := 0; e < entries; e++ {
		off := ifdOffset + 2 + e*12
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

// applyOrientation rotates/flips img so that it is displayed upright.
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
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
			dst.Set(nx, ny, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
