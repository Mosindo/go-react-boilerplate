package photos

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", "format non pris en charge (JPEG, PNG ou WebP)")
	case errors.Is(err, ErrInvalidImage):
		httpx.BadRequest(c, "invalid_image", "image illisible")
	case errors.Is(err, ErrImageTooSmall):
		httpx.BadRequest(c, "image_too_small", "image trop petite (200 × 200 px minimum)")
	case errors.Is(err, ErrImageTooLarge):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "image_too_large", "image trop volumineuse (10 Mo maximum)")
	case errors.Is(err, ErrLimitReached):
		httpx.Fail(c, http.StatusConflict, "photo_limit", "6 photos maximum")
	case errors.Is(err, ErrLastPhoto):
		httpx.Fail(c, http.StatusConflict, "last_photo", "gardez au moins une photo sur votre profil")
	case errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, "invalid_order", "ordre des photos invalide")
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "photo_not_found", "photo introuvable")
	default:
		httpx.Internal(c, op, err)
	}
}

// readUpload reads the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "image_too_large", "image trop volumineuse (10 Mo maximum)")
			return nil, false
		}
		httpx.BadRequest(c, "missing_photo", "fichier photo manquant")
		return nil, false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		httpx.BadRequest(c, "invalid_image", "lecture du fichier impossible")
		return nil, false
	}
	if len(raw) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "image_too_large", "image trop volumineuse (10 Mo maximum)")
		return nil, false
	}
	return raw, true
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	if err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, raw); err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	h.List(c)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	h.List(c)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	for i, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.BadRequest(c, "invalid_order", "ordre des photos invalide")
			return
		}
		req.PhotoIDs[i] = strings.ToLower(id)
	}
	if err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	h.List(c)
}

// Serve streams a photo for a valid signed URL. Responses are private and
// cacheable by the client only.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "photo_not_found", "photo introuvable")
			return
		}
		httpx.Internal(c, "photos.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Header("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
