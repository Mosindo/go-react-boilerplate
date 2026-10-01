package photos

import (
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "photo introuvable")
	case errors.Is(err, ErrLimitReached):
		httpx.Error(c, http.StatusConflict, "photo_limit", "vous pouvez ajouter 6 photos au maximum")
	case errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, "l'ordre doit contenir chacune de vos photos exactement une fois")
	case errors.Is(err, ErrFileRequired):
		httpx.BadRequest(c, "fichier requis")
	case errors.Is(err, ErrFileTooLarge):
		httpx.Error(c, http.StatusRequestEntityTooLarge, "file_too_large", "fichier trop volumineux (8 Mo maximum)")
	case errors.Is(err, ErrUnsupportedType), errors.Is(err, ErrInvalidImage), errors.Is(err, ErrImageTooLarge), errors.Is(err, ErrImageTooSmall):
		httpx.Error(c, http.StatusUnprocessableEntity, "invalid_image", "image refusée : "+err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) readUpload(c *gin.Context) (ok bool) {
	// Cap the whole body (file + multipart overhead) before parsing.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	return true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) Add(c *gin.Context) {
	h.readUpload(c)
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			h.fail(c, "photos.add", ErrFileTooLarge)
			return
		}
		h.fail(c, "photos.add", ErrFileRequired)
		return
	}
	defer file.Close()
	photo, err := h.service.Add(c.Request.Context(), httpx.UserID(c), file)
	if err != nil {
		h.fail(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	h.readUpload(c)
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		h.fail(c, "photos.replace", ErrFileRequired)
		return
	}
	defer file.Close()
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), c.Param("id"), file)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "liste de photos requise")
		return
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) serve(thumb bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		f, err := h.service.Open(c.Request.Context(), httpx.UserID(c), c.Param("id"), thumb)
		if err != nil {
			h.fail(c, "photos.serve", err)
			return
		}
		defer f.Close()
		// Photo ids are unguessable and never reused (replace creates a new id), so the bytes
		// behind a URL are immutable; "private" keeps shared caches from storing them.
		c.Header("Cache-Control", "private, max-age=86400, immutable")
		c.Header("Content-Type", "image/jpeg")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Disposition", "inline")
		http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
	}
}

func (h *Handler) Image(c *gin.Context) { h.serve(false)(c) }
func (h *Handler) Thumb(c *gin.Context) { h.serve(true)(c) }
