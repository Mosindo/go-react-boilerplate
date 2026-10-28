package photos

import (
	"io"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Upload(c *gin.Context) {
	// Cap the whole request (file + multipart framing) before parsing.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(256<<10))
	header, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if asMaxBytes(err, &tooBig) {
			httpx.Fail(c, "photos.upload", platformerrors.New(http.StatusRequestEntityTooLarge, "file_too_large", "the image must be at most 8 MB"))
			return
		}
		httpx.Fail(c, "photos.upload", platformerrors.BadRequest("missing_photo", "send the image in a multipart field named \"photo\""))
		return
	}
	if header.Size > maxUploadBytes {
		httpx.Fail(c, "photos.upload", platformerrors.New(http.StatusRequestEntityTooLarge, "file_too_large", "the image must be at most 8 MB"))
		return
	}
	f, err := header.Open()
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) File(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	f, info, err := h.service.Open(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "photos.file", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable") // ids are never reused for new content
	c.Header("Content-Disposition", "inline")
	http.ServeContent(c.Writer, c.Request, "", info.ModTime(), f)
}
