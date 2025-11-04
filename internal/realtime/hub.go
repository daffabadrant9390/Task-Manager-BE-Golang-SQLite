package realtime

import (
	"sync"
)

// Client represents a single websocket client connection.
// We keep it minimal here; the actual network conn is managed in the ws handler.
type Client interface {
	Send(message []byte) bool
	Close()
}

// Hub maintains active user connections and broadcasts events to them.
type Hub struct {
	mu              sync.RWMutex
	userIdToClients map[string]map[Client]struct{}
}

var hubInstance *Hub
var once sync.Once

// GetHub returns a singleton hub instance.
func GetHub() *Hub {
	once.Do(func() {
		hubInstance = &Hub{
			userIdToClients: make(map[string]map[Client]struct{}),
		}
	})
	return hubInstance
}

// Register adds a client under a user ID.
func (h *Hub) Register(userID string, client Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.userIdToClients[userID]; !ok {
		h.userIdToClients[userID] = make(map[Client]struct{})
	}
	h.userIdToClients[userID][client] = struct{}{}
}

// Unregister removes a client; if user has no more clients, cleans up map.
func (h *Hub) Unregister(userID string, client Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.userIdToClients[userID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.userIdToClients, userID)
		}
	}
}

// Broadcast sends a message to all clients of a user.
func (h *Hub) Broadcast(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := h.userIdToClients[userID]
	for c := range clients {
		if ok := c.Send(message); !ok {
			// client write failed; let the handler clean it up on its side
		}
	}
}
