package ws

import (
	"encoding/json"
	"sync"
	"time"
)

// Hub manages all active WebSocket clients and routes broadcasts.
type Hub struct {
	// clients holds every connected client; only accessed from Run().
	clients map[*Client]bool

	// userClients maps userID → clients; protected by mu.
	userClients map[string][]*Client
	mu          sync.RWMutex

	// Broadcast sends a message to every connected client.
	Broadcast chan []byte

	// Register adds a client to the hub.
	Register chan *Client

	// Unregister removes a client from the hub.
	Unregister chan *Client

	// pendingAcks tracks in-flight deliveries waiting for client ACK.
	pendingAcks map[int64]chan struct{}
	ackMu       sync.Mutex
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		userClients: make(map[string][]*Client),
		Broadcast:   make(chan []byte, 256),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		pendingAcks: make(map[int64]chan struct{}),
	}
}

// Run starts the Hub event loop. Must be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.clients[client] = true
			if client.UserID != "" {
				h.mu.Lock()
				h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
				h.mu.Unlock()
			}

		case client := <-h.Unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if client.UserID != "" {
					h.mu.Lock()
					h.removeUserClient(client.UserID, client)
					h.mu.Unlock()
				}
			}

		case message := <-h.Broadcast:
			for client := range h.clients {
				client.Send(message)
			}
		}
	}
}

// SendToUser sends a message to all active connections belonging to userID.
// Safe to call from any goroutine.
func (h *Hub) SendToUser(userID string, msg []byte) {
	h.mu.RLock()
	clients := h.userClients[userID]
	h.mu.RUnlock()

	for _, c := range clients {
		c.Send(msg)
	}
}

// PushAndWaitAck sends a typed message with a delivery ID to a user and waits for the client ACK.
// Returns true if ACK was received within timeout, false otherwise.
func (h *Hub) PushAndWaitAck(userID string, deliveryID int64, msgType string, payload []byte, timeout time.Duration) bool {
	ackCh := make(chan struct{}, 1)

	h.ackMu.Lock()
	h.pendingAcks[deliveryID] = ackCh
	h.ackMu.Unlock()

	msg := Message{
		Type:       msgType,
		Payload:    json.RawMessage(payload),
		DeliveryID: &deliveryID,
	}
	data, err := json.Marshal(msg)
	if err == nil {
		h.SendToUser(userID, data)
	}

	select {
	case <-ackCh:
		h.ackMu.Lock()
		delete(h.pendingAcks, deliveryID)
		h.ackMu.Unlock()
		return true
	case <-time.After(timeout):
		h.ackMu.Lock()
		delete(h.pendingAcks, deliveryID)
		h.ackMu.Unlock()
		return false
	}
}

// ResolveAck closes the pending ACK channel for the given deliveryID.
func (h *Hub) ResolveAck(deliveryID int64) {
	h.ackMu.Lock()
	ch, ok := h.pendingAcks[deliveryID]
	h.ackMu.Unlock()
	if ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// removeUserClient removes target from h.userClients[userID].
// Caller must hold h.mu.Lock().
func (h *Hub) removeUserClient(userID string, target *Client) {
	clients := h.userClients[userID]
	for i, c := range clients {
		if c == target {
			h.userClients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.userClients[userID]) == 0 {
		delete(h.userClients, userID)
	}
}
