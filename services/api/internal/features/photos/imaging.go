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

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrImageTooSmall   = errors.New("image dimensions too small")
	ErrCorruptImage    = errors.New("image could not be decoded")
)

var allowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

type processedImage struct {
	Data   []byte
	Width  int
	Height int
}

// processImage validates an untrusted upload and re-encodes it as a clean
// JPEG: the type is sniffed from bytes (never trusted from the client),
// dimensions are checked before decoding to stop decompression bombs, EXIF
// orientation is applied, and re-encoding drops every metadata block
// (including GPS) as well as any polyglot payload.
func processImage(raw []byte) (processedImage, error) {
	if !allowedMIME[http.DetectContentType(raw)] {
		return processedImage{}, ErrUnsupportedType
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processedImage{}, ErrCorruptImage
	}
	if cfg.Width > MaxSourceSide || cfg.Height > MaxSourceSide || cfg.Width*cfg.Height > MaxSourcePixels {
		return processedImage{}, ErrImageTooLarge
	}
	if cfg.Width < MinSourceSide || cfg.Height < MinSourceSide {
		return processedImage{}, ErrImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return processedImage{}, ErrCorruptImage
	}
	// Resize first, then rotate the (much smaller) RGBA result.
	dst := resize(src, OutputMaxSide)
	if format == "jpeg" {
		dst = applyOrientation(dst, jpegOrientation(raw))
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: OutputQuality}); err != nil {
		return processedImage{}, err
	}
	b := dst.Bounds()
	return processedImage{Data: out.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

func resize(src image.Image, maxSide int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := 1.0
	if w >= h && w > maxSide {
		scale = float64(maxSide) / float64(w)
	} else if h > w && h > maxSide {
		scale = float64(maxSide) / float64(h)
	}
	nw, nh := int(float64(w)*scale+0.5), int(float64(h)*scale+0.5)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	// Always redraw into a fresh RGBA canvas (also flattens transparency).
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}

// jpegOrientation extracts the EXIF orientation tag (1-8), defaulting to 1.
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
		seg := data[i+4 : i+2+segLen]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return parseTIFFOrientation(seg[6:])
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
	offset := int(order.Uint32(tiff[4:8]))
	if offset < 8 || offset+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	for n := 0; n < count; n++ {
		entry := offset + 2 + n*12
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

// applyOrientation rotates/flips pixels according to EXIF orientation,
// copying raw RGBA pixels (no per-pixel interface calls).
func applyOrientation(src *image.RGBA, orientation int) *image.RGBA {
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
			si := src.PixOffset(b.Min.X+x, b.Min.Y+y)
			di := dst.PixOffset(dx, dy)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}
