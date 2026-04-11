package ws

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// ---- Client.Send --------------------------------------------------------

func TestClient_Send_EnqueuesMessage(t *testing.T) {
	client := newTestClient(nil, "")

	client.Send([]byte("hello"))

	select {
	case got := <-client.send:
		assert.Equal(t, []byte("hello"), got)
	default:
		t.Error("message not enqueued")
	}
}

func TestClient_Send_DropsWhenBufferFull(t *testing.T) {
	client := &Client{send: make(chan []byte, 1)}
	client.send <- []byte("existing") // fill the 1-slot buffer

	// Must not block or panic.
	assert.NotPanics(t, func() { client.Send([]byte("dropped")) })

	// Original message is still there; dropped message is gone.
	assert.Equal(t, []byte("existing"), <-client.send)
	assert.Empty(t, client.send)
}

// ---- ReadPump -----------------------------------------------------------

func TestClient_ReadPump_RoutesValidMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn := testWSPair(t)
	defer clientConn.Close()

	// Track handler invocations via a channel.
	received := make(chan json.RawMessage, 1)
	handler := &funcHandler{fn: func(_ *Client, p json.RawMessage) error {
		received <- p
		return nil
	}}

	router := NewMessageRouter()
	router.Register("chat.send", handler)

	client := NewClient(hub, serverConn, "user1", "")
	hub.Register <- client
	go client.ReadPump(router)

	payload := json.RawMessage(`{"room_id":"r1","content":"hi"}`)
	msg, _ := json.Marshal(Message{Type: "chat.send", Payload: payload})
	assert.NoError(t, clientConn.WriteMessage(websocket.TextMessage, msg))

	select {
	case got := <-received:
		assert.JSONEq(t, string(payload), string(got))
	case <-time.After(500 * time.Millisecond):
		t.Fatal("handler was not called within timeout")
	}
}

func TestClient_ReadPump_IgnoresMalformedJSON(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn := testWSPair(t)
	defer clientConn.Close()

	handler := &mockHandler{}
	router := NewMessageRouter()
	router.Register("chat.send", handler)

	client := NewClient(hub, serverConn, "", "")
	hub.Register <- client
	go client.ReadPump(router)

	// Send malformed JSON — handler must not be called.
	assert.NoError(t, clientConn.WriteMessage(websocket.TextMessage, []byte(`not-json`)))

	// Send a valid follow-up to confirm the pump is still running.
	validMsg, _ := json.Marshal(Message{Type: "chat.send", Payload: json.RawMessage(`{}`)})
	assert.NoError(t, clientConn.WriteMessage(websocket.TextMessage, validMsg))

	assert.Eventually(t, func() bool { return handler.Calls() == 1 },
		500*time.Millisecond, 5*time.Millisecond,
		"pump should continue after malformed message")
}

func TestClient_ReadPump_UnregistersOnDisconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn := testWSPair(t)

	client := NewClient(hub, serverConn, "user1", "")
	hub.Register <- client
	go client.ReadPump(NewMessageRouter())

	// Closing the client side causes ReadPump to receive an error and exit.
	clientConn.Close()

	// After disconnect, hub should have unregistered the client; its send
	// channel must be closed.
	assert.Eventually(t, func() bool {
		select {
		case _, ok := <-client.send:
			return !ok
		default:
			return false
		}
	}, 500*time.Millisecond, 5*time.Millisecond,
		"client send channel should be closed after disconnect")
}

// ---- WritePump ----------------------------------------------------------

func TestClient_WritePump_DeliversQueuedMessage(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn := testWSPair(t)
	defer clientConn.Close()

	client := NewClient(hub, serverConn, "", "")
	hub.Register <- client
	go client.WritePump()

	want := []byte(`{"type":"pong"}`)
	client.Send(want)

	clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, got, err := clientConn.ReadMessage()
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestClient_WritePump_DeliversMultipleMessages(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	serverConn, clientConn := testWSPair(t)
	defer clientConn.Close()

	client := NewClient(hub, serverConn, "", "")
	hub.Register <- client
	go client.WritePump()

	messages := [][]byte{[]byte("msg1"), []byte("msg2"), []byte("msg3")}
	for _, m := range messages {
		client.Send(m)
	}

	// Each message arrives in its own frame.
	for _, want := range messages {
		clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, got, err := clientConn.ReadMessage()
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

// ---- helpers -----------------------------------------------------------

// funcHandler adapts a function to the MessageHandler interface.
type funcHandler struct {
	mu sync.Mutex
	fn func(*Client, json.RawMessage) error
}

func (h *funcHandler) Handle(c *Client, p json.RawMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.fn(c, p)
}
