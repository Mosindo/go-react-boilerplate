package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		httpx.Fail(c, op, http.StatusUnsupportedMediaType, "unsupported_type", err.Error(), err)
	case errors.Is(err, ErrInvalidImage), errors.Is(err, ErrImageTooSmall):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, "invalid_image", err.Error(), err)
	case errors.Is(err, ErrLimitReached):
		httpx.Fail(c, op, http.StatusConflict, "photo_limit", "you can add up to 6 photos", err)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "not_found", "photo not found", err)
	case errors.Is(err, ErrBadOrder):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_order", err.Error(), err)
	default:
		httpx.Internal(c, op, err)
	}
}

// readUpload extracts the "photo" part with a hard size cap. It returns false after responding on error.
func readUpload(c *gin.Context, op string) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	header, err := c.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be at most 8 MB", err)
			return nil, false
		}
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_request", "multipart field \"photo\" is required", err)
		return nil, false
	}
	if header.Size > MaxUploadBytes {
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be at most 8 MB", nil)
		return nil, false
	}
	f, err := header.Open()
	if err != nil {
		httpx.Invalid(c, op, err)
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be at most 8 MB", err)
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c, "photos.upload")
	if !ok {
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photos.replace", "photoId")
	if !ok {
		return
	}
	data, ok := readUpload(c, "photos.replace")
	if !ok {
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photos.delete", "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, "photos.reorder", &req) {
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, "photos.reorder", http.StatusBadRequest, "invalid_id", "invalid photo id", nil)
			return
		}
	}
	userID := c.GetString("userID")
	if err := h.service.Reorder(c.Request.Context(), userID, req.PhotoIDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	h.List(c)
}

// Image serves a photo for a valid signed URL. It sits outside the bearer-auth group because image
// widgets cannot send headers; authorisation was checked when the URL was issued and the URL expires.
func (h *Handler) Image(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photos.image", "photoId")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), id, c.Query("exp"), c.Query("sig"))
	if err != nil {
		// Same answer for bad signature and unknown photo: do not reveal which photo ids exist.
		if errors.Is(err, ErrNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		httpx.Internal(c, "photos.image", err)
		return
	}
	defer f.Close()
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Type", storedMime)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
