package files

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	userID := c.GetString("userID")
	limit := parseBoundedLimit(c.Query("limit"), 20, 100)
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	files, err := h.service.ListByOwner(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "files.list", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch files"})
		return
	}

	payload := make([]FileResponse, 0, len(files))
	for _, file := range files {
		payload = append(payload, mapFileResponse(file))
	}

	c.JSON(http.StatusOK, FilesResponse{Files: payload})
}

func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("userID")

	var req CreateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "files.create.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	file, err := h.service.Create(c.Request.Context(), userID, req.Filename, req.MimeType, req.SizeBytes, req.StorageKey)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrFilenameRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "filename required"})
		case errors.Is(err, ErrMimeTypeRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "mime type required"})
		case errors.Is(err, ErrStorageKeyNeeded):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "storage key required"})
		case errors.Is(err, ErrInvalidFileSize):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "invalid file size"})
		case errors.Is(err, ErrFilenameTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "filename too long"})
		case errors.Is(err, ErrMimeTypeTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "mime type too long"})
		case errors.Is(err, ErrStorageKeyTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "storage key too long"})
		case errors.Is(err, ErrFileTooLarge):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "file too large"})
		default:
			c.JSON(status, gin.H{"error": "could not create file"})
		}
		logger.LogHandlerError(c, "files.create", status, err)
		return
	}

	status := http.StatusCreated
	c.JSON(status, mapFileResponse(file))
	logger.LogHandlerEvent(c, "files.create.success", status, map[string]string{"file_id": file.ID})
}

func (h *Handler) GetByID(c *gin.Context) {
	userID := c.GetString("userID")
	fileID := strings.TrimSpace(c.Param("fileId"))
	if !isUUIDLike(fileID) {
		logger.LogHandlerError(c, "files.get.validate_file_id", http.StatusBadRequest, errors.New("invalid file id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file id"})
		return
	}

	file, err := h.service.GetByID(c.Request.Context(), userID, fileID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrFileNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "file not found"})
		default:
			c.JSON(status, gin.H{"error": "could not fetch file"})
		}
		logger.LogHandlerError(c, "files.get", status, err)
		return
	}

	c.JSON(http.StatusOK, mapFileResponse(file))
}

func mapFileResponse(file File) FileResponse {
	return FileResponse{
		ID:          file.ID,
		OwnerUserID: file.OwnerUserID,
		Filename:    file.Filename,
		MimeType:    file.MimeType,
		SizeBytes:   file.SizeBytes,
		StorageKey:  file.StorageKey,
		CreatedAt:   file.CreatedAt,
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseBoundedLimit(raw string, fallback, max int) int {
	parsed := parsePositiveInt(raw, fallback)
	if parsed > max {
		return max
	}
	return parsed
}

func parseNonNegativeInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func isUUIDLike(v string) bool {
	if len(v) != 36 {
		return false
	}
	hyphenPos := map[int]struct{}{8: {}, 13: {}, 18: {}, 23: {}}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if _, ok := hyphenPos[i]; ok {
			if ch != '-' {
				return false
			}
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}
