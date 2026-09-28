package photos

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// readUpload streams the multipart part named "file" into memory, refusing
// anything above MaxUploadBytes. The client-declared filename and content type
// are ignored; the service sniffs the bytes.
func readUpload(c *gin.Context) ([]byte, error) {
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, httpx.BadRequest("expected a multipart/form-data body with a 'file' field")
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, httpx.BadRequest("missing 'file' field")
		}
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				return nil, httpx.PayloadTooLarge("upload too large (max 8 MB)")
			}
			return nil, httpx.BadRequest("malformed multipart body")
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		return readPart(part)
	}
}

func readPart(part *multipart.Part) ([]byte, error) {
	defer part.Close()
	data, err := io.ReadAll(io.LimitReader(part, MaxUploadBytes+1))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, httpx.PayloadTooLarge("upload too large (max 8 MB)")
		}
		return nil, httpx.BadRequest("malformed multipart body")
	}
	if len(data) > MaxUploadBytes {
		return nil, httpx.PayloadTooLarge("upload too large (max 8 MB)")
	}
	if len(data) == 0 {
		return nil, httpx.BadRequest("empty file")
	}
	return data, nil
}

func (h *Handler) Upload(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	p, err := h.service.Add(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	p, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), c.Param("id"), data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Content(c *gin.Context) {
	data, err := h.service.Content(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "image/jpeg", data)
}
