package photos

import (
	"errors"
	"mime/multipart"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func uploadedFile(c *gin.Context) (multipart.File, error) {
	fh, err := c.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, ErrTooLarge
		}
		return nil, ErrMissingUpload
	}
	f, err := fh.Open()
	if err != nil {
		return nil, ErrMissingUpload
	}
	return f, nil
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := uploadedFile(c)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	defer file.Close()
	photos, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), file)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"photos": photos})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	file, err := uploadedFile(c)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	defer file.Close()
	photos, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, file)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
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
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, "photos.reorder", ErrInvalidOrder)
			return
		}
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

// Media serves a stored photo for a valid signed URL. No auth header is needed, which lets
// native and web image components load photos directly, but URLs expire within two hours.
func (h *Handler) Media(c *gin.Context) {
	key := c.Param("key")
	f, modTime, ok := h.service.OpenMedia(key, c.Query("exp"), c.Query("sig"))
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, httpx.ErrorResponse{Error: "Média introuvable.", Code: "media_not_found"})
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, key, modTime, f)
}
