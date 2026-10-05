package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"math"
	"net/http"
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrInvalidImage    = errors.New("image is corrupted or has unsupported dimensions")
	ErrImageTooSmall   = errors.New("image is too small")
)

type processed struct {
	Data          []byte
	Width, Height int
}

// processImage validates untrusted bytes and returns a clean, re-encoded JPEG:
//   - the type is decided from the content (magic bytes), never from the filename or client MIME type;
//   - pixel dimensions are checked before decoding to defuse decompression bombs;
//   - EXIF orientation is applied, then every metadata block (GPS included) is dropped by re-encoding;
//   - the long edge is capped (compression) and transparency is flattened on white.
func processImage(raw []byte) (processed, error) {
	head := raw
	if len(head) > 512 {
		head = head[:512]
	}
	if t := http.DetectContentType(head); t != "image/jpeg" && t != "image/png" {
		return processed{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return processed{}, ErrInvalidImage
	}
	if cfg.Width > maxSourceEdge || cfg.Height > maxSourceEdge || cfg.Width*cfg.Height > maxSourcePixels {
		return processed{}, ErrInvalidImage
	}
	if cfg.Width < minEdgePixels || cfg.Height < minEdgePixels {
		return processed{}, ErrImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return processed{}, ErrInvalidImage
	}

	out := resample(src, exifOrientation(raw), MaxEdgePixels)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return processed{}, err
	}
	b := out.Bounds()
	return processed{Data: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// resample applies the EXIF orientation (1-8) and downsizes so the long edge is at most maxEdge,
// averaging n×n samples per output pixel when shrinking.
func resample(src image.Image, orientation, maxEdge int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()

	// Dimensions after orientation (orientations 5-8 swap axes).
	ow, oh := sw, sh
	if orientation >= 5 && orientation <= 8 {
		ow, oh = sh, sw
	}
	scale := 1.0
	if long := max(ow, oh); long > maxEdge {
		scale = float64(maxEdge) / float64(long)
	}
	dw := max(1, int(math.Round(float64(ow)*scale)))
	dh := max(1, int(math.Round(float64(oh)*scale)))
	n := min(4, max(1, int(math.Ceil(1/scale))))

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	cnt := uint32(n * n)
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var r, g, b, a uint32
			for j := 0; j < n; j++ {
				for i := 0; i < n; i++ {
					ox := min(ow-1, int((float64(x)+(float64(i)+0.5)/float64(n))/float64(dw)*float64(ow)))
					oy := min(oh-1, int((float64(y)+(float64(j)+0.5)/float64(n))/float64(dh)*float64(oh)))
					rx, ry := orientedToRaw(ox, oy, sw, sh, orientation)
					cr, cg, cb, ca := src.At(sb.Min.X+rx, sb.Min.Y+ry).RGBA()
					r, g, b, a = r+cr, g+cg, b+cb, a+ca
				}
			}
			r, g, b, a = r/cnt, g/cnt, b/cnt, a/cnt
			// Flatten transparency on white (premultiplied alpha: c + (1-a)*white).
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8(min32(r+0xffff-a, 0xffff) >> 8),
				G: uint8(min32(g+0xffff-a, 0xffff) >> 8),
				B: uint8(min32(b+0xffff-a, 0xffff) >> 8),
				A: 255,
			})
		}
	}
	return dst
}

func min32(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// orientedToRaw maps a coordinate in the displayed (oriented) image back to the stored raster.
func orientedToRaw(x, y, w, h, orientation int) (int, int) {
	switch orientation {
	case 2: // mirror horizontal
		return w - 1 - x, y
	case 3: // rotate 180
		return w - 1 - x, h - 1 - y
	case 4: // mirror vertical
		return x, h - 1 - y
	case 5: // transpose
		return y, x
	case 6: // rotate 90 CW
		return y, h - 1 - x
	case 7: // transverse
		return w - 1 - y, h - 1 - x
	case 8: // rotate 90 CCW
		return w - 1 - y, x
	default:
		return x, y
	}
}

// exifOrientation reads the EXIF orientation tag of a JPEG, returning 1 (normal) when absent or malformed.
func exifOrientation(data []byte) int {
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
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 && size >= 16 {
			seg := data[i+4 : i+2+size]
			if bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
				return parseTIFFOrientation(seg[6:])
			}
		}
		i += 2 + size
	}
	return 1
}

func parseTIFFOrientation(t []byte) int {
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
	ifd := int(order.Uint32(t[4:8]))
	if ifd < 8 || ifd+2 > len(t) {
		return 1
	}
	count := int(order.Uint16(t[ifd : ifd+2]))
	for k := 0; k < count; k++ {
		off := ifd + 2 + k*12
		if off+12 > len(t) {
			return 1
		}
		if order.Uint16(t[off:off+2]) == 0x0112 {
			v := int(order.Uint16(t[off+8 : off+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}
