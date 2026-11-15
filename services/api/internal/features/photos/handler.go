package photos

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	store   storage.Store
	signer  *storage.URLSigner
}

func NewHandler(service *Service, store storage.Store, signer *storage.URLSigner) *Handler {
	return &Handler{service: service, store: store, signer: signer}
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
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
	c.JSON(http.StatusCreated, PhotosResponse{Photos: photos})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	photos, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.Bind(c, "photos.reorder", &req) {
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

// Media serves a stored photo when the signed URL is valid.
func (h *Handler) Media(c *gin.Context) {
	key := c.Param("key")
	if !storage.ValidKey(key) || !h.signer.Verify(key, c.Query("exp"), c.Query("sig")) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	f, modTime, err := h.store.Open(c.Request.Context(), key)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			httpx.Fail(c, "photos.media", err)
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer f.Close()

	maxAge := 3600
	if exp, err := strconv.ParseInt(c.Query("exp"), 10, 64); err == nil {
		if remaining := int(exp - time.Now().Unix()); remaining < maxAge {
			maxAge = max(remaining, 0)
		}
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age="+strconv.Itoa(maxAge))
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Header("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(c.Writer, c.Request, key, modTime, f)
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, "photos.read", ErrImageTooLarge)
			return nil, false
		}
		httpx.Fail(c, "photos.read", httpx.ErrInvalidRequest)
		return nil, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, "photos.read", httpx.ErrInvalidRequest)
		return nil, false
	}
	if len(data) > MaxUploadBytes {
		httpx.Fail(c, "photos.read", ErrImageTooLarge)
		return nil, false
	}
	return data, true
}
