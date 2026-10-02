package photos

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found")
	case errors.Is(err, ErrLimitReached):
		httpx.Fail(c, http.StatusConflict, "photo_limit", "you can add up to 6 photos")
	case errors.Is(err, ErrInvalidImage):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "invalid_image", "only JPEG, PNG or WebP images are accepted")
	case errors.Is(err, ErrTooLarge):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "image_too_large", "image is too large (max 10 MB)")
	case errors.Is(err, ErrTooSmall):
		httpx.Fail(c, http.StatusBadRequest, "image_too_small", "image must be at least 200 px on each side")
	case errors.Is(err, ErrInvalidOrder):
		httpx.Fail(c, http.StatusBadRequest, "invalid_order", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	uid := httpx.UserID(c)
	m, err := h.service.ListForUsers(c.Request.Context(), []string{uid})
	if err != nil {
		httpx.Internal(c, "photos.list", err)
		return
	}
	list := m[uid]
	if list == nil {
		list = []View{}
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) openUpload(c *gin.Context) (file interface{ Read([]byte) (int, error) }, closeFn func(), ok bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			h.fail(c, "photos.upload", ErrTooLarge)
		} else {
			httpx.BadRequest(c, "multipart field \"file\" is required")
		}
		return nil, nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.BadRequest(c, "cannot read file")
		return nil, nil, false
	}
	return f, func() { _ = f.Close() }, true
}

func (h *Handler) Upload(c *gin.Context) {
	f, closeFn, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer closeFn()
	v, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), f)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, v)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	f, closeFn, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer closeFn()
	v, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, f)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs); err != nil {
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
	if err := h.service.MakePrimary(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "photos.primary", err)
		return
	}
	h.List(c)
}

// File serves bytes for a signed link. No bearer token: the HMAC link itself is the
// capability, so plain <Image> components can load it and HTTP caches work.
func (h *Handler) File(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), id, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrBadSignature) {
			httpx.Fail(c, http.StatusForbidden, "forbidden", err.Error())
		} else {
			httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found")
		}
		return
	}
	defer f.Close()
	hd := c.Writer.Header()
	hd.Set("Content-Type", "image/jpeg")
	hd.Set("Cache-Control", "private, max-age=3600")
	hd.Set("Content-Security-Policy", "default-src 'none'")
	hd.Set("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
