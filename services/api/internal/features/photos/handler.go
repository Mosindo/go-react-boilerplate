package photos

import (
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize+(1<<20))
	file, err := c.FormFile("file")
	if err != nil {
		h.uploadFailed(c, err)
		return
	}
	src, err := file.Open()
	if err != nil {
		httpx.Fail(c, "photos.open", platformerrors.Invalid("invalid upload"))
		return
	}
	defer src.Close()
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), src)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize+(1<<20))
	file, err := c.FormFile("file")
	if err != nil {
		h.uploadFailed(c, err)
		return
	}
	src, err := file.Open()
	if err != nil {
		httpx.Fail(c, "photos.open", platformerrors.Invalid("invalid upload"))
		return
	}
	defer src.Close()
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, src)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) uploadFailed(c *gin.Context, err error) {
	if IsMultipartTooLarge(err) {
		httpx.Fail(c, "photos.size", platformerrors.New(http.StatusRequestEntityTooLarge, "file_too_large", "photo must be 8 MB or smaller"))
		return
	}
	httpx.Fail(c, "photos.form", platformerrors.Invalid("multipart field \"file\" is required"))
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs); err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Serve(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "photos.serve", err)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		httpx.Fail(c, "photos.stat", err)
		return
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	http.ServeContent(c.Writer, c.Request, "", stat.ModTime(), f)
}
