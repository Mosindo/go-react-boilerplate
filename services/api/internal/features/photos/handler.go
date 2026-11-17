package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// readUpload reads the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "La photo dépasse 10 Mo.")
			return nil, false
		}
		httpx.Fail(c, http.StatusBadRequest, "photo_required", "Ajoute une photo (champ « photo »).")
		return nil, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_upload", "Envoi de la photo impossible.")
		return nil, false
	}
	if len(data) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "La photo dépasse 10 Mo.")
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "photos.list", http.StatusOK, photos, err)
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	h.respond(c, "photos.upload", http.StatusCreated, photos, err)
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
	photos, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	h.respond(c, "photos.replace", http.StatusOK, photos, err)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	photos, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	h.respond(c, "photos.delete", http.StatusOK, photos, err)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, "photos.reorder", &req) {
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, http.StatusBadRequest, "invalid_id", "Identifiant invalide.")
			return
		}
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	h.respond(c, "photos.reorder", http.StatusOK, photos, err)
}

// Serve streams a photo for a valid signed URL. No bearer token is needed so that
// image components can load it; the HMAC signature and expiry protect access.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrPhotoNotFound) {
			httpx.Fail(c, http.StatusNotFound, "photo_not_found", "Photo introuvable.")
			return
		}
		httpx.FailInternal(c, "photos.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", outputMediaType)
	c.Header("Cache-Control", "private, max-age=21600, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}

func (h *Handler) respond(c *gin.Context, op string, status int, photos []profiles.Photo, err error) {
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedType):
			httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", "Format non pris en charge (JPEG ou PNG).")
		case errors.Is(err, ErrImageTooSmall):
			httpx.Fail(c, http.StatusBadRequest, "image_too_small", "La photo est trop petite (200 px minimum).")
		case errors.Is(err, ErrImageTooLarge):
			httpx.Fail(c, http.StatusBadRequest, "image_too_large", "La photo est trop grande (8000 px maximum).")
		case errors.Is(err, ErrCorruptImage):
			httpx.Fail(c, http.StatusBadRequest, "corrupt_image", "Fichier image illisible.")
		case errors.Is(err, ErrTooManyPhotos):
			httpx.Fail(c, http.StatusConflict, "too_many_photos", "6 photos maximum.")
		case errors.Is(err, ErrPhotoNotFound):
			httpx.Fail(c, http.StatusNotFound, "photo_not_found", "Photo introuvable.")
		case errors.Is(err, ErrRepoUserAbsent):
			httpx.Fail(c, http.StatusUnauthorized, "user_not_found", "Compte introuvable.")
		case errors.Is(err, ErrInvalidOrder):
			httpx.Fail(c, http.StatusBadRequest, "invalid_order", "L'ordre doit contenir toutes tes photos, une seule fois.")
		default:
			httpx.FailInternal(c, op, err)
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, PhotosResponse{Photos: photos})
}
