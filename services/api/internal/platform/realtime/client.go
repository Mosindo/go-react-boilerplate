package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout  = 5 * time.Second
	writeTimeout = 10 * time.Second
	pongTimeout  = 60 * time.Second
	pingInterval = 25 * time.Second
	maxReadBytes = 4096
)

// Authenticator validates the token sent in the first WebSocket frame and returns the user ID.
type Authenticator func(ctx context.Context, token string) (userID string, err error)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	userID    string
	send      chan []byte
	closeOnce sync.Once
	done      chan struct{}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

// Handler upgrades GET /ws. Browsers cannot set headers on WebSocket requests and
// tokens in URLs end up in logs, so the client authenticates with its first
// frame: {"type":"auth","token":"<access token>"}.
func Handler(hub *Hub, authenticate Authenticator, allowedOrigins []string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" { // native clients send no Origin
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.SetReadLimit(maxReadBytes)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var hello struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err := conn.ReadJSON(&hello); err != nil || hello.Type != "auth" {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "auth required"), time.Now().Add(time.Second))
			_ = conn.Close()
			return
		}
		userID, err := authenticate(c.Request.Context(), hello.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"), time.Now().Add(time.Second))
			_ = conn.Close()
			return
		}

		client := &Client{hub: hub, conn: conn, userID: userID, send: make(chan []byte, 32), done: make(chan struct{})}
		hub.register(client)
		go client.writePump()
		client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.close()
	}()
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	})
	for {
		// Clients only listen; any frame (e.g. an app-level ping) keeps the connection alive.
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	ready, _ := json.Marshal(Event{Type: "ready"})
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := c.conn.WriteMessage(websocket.TextMessage, ready); err != nil {
		return
	}
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
