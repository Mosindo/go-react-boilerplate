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
	// MaxUploadBytes is the largest accepted upload.
	MaxUploadBytes = 5 << 20
	// MaxSourceDim / MaxSourcePixels reject decompression bombs before any pixel is decoded.
	MaxSourceDim    = 8000
	MaxSourcePixels = 40_000_000
	// MaxSide is the longest side of the stored image.
	MaxSide     = 1080
	JPEGQuality = 82
)

var (
	ErrUnsupportedImage = errors.New("only JPEG, PNG and WebP images are accepted")
	ErrCorruptImage     = errors.New("the image could not be decoded")
	ErrImageTooBig      = errors.New("image dimensions are too large")
)

// ProcessImage validates untrusted image bytes and returns a re-encoded JPEG (no metadata) whose
// longest side is at most MaxSide, together with its dimensions.
func ProcessImage(data []byte) (out []byte, width, height int, err error) {
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	kind := http.DetectContentType(head)
	if kind != "image/jpeg" && kind != "image/png" && kind != "image/webp" {
		return nil, 0, 0, ErrUnsupportedImage
	}

	// Header-only decode first: never allocate pixels for an oversized image.
	var cfg image.Config
	switch kind {
	case "image/jpeg":
		cfg, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case "image/png":
		cfg, err = png.DecodeConfig(bytes.NewReader(data))
	case "image/webp":
		cfg, err = webp.DecodeConfig(bytes.NewReader(data))
	}
	if err != nil {
		return nil, 0, 0, ErrCorruptImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, 0, 0, ErrCorruptImage
	}
	if cfg.Width > MaxSourceDim || cfg.Height > MaxSourceDim || cfg.Width*cfg.Height > MaxSourcePixels {
		return nil, 0, 0, ErrImageTooBig
	}

	var src image.Image
	switch kind {
	case "image/jpeg":
		src, err = jpeg.Decode(bytes.NewReader(data))
	case "image/png":
		src, err = png.Decode(bytes.NewReader(data))
	case "image/webp":
		src, err = webp.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, 0, 0, ErrCorruptImage
	}

	w, h := fit(cfg.Width, cfg.Height, MaxSide)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// JPEG has no alpha: flatten transparency over white.
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

	var final image.Image = dst
	if kind == "image/jpeg" {
		if o := exifOrientation(data); o > 1 {
			final = orient(dst, o)
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, final, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, 0, 0, err
	}
	b := final.Bounds()
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// fit scales (w,h) down so that the longest side is at most limit; it never upscales.
func fit(w, h, limit int) (int, int) {
	if w <= limit && h <= limit {
		return w, h
	}
	if w >= h {
		return limit, max(1, (h*limit+w/2)/w)
	}
	return max(1, (w*limit+h/2)/h), limit
}

// exifOrientation extracts the EXIF orientation (1-8) from a JPEG, or 0 when absent/invalid.
// Metadata is stripped on re-encode, so the rotation must be baked into the pixels first.
func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 0
		}
		marker := data[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 0
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 0
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) >= 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 0
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	if bo.Uint16(t[2:4]) != 42 {
		return 0
	}
	off := int(bo.Uint32(t[4:8]))
	if off < 8 || off+2 > len(t) {
		return 0
	}
	n := int(bo.Uint16(t[off : off+2]))
	for k := 0; k < n; k++ {
		e := off + 2 + k*12
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 {
			v := int(bo.Uint16(t[e+8 : e+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// orient applies an EXIF orientation (2-8) and returns the upright image.
func orient(src *image.RGBA, o int) image.Image {
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
			default:
				nx, ny = x, y
			}
			si := src.PixOffset(x, y)
			di := dst.PixOffset(nx, ny)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}
