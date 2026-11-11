package realtime

import (
	"net/http"
	"net/url"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/middleware"
	hub "example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/tokens"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Handler struct {
	service  *Service
	hub      *hub.Hub
	sessions middleware.SessionChecker
	upgrader websocket.Upgrader
}

// NewHandler builds the WebSocket handler. Browser origins are checked against
// allowedOrigins; native apps send no Origin header and are accepted.
func NewHandler(service *Service, h *hub.Hub, sessions middleware.SessionChecker, allowedOrigins []string) *Handler {
	allowed := map[string]bool{}
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return &Handler{
		service:  service,
		hub:      h,
		sessions: sessions,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" || allowed[origin] {
					return true
				}
				u, err := url.Parse(origin)
				if err != nil {
					return false
				}
				// Same host as the API (e.g. served behind the same domain).
				return strings.EqualFold(u.Host, r.Host)
			},
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	t, err := h.service.IssueTicket(httpx.UserID(c), httpx.SessionID(c))
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	c.JSON(http.StatusOK, TicketResponse{Ticket: t, ExpiresIn: int(tokens.RealtimeTTL.Seconds())})
}

func (h *Handler) Connect(c *gin.Context) {
	claims, err := h.service.ParseTicket(c.Query("ticket"))
	if err != nil {
		httpx.Fail(c, http.StatusUnauthorized, "invalid_ticket", "Ticket temps réel invalide.", nil)
		return
	}
	active, err := h.sessions.IsSessionActive(c.Request.Context(), claims.SessionID, claims.UserID)
	if err != nil {
		httpx.Internal(c, err)
		return
	}
	if !active {
		httpx.Fail(c, http.StatusUnauthorized, "invalid_ticket", "Session expirée.", nil)
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // the upgrader already wrote the HTTP error
	}
	h.hub.Serve(conn, claims.UserID)
}
