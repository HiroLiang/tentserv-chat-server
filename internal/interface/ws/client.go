package ws

import (
	"encoding/json"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// [EN] Client represents one WebSocket connection. Each client has two goroutines:
//      ReadPump: reads inbound messages and routes them via MessageRouter.
//      WritePump: drains the send channel and writes to the socket; also sends periodic pings (54s).
// [中] Client 代表一條 WebSocket 連線。每個 Client 有兩個 goroutine：
//      ReadPump：讀取入站訊息並透過 MessageRouter 分發；
//      WritePump：消費 send channel 並寫入 socket，同時定期發送 ping（54 秒）。
// [日] Client は 1 つの WebSocket 接続を表す。各 Client に 2 つの goroutine を持つ：
//      ReadPump：受信メッセージを読み取り MessageRouter でルーティングする；
//      WritePump：send チャネルをドレインして socket に書き込み、定期的に ping（54 秒）を送信する。

// Client represents a single WebSocket connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	UserID string
}

// NewClient creates a new Client and registers it with the hub.
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		UserID: userID,
	}
}

// ReadPump reads messages from the WebSocket connection and routes them.
// Must be called in a goroutine. Exits when the connection is closed.
func (c *Client) ReadPump(router *MessageRouter) {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		_ = router.Route(c, &msg)
	}
}

// WritePump writes messages from the send channel to the WebSocket connection.
// Must be called in a goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send enqueues msg for delivery. Drops with a warning log if the buffer is full.
func (c *Client) Send(msg []byte) {
	select {
	case c.send <- msg:
	default:
		logger.Log.Warn("ws: send buffer full, dropping message", zap.String("user_id", c.UserID))
	}
}
