package ws

import (
	"encoding/json"
	"sync"
	"time"
)

// [EN] Hub is the central WebSocket message broker. It maintains:
//      - clients: all connected Client instances (accessed only from Run() goroutine — no lock needed).
//      - userClients: userID → []*Client mapping (protected by mu RWMutex for concurrent access).
//      - pendingAcks: tracks in-flight messages waiting for client ACK (protected by ackMu).
//      - onConnect: hook called in a goroutine when a new client registers (used for replay of queued keys).
// [中] Hub 為中央 WebSocket 訊息代理。維護：
//      clients（所有連線，僅在 Run() goroutine 中存取，無需鎖定）；
//      userClients（userID → []*Client，以 mu RWMutex 保護並發存取）；
//      pendingAcks（追蹤等待 ACK 的訊息，以 ackMu 保護）；
//      onConnect（新客戶端連線時在 goroutine 中執行的鉤子，用於重播佇列金鑰）。
// [日] Hub は WebSocket メッセージブローカーの中核。以下を管理する：
//      clients（全接続 Client、Run() goroutine からのみアクセス — ロック不要）；
//      userClients（userID → []*Client、mu RWMutex で並行アクセスを保護）；
//      pendingAcks（ACK 待ちの配信を追跡、ackMu で保護）；
//      onConnect（新クライアント登録時に goroutine で呼び出されるフック、キュー鍵の再配信に使用）。

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

	// onConnect is an optional hook called (in a goroutine) when a new client connects.
	// userID is the string user ID of the newly connected client.
	onConnect func(userID string)
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

// [EN] Run: event loop (select) for Register/Unregister/Broadcast channels. Single goroutine owns clients map.
// [中] Run：Register/Unregister/Broadcast channel 的事件迴圈，單一 goroutine 擁有 clients map。
// [日] Run：Register/Unregister/Broadcast チャネルのイベントループ。clients map は単一 goroutine が所有する。
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
				if h.onConnect != nil {
					go h.onConnect(client.UserID)
				}
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

// [EN] PushAndWaitAck: sends a message with delivery_id and blocks until ACK received or timeout.
//      Used by the retry scheduler for guaranteed delivery to online clients.
// [中] PushAndWaitAck：傳送含 delivery_id 的訊息，阻塞直到收到 ACK 或逾時。
//      由重試排程器用於對線上客戶端的保證送達。
// [日] PushAndWaitAck：delivery_id 付きメッセージを送信し、ACK 受信またはタイムアウトまでブロックする。
//      オンラインクライアントへの配信保証のためにリトライスケジューラから使用される。
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

// SetOnConnect registers a hook function called (in a goroutine) each time a new client connects.
func (h *Hub) SetOnConnect(fn func(userID string)) {
	h.onConnect = fn
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
