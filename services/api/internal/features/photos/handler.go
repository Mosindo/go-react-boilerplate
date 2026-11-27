package photos

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_image", "Formats acceptés : JPEG, PNG ou WebP.", err)
	case errors.Is(err, ErrImageTooLarge):
		httpx.BadRequest(c, "image_too_large", "Image trop grande (10 000 px maximum par côté).")
	case errors.Is(err, ErrImageTooSmall):
		httpx.BadRequest(c, "image_too_small", "Image trop petite (200 px minimum par côté).")
	case errors.Is(err, ErrCorruptImage):
		httpx.BadRequest(c, "corrupt_image", "Image illisible ou corrompue.")
	case errors.Is(err, ErrTooManyPhotos):
		httpx.Fail(c, http.StatusConflict, "too_many_photos", "Tu peux ajouter jusqu'à 6 photos.", err)
	case errors.Is(err, ErrProfileRequired):
		httpx.Fail(c, http.StatusConflict, "profile_required", "Crée d'abord ton profil.", err)
	case errors.Is(err, ErrPhotoNotFound):
		httpx.NotFound(c, "Photo introuvable.")
	case errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, "invalid_order", "L'ordre doit contenir chacune de tes photos une seule fois.")
	default:
		httpx.Internal(c, err)
	}
}

// readUpload reads the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64*1024)
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "Photo trop lourde (10 Mo maximum).", err)
			return nil, false
		}
		httpx.BadRequest(c, "photo_required", "Ajoute une photo (champ « photo »).")
		return nil, false
	}
	defer file.Close()
	if header.Size > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "Photo trop lourde (10 Mo maximum).", nil)
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil || len(data) == 0 {
		httpx.BadRequest(c, "photo_required", "Ajoute une photo (champ « photo »).")
		return nil, false
	}
	if len(data) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "Photo trop lourde (10 Mo maximum).", nil)
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	list, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: list})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	list, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	list, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	ids := make([]string, 0, len(req.PhotoIDs))
	for _, id := range req.PhotoIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if !httpx.IsUUID(id) {
			httpx.BadRequest(c, "invalid_id", "Identifiant invalide.")
			return
		}
		ids = append(ids, id)
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), ids)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

// Media serves a photo for a valid signed URL. Signatures expire, so a
// leaked link stops working; responses are private-cacheable.
func (h *Handler) Media(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	f, mod, err := h.service.OpenSigned(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrPhotoNotFound) {
			httpx.NotFound(c, "Photo introuvable.")
			return
		}
		httpx.Internal(c, err)
		return
	}
	defer f.Close()

	headers := c.Writer.Header()
	headers.Set("Content-Type", "image/jpeg")
	headers.Set("Cache-Control", "private, max-age=21600")
	headers.Set("Cross-Origin-Resource-Policy", "cross-origin")
	headers.Set("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(c.Writer, c.Request, "", mod, f)
}
