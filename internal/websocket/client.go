package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	mu       sync.RWMutex
	isClosed bool
	isRunCmd bool // Ensure that one command is processed at a time
}

// NewClient creates a new WebSocket client instance.
func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		conn:     conn,
		send:     make(chan []byte, 4096), // Buffered channel for outgoing messages
		ctx:      ctx,
		isClosed: false,
		cancel:   cancel,
		isRunCmd: false,
	}
}

// Send a message to the WebSocket client.
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
		return
	}
}

// Close gracefully closes the client connection.
func (c *Client) Close() {
	c.once.Do(func() {
		c.mu.Lock()
		c.isClosed = true
		c.mu.Unlock()

		c.cancel()
		close(c.send)
		c.conn.Close()
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
			if !ok {
				return
			}

			// Send data in json format
			type Message struct {
				Type string `json:"type"`
				Data string `json:"data"`
			}

			msg := Message{
				Type: "output",
				Data: string(message),
			}

			jsonData, err := json.Marshal(msg)
			if err != nil {
				utils.Error(err.Error())
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				utils.Error("Error writing message to WebSocket:", err)
				return
			}

			// if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			// 	utils.Error("Error writing message to WebSocket:", err)
			// 	return
			// }
		case <-ticker.C: // Send ping message to keep connection alive
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				utils.Error("Error sending ping message to WebSocket:", err)
				return
			}
		case <-c.ctx.Done(): // Context cancelled, signaling client disconnection
			utils.Info("WebSocket WriteLoop context cancelled.")
			c.Close()
			return
		}
	}
}

// ReadLoop continuously reads messages from the WebSocket in base64 format
func (c *Client) ReadLoop(
	cmdRunner *cmdrunner.CommandRunner,
	targetHost string,
	onError func(err error), // Callback for handling read errors
) {
	defer func() {
		c.Close()
		utils.Info("WebSocket ReadLoop finished for target:", targetHost)
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(time.Second * 60))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(time.Second * 60))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				utils.Error("Unexpected WebSocket close error for client connected to", targetHost, ":", err)
			}
			onError(err)
			return
		}

		if c.isRunCmd {
			utils.Info("Ignoring subsequent message from client as command is executing.")
			c.Send([]byte("Server: Only one command can be executed at a time. Ignoring subsequent messages.\r\n"))
			continue
		}

		// Unmarshal the json data receive from client
		type ReceiveMsg struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		}

		var msg ReceiveMsg
		err = json.Unmarshal(message, &msg)
		if err != nil {
			utils.Error(err.Error())
		}

		// If the type is not "command" --> exit
		if msg.Type != "command" {
			return
		}

		// Decoded the base64 command
		cmdEncodedString := string(msg.Command)
		utils.Info("Received Base64-encoded command from WebSocket client for target", targetHost, ":", cmdEncodedString)

		c.isRunCmd = true

		go func() {
			cmdExecCtx, cmdExecCancel := context.WithCancel(c.ctx)
			defer func() {
				cmdExecCancel()
				c.UpdateState(false)
			}()

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

// A func to update the isRunCmd attributes
func (c *Client) UpdateState(newState bool) {
	c.isRunCmd = newState
}
