package photos

import (
	"errors"
	"mime/multipart"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/imaging"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// uploadedFile enforces the body cap before parsing the multipart form.
func uploadedFile(c *gin.Context) (multipart.File, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imaging.MaxInputBytes+(512<<10))
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, "photos.upload", http.StatusRequestEntityTooLarge, "image too large (max 8 MB)", nil)
		} else {
			httpx.Fail(c, "photos.upload", http.StatusBadRequest, "multipart field \"file\" is required", nil)
		}
		return nil, false
	}
	return file, true
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, imaging.ErrUnsupportedType):
		httpx.Fail(c, op, http.StatusUnsupportedMediaType, "only JPEG and PNG images are accepted", nil)
	case errors.Is(err, imaging.ErrTooLarge):
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "image too large", nil)
	case errors.Is(err, imaging.ErrCorrupt):
		httpx.Fail(c, op, http.StatusBadRequest, "image could not be read", nil)
	case errors.Is(err, ErrLimit):
		httpx.Fail(c, op, http.StatusConflict, "photo limit reached (6)", nil)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "photo not found", nil)
	case errors.Is(err, ErrBadReorder):
		httpx.Fail(c, op, http.StatusBadRequest, ErrBadReorder.Error(), nil)
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Upload(c *gin.Context) {
	file, ok := uploadedFile(c)
	if !ok {
		return
	}
	defer file.Close()
	list, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), file)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: list})
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	file, ok := uploadedFile(c)
	if !ok {
		return
	}
	defer file.Close()
	list, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, file)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	list, err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "photos.reorder.bind", err)
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, "photos.reorder", http.StatusBadRequest, "invalid photo id", nil)
			return
		}
	}
	list, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

// File serves an image to whoever holds a valid signed URL. The signature is
// only ever handed to users allowed to see the owner's profile.
func (h *Handler) File(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), id, c.Query("exp"), c.Query("sig"))
	if err != nil {
		h.fail(c, "photos.file", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
