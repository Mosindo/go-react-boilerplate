package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		logger.LogHandlerError(c, "photos.list", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not load photos")
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(1<<20))
	file, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, ErrFileTooLarge.Error())
		} else {
			httpx.Error(c, http.StatusBadRequest, "multipart field \"file\" is required")
		}
		return nil, false
	}
	if file.Size > maxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, ErrFileTooLarge.Error())
		return nil, false
	}
	f, err := file.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "could not read upload")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes+1))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "could not read upload")
		return nil, false
	}
	return data, true
}

func writeError(c *gin.Context, op string, err error) {
	status, message := http.StatusInternalServerError, "could not process photo"
	switch {
	case errors.Is(err, ErrUnsupportedImage), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrImageTooLarge):
		status, message = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, ErrFileTooLarge):
		status, message = http.StatusRequestEntityTooLarge, err.Error()
	case errors.Is(err, ErrLimit):
		status, message = http.StatusConflict, "you can upload up to 6 photos"
	case errors.Is(err, ErrNotFound):
		status, message = http.StatusNotFound, "photo not found"
	case errors.Is(err, ErrBadReorder):
		status, message = http.StatusBadRequest, err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Error(c, status, message)
}

func (h *Handler) Add(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Add(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		writeError(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, data)
	if err != nil {
		writeError(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		writeError(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	userID := c.GetString("userID")
	if err := h.service.Reorder(c.Request.Context(), userID, req.PhotoIDs); err != nil {
		writeError(c, "photos.reorder", err)
		return
	}
	h.List(c)
}

// Serve streams a file for a signed URL. A bad signature, an expired URL and an
// unknown id are indistinguishable (404) so ids cannot be probed.
func (h *Handler) Serve(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		c.Status(http.StatusNotFound)
		return
	}
	f, err := h.service.Open(c.Request.Context(), id, c.Param("variant"), c.Query("exp"), c.Query("sig"))
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			logger.LogHandlerError(c, "photos.serve", http.StatusInternalServerError, err)
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNotFound)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
