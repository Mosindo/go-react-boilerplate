package photos

import (
	"errors"
	"net/http"
	"regexp"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrInvalidImage):
		httpx.Error(c, http.StatusBadRequest, "invalid image: send a JPEG or PNG between 64px and 40 megapixels, up to 8 MB")
	case errors.Is(err, ErrLimitReached):
		httpx.Error(c, http.StatusConflict, "you can upload up to 6 photos")
	case errors.Is(err, ErrOrderMismatch):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "photo not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) validID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !uuidRe.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return "", false
	}
	return id, true
}

func (h *Handler) openUpload(c *gin.Context) (interface {
	Read([]byte) (int, error)
	Close() error
}, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize+(1<<20))
	fh, err := c.FormFile("photo")
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "multipart field \"photo\" is required (max 8 MB)")
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "could not read upload")
		return nil, false
	}
	return f, true
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
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
	id, ok := h.validID(c)
	if !ok {
		return
	}
	f, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer f.Close()
	if err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, f); err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.replace.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := h.validID(c)
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
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	for _, id := range req.IDs {
		if !uuidRe.MatchString(id) {
			httpx.Error(c, http.StatusBadRequest, "invalid photo id")
			return
		}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	list, _ := h.service.List(c.Request.Context(), c.GetString("userID"))
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) MakePrimary(c *gin.Context) {
	id, ok := h.validID(c)
	if !ok {
		return
	}
	if err := h.service.MakePrimary(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.primary", err)
		return
	}
	list, _ := h.service.List(c.Request.Context(), c.GetString("userID"))
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

// Serve returns image bytes only to users allowed to see the owner (not blocked, visible or matched).
func (h *Handler) Serve(c *gin.Context) {
	id, ok := h.validID(c)
	if !ok {
		return
	}
	data, err := h.service.Read(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, "image/jpeg", data)
}
