package realtime

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
	maxInbound = 512
)

type Handler struct {
	service  *Service
	upgrader websocket.Upgrader
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 4096,
			// Authentication relies on the single-use-window ticket, not on
			// ambient cookies, so cross-site WebSocket hijacking is not a
			// concern and any origin (native apps send none) is accepted.
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
}

func (h *Handler) Ticket(c *gin.Context) {
	resp, err := h.service.IssueTicket(httpx.UserID(c), httpx.SessionID(c))
	if err != nil {
		httpx.Fail(c, "realtime.ticket", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Connect(c *gin.Context) {
	userID, err := h.service.Authenticate(c.Request.Context(), c.Query("ticket"))
	if err != nil {
		httpx.Fail(c, "realtime.connect", err)
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	hub := h.service.Hub()
	client := hub.Register(userID)

	// Reader: only used for pong / close detection; clients send nothing else.
	go func() {
		defer func() {
			hub.Unregister(client)
			conn.Close()
		}()
		conn.SetReadLimit(maxInbound)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Writer: forwards hub events and keeps the connection alive.
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			conn.Close()
		}()
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ready","data":{}}`)); err != nil {
			return
		}
		for {
			select {
			case payload, ok := <-client.Send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()
}
