package photos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
)

type MineProvider interface {
	GetMine(ctx context.Context, userID string) (profiles.MyProfileResponse, error)
}

type Handler struct {
	service *Service
	mine    MineProvider
	signer  *media.Signer
}

func NewHandler(service *Service, mine MineProvider, signer *media.Signer) *Handler {
	return &Handler{service: service, mine: mine, signer: signer}
}

func readUpload(c *gin.Context) ([]byte, bool) {
	fh, err := c.FormFile("photo")
	if err != nil {
		httpx.BadRequest(c, "photo_required", "Ajoute une photo (champ « photo »).")
		return nil, false
	}
	if fh.Size > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "photo_too_large", "Photo trop lourde (10 Mo maximum).", nil)
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Internal(c, err)
		return nil, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		httpx.Internal(c, err)
		return nil, false
	}
	if len(raw) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "photo_too_large", "Photo trop lourde (10 Mo maximum).", nil)
		return nil, false
	}
	return raw, true
}

func (h *Handler) respond(c *gin.Context, status int) {
	resp, err := h.mine.GetMine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(status, resp)
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	if err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw); err != nil {
		fail(c, err)
		return
	}
	h.respond(c, http.StatusCreated)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	if err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, raw); err != nil {
		fail(c, err)
		return
	}
	h.respond(c, http.StatusOK)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID); err != nil {
		fail(c, err)
		return
	}
	h.respond(c, http.StatusOK)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs); err != nil {
		fail(c, err)
		return
	}
	h.respond(c, http.StatusOK)
}

// Serve streams a stored photo when the signed URL is valid.
func (h *Handler) Serve(c *gin.Context) {
	key := c.Param("key")
	exp, sig := c.Query("exp"), c.Query("sig")
	if !storage.ValidKey(key) || !h.signer.Verify(key, exp, sig) {
		httpx.Fail(c, http.StatusForbidden, "invalid_signature", "Lien expiré ou invalide.", nil)
		return
	}
	exists, err := h.service.Exists(c.Request.Context(), key)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	if !exists {
		httpx.Fail(c, http.StatusNotFound, "not_found", "Photo introuvable.", nil)
		return
	}
	f, err := h.service.Store().Open(key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "Photo introuvable.", nil)
			return
		}
		httpx.Internal(c, err)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Length", strconv.FormatInt(info.Size(), 10))
	c.Header("Cache-Control", "private, max-age="+strconv.Itoa(int(h.signer.MaxAge(exp).Seconds())))
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}

func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", "Format non supporté (JPEG, PNG ou WebP).", nil)
	case errors.Is(err, ErrImageTooSmall):
		httpx.BadRequest(c, "image_too_small", "Photo trop petite (200×200 minimum).")
	case errors.Is(err, ErrImageTooLarge):
		httpx.BadRequest(c, "image_too_large", "Dimensions de la photo trop grandes.")
	case errors.Is(err, ErrCorruptImage):
		httpx.BadRequest(c, "corrupt_image", "Le fichier n'est pas une image valide.")
	case errors.Is(err, ErrLimitReached):
		httpx.Fail(c, http.StatusConflict, "photo_limit", "Tu as déjà 6 photos.", nil)
	case errors.Is(err, ErrNoProfile):
		httpx.Fail(c, http.StatusConflict, "profile_required", "Crée d'abord ton profil.", nil)
	case errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, "invalid_order", "Ordre des photos invalide.")
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "Photo introuvable.", nil)
	default:
		httpx.Internal(c, err)
	}
}
