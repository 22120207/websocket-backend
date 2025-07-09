package websocket

import (
	"encoding/json"
	"net/http"
	"websocket-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
			utils.Error("WebSocket upgrade failed:", err)
			return
		}
		utils.Info("WebSocket connection established.")

		client := NewClient(conn)

		go client.WriteLoop()

		go client.ReadLoop(
			func(readErr error) {
				utils.Error("WebSocket ReadLoop error", ":", readErr)
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
			return
		}

		if err := client.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			return
		}
	}
}
