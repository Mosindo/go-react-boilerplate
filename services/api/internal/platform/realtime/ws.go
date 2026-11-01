package realtime

import (
	"net/http"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	authTimeout = 5 * time.Second
	pongWait    = 60 * time.Second
	pingEvery   = 25 * time.Second
	writeWait   = 10 * time.Second
	maxReadSize = 4096
)

// Handler gere GET /ws. Le client doit envoyer, en premier message,
// {"type":"auth","token":"<access token>"} dans les 5 secondes (le token n'est
// ainsi jamais place dans l'URL ni dans les logs).
func (h *Hub) Handler(secret []byte, allowedOrigins []string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" { // clients natifs: pas d'Origin
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
		conn.SetReadLimit(maxReadSize)

		_ = conn.SetReadDeadline(time.Now().Add(authTimeout))
		var hello struct {
			Type  string `json:"type"`
			Token string `json:"token"`
		}
		if err := conn.ReadJSON(&hello); err != nil || hello.Type != "auth" {
			_ = conn.Close()
			return
		}
		identity, err := middleware.ParseAccessToken(secret, hello.Token)
		if err != nil {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "unauthorized"), time.Now().Add(writeWait))
			_ = conn.Close()
			return
		}

		cl := &client{userID: identity.UserID, send: make(chan []byte, 32)}
		h.register(cl)
		defer func() {
			h.unregister(cl)
			_ = conn.Close()
		}()

		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		})

		done := make(chan struct{})
		go func() { // lecteur: ignore les messages (canal serveur -> client), detecte la fermeture
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
				_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			}
		}()

		_ = conn.WriteJSON(Event{Type: "ready", Payload: gin.H{}})
		ticker := time.NewTicker(pingEvery)
		defer ticker.Stop()
		for {
			select {
			case msg := <-cl.send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}
}
