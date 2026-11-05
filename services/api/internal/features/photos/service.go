package photos

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	xdraw "golang.org/x/image/draw"
)

var (
	ErrNotFound        = errors.New("photo not found")
	ErrTooMany         = errors.New("photo limit reached")
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrInvalidImage    = errors.New("invalid or too large image")
	ErrInvalidOrder    = errors.New("order must list every photo exactly once")
	ErrNeedProfile     = errors.New("create your profile before adding photos")
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Add(ctx context.Context, userID string, raw []byte) (Photo, error) {
	data, w, h, err := Sanitize(raw)
	if err != nil {
		return Photo{}, err
	}
	return s.repo.Add(ctx, userID, data, w, h)
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, raw []byte) (Photo, error) {
	data, w, h, err := Sanitize(raw)
	if err != nil {
		return Photo{}, err
	}
	return s.repo.Replace(ctx, userID, photoID, data, w, h)
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	return s.repo.Delete(ctx, userID, photoID)
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	return s.repo.Reorder(ctx, userID, ids)
}

// Content returns image bytes when viewerID may see the photo.
func (s *Service) Content(ctx context.Context, viewerID, photoID string) (Content, error) {
	return s.repo.Content(ctx, viewerID, photoID)
}

// Sanitize validates an upload by content sniffing (never trusting the client
// MIME type or extension), bounds its dimensions, downsizes it and re-encodes
// it as JPEG. Re-encoding drops EXIF (GPS!) and any trailing/polyglot payload.
func Sanitize(raw []byte) (data []byte, width, height int, err error) {
	if len(raw) == 0 || len(raw) > MaxUploadSize {
		return nil, 0, 0, ErrInvalidImage
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return nil, 0, 0, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width*cfg.Height > maxPixels {
		return nil, 0, 0, ErrInvalidImage
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxSide || h > maxSide {
		if w >= h {
			h, w = h*maxSide/w, maxSide
		} else {
			w, h = w*maxSide/h, maxSide
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	// Flatten onto white so transparent PNGs encode predictably as JPEG.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, 0, 0, ErrInvalidImage
	}
	return out.Bytes(), w, h, nil
}
