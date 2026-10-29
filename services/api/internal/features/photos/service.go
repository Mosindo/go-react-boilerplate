package photos

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"log"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/storage"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

type Service struct {
	repo  Repository
	store storage.Storage
}

func NewService(repo Repository, store storage.Storage) *Service {
	return &Service{repo: repo, store: store}
}

// Upload validates and re-encodes an image. Whatever the client sends, only a
// freshly encoded JPEG produced by this server is ever stored: this strips EXIF
// metadata (GPS!), neutralises polyglot files and bounds the file size.
func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) (Photo, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return Photo{}, platformerrors.Invalid("could not read the uploaded file")
	}
	if len(data) > MaxUploadBytes {
		return Photo{}, platformerrors.Invalid("photo must be smaller than %d MB", MaxUploadBytes>>20)
	}
	if len(data) == 0 {
		return Photo{}, platformerrors.Invalid("empty file")
	}

	encoded, w, h, err := normalizeImage(data)
	if err != nil {
		return Photo{}, err
	}

	key, err := storage.NewKey()
	if err != nil {
		return Photo{}, err
	}
	if err := s.store.Save(key, encoded); err != nil {
		return Photo{}, err
	}
	photo, err := s.repo.Add(ctx, Photo{
		UserID: userID, StorageKey: key, ContentType: "image/jpeg", SizeBytes: len(encoded), Width: w, Height: h,
	})
	if err != nil {
		_ = s.store.Delete(key)
		if errors.Is(err, ErrLimitReached) {
			return Photo{}, platformerrors.Conflict("you can upload at most %d photos", MaxPhotosPerUser)
		}
		return Photo{}, err
	}
	return photo, nil
}

func normalizeImage(data []byte) (encoded []byte, width, height int, err error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") {
		return nil, 0, 0, platformerrors.Invalid("unsupported image: use JPEG, PNG or WebP")
	}
	if cfg.Width < 200 || cfg.Height < 200 {
		return nil, 0, 0, platformerrors.Invalid("image is too small (minimum 200x200)")
	}
	// Reject decompression bombs before allocating pixel buffers.
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePixels {
		return nil, 0, 0, platformerrors.Invalid("image dimensions are too large")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, platformerrors.Invalid("the image file is corrupted")
	}

	b := img.Bounds()
	tw, th := b.Dx(), b.Dy()
	if tw > maxDimension || th > maxDimension {
		if tw >= th {
			th, tw = th*maxDimension/tw, maxDimension
		} else {
			tw, th = tw*maxDimension/th, maxDimension
		}
	}
	// Draw on an opaque white canvas: flattens transparency and yields a plain RGBA image.
	canvas := image.NewRGBA(image.Rect(0, 0, tw, th))
	draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(canvas, canvas.Bounds(), img, b, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, canvas, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), tw, th, nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Remove(ctx, userID, photoID)
	switch {
	case errors.Is(err, ErrNotFound):
		return platformerrors.NotFound("photo not found")
	case errors.Is(err, ErrLastPhoto):
		return platformerrors.Conflict("keep at least one photo on your profile")
	case err != nil:
		return err
	}
	if err := s.store.Delete(key); err != nil {
		log.Printf(`{"event":"photo_file_delete_failed","error":%q}`, err.Error())
	}
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) ([]Photo, error) {
	for _, id := range ids {
		if !profiles.IsUUID(id) {
			return nil, platformerrors.Invalid("invalid photo id")
		}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrBadOrder) {
			return nil, platformerrors.Invalid("%s", err.Error())
		}
		return nil, err
	}
	return s.repo.List(ctx, userID)
}

func (s *Service) List(ctx context.Context, userID string) ([]Photo, error) {
	return s.repo.List(ctx, userID)
}

// Open returns the stored bytes of a photo if viewerID is allowed to see it.
func (s *Service) Open(ctx context.Context, viewerID, photoID string) (io.ReadSeekCloser, Photo, error) {
	if !profiles.IsUUID(photoID) {
		return nil, Photo{}, platformerrors.NotFound("photo not found")
	}
	p, err := s.repo.GetForViewer(ctx, viewerID, photoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, Photo{}, platformerrors.NotFound("photo not found")
		}
		return nil, Photo{}, err
	}
	f, err := s.store.Open(p.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, Photo{}, platformerrors.NotFound("photo not found")
		}
		return nil, Photo{}, err
	}
	return f, p, nil
}

// RemoveFiles deletes stored files, used after account deletion.
func (s *Service) RemoveFiles(keys []string) {
	for _, k := range keys {
		if err := s.store.Delete(k); err != nil {
			log.Printf(`{"event":"photo_file_delete_failed","error":%q}`, err.Error())
		}
	}
}
