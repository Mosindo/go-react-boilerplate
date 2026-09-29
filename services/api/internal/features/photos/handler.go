package photos

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

// multipartOverhead leaves room for boundaries and headers around the 5 MB file.
const multipartOverhead = 64 << 10

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func jsonErr(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

func serverErr(c *gin.Context, op string, err error) {
	logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
	jsonErr(c, http.StatusInternalServerError, "internal error")
}

func isTooLarge(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

func (h *Handler) Upload(c *gin.Context) {
	if c.Request.ContentLength > MaxUploadBytes+multipartOverhead {
		jsonErr(c, http.StatusRequestEntityTooLarge, "file is too large (max 5 MB)")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+multipartOverhead)
	mr, err := c.Request.MultipartReader()
	if err != nil {
		jsonErr(c, http.StatusBadRequest, "expected multipart/form-data with a file field")
		return
	}
	var data []byte
	found := false
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			if isTooLarge(err) {
				jsonErr(c, http.StatusRequestEntityTooLarge, "file is too large (max 5 MB)")
				return
			}
			jsonErr(c, http.StatusBadRequest, "invalid multipart body")
			return
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		data, err = io.ReadAll(io.LimitReader(part, MaxUploadBytes+1))
		_ = part.Close()
		if err != nil {
			if isTooLarge(err) {
				jsonErr(c, http.StatusRequestEntityTooLarge, "file is too large (max 5 MB)")
				return
			}
			jsonErr(c, http.StatusBadRequest, "invalid multipart body")
			return
		}
		if len(data) > MaxUploadBytes {
			jsonErr(c, http.StatusRequestEntityTooLarge, "file is too large (max 5 MB)")
			return
		}
		found = true
		break
	}
	if !found || len(data) == 0 {
		jsonErr(c, http.StatusBadRequest, "file field is required")
		return
	}

	p, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), data)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, p)
	case errors.Is(err, ErrLimitReached):
		jsonErr(c, http.StatusUnprocessableEntity, "you can upload at most 6 photos")
	case errors.Is(err, ErrUnsupportedImage), errors.Is(err, ErrCorruptImage):
		jsonErr(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrImageTooBig):
		jsonErr(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrNotFound):
		jsonErr(c, http.StatusUnauthorized, "account not found")
	default:
		serverErr(c, "photos.upload", err)
	}
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonErr(c, http.StatusBadRequest, "invalid request")
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"photos": list})
	case errors.Is(err, ErrInvalidIDs):
		jsonErr(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotAPermutation):
		jsonErr(c, http.StatusUnprocessableEntity, err.Error())
	default:
		serverErr(c, "photos.reorder", err)
	}
}

func (h *Handler) Delete(c *gin.Context) {
	err := h.service.Delete(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		jsonErr(c, http.StatusNotFound, "photo not found")
	default:
		serverErr(c, "photos.delete", err)
	}
}

// File serves a photo to anyone holding a valid signed URL.
func (h *Handler) File(c *gin.Context) {
	notFound := func() { jsonErr(c, http.StatusNotFound, "not found") }
	exp, err := strconv.ParseInt(c.Query("exp"), 10, 64)
	if err != nil {
		notFound()
		return
	}
	rc, size, err := h.service.Open(c.Request.Context(), c.Param("id"), exp, c.Query("sig"))
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			logger.LogHandlerError(c, "photos.file", http.StatusNotFound, err)
		}
		notFound()
		return
	}
	defer rc.Close()
	hd := c.Writer.Header()
	hd.Set("Content-Type", "image/jpeg")
	hd.Set("Cache-Control", "private, max-age=3600")
	hd.Set("X-Content-Type-Options", "nosniff")
	// Signed, private URL loaded by the web client from another origin.
	hd.Set("Cross-Origin-Resource-Policy", "cross-origin")
	if size > 0 {
		hd.Set("Content-Length", strconv.FormatInt(size, 10))
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}
