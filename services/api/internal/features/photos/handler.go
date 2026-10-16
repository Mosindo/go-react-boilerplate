package photos

import (
	"errors"
	"mime/multipart"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/mediaurl"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	signer  *mediaurl.Signer
}

func NewHandler(service *Service, signer *mediaurl.Signer) *Handler {
	return &Handler{service: service, signer: signer}
}

type orderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=9"`
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "photo not found", err)
	case errors.Is(err, ErrLimit):
		httpx.Fail(c, op, http.StatusConflict, "you can have at most 6 photos", err)
	case errors.Is(err, ErrBadOrder):
		httpx.Fail(c, op, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrTooLarge):
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "image must be 8 MB or less", err)
	case errors.Is(err, ErrUnsupportedType), errors.Is(err, ErrBadDimensions), errors.Is(err, ErrCorruptImage):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, err.Error(), err)
	default:
		httpx.Fail(c, op, http.StatusInternalServerError, "internal error", err)
	}
}

// readUpload bounds the request body before parsing the multipart form.
func readUpload(c *gin.Context) (multipart.File, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(512<<10))
	fh, err := c.FormFile("photo")
	if err != nil {
		return nil, err
	}
	return fh.Open()
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := readUpload(c)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, "photos.upload.size", http.StatusRequestEntityTooLarge, "image must be 8 MB or less", err)
			return
		}
		httpx.Fail(c, "photos.upload.form", http.StatusBadRequest, "send the image in the \"photo\" form field", err)
		return
	}
	defer file.Close()

	id, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), file)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "url": h.signer.URL(id)})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		httpx.Fail(c, "photos.replace.id", http.StatusBadRequest, "invalid photo id", errors.New("invalid photo id"))
		return
	}
	file, err := readUpload(c)
	if err != nil {
		httpx.Fail(c, "photos.replace.form", http.StatusBadRequest, "send the image in the \"photo\" form field", err)
		return
	}
	defer file.Close()

	if err := h.service.Replace(c.Request.Context(), c.GetString("userID"), photoID, file); err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": photoID, "url": h.signer.URL(photoID)})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		httpx.Fail(c, "photos.delete.id", http.StatusBadRequest, "invalid photo id", errors.New("invalid photo id"))
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), photoID); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req orderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, "photos.reorder.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, "photos.reorder.id", http.StatusBadRequest, "invalid photo id", errors.New("invalid photo id"))
			return
		}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) MakeMain(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		httpx.Fail(c, "photos.main.id", http.StatusBadRequest, "invalid photo id", errors.New("invalid photo id"))
		return
	}
	if err := h.service.MakeMain(c.Request.Context(), c.GetString("userID"), photoID); err != nil {
		h.fail(c, "photos.main", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Serve streams a photo for a valid signed URL. It carries no Authorization header
// on purpose so plain image components can load it; the signature is the credential.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok || !h.signer.Verify(photoID, c.Query("exp"), c.Query("sig")) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	f, err := h.service.Open(c.Request.Context(), photoID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		httpx.Fail(c, "photos.serve", http.StatusInternalServerError, "internal error", err)
		return
	}
	defer f.Close()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
