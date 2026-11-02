package photos

import (
	"errors"
	"net/http"
	"strings"

	perr "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
)

const MaxPhotos = 6

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func translate(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return perr.NotFound("Photo introuvable.")
	case errors.Is(err, ErrLimit):
		return perr.Conflict("photo_limit", "6 photos maximum.")
	case errors.Is(err, ErrLastPhoto):
		return perr.Conflict("last_photo", "Gardez au moins une photo, remplacez-la plutôt.")
	case errors.Is(err, ErrBadOrder):
		return perr.BadRequest("invalid_order", "L'ordre doit contenir chaque photo exactement une fois.")
	}
	return err
}

func (h *Handler) process(c *gin.Context) (Processed, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Champ « file » manquant ou trop volumineux."))
		return Processed{}, false
	}
	f, err := fh.Open()
	if err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Fichier illisible."))
		return Processed{}, false
	}
	defer f.Close()
	p, err := Process(f)
	if err != nil {
		perr.Respond(c, err)
		return Processed{}, false
	}
	return p, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.repo.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		perr.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Upload(c *gin.Context) {
	p, ok := h.process(c)
	if !ok {
		return
	}
	ph, err := h.repo.Add(c.Request.Context(), c.GetString("userID"), p, MaxPhotos)
	if err != nil {
		perr.Respond(c, translate(err))
		return
	}
	c.JSON(http.StatusCreated, ph)
}

func (h *Handler) Replace(c *gin.Context) {
	p, ok := h.process(c)
	if !ok {
		return
	}
	ph, err := h.repo.Replace(c.Request.Context(), c.GetString("userID"), c.Param("id"), p)
	if err != nil {
		perr.Respond(c, translate(err))
		return
	}
	c.JSON(http.StatusOK, ph)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.repo.Delete(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		perr.Respond(c, translate(err))
		return
	}
	c.Status(http.StatusNoContent)
}

type orderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}

func (h *Handler) Reorder(c *gin.Context) {
	var req orderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		perr.Respond(c, perr.BadRequest("invalid_request", "Requête invalide."))
		return
	}
	if err := h.repo.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs); err != nil {
		perr.Respond(c, translate(err))
		return
	}
	h.List(c)
}

// SetMain moves the photo to position 0, shifting the others down.
func (h *Handler) SetMain(c *gin.Context) {
	userID := c.GetString("userID")
	items, err := h.repo.List(c.Request.Context(), userID)
	if err != nil {
		perr.Respond(c, err)
		return
	}
	ids := []string{c.Param("id")}
	found := false
	for _, it := range items {
		if it.ID == c.Param("id") {
			found = true
			continue
		}
		ids = append(ids, it.ID)
	}
	if !found {
		perr.Respond(c, perr.NotFound("Photo introuvable."))
		return
	}
	if err := h.repo.Reorder(c.Request.Context(), userID, ids); err != nil {
		perr.Respond(c, translate(err))
		return
	}
	h.List(c)
}

func (h *Handler) Serve(c *gin.Context) {
	id := c.Param("id")
	if len(id) != 36 || strings.ContainsAny(id, "/\\ ") {
		perr.Respond(c, perr.NotFound("Photo introuvable."))
		return
	}
	b, err := h.repo.Fetch(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		perr.Respond(c, translate(err))
		return
	}
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, b.ContentType, b.Data)
}
