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
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "photo not found")
	case errors.Is(err, ErrLimitReached):
		httpx.Error(c, http.StatusConflict, "photo limit reached (6)")
	case errors.Is(err, ErrInvalidImage):
		httpx.Error(c, http.StatusUnsupportedMediaType, "unsupported or invalid image (JPEG, PNG or WebP, min 200x200)")
	case errors.Is(err, ErrTooLarge):
		httpx.Error(c, http.StatusRequestEntityTooLarge, "image too large (max 8 MB)")
	case errors.Is(err, ErrInvalidOrder):
		httpx.Error(c, http.StatusBadRequest, "invalid order")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Upload(c *gin.Context) {
	replaceID := c.Query("replace")
	if replaceID != "" && !httpx.IsUUID(replaceID) {
		httpx.Error(c, http.StatusBadRequest, "invalid replace id")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(64<<10))
	fh, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			h.fail(c, "photos.upload", ErrTooLarge)
			return
		}
		httpx.Error(c, http.StatusBadRequest, "multipart field 'file' is required")
		return
	}
	if fh.Size > MaxUploadBytes {
		h.fail(c, "photos.upload", ErrTooLarge)
		return
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "cannot read file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "cannot read file")
		return
	}
	p, err := h.service.Add(c.Request.Context(), c.GetString("userID"), data, replaceID)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
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
		if !httpx.IsUUID(id) {
			httpx.Error(c, http.StatusBadRequest, "invalid request")
			return
		}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	h.List(c)
}

func (h *Handler) MakePrimary(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.MakePrimary(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.primary", err)
		return
	}
	h.List(c)
}

// Serve renvoie les octets d'une photo apres controle d'acces (proprietaire, ou
// profil visible/matche et non bloque).
func (h *Handler) Serve(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	file, err := h.service.Open(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	defer file.Close()
	st, err := file.Stat()
	if err != nil {
		httpx.Internal(c)
		return
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(c.Writer, c.Request, "", st.ModTime(), file)
}
