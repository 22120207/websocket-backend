package websocket

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client is a middleman between the websocket connection and the cmdrunner
type Client struct {
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Context for managing the lifecycle of the client and its associated command
	ctx    context.Context
	cancel context.CancelFunc
}

// Creates a new WebSocket client
func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		conn:   conn,
		send:   make(chan []byte, 256), // Buffer for outgoing messages
		ctx:    ctx,
		cancel: cancel,
	}
}

// ReadLoop reads messages from the WebSocket connection.
// It typically receives client commands (like the Base64 encoded cmd, or a "stop" signal).
func (c *Client) ReadLoop(onCommand func(encodedCmd, targetHost string), onError func(error)) {
	defer func() {
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		select {
		case <-c.ctx.Done(): // Context cancelled, stop reading
			return
		default:
			messageType, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					onError(err)
				}
				return
			}

			if messageType == websocket.TextMessage {
				// The client is expected to send a Base64 encoded command string.
				// You'll need to parse this message to extract the command and target.
				// For example, if the client sends {"cmd": "b64encodedcmd", "target": "host"},
				// you'd decode the JSON here.
				//
				// For simplicity in this outline, let's assume the first message *is* the b64 command
				// and the target host might come from the initial WebSocket URL query params.

				// Example: Assuming message is just the base64 encoded command string directly.
				// In a real app, you'd likely parse a JSON message like:
				/*
					var req struct {
						EncodedCmd string `json:"cmd"`
						Target     string `json:"target"`
					}
					if err := json.Unmarshal(message, &req); err != nil {
						onError(fmt.Errorf("failed to parse client message: %w", err))
						continue
					}
					onCommand(req.EncodedCmd, req.Target)
				*/

				// For now, let's assume the command and target are passed via the initial URL query.
				// The `onCommand` callback will be triggered from the `handler.go` after extracting this.
				// So, ReadLoop here is more for future client-sent controls (like "stop streaming").
				// If you want the command from the message, you'd process `message` here.
				// For this outline, let's assume `onCommand` is called just once from handler.
			}
		}
	}
}

// WriteLoop writes messages from the send channel to the WebSocket connection.
// This is where you send the streamed output from the remote command.
func (c *Client) WriteLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The send channel has been closed.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				// Handle write error (e.g., connection closed unexpectedly)
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.ctx.Done(): // Context cancelled, stop writing
			return
		}
	}
}

// Send sends a message to the client's outbound buffer.
func (c *Client) Send(message []byte) {
	select {
	case c.send <- message:
	case <-c.ctx.Done(): // Context cancelled, don't send
		// Log that message couldn't be sent because context is done
	default:
		// Drop message if send buffer is full, or implement backpressure/logging
	}
}

// Close gracefully closes the client connection and cancels its context.
func (c *Client) Close() {
	select {
	case <-c.ctx.Done():
		// Already closed
		return
	default:
		c.cancel() // Signal all goroutines associated with this client to stop
		c.conn.Close()
	}
}

// GetContext returns the client's context for external cancellation signals.
func (c *Client) GetContext() context.Context {
	return c.ctx
}
