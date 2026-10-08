package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			platformerrors.Respond(c, errTooLarge)
		} else {
			platformerrors.Respond(c, platformerrors.BadRequest("missing_file", "multipart field \"file\" is required"))
		}
		return nil, false
	}
	if fh.Size > MaxUploadBytes {
		platformerrors.Respond(c, errTooLarge)
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		platformerrors.Respond(c, err)
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		platformerrors.Respond(c, err)
		return nil, false
	}
	return data, true
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	list, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), data)
	h.reply(c, http.StatusCreated, list, err)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	list, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, data)
	h.reply(c, http.StatusOK, list, err)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	list, err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id)
	h.reply(c, http.StatusOK, list, err)
}

type reorderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6,dive,max=40"`
}

func (h *Handler) Reorder(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platformerrors.RespondBind(c, err)
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs)
	h.reply(c, http.StatusOK, list, err)
}

func (h *Handler) Serve(c *gin.Context) {
	id, ok := platformerrors.ParamUUID(c, "id")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), c.GetString("userID"), id, c.Query("size") == "thumb")
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	// Photo ids are immutable: replacing a photo issues a new id.
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}

func (h *Handler) reply(c *gin.Context, status int, list any, err error) {
	if err != nil {
		platformerrors.Respond(c, err)
		return
	}
	c.JSON(status, gin.H{"photos": list})
}
