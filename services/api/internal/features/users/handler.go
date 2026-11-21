package users

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func toResponse(user User) UserResponse {
	return UserResponse{
		ID:             user.ID,
		DisplayName:    user.DisplayName,
		OrganizationID: user.OrganizationID,
		CreatedAt:      user.CreatedAt,
	}
}

func (h *Handler) List(c *gin.Context) {
	organizationID := httpx.OrganizationID(c)
	if organizationID == "" {
		httpx.Error(c, http.StatusUnauthorized, "missing_organization", "missing organization")
		return
	}
	limit := httpx.ParseLimit(c.Query("limit"), defaultUsersLimit, maxUsersLimit)
	offset := httpx.ParseOffset(c.Query("offset"))
	users, err := h.service.ListWithPagination(c.Request.Context(), organizationID, httpx.UserID(c), limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "users.list", http.StatusInternalServerError, err)
		httpx.Internal(c, "could not fetch users")
		return
	}

	payload := make([]UserResponse, 0, len(users))
	for _, user := range users {
		payload = append(payload, toResponse(user))
	}
	c.JSON(http.StatusOK, UsersResponse{Users: payload})
}

func (h *Handler) GetByID(c *gin.Context) {
	organizationID := httpx.OrganizationID(c)
	if organizationID == "" {
		httpx.Error(c, http.StatusUnauthorized, "missing_organization", "missing organization")
		return
	}
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), organizationID, httpx.UserID(c), userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrUserNotFound) {
			status = http.StatusNotFound
			httpx.Error(c, status, "user_not_found", "user not found")
		} else {
			httpx.Internal(c, "could not fetch user")
		}
		logger.LogHandlerError(c, "users.get", status, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(user))
}
