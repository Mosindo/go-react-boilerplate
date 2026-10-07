package photos

import (
	"io"
	"net/http"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	signer  *media.Signer
}

func NewHandler(service *Service, signer *media.Signer) *Handler {
	return &Handler{service: service, signer: signer}
}

type reorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}

// readUpload reads the multipart "file" field with a hard size bound.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(256<<10))
	header, err := c.FormFile("file")
	if err != nil {
		httpx.Respond(c, "photos.upload.form", platformerrors.Validation("multipart field 'file' is required (max 5 MB)"))
		return nil, false
	}
	if header.Size > maxUploadBytes {
		httpx.Respond(c, "photos.upload.size", platformerrors.Validation("the image must be at most 5 MB"))
		return nil, false
	}
	f, err := header.Open()
	if err != nil {
		httpx.Respond(c, "photos.upload.open", platformerrors.Validation("could not read the uploaded file"))
		return nil, false
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes+1))
	if err != nil {
		httpx.Respond(c, "photos.upload.read", platformerrors.Validation("could not read the uploaded file"))
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "photos.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, httpx.Items(items))
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Respond(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, data)
	if err != nil {
		httpx.Respond(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Respond(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req reorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Respond(c, "photos.reorder", platformerrors.Validation("photoIds must be UUIDs"))
			return
		}
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Respond(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, httpx.Items(items))
}

// Serve streams an image after verifying its signed URL. Visibility was decided when the
// URL was issued (only self, matches and eligible candidates ever receive one).
func (h *Handler) Serve(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) || !h.signer.Verify(id, c.Query("exp"), c.Query("sig")) {
		httpx.Fail(c, http.StatusNotFound, platformerrors.CodeNotFound, "not found")
		return
	}
	file, err := h.service.Open(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, http.StatusNotFound, platformerrors.CodeNotFound, "not found")
		return
	}
	defer func() { _ = file.Close() }()

	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=21600, immutable")
	c.Header("Content-Disposition", "inline")
	c.Header("X-Content-Type-Options", "nosniff")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, file)
}
