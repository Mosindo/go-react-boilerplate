package photos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register decoders
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/api/internal/platform/storage"
	"github.com/google/uuid"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register decoder
)

var (
	ErrNotFound      = errors.New("photo not found")
	ErrLimitReached  = errors.New("photo limit reached")
	ErrInvalidImage  = errors.New("unsupported or corrupted image")
	ErrTooLarge      = errors.New("image too large")
	ErrTooSmall      = errors.New("image too small")
	ErrInvalidOrder  = errors.New("order must list each of your photos exactly once")
	ErrBadSignature  = errors.New("invalid or expired link")
	allowedMIMETypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}
)

const urlWindow = 6 * time.Hour

type Service struct {
	repo    Repository
	store   storage.Store
	sigKey  []byte
	nowFunc func() time.Time
}

func NewService(repo Repository, store storage.Store, jwtSecret []byte) *Service {
	k := sha256.Sum256(append([]byte("photo-url|"), jwtSecret...))
	return &Service{repo: repo, store: store, sigKey: k[:], nowFunc: time.Now}
}

// Process validates untrusted bytes and returns a sanitised JPEG: the type is sniffed
// from content (never from the filename), the image is fully decoded, downscaled and
// re-encoded, which strips EXIF/GPS metadata and any polyglot payload.
func Process(data []byte) (out []byte, width, height int, err error) {
	if len(data) == 0 || len(data) > MaxUploadBytes {
		return nil, 0, 0, ErrTooLarge
	}
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	mime := http.DetectContentType(sniff)
	if !allowedMIMETypes[mime] {
		return nil, 0, 0, ErrInvalidImage
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return nil, 0, 0, ErrTooLarge
	}
	if cfg.Width < MinSide || cfg.Height < MinSide {
		return nil, 0, 0, ErrTooSmall
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}

	w, h := cfg.Width, cfg.Height
	if w > MaxOutputSide || h > MaxOutputSide {
		if w >= h {
			h = h * MaxOutputSide / w
			w = MaxOutputSide
		} else {
			w = w * MaxOutputSide / h
			h = MaxOutputSide
		}
	}
	// Flatten onto white so transparent PNGs do not turn black.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}

func (s *Service) saveProcessed(ctx context.Context, userID string, data []byte) (NewPhoto, error) {
	out, w, h, err := Process(data)
	if err != nil {
		return NewPhoto{}, err
	}
	key := fmt.Sprintf("photos/%s/%s.jpg", strings.ToLower(userID[:2]), uuid.NewString())
	if err := s.store.Put(ctx, key, out); err != nil {
		return NewPhoto{}, err
	}
	return NewPhoto{StorageKey: key, ContentType: "image/jpeg", SizeBytes: len(out), Width: w, Height: h}, nil
}

func (s *Service) Upload(ctx context.Context, userID string, r io.Reader) (View, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return View{}, err
	}
	np, err := s.saveProcessed(ctx, userID, data)
	if err != nil {
		return View{}, err
	}
	stored, err := s.repo.Add(ctx, userID, np)
	if err != nil {
		_ = s.store.Delete(ctx, np.StorageKey)
		if errors.Is(err, ErrRepoFull) {
			return View{}, ErrLimitReached
		}
		return View{}, err
	}
	return s.view(stored), nil
}

func (s *Service) Replace(ctx context.Context, userID, photoID string, r io.Reader) (View, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return View{}, err
	}
	np, err := s.saveProcessed(ctx, userID, data)
	if err != nil {
		return View{}, err
	}
	oldKey, stored, err := s.repo.Replace(ctx, userID, photoID, np)
	if err != nil {
		_ = s.store.Delete(ctx, np.StorageKey)
		if errors.Is(err, ErrRepoNotFound) {
			return View{}, ErrNotFound
		}
		return View{}, err
	}
	_ = s.store.Delete(ctx, oldKey)
	return s.view(stored), nil
}

func (s *Service) Delete(ctx context.Context, userID, photoID string) error {
	key, err := s.repo.Delete(ctx, userID, photoID)
	if err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	_ = s.store.Delete(ctx, key)
	return nil
}

func (s *Service) Reorder(ctx context.Context, userID string, ids []string) error {
	for i := range ids {
		ids[i] = strings.ToLower(strings.TrimSpace(ids[i]))
		if _, err := uuid.Parse(ids[i]); err != nil {
			return ErrInvalidOrder
		}
	}
	if err := s.repo.Reorder(ctx, userID, ids); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrInvalidOrder
		}
		return err
	}
	return nil
}

// MakePrimary moves a photo to position 0 and keeps the others in their relative order.
func (s *Service) MakePrimary(ctx context.Context, userID, photoID string) error {
	list, err := s.ListForUsers(ctx, []string{userID})
	if err != nil {
		return err
	}
	mine := list[userID]
	order := []string{photoID}
	found := false
	for _, p := range mine {
		if p.ID == photoID {
			found = true
			continue
		}
		order = append(order, p.ID)
	}
	if !found {
		return ErrNotFound
	}
	return s.Reorder(ctx, userID, order)
}

// ListForUsers returns photo views for many users in one query (no N+1).
func (s *Service) ListForUsers(ctx context.Context, userIDs []string) (map[string][]View, error) {
	out := make(map[string][]View, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := s.repo.ListByUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], s.view(r))
	}
	return out, nil
}

func (s *Service) view(p Stored) View {
	exp := s.expiry()
	return View{ID: p.ID, Position: p.Position, URL: fmt.Sprintf("/photos/%s/file?exp=%d&sig=%s", p.ID, exp, s.sign(p.ID, exp))}
}

// expiry is constant inside a 6h window so that URLs stay cacheable by the client.
func (s *Service) expiry() int64 {
	w := int64(urlWindow / time.Second)
	return (s.nowFunc().Unix()/w + 2) * w
}

func (s *Service) sign(photoID string, exp int64) string {
	m := hmac.New(sha256.New, s.sigKey)
	m.Write([]byte(photoID + "|" + strconv.FormatInt(exp, 10)))
	return hex.EncodeToString(m.Sum(nil))[:40]
}

// Open verifies a signed link and returns the stored file.
func (s *Service) Open(ctx context.Context, photoID, expRaw, sig string) (io.ReadSeekCloser, error) {
	exp, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil || exp < s.nowFunc().Unix() || !hmac.Equal([]byte(sig), []byte(s.sign(photoID, exp))) {
		return nil, ErrBadSignature
	}
	key, err := s.repo.StorageKey(ctx, photoID)
	if err != nil {
		return nil, ErrNotFound
	}
	f, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, ErrNotFound
	}
	return f, nil
}
