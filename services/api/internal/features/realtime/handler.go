package realtime

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	platformrealtime "example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	service  *Service
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket handler. Authentication relies on a
// single-use ticket obtained with the bearer token, never on cookies, so a
// cross-site page cannot ride on ambient credentials: the origin check is
// therefore not a security boundary and is left permissive for native apps.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	ticket, err := h.service.IssueTicket(httpx.UserID(c), c.GetString("sessionID"))
	if err != nil {
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not issue ticket", "realtime.ticket", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, TicketResponse{Ticket: ticket, ExpiresIn: int(platformrealtime.TicketTTL.Seconds())})
}

func (h *Handler) Connect(c *gin.Context) {
	ticket, err := h.service.Redeem(c.Request.Context(), c.Query("ticket"))
	if err != nil {
		if errors.Is(err, ErrInvalidTicket) {
			httpx.Fail(c, http.StatusUnauthorized, "invalid_ticket", err.Error(), "realtime.connect", nil)
			return
		}
		httpx.Fail(c, http.StatusServiceUnavailable, "session_check_failed", "could not verify session", "realtime.connect", err)
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	c.Set("userID", ticket.UserID)
	h.service.Serve(ticket, conn)
}
