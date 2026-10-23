package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const (
	fullMaxSide  = 1080
	thumbMaxSide = 320
	jpegQuality  = 82
	minSide      = 64
	maxSide      = 8000
	maxPixels    = 40_000_000 // refuse decompression bombs before decoding
)

var (
	errUnsupported = platformerrors.New(http.StatusUnsupportedMediaType, "unsupported_media_type", "only JPEG, PNG and WebP images are accepted")
	errBadImage    = platformerrors.BadRequest("invalid_image", "the image is corrupted or has unsupported dimensions")
)

type Processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// Process validates an uploaded image by content (never by filename or declared
// type), strips all metadata by re-encoding, fixes orientation, and renders the
// full-size and thumbnail JPEG variants.
func Process(data []byte) (Processed, error) {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	switch http.DetectContentType(head) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return Processed{}, errUnsupported
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Processed{}, errBadImage
	}
	if cfg.Width < minSide || cfg.Height < minSide || cfg.Width > maxSide || cfg.Height > maxSide || cfg.Width*cfg.Height > maxPixels {
		return Processed{}, errBadImage
	}

	img, err := decode(data)
	if err != nil {
		return Processed{}, errBadImage
	}
	if http.DetectContentType(head) == "image/jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}
	flat := flatten(img)

	full, w, h, err := renderJPEG(flat, fullMaxSide)
	if err != nil {
		return Processed{}, err
	}
	thumb, _, _, err := renderJPEG(flat, thumbMaxSide)
	if err != nil {
		return Processed{}, err
	}
	return Processed{Full: full, Thumb: thumb, Width: w, Height: h}, nil
}

func decode(data []byte) (image.Image, error) {
	switch http.DetectContentType(data[:min(len(data), 512)]) {
	case "image/jpeg":
		return jpeg.Decode(bytes.NewReader(data))
	case "image/png":
		return png.Decode(bytes.NewReader(data))
	default:
		return webp.Decode(bytes.NewReader(data))
	}
}

// flatten composites onto white so transparent PNGs do not turn black as JPEG.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func renderJPEG(src *image.RGBA, maxSideLen int) (data []byte, width, height int, err error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxSideLen || h > maxSideLen {
		if w >= h {
			h = max(1, h*maxSideLen/w)
			w = maxSideLen
		} else {
			w = max(1, w*maxSideLen/h)
			h = maxSideLen
		}
	}
	out := src
	if w != b.Dx() || h != b.Dy() {
		out = image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(out, out.Bounds(), src, b, xdraw.Src, nil)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}

// jpegOrientation reads the EXIF orientation tag (1..8), defaulting to 1.
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
			if o := exifOrientation(data[i+4 : i+2+segLen]); o != 0 {
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
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	n := int(order.Uint16(tiff[ifd : ifd+2]))
	for k := 0; k < n; k++ {
		off := ifd + 2 + k*12
		if off+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[off:off+2]) == 0x0112 {
			if v := int(order.Uint16(tiff[off+8 : off+10])); v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// applyOrientation rotates/flips src so that it displays upright.
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
