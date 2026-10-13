package photos

import (
	"errors"
	"mime/multipart"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrTooLarge):
		httpx.Error(c, http.StatusRequestEntityTooLarge, err.Error())
	case IsUploadError(err), errors.Is(err, ErrBadReorder):
		httpx.Error(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrLimit):
		httpx.Error(c, http.StatusConflict, "you can have at most 6 photos")
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "photo not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) openUpload(c *gin.Context) (src multipart.File, ok bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(256<<10))
	fh, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, ErrTooLarge.Error())
		} else {
			httpx.BadRequest(c, "multipart field \"file\" is required")
		}
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.BadRequest(c, "could not read upload")
		return nil, false
	}
	return f, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) Upload(c *gin.Context) {
	f, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer f.Close()
	p, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), f)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return
	}
	f, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer f.Close()
	p, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, f)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
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
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "ids is required")
		return
	}
	for _, id := range req.IDs {
		if !httpx.IsUUID(id) {
			httpx.BadRequest(c, "invalid photo id")
			return
		}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.reorder.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) File(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return
	}
	thumb := c.Query("size") == "thumb"
	img, err := h.service.Image(c.Request.Context(), c.GetString("userID"), id, thumb)
	if err != nil {
		h.fail(c, "photos.file", err)
		return
	}
	// Ids are immutable (replacement creates a new id), so clients may cache,
	// but never shared caches: access is per viewer.
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, "image/jpeg", img.Data)
}
