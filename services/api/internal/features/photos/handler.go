package photos

import (
	"io"
	"net/http"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := openUpload(c)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	defer file.Close()
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), file)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	file, err := openUpload(c)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	defer file.Close()
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), c.Param("id"), file)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.Bind(c, &req) {
		return
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Media serves a photo through a signed, expiring URL.
func (h *Handler) Media(c *gin.Context) {
	rc, err := h.service.Open(c.Request.Context(), c.Param("id"), c.DefaultQuery("size", "full"), c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, "photos.media", err)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}

// openUpload extracts the "photo" file part, enforcing the request size cap.
func openUpload(c *gin.Context) (io.ReadCloser, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(1<<20))
	fh, err := c.FormFile("photo")
	if err != nil {
		return nil, perr.Invalid("invalid_upload", "send the image in the multipart field \"photo\"")
	}
	if fh.Size > maxUploadBytes {
		return nil, perr.Invalid("file_too_large", "photo must be at most 8 MB")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, perr.Invalid("invalid_upload", "could not read the uploaded file")
	}
	return f, nil
}
