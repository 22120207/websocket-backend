package websocket

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// NewWebSocketHandler creates a Gin handler function for WebSocket connections.
func NewWebSocketHandler() gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error("WebSocket upgrade failed:", err)
			return
		}
		log.Info("WebSocket connection established.")

		client := NewClient(conn)

		go client.WriteLoop()

		go client.ReadLoop(
			func(readErr error) {
				log.Error("WebSocket ReadLoop error", ":", readErr)
			},
		)

		// Send data in json format
		type MsgStatus struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		}

		msg := MsgStatus{
			Status:  "ok",
			Message: "websocket connect success",
		}

		jsonData, err := json.Marshal(msg)
		if err != nil {
			log.Error(err.Error())
			return
		}

		if err := client.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			log.Error("Error writing message to WebSocket:", err)
			return
		}
	}
}
