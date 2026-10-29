package photos

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/features/profiles"
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

func toResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, Position: p.Position, URL: profiles.PhotoURL(p.ID), Width: p.Width, Height: p.Height}
}

func toResponses(ps []Photo) []PhotoResponse {
	out := make([]PhotoResponse, len(ps))
	for i, p := range ps {
		out[i] = toResponse(p)
	}
	return out
}

func (h *Handler) List(c *gin.Context) {
	ps, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"photos": toResponses(ps)})
}

func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	file, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, "photos.upload", platformerrors.Invalid("photo must be smaller than %d MB", MaxUploadBytes>>20))
			return
		}
		httpx.Fail(c, "photos.upload", platformerrors.Invalid("multipart field \"photo\" is required"))
		return
	}
	f, err := file.Open()
	if err != nil {
		httpx.Fail(c, "photos.upload", platformerrors.Invalid("could not read the uploaded file"))
		return
	}
	defer f.Close()

	photo, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), f)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, toResponse(photo))
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		httpx.Fail(c, "photos.delete", platformerrors.NotFound("photo not found"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	ps, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": toResponses(ps)})
}

func (h *Handler) File(c *gin.Context) {
	f, _, err := h.service.Open(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		httpx.Fail(c, "photos.file", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	// Photo bytes never change for a given id, so clients may cache them privately.
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
