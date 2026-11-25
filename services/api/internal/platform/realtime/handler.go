package realtime

import (
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

type TicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresAt string `json:"expiresAt"`
}

// RegisterRoutes exposes:
//   - POST /realtime/ticket (authenticated): returns a single-use ticket
//   - GET  /realtime?ticket=...: upgrades to a WebSocket stream of events
func RegisterRoutes(r gin.IRouter, hub *Hub, requireUser, ticketLimit gin.HandlerFunc) {
	r.POST("/realtime/ticket", requireUser, ticketLimit, func(c *gin.Context) {
		value, expiresAt, err := hub.IssueTicket(httpx.UserID(c), c.GetString("sessionID"))
		if err != nil {
			httpx.Fail(c, "realtime.ticket", err)
			return
		}
		httpx.NoStore(c)
		c.JSON(http.StatusOK, TicketResponse{Ticket: value, ExpiresAt: expiresAt.UTC().Format("2006-01-02T15:04:05Z07:00")})
	})

	r.GET("/realtime", func(c *gin.Context) {
		userID, sessionID, ok := hub.RedeemTicket(strings.TrimSpace(c.Query("ticket")))
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, httpx.ErrorBody{Error: "invalid ticket", Code: "invalid_ticket"})
			return
		}
		// The single-use ticket (obtained with a bearer token) authenticates
		// the upgrade, so cross-origin upgrades cannot be abused and the
		// origin check is not needed (Expo web runs on another origin).
		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		hub.Serve(c.Request.Context(), conn, userID, sessionID)
	})
}
