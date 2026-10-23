package photos

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

var errTooLarge = platformerrors.New(http.StatusRequestEntityTooLarge, "file_too_large", "image must be at most 8 MB")

// readUpload extracts the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, error) {
	// Slack for multipart framing on top of the file itself.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, errTooLarge
		}
		return nil, platformerrors.BadRequest("invalid_request", `multipart field "photo" is required`)
	}
	defer file.Close()
	if header.Size > MaxUploadBytes {
		return nil, errTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		return nil, platformerrors.BadRequest("invalid_request", "could not read upload")
	}
	if len(data) > MaxUploadBytes {
		return nil, errTooLarge
	}
	return data, nil
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if list == nil {
		list = []Photo{}
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) Upload(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	data, err := readUpload(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": list})
}

func (h *Handler) Serve(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	size := c.DefaultQuery("size", "full")
	if size != "full" && size != "thumb" {
		httpx.Fail(c, platformerrors.BadRequest("invalid_request", "size must be thumb or full"))
		return
	}
	blob, err := h.service.Blob(c.Request.Context(), httpx.UserID(c), id, size == "thumb")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	etag := fmt.Sprintf(`"%s-%s-%s"`, id, size, strconv.FormatInt(blob.UpdatedAt.UnixNano(), 36))
	c.Header("ETag", etag)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "image/jpeg", blob.Data)
}
