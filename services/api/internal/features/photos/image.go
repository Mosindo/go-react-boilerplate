package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	xdraw "golang.org/x/image/draw"
)

const (
	MaxUploadBytes = 8 << 20
	maxPixels      = 40_000_000
	minSide        = 200
	fullMaxSide    = 1280
	thumbMaxSide   = 360
	jpegQuality    = 82
)

var (
	errBadType   = platformerrors.New(http.StatusUnsupportedMediaType, "unsupported_media_type", "only JPEG and PNG images are accepted")
	errBadImage  = platformerrors.BadRequest("invalid_image", "the file is not a valid image")
	errTooSmall  = platformerrors.BadRequest("image_too_small", "the image must be at least 200x200 pixels")
	errTooLarge  = platformerrors.New(http.StatusRequestEntityTooLarge, "image_too_large", "the image is too large")
	errMaxPhotos = platformerrors.Conflict("too_many_photos", "you can add up to 6 photos")
)

type processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// processImage validates untrusted bytes by content (never by filename or client MIME type),
// bounds decoder memory via DecodeConfig first, bakes EXIF orientation, strips all metadata by
// re-encoding, and produces a display size and a thumbnail as JPEG.
func processImage(data []byte) (processed, error) {
	if len(data) > MaxUploadBytes {
		return processed{}, errTooLarge
	}
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	switch http.DetectContentType(head) {
	case "image/jpeg", "image/png":
	default:
		return processed{}, errBadType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return processed{}, errBadImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return processed{}, errTooLarge
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return processed{}, errTooSmall
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return processed{}, errBadImage
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}
	flat := flatten(img)
	full := fit(flat, fullMaxSide)
	thumb := fit(flat, thumbMaxSide)
	fullBytes, err := encodeJPEG(full)
	if err != nil {
		return processed{}, err
	}
	thumbBytes, err := encodeJPEG(thumb)
	if err != nil {
		return processed{}, err
	}
	b := full.Bounds()
	return processed{Full: fullBytes, Thumb: thumbBytes, Width: b.Dx(), Height: b.Dy()}, nil
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// flatten composites transparency over white, since JPEG has no alpha channel.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
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
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

// jpegOrientation reads the EXIF orientation tag (1-8); 1 when absent or malformed.
func jpegOrientation(data []byte) int {
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
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	if len(t) < 8 {
		return 1
	}
	off := int(bo.Uint32(t[4:8]))
	if off < 8 || off+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[off : off+2]))
	for k := 0; k < n; k++ {
		e := off + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 {
			if v := int(bo.Uint16(t[e+8 : e+10])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
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
