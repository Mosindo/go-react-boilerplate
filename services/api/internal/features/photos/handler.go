package photos

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	switch {
	case errors.Is(err, ErrNotFound):
		status, msg = http.StatusNotFound, "photo not found"
	case errors.Is(err, ErrLimit), errors.Is(err, ErrBadOrder):
		status, msg = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, ErrFileTooBig):
		status, msg = http.StatusRequestEntityTooLarge, err.Error()
	case errors.Is(err, ErrUnsupportedType):
		status, msg = http.StatusUnsupportedMediaType, err.Error()
	case errors.Is(err, ErrInvalidImage), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrImageTooLarge):
		status, msg = http.StatusUnprocessableEntity, err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, gin.H{"error": msg})
}

// uploadReader returns the multipart "file" part with the body capped in size.
func uploadReader(c *gin.Context) (*bytes.Reader, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, ErrFileTooBig
		}
		return nil, ErrInvalidImage
	}
	if fh.Size > MaxUploadBytes {
		return nil, ErrFileTooBig
	}
	f, err := fh.Open()
	if err != nil {
		return nil, ErrInvalidImage
	}
	defer f.Close()
	data, err := readLimited(f)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (h *Handler) List(c *gin.Context) {
	out, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Add(c *gin.Context) {
	file, err := uploadReader(c)
	if err != nil {
		h.fail(c, "photos.add.read", err)
		return
	}
	out, err := h.service.Add(c.Request.Context(), c.GetString("userID"), file)
	if err != nil {
		h.fail(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *Handler) Replace(c *gin.Context) {
	file, err := uploadReader(c)
	if err != nil {
		h.fail(c, "photos.replace.read", err)
		return
	}
	out, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), c.Param("id"), file)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Delete(c *gin.Context) {
	out, err := h.service.Delete(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 || len(req.IDs) > MaxPhotos {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	out, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// Serve streams a photo for a signed URL (no bearer token needed so that image
// components can load it directly).
func (h *Handler) Serve(c *gin.Context) {
	f, err := h.service.Open(c.Request.Context(), c.Param("id"), c.Query("exp"), c.Query("sig"))
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	hdr := c.Writer.Header()
	hdr.Set("Content-Type", f.ContentType)
	hdr.Set("Content-Length", strconv.Itoa(len(f.Data)))
	hdr.Set("Cache-Control", "private, max-age=3600")
	hdr.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	hdr.Set("Cross-Origin-Resource-Policy", "cross-origin")
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(f.Data)
}
