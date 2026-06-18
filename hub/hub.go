package hub

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

// Client represents a connected SSE client
type Client struct {
	UserID int
	Chan   chan []byte
}

// Hub manages all connected clients for SSE push
type Hub struct {
	clients map[int]map[*Client]bool
	closed  map[*Client]bool
	mu      sync.RWMutex
}

var DefaultHub = &Hub{
	clients: make(map[int]map[*Client]bool),
	closed:  make(map[*Client]bool),
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.closed, client)
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	if clients, ok := h.clients[client.UserID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			h.closed[client] = true
			close(client.Chan)
		}
		if len(clients) == 0 {
			delete(h.clients, client.UserID)
		}
	}
	h.mu.Unlock()
}

// PushToUser sends a message to all connections of a specific user
// 非阻塞但带超时：避免channel满时直接丢消息；超时后才丢弃，保证大多数场景消息可达
func (h *Hub) PushToUser(userID int, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients, ok := h.clients[userID]
	if !ok {
		return
	}
	for client := range clients {
		if h.closed[client] {
			continue
		}
		select {
		case client.Chan <- message:
		case <-time.After(200 * time.Millisecond):
			// 200ms 仍未写入（消费端太慢），丢弃本条，避免发送方被卡住
			fmt.Printf("[hub] user %d channel full, drop 1 message\n", userID)
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
	var buf bytes.Buffer
	buf.WriteString("event: ")
	buf.WriteString(eventType)
	buf.WriteString("\ndata: ")
	buf.WriteString(data)
	buf.WriteString("\n\n")
	h.PushToUser(userID, buf.Bytes())
}
