package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gorilla/websocket"
)

func TestMain(m *testing.M) {
	logger.InitTestEnv()
	os.Exit(m.Run())
}

// mockHandler is a test double for MessageHandler.
type mockHandler struct {
	mu      sync.Mutex
	calls   int
	payload json.RawMessage
	err     error
}

func (m *mockHandler) Handle(_ *Client, payload json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.payload = payload
	return m.err
}

func (m *mockHandler) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// newTestClient creates a minimal Client suitable for hub tests (no real connection).
func newTestClient(hub *Hub, userID string) *Client {
	client := &Client{hub: hub, send: make(chan []byte, 256), UserID: userID}
	client.Touch()
	return client
}

// testWSPair spins up an in-process HTTP server and returns a connected pair of
// WebSocket connections. The returned serverConn is the one that the Client
// under test should wrap.
func testWSPair(t *testing.T) (serverConn, clientConn *websocket.Conn) {
	t.Helper()

	connCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		conn, err := u.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server upgrade: %v", err)
			return
		}
		connCh <- conn
	}))
	t.Cleanup(srv.Close)

	wsURL := "ws" + srv.URL[4:]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	serverConn = <-connCh
	return serverConn, clientConn
}
