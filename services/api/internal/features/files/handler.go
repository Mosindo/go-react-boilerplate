package files

import (
	"errors"
	"io"
	"net/http"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// readUpload extracts the multipart "file" field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, "files.upload", ErrImageTooLarge)
			return nil, false
		}
		httpx.Fail(c, "files.upload", platformerrors.BadRequest("missing_file", "a 'file' field is required"))
		return nil, false
	}
	if header.Size > MaxUploadBytes {
		httpx.Fail(c, "files.upload", ErrImageTooLarge)
		return nil, false
	}
	f, err := header.Open()
	if err != nil {
		httpx.Fail(c, "files.upload", platformerrors.BadRequest("missing_file", "could not read the file"))
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, "files.upload", err)
		return nil, false
	}
	if len(data) > MaxUploadBytes {
		httpx.Fail(c, "files.upload", ErrImageTooLarge)
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusCreated, gin.H{"photos": photos})
}

func (h *Handler) Replace(c *gin.Context) {
	if !httpx.IsUUID(c.Param("photoId")) {
		httpx.Fail(c, "photos.replace", ErrPhotoNotFound)
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), c.Param("photoId"), data)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Delete(c *gin.Context) {
	if !httpx.IsUUID(c.Param("photoId")) {
		httpx.Fail(c, "photos.delete", ErrPhotoNotFound)
		return
	}
	photos, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("photoId"))
	if err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.Bind(c, &req) {
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) SetMain(c *gin.Context) {
	if !httpx.IsUUID(c.Param("photoId")) {
		httpx.Fail(c, "photos.set_main", ErrPhotoNotFound)
		return
	}
	photos, err := h.service.SetMain(c.Request.Context(), httpx.UserID(c), c.Param("photoId"))
	if err != nil {
		httpx.Fail(c, "photos.set_main", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

// File serves image bytes for a signed URL. The signature, not a bearer
// token, authorizes it, so plain image components can load it.
func (h *Handler) File(c *gin.Context) {
	if !httpx.IsUUID(c.Param("photoId")) {
		httpx.Fail(c, "photos.file", ErrPhotoNotFound)
		return
	}
	f, err := h.service.Open(c.Request.Context(), c.Param("photoId"), c.Query("size"), c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, "photos.file", err)
		return
	}
	defer f.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
