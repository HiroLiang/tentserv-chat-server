package ws

import (
	"encoding/json"
	"sync"
	"time"

	chatPort "github.com/HiroLiang/tentserv-chat-server/internal/application/chat/port"
)

const (
	defaultPresenceTTL     = pongWait + 15*time.Second
	defaultCleanupInterval = 15 * time.Second
)

// [EN] Hub is the central WebSocket message broker. It now also tracks per-user presence
//
//	from active WS connections and exposes snapshots for direct-room summaries.
//
// [中] Hub 為中央 WebSocket 訊息代理，現在也根據活躍 WS 連線追蹤每位使用者的 presence，
//
//	並提供 direct room 摘要查詢。
//
// [日] Hub は WebSocket メッセージブローカーの中核であり、現在はアクティブな WS 接続から
//
//	ユーザーごとの presence も追跡し、ダイレクトルーム概要向けのスナップショットを提供する。
type Hub struct {
	clients map[*Client]bool

	userClients    map[string][]*Client
	lastSeenByUser map[string]time.Time
	mu             sync.RWMutex

	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client

	pendingAcks map[int64]chan struct{}
	ackMu       sync.Mutex

	presenceTTL     time.Duration
	cleanupInterval time.Duration

	onConnect         func(userID string)
	onPresenceChanged func(userID string, snapshot chatPort.PresenceSnapshot)
}

var _ chatPort.PresenceReader = (*Hub)(nil)

func NewHub() *Hub {
	return &Hub{
		clients:         make(map[*Client]bool),
		userClients:     make(map[string][]*Client),
		lastSeenByUser:  make(map[string]time.Time),
		Broadcast:       make(chan []byte, 256),
		Register:        make(chan *Client),
		Unregister:      make(chan *Client),
		pendingAcks:     make(map[int64]chan struct{}),
		presenceTTL:     defaultPresenceTTL,
		cleanupInterval: defaultCleanupInterval,
	}
}

func (h *Hub) Run() {
	cleanupTicker := time.NewTicker(h.cleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client, time.Now().UTC())

		case message := <-h.Broadcast:
			for client := range h.clients {
				client.Send(message)
			}

		case now := <-cleanupTicker.C:
			h.pruneStaleClients(now.UTC())
		}
	}
}

func (h *Hub) SendToUser(userID string, msg []byte) {
	h.sendToUser(userID, msg)
}

func (h *Hub) SendToUsers(userIDs []string, msg []byte) {
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		h.sendToUser(userID, msg)
	}
}

func (h *Hub) GetPresence(userID string) chatPort.PresenceSnapshot {
	now := time.Now().UTC()

	h.mu.RLock()
	clients := append([]*Client(nil), h.userClients[userID]...)
	lastSeen, hasLastSeen := h.lastSeenByUser[userID]
	h.mu.RUnlock()

	staleClients := make([]*Client, 0)
	for _, client := range clients {
		if client.IsStale(now, h.presenceTTL) {
			staleClients = append(staleClients, client)
			continue
		}
		return chatPort.PresenceSnapshot{Status: chatPort.PresenceStatusOnline}
	}

	h.queueStaleUnregisters(staleClients)

	if !hasLastSeen || lastSeen.IsZero() {
		return chatPort.PresenceSnapshot{Status: chatPort.PresenceStatusOffline}
	}

	lastSeenCopy := lastSeen
	return chatPort.PresenceSnapshot{
		Status:     chatPort.PresenceStatusOffline,
		LastSeenAt: &lastSeenCopy,
	}
}

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
	if err != nil {
		h.deletePendingAck(deliveryID)
		return false
	}
	if delivered := h.sendToUser(userID, data); delivered == 0 {
		h.deletePendingAck(deliveryID)
		return false
	}

	select {
	case <-ackCh:
		h.deletePendingAck(deliveryID)
		return true
	case <-time.After(timeout):
		h.deletePendingAck(deliveryID)
		return false
	}
}

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

func (h *Hub) SetOnConnect(fn func(userID string)) {
	h.onConnect = fn
}

func (h *Hub) SetOnPresenceChanged(fn func(userID string, snapshot chatPort.PresenceSnapshot)) {
	h.onPresenceChanged = fn
}

func (h *Hub) deletePendingAck(deliveryID int64) {
	h.ackMu.Lock()
	delete(h.pendingAcks, deliveryID)
	h.ackMu.Unlock()
}

func (h *Hub) sendToUser(userID string, msg []byte) int {
	now := time.Now().UTC()

	h.mu.RLock()
	clients := append([]*Client(nil), h.userClients[userID]...)
	h.mu.RUnlock()

	delivered := 0
	staleClients := make([]*Client, 0)
	for _, client := range clients {
		if client.IsStale(now, h.presenceTTL) {
			staleClients = append(staleClients, client)
			continue
		}
		client.Send(msg)
		delivered++
	}

	h.queueStaleUnregisters(staleClients)
	return delivered
}

func (h *Hub) queueStaleUnregisters(clients []*Client) {
	for _, client := range clients {
		go func(target *Client) {
			h.Unregister <- target
		}(client)
	}
}

func (h *Hub) registerClient(client *Client) {
	now := time.Now().UTC()
	h.clients[client] = true

	var becameOnline bool
	var onConnect func(string)
	var onPresenceChanged func(string, chatPort.PresenceSnapshot)

	if client.UserID != "" {
		h.mu.Lock()
		wasOnline := h.hasActiveUserLocked(client.UserID, now)
		h.userClients[client.UserID] = append(h.userClients[client.UserID], client)
		delete(h.lastSeenByUser, client.UserID)
		becameOnline = !wasOnline
		onConnect = h.onConnect
		onPresenceChanged = h.onPresenceChanged
		h.mu.Unlock()
	}

	if onConnect != nil && client.UserID != "" {
		go onConnect(client.UserID)
	}
	if becameOnline && onPresenceChanged != nil {
		go onPresenceChanged(client.UserID, chatPort.PresenceSnapshot{Status: chatPort.PresenceStatusOnline})
	}
}

func (h *Hub) unregisterClient(client *Client, now time.Time) {
	if _, ok := h.clients[client]; !ok {
		return
	}

	delete(h.clients, client)
	close(client.send)

	if client.UserID == "" {
		return
	}

	var becameOffline bool
	var snapshot chatPort.PresenceSnapshot
	var onPresenceChanged func(string, chatPort.PresenceSnapshot)

	h.mu.Lock()
	h.removeUserClientLocked(client.UserID, client)
	if !h.hasActiveUserLocked(client.UserID, now) {
		lastSeen := now
		h.lastSeenByUser[client.UserID] = lastSeen
		lastSeenCopy := lastSeen
		snapshot = chatPort.PresenceSnapshot{
			Status:     chatPort.PresenceStatusOffline,
			LastSeenAt: &lastSeenCopy,
		}
		becameOffline = true
	}
	onPresenceChanged = h.onPresenceChanged
	h.mu.Unlock()

	if becameOffline && onPresenceChanged != nil {
		go onPresenceChanged(client.UserID, snapshot)
	}
}

func (h *Hub) pruneStaleClients(now time.Time) {
	for client := range h.clients {
		if !client.IsStale(now, h.presenceTTL) {
			continue
		}
		h.unregisterClient(client, now)
	}
}

func (h *Hub) hasActiveUserLocked(userID string, now time.Time) bool {
	for _, client := range h.userClients[userID] {
		if !client.IsStale(now, h.presenceTTL) {
			return true
		}
	}
	return false
}

func (h *Hub) removeUserClientLocked(userID string, target *Client) {
	clients := h.userClients[userID]
	for i, client := range clients {
		if client == target {
			h.userClients[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.userClients[userID]) == 0 {
		delete(h.userClients, userID)
	}
}
