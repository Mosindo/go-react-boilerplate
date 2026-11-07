package profiles

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Reader interface {
	Read(key string) ([]byte, error)
}

type Handler struct {
	service *Service
	files   Reader
}

func NewHandler(service *Service, files Reader) *Handler {
	return &Handler{service: service, files: files}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.BadRequest(c, ve.Message)
	case errors.Is(err, ErrProfileMissing):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "profile not created yet")
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden):
		httpx.NotFound(c)
	default:
		logger.LogHandlerError(c, op, 500, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Catalog(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) GetMine(c *gin.Context) {
	p, err := h.service.GetMine(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get_mine", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UploadPhoto(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(64<<10))
	fh, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be under 5 MB")
			return
		}
		httpx.BadRequest(c, "multipart field 'photo' is required")
		return
	}
	if fh.Size > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be under 5 MB")
		return
	}
	f, err := fh.Open()
	if err != nil {
		httpx.BadRequest(c, "could not read upload")
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil || len(raw) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be under 5 MB")
		return
	}
	photo, err := h.service.AddPhoto(c.Request.Context(), c.GetString("userID"), raw)
	if err != nil {
		h.fail(c, "profiles.upload_photo", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) DeletePhoto(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.NotFound(c)
		return
	}
	if err := h.service.DeletePhoto(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "profiles.delete_photo", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReorderPhotos(c *gin.Context) {
	var req ReorderPhotosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "ids must be a list of photo ids")
		return
	}
	if err := h.service.ReorderPhotos(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		h.fail(c, "profiles.reorder_photos", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) View(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		httpx.NotFound(c)
		return
	}
	card, err := h.service.ViewProfile(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, card)
}

// Media serves a photo for a valid signed URL. The signature is the capability; no bearer token needed
// so that plain <Image> components can load it.
func (h *Handler) Media(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil || !h.service.Signer().Verify(id, c.Query("e"), c.Query("s")) {
		httpx.NotFound(c)
		return
	}
	key, err := h.service.PhotoKey(c.Request.Context(), id)
	if err != nil {
		h.fail(c, "profiles.media", err)
		return
	}
	data, err := h.files.Read(key)
	if err != nil {
		httpx.NotFound(c)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, "image/jpeg", data)
}
