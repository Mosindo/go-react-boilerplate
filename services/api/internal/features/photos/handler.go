package photos

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func openUpload(c *gin.Context) (func() error, interface{ Read([]byte) (int, error) }, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, media.MaxUploadBytes+(1<<20))
	fh, err := c.FormFile("photo")
	if err != nil {
		httpx.BadRequest(c, "invalid_request", "multipart field 'photo' is required (max 8 MB)")
		return nil, nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.BadRequest(c, "invalid_request", "could not read upload")
		return nil, nil, false
	}
	return f.Close, f, true
}

func (h *Handler) Upload(c *gin.Context) {
	closeFn, f, ok := openUpload(c)
	if !ok {
		return
	}
	defer func() { _ = closeFn() }()
	p, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), f)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	closeFn, f, ok := openUpload(c)
	if !ok {
		return
	}
	defer func() { _ = closeFn() }()
	p, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, f)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	for _, id := range req.IDs {
		if !httpx.IsUUID(id) {
			httpx.BadRequest(c, "validation", "invalid photo id")
			return
		}
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) MakePrimary(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	list, err := h.service.MakePrimary(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "photos.primary", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

// File serves bytes for a signed URL (no bearer token so plain <Image> works).
func (h *Handler) File(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	path, err := h.service.FilePath(c.Request.Context(), id, c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, "photos.file", err)
		return
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=21600, immutable")
	c.Header("Content-Disposition", "inline")
	c.File(path)
}
