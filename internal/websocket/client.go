package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
)

// Client represents a single WebSocket client connection.
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	ctx       context.Context
	cancel    context.CancelFunc
	cmdCancel context.CancelFunc
	once      sync.Once
	mu        sync.RWMutex
	isClosed  bool
	isRunCmd  bool // Ensure that one command is processed at a time
}

// NewClient creates a new WebSocket client instance.
func NewClient(conn *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		conn:     conn,
		send:     make(chan []byte, 4096), // Buffered channel for outgoing messages
		ctx:      ctx,
		cancel:   cancel,
		isClosed: false,
		isRunCmd: false,
	}
}

// Send a message to the WebSocket client.
func (c *Client) Send(message []byte) {
	c.mu.Lock()
	closed := c.isClosed
	c.mu.Unlock()

	if closed {
		log.Debug("Attempted to send message to a closed WebSocket client.")
		return
	}

	select {
	case c.send <- message:
		// Message sent to channel
	case <-c.ctx.Done():
		log.Debug("Context done, not sending message to WebSocket client.")
	default:
		// If channel is full and context not done, log a warning
		log.Error("WebSocket send channel is full, dropping message.")
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

		log.Info("WebSocket client connection closed.")
	})
}

// WriteLoop continuously reads messages from the send channel and writes them to the WebSocket.
func (c *Client) WriteLoop() {
	ticker := time.NewTicker(time.Second * 9) // Ping interval for WebSocket keep-alive
	defer func() {
		ticker.Stop()
		log.Info("WebSocket WriteLoop finished.")
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

			if string(message) == "command finished successfully" {
				msg.Type = "finishied"
			}

			jsonData, err := json.Marshal(msg)
			if err != nil {
				log.Error(err.Error())
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
				log.Error("Error writing message to WebSocket:", err)
				return
			}
		case <-ticker.C: // Send ping message to keep connection alive
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error("Error sending ping message to WebSocket:", err)
				return
			}
		case <-c.ctx.Done(): // Context cancelled, signaling client disconnection
			log.Info("WebSocket WriteLoop context cancelled.")
			c.Close()
			return
		}
	}
}

// ReadLoop continuously reads messages from the WebSocket in base64 format
func (c *Client) ReadLoop(
	onError func(err error), // Callback for handling read errors
) {
	defer func() {
		c.Close()
		log.Info("WebSocket ReadLoop finished:")
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
				log.Error("Unexpected WebSocket close error for client connected to", ":", err)
			}
			onError(err)
			return
		}

		if c.isRunCmd {
			log.Info("Interrupted the previous command. Run the new command.")
			c.Send([]byte("Server: Interrupted the previous command. Run the new command."))
			c.cmdCancel()
		}

		// Unmarshal the json data receive from client
		type ReceiveMsg struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		}

		var msg ReceiveMsg
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Error(err.Error())
		}

		// If the type is not "command" --> exit
		if msg.Type != "command" {
			log.Warnf("Unsupported message type received: %s", msg.Type)
			c.Send([]byte("Server: Unsupported message type."))
			continue
		}

		// Decoded the base64 command
		cmdEncodedString := string(msg.Command)
		log.Info("Received Base64-encoded command from WebSocket client:", cmdEncodedString)

		c.isRunCmd = true

		go func() {
			cmdExecCtx, cmdExecCancel := context.WithCancel(c.ctx)
			defer func() {
				cmdExecCancel()
				c.UpdateState(false)
			}()

			// Pass the Base64-encoded string directly to RunAndStream
			execErr := runAndStream(cmdExecCtx, cmdEncodedString, c)

			if execErr != nil {
				decodedCmd, decodeErr := base64.StdEncoding.DecodeString(cmdEncodedString)
				cmdForErrorMsg := cmdEncodedString
				if decodeErr == nil {
					cmdForErrorMsg = string(decodedCmd)
				}
				errMsg := fmt.Sprintf("Error executing command '%s': %v\r\n", cmdForErrorMsg, execErr)
				log.Error(errMsg)
				c.Send([]byte(fmt.Sprintf("ERROR: %s", errMsg)))
			}
		}()
	}
}

// A func to update the isRunCmd attributes
func (c *Client) UpdateState(newState bool) {
	c.isRunCmd = newState
}

// A func to set ctx and cancelFunc
func (c *Client) SetCmdCancelFunc(cancel context.CancelFunc) {
	c.cmdCancel = cancel
}
