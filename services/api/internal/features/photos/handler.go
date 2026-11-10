package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	signer  *media.Signer
	store   storage.Store
}

func NewHandler(service *Service, signer *media.Signer, store storage.Store) *Handler {
	return &Handler{service: service, signer: signer, store: store}
}

type photoResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func (h *Handler) list(c *gin.Context, status int) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "photos.list", err)
		return
	}
	out := make([]photoResponse, 0, len(photos))
	for _, p := range photos {
		out = append(out, photoResponse{ID: p.ID, Position: p.Position, URL: h.signer.URL(p.StorageKey), Width: p.Width, Height: p.Height})
	}
	httpx.NoStore(c)
	c.JSON(status, gin.H{"photos": out})
}

func (h *Handler) List(c *gin.Context) { h.list(c, http.StatusOK) }

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 10 MB or less")
			return nil, false
		}
		httpx.BadRequest(c, "photo_required", "multipart field \"photo\" is required")
		return nil, false
	}
	defer file.Close()
	if header.Size > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 10 MB or less")
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 10 MB or less")
		return nil, false
	}
	return data, true
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	if _, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data); err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	h.list(c, http.StatusCreated)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	if err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data); err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	h.list(c, http.StatusOK)
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
	h.list(c, http.StatusOK)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "photoIds must list your photo ids")
		return
	}
	if err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	h.list(c, http.StatusOK)
}

// Serve streams a photo for a valid signed URL. The signature is the
// authorization: it is only issued to viewers allowed to see the profile.
func (h *Handler) Serve(c *gin.Context) {
	key := c.Param("key")
	if !storage.ValidKey(key) || !h.signer.Verify(key, c.Query("exp"), c.Query("sig")) {
		httpx.Fail(c, http.StatusForbidden, "invalid_signature", "link expired or invalid")
		return
	}
	f, err := h.store.Open(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found")
			return
		}
		httpx.Internal(c, "photos.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrImageTooLarge), errors.Is(err, ErrCorruptImage):
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid_image", err.Error())
	case errors.Is(err, ErrLimitReached):
		httpx.Fail(c, http.StatusConflict, "photo_limit", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrInvalidOrder), errors.Is(err, ErrDuplicateID):
		httpx.BadRequest(c, "invalid_order", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
