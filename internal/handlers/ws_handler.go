package handlers

import (
	"log"
	"net/http"
	"time"

	"task-management-api/internal/realtime"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// wsClient implements realtime.Client by wrapping a websocket connection.
type wsClient struct {
	conn *websocket.Conn
}

func (c *wsClient) Send(message []byte) bool {
	if c == nil || c.conn == nil {
		return false
	}
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return false
	}
	return true
}

func (c *wsClient) Close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Close()
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// CORS is already handled at Gin level; allow upgrade from any origin here
		return true
	},
}

// WebSocketHandler upgrades the connection and registers the client to the hub.
// It requires JWT middleware to have set "user_id" in context.
func WebSocketHandler(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authorized"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("websocket upgrade error:", err)
		return
	}

	client := &wsClient{conn: conn}
	hub := realtime.GetHub()
	hub.Register(userID, client)

	// Heartbeat: send periodic pings; close on error
	pingTicker := time.NewTicker(30 * time.Second)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-pingTicker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					// ping failed; reader loop will exit on next error
					return
				}
			}
		}
	}()
	defer func() {
		close(done)
		pingTicker.Stop()
		hub.Unregister(userID, client)
		client.Close()
	}()

	// Reader loop: drain messages and keep connection alive via pong handler
	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			// Normal close or error; exit loop
			return
		}
	}
}
