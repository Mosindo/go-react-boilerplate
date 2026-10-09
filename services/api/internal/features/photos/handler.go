package photos

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrLimit):
		httpx.Fail(c, http.StatusConflict, "photo_limit", "you can upload up to 6 photos")
	case errors.Is(err, ErrTooBig):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "photo is larger than 8 MB")
	case errors.Is(err, ErrUnsupportedImage):
		httpx.BadRequest(c, "file must be a valid JPEG, PNG or WebP image")
	case errors.Is(err, ErrImageTooSmall):
		httpx.BadRequest(c, "photo must be at least 200x200 pixels")
	case errors.Is(err, ErrImageTooLarge):
		httpx.BadRequest(c, "photo dimensions are too large")
	case errors.Is(err, ErrBadOrder):
		httpx.BadRequest(c, "ids must list every photo exactly once")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
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

func (h *Handler) upload(c *gin.Context, replaceID string) {
	// Allow multipart overhead on top of the 8 MB file cap.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			h.fail(c, "photos.upload", ErrTooBig)
			return
		}
		httpx.BadRequest(c, "multipart field \"file\" is required")
		return
	}
	if header.Size > MaxUploadBytes {
		h.fail(c, "photos.upload", ErrTooBig)
		return
	}
	f, err := header.Open()
	if err != nil {
		httpx.BadRequest(c, "could not read upload")
		return
	}
	defer f.Close()

	var photo Photo
	if replaceID == "" {
		photo, err = h.service.Upload(c.Request.Context(), c.GetString("userID"), f)
	} else {
		photo, err = h.service.Replace(c.Request.Context(), c.GetString("userID"), replaceID, f)
	}
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	status := http.StatusCreated
	if replaceID != "" {
		status = http.StatusOK
	}
	c.JSON(status, photo)
}

func (h *Handler) Upload(c *gin.Context)  { h.upload(c, "") }
func (h *Handler) Replace(c *gin.Context) { h.upload(c, c.Param("id")) }

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) SetMain(c *gin.Context) {
	list, err := h.service.SetMain(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "photos.main", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) File(c *gin.Context) {
	p, f, err := h.service.Open(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "photos.file", err)
		return
	}
	defer f.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("ETag", `"`+p.StorageKey+`"`)
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
