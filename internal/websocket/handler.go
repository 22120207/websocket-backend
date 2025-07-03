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
			// For development, allow all origins.
			// In production, restrict this to your actual frontend domain(s).
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
		// No need for a global pool unless you explicitly manage connections elsewhere
		// wsPool.Register(client) // if you have a pool, uncomment this

		// Start goroutines for reading from and writing to the WebSocket
		go client.WriteLoop()
		// Pass dependencies needed for command execution to the ReadLoop
		go client.ReadLoop(
			deps.CmdRunner, // Pass the CommandRunner
			targetHost,     // Pass the target host
			func(readErr error) { // Error callback for ReadLoop
				utils.Error("WebSocket ReadLoop error for client connected to", targetHost, ":", readErr)
				// wsPool.Unregister(client) // if you have a pool, uncomment this
			},
		)
	}
}
