package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Upload(c *gin.Context) {
	// Bound the whole request (file + multipart overhead) before parsing.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "image is too large (max 8 MB)")
			return
		}
		httpx.Fail(c, http.StatusBadRequest, "multipart field \"file\" is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "could not read upload")
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Respond(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Respond(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Respond(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) File(c *gin.Context) {
	data, err := h.service.Open(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Respond(c, "photos.file", err)
		return
	}
	// Photo ids are immutable (content never changes for an id), so private caching is safe.
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, "image/jpeg", data)
}
