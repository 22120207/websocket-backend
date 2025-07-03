package websocket

import (
	"context"
	"encoding/base64" // Keep import, as cmdrunner still uses it for decoding
	"fmt"
	"sync"
	"time"
	"websocket-backend/internal/cmdrunner"
	"websocket-backend/pkg/utils"

	"github.com/gorilla/websocket"
)

// Client represents a single WebSocket client connection.
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once
	mu       sync.Mutex // For protecting access to conn (e.g., Close)
	isClosed bool
}

// NewClient creates a new WebSocket client instance.
func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256), // Buffered channel for outgoing messages
		ctx:    ctx,
		cancel: cancel,
	}
}

// Send queues a message to be sent to the WebSocket client.
// This is used to send command output and status messages back to the client.
func (c *Client) Send(message []byte) {
	c.mu.Lock()
	closed := c.isClosed
	c.mu.Unlock()

	if closed {
		utils.Debug("Attempted to send message to a closed WebSocket client.")
		return
	}

	select {
	case c.send <- message:
		// Message sent to channel
	case <-c.ctx.Done():
		utils.Debug("Context done, not sending message to WebSocket client.")
	default:
		// If channel is full and context not done, log a warning
		utils.Error("WebSocket send channel is full, dropping message.")
	}
}

// GetContext returns the client's context, useful for external goroutines
// to listen for client disconnection.
func (c *Client) GetContext() context.Context {
	return c.ctx
}

// Close gracefully closes the client connection.
func (c *Client) Close() {
	c.once.Do(func() {
		c.mu.Lock()
		c.isClosed = true
		c.mu.Unlock()

		c.cancel()     // Signal context cancellation
		close(c.send)  // Close the send channel
		c.conn.Close() // Close the underlying WebSocket connection
		utils.Info("WebSocket client connection closed.")
	})
}

// WriteLoop continuously reads messages from the send channel and writes them to the WebSocket.
func (c *Client) WriteLoop() {
	ticker := time.NewTicker(time.Second * 9) // Ping interval for WebSocket keep-alive
	defer func() {
		ticker.Stop()
		utils.Info("WebSocket WriteLoop finished.")
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok { // Channel closed, indicating client.Close() was called
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				utils.Error("Error writing message to WebSocket:", err)
				return // Exit loop on write error
			}
		case <-ticker.C: // Send ping message to keep connection alive
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				utils.Error("Error sending ping message to WebSocket:", err)
				return // Exit loop on ping error
			}
		case <-c.ctx.Done(): // Context cancelled, signaling client disconnection
			utils.Info("WebSocket WriteLoop context cancelled.")
			return
		}
	}
}

// ReadLoop continuously reads messages from the WebSocket.
// It expects the first message to be the Base64-encoded command to execute.
func (c *Client) ReadLoop(
	cmdRunner *cmdrunner.CommandRunner, // Dependency: CommandRunner to execute commands
	targetHost string, // Dependency: Target host for SSH
	onError func(err error), // Callback for handling read errors
) {
	defer func() {
		c.Close() // Ensure client connection is closed when ReadLoop exits
		utils.Info("WebSocket ReadLoop finished for target:", targetHost)
	}()

	c.conn.SetReadLimit(512)                                 // Max message size
	c.conn.SetReadDeadline(time.Now().Add(time.Second * 60)) // Set initial read deadline
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(time.Second * 60))
		return nil
	})

	// Ensure that one command is processed at a time
	var commandReceived bool

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				utils.Error("Unexpected WebSocket close error for client connected to", targetHost, ":", err)
			}
			onError(err)
			return
		}

		if commandReceived {
			utils.Info("Ignoring subsequent message from client as command already provided/executing.")
			c.Send([]byte("Server: Only one command can be executed per WebSocket connection. Ignoring subsequent messages.\r\n"))
			continue
		}

		cmdEncodedString := string(message)
		utils.Info("Received Base64-encoded command from WebSocket client for target", targetHost, ":", cmdEncodedString)

		commandReceived = true

		go func() {
			cmdExecCtx, cmdExecCancel := context.WithCancel(c.ctx)
			defer cmdExecCancel()

			// Pass the Base64-encoded string directly to RunAndStream
			execErr := cmdRunner.RunAndStream(cmdExecCtx, cmdEncodedString, targetHost, c)

			if execErr != nil {
				decodedCmd, decodeErr := base64.StdEncoding.DecodeString(cmdEncodedString)
				cmdForErrorMsg := cmdEncodedString
				if decodeErr == nil {
					cmdForErrorMsg = string(decodedCmd)
				}
				errMsg := fmt.Sprintf("Error executing command '%s' on %s: %v\r\n", cmdForErrorMsg, targetHost, execErr)
				utils.Error(errMsg)
				c.Send([]byte(fmt.Sprintf("ERROR: %s", errMsg)))
			} else {
				c.Send([]byte(fmt.Sprintf("\r\nBase64-encoded command received on %s finished successfully.\r\n", targetHost)))
			}
		}()
	}
}
