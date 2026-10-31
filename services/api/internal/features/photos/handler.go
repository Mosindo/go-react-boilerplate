package photos

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Internal(c, "photos.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Upload(c *gin.Context) {
	file, ok := h.openUpload(c, "photos.upload")
	if !ok {
		return
	}
	defer file.Close()
	list, err := h.service.Add(c.Request.Context(), c.GetString("userID"), file)
	if h.fail(c, "photos.upload", err) {
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: list})
}

func (h *Handler) Replace(c *gin.Context) {
	file, ok := h.openUpload(c, "photos.replace")
	if !ok {
		return
	}
	defer file.Close()
	list, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), c.Param("id"), file)
	if h.fail(c, "photos.replace", err) {
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Delete(c *gin.Context) {
	list, err := h.service.Delete(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if h.fail(c, "photos.delete", err) {
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "photos.reorder.bind", "invalid request", err)
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs)
	if h.fail(c, "photos.reorder", err) {
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Get(c *gin.Context) {
	data, err := h.service.Fetch(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if h.fail(c, "photos.get", err) {
		return
	}
	// Replacing a photo issues a new id, so a given id's bytes never change.
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, "image/jpeg", data)
}

type closableFile interface {
	Read(p []byte) (int, error)
	Close() error
}

func (h *Handler) openUpload(c *gin.Context, op string) (closableFile, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(64<<10))
	fh, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "photo too large (max 5 MB)", err)
		} else {
			httpx.BadRequest(c, op, "multipart field \"photo\" is required", err)
		}
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Internal(c, op, err)
		return nil, false
	}
	return f, true
}

func (h *Handler) fail(c *gin.Context, op string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "photo not found", nil)
	case errors.Is(err, ErrLimit):
		httpx.Fail(c, op, http.StatusConflict, "photo limit reached (6)", err)
	case errors.Is(err, ErrBadOrder):
		httpx.BadRequest(c, op, err.Error(), err)
	case errors.Is(err, ErrImageTooLarge):
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "image too large (max 5 MB, 16 megapixels)", err)
	case errors.Is(err, ErrUnsupportedImage), errors.Is(err, ErrCorruptImage):
		httpx.Fail(c, op, http.StatusUnsupportedMediaType, err.Error(), err)
	default:
		httpx.Internal(c, op, err)
	}
	return true
}
