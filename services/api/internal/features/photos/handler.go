package photos

import (
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// readUpload reads the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(64<<10))
	file, err := c.FormFile("photo")
	if err != nil {
		httpx.BadRequest(c, "multipart field 'photo' is required (max 8 MB)")
		return nil, false
	}
	if file.Size > MaxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large (max 8 MB)"})
		return nil, false
	}
	f, err := file.Open()
	if err != nil {
		httpx.BadRequest(c, "unreadable file")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large (max 8 MB)"})
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), c.Param("photoId"), data)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), c.Param("photoId")); err != nil {
		httpx.Fail(c, "photos.delete", err)
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
	items, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) File(c *gin.Context) {
	f, err := h.service.Open(c.Request.Context(), c.GetString("userID"), c.Param("photoId"))
	if err != nil {
		httpx.Fail(c, "photos.file", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	// The URL carries a version token, so the bytes behind it never change.
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
