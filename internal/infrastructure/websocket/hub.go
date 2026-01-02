package websocket

import (
	"encoding/json"
	"sync"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
)

// Message represents a WebSocket message sent to clients
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients by user ID
	clients map[string]map[*Client]bool

	// All clients (for broadcast to all)
	allClients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast to all clients
	broadcast chan []byte

	// Broadcast to specific user
	userBroadcast chan *userMessage

	// Mutex for client maps
	mu sync.RWMutex

	// Done channel for shutdown
	done chan struct{}
}

type userMessage struct {
	userID  string
	message []byte
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[string]map[*Client]bool),
		allClients:    make(map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan []byte, 256),
		userBroadcast: make(chan *userMessage, 256),
		done:          make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	logger.Info().Msg("WebSocket hub started")

	for {
		select {
		case <-h.done:
			logger.Info().Msg("WebSocket hub shutting down")
			h.closeAllClients()
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToAll(message)

		case um := <-h.userBroadcast:
			h.broadcastToUser(um.userID, um.message)
		}
	}
}

// Stop gracefully shuts down the hub
func (h *Hub) Stop() {
	close(h.done)
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal broadcast message")
		return
	}
	h.broadcast <- data
}

// BroadcastToUser sends a message to all clients of a specific user
func (h *Hub) BroadcastToUser(userID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal user message")
		return
	}
	h.userBroadcast <- &userMessage{userID: userID, message: data}
}

// BroadcastRaw sends raw JSON bytes to all connected clients
func (h *Hub) BroadcastRaw(data []byte) {
	h.broadcast <- data
}

// ClientCount returns the total number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.allClients)
}

// UserClientCount returns the number of clients for a specific user
func (h *Hub) UserClientCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}
	return 0
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Add to all clients
	h.allClients[client] = true

	// Add to user-specific map
	if _, ok := h.clients[client.userID]; !ok {
		h.clients[client.userID] = make(map[*Client]bool)
	}
	h.clients[client.userID][client] = true

	logger.Debug().
		Str("user_id", client.userID).
		Int("total_clients", len(h.allClients)).
		Msg("Client registered")
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.allClients[client]; !ok {
		return // Client not registered
	}

	// Remove from all clients
	delete(h.allClients, client)

	// Remove from user-specific map
	if userClients, ok := h.clients[client.userID]; ok {
		delete(userClients, client)
		if len(userClients) == 0 {
			delete(h.clients, client.userID)
		}
	}

	// Close the client's send channel
	close(client.send)

	logger.Debug().
		Str("user_id", client.userID).
		Int("total_clients", len(h.allClients)).
		Msg("Client unregistered")
}

func (h *Hub) broadcastToAll(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.allClients {
		select {
		case client.send <- message:
		default:
			// Client's buffer is full, schedule for removal
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

func (h *Hub) broadcastToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	userClients, ok := h.clients[userID]
	if !ok {
		return
	}

	for client := range userClients {
		select {
		case client.send <- message:
		default:
			// Client's buffer is full, schedule for removal
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.allClients {
		close(client.send)
		delete(h.allClients, client)
	}
	h.clients = make(map[string]map[*Client]bool)

	logger.Info().Msg("All WebSocket clients closed")
}
