package media

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
	MaxUploadBytes    = 10 << 20 // 10 MiB raw upload
	maxSourcePixels   = 24_000_000
	maxOutputEdge     = 1440
	outputJPEGQuality = 82
)

// processSlots limits how many images are decoded at the same time.
var processSlots = make(chan struct{}, 4)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrImageTooSmall   = errors.New("image dimensions too small")
	ErrCorruptImage    = errors.New("corrupt image")
)

type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// Process sanitises an uploaded photo: the type is detected from magic bytes (never from the
// client-provided name or header), dimensions are checked before full decode (decompression
// bombs), EXIF orientation is applied, and the image is re-encoded as a bounded JPEG.
// Re-encoding drops every metadata block, including EXIF GPS coordinates.
func Process(data []byte) (ProcessedImage, error) {
	// Bound concurrent decodes so parallel uploads cannot exhaust memory.
	processSlots <- struct{}{}
	defer func() { <-processSlots }()

	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width*cfg.Height > maxSourcePixels || cfg.Width > 12000 || cfg.Height > 12000 {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if cfg.Width < 200 || cfg.Height < 200 {
		return ProcessedImage{}, ErrImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	orientation := jpegOrientation(data)

	// Downscale first, then rotate the small image: rotating a full-size decode is costly.
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxOutputEdge || h > maxOutputEdge {
		if w >= h {
			h = h * maxOutputEdge / w
			w = maxOutputEdge
		} else {
			w = w * maxOutputEdge / h
			h = maxOutputEdge
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)
	final := applyOrientation(dst, orientation)
	fb := final.Bounds()

	var out bytes.Buffer
	if err := jpeg.Encode(&out, final, &jpeg.Options{Quality: outputJPEGQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: out.Bytes(), Width: fb.Dx(), Height: fb.Dy()}, nil
}

// jpegOrientation extracts the EXIF orientation tag (1-8) from a JPEG, defaulting to 1.
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
		segment := data[i+4 : i+2+segLen]
		if marker == 0xE1 && len(segment) > 14 && string(segment[:6]) == "Exif\x00\x00" {
			return parseTIFFOrientation(segment[6:])
		}
		i += 2 + segLen
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
	if ifd+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	for e := 0; e < count; e++ {
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
