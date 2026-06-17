package hub

import (
	"fmt"
	"sync"
)

// Client represents a connected SSE client
type Client struct {
	UserID int
	Chan   chan []byte
}

// Hub manages all connected clients for SSE push
type Hub struct {
	clients map[int]map[*Client]bool
	mu      sync.RWMutex
}

var DefaultHub = &Hub{
	clients: make(map[int]map[*Client]bool),
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[client.UserID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.Chan)
		}
		if len(clients) == 0 {
			delete(h.clients, client.UserID)
		}
	}
}

// PushToUser sends a message to all connections of a specific user
func (h *Hub) PushToUser(userID int, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.clients[userID]; ok {
		for client := range clients {
			select {
			case client.Chan <- message:
			default:
				// channel full, skip (weak network handling)
			}
		}
	}
}

// IsOnline checks if a user has any active connection
func (h *Hub) IsOnline(userID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

// PushEvent formats and pushes an SSE event
func (h *Hub) PushEvent(userID int, eventType string, data string) {
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
	h.PushToUser(userID, []byte(msg))
}
