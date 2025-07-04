package websocket

import (
	"net/http"
	"websocket-backend/internal/cmdrunner"
	"websocket-backend/internal/config" // Make sure config is imported if not already
	"websocket-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// HandlerDependencies holds dependencies required by the WebSocket handler.
type HandlerDependencies struct {
	Config    *config.Config
	CmdRunner *cmdrunner.CommandRunner
}

// NewWebSocketHandler creates a Gin handler function for WebSocket connections.
func NewWebSocketHandler(deps *HandlerDependencies) gin.HandlerFunc {
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

		targetHost := c.Query("target")
		if targetHost == "" {
			utils.Error("WebSocket connection rejected: 'target' query parameter missing.")
			conn.WriteMessage(websocket.TextMessage, []byte("Error: 'target' query parameter is required to identify the remote host."))
			conn.Close()
			return
		}
		utils.Info("Target host for WebSocket connection:", targetHost)

		client := NewClient(conn)

		go client.WriteLoop()

		go client.ReadLoop(
			deps.CmdRunner,
			targetHost,
			func(readErr error) {
				utils.Error("WebSocket ReadLoop error for client connected to", targetHost, ":", readErr)
			},
		)
	}
}
