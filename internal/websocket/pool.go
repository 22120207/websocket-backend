package websocket

import (
	"sync"
	"websocket-backend/pkg/utils"
)

// Pool manages the collection of active WebSocket clients.
type Pool struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

// NewPool creates and returns a new Pool instance.
func NewPool() *Pool {
	return &Pool{
		clients: make(map[*Client]bool),
	}
}

// Register adds a client to the pool.
func (p *Pool) Register(client *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients[client] = true
	utils.Info("Client registered to WebSocket pool. Total clients:", len(p.clients))
}

// Unregister removes a client from the pool.
func (p *Pool) Unregister(client *Client) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.clients[client]; ok {
		delete(p.clients, client)
		client.Close()
		utils.Info("Client unregistered from WebSocket pool. Total clients:", len(p.clients))
	}
}

// // Broadcast sends a message to all registered clients.
// // This function is not directly used in the current streaming command execution model,
// // as command output is streamed directly to a single client.
// // However, it's a common pattern for WebSocket pools.
// func (p *Pool) Broadcast(message []byte) {
// 	p.mu.RLock()
// 	defer p.mu.RUnlock()
// 	for client := range p.clients {
// 		select {
// 		case client.send <- message:
// 			// Message sent
// 		default:
// 			utils.Error("Failed to send message to client, send channel full. Unregistering client.")
// 			// This client's channel is full, implying it's not reading fast enough or is stuck.
// 			// We could consider unregistering it, but that requires a write lock,
// 			// so it's usually handled in the WriteLoop or a separate cleanup routine.
// 		}
// 	}
// }
