package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	if v, ok := httpx.AsValidation(err); ok {
		httpx.Error(c, http.StatusBadRequest, v.Message)
		return
	}
	switch {
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Error(c, http.StatusNotFound, "photo not found")
	case errors.Is(err, ErrPhotoLimit):
		httpx.Error(c, http.StatusConflict, "you can upload up to 6 photos")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

// readUpload reads the multipart "photo" field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(512<<10))
	header, err := c.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, "photo exceeds 5 MB")
			return nil, false
		}
		httpx.Error(c, http.StatusBadRequest, "photo file is required (multipart field \"photo\", max 5 MB)")
		return nil, false
	}
	if header.Size > MaxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "photo exceeds 5 MB")
		return nil, false
	}
	f, err := header.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "could not read photo")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "photo exceeds 5 MB")
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Add(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Add(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		h.fail(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.ParamUUID(c, "photoId")
	if !ok {
		return
	}
	data, uploaded := readUpload(c)
	if !uploaded {
		return
	}
	p, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.ParamUUID(c, "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) File(c *gin.Context) {
	photoID, ok := httpx.ParamUUID(c, "photoId")
	if !ok {
		return
	}
	rc, err := h.service.Open(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		h.fail(c, "photos.file", err)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}
