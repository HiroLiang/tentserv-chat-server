package chat

import (
	"encoding/json"
	"testing"

	"github.com/HiroLiang/tentserv-chat-server/internal/interface/ws"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitTestEnv()
	m.Run()
}

func TestChatMessageHandler_Handle_InvalidJSON_ReturnsError(t *testing.T) {
	handler := NewMessageHandler(nil)
	client := ws.NewClient(nil, nil, "1")

	err := handler.Handle(client, json.RawMessage(`not-valid-json`))

	assert.Error(t, err)
}

func TestChatMessageHandler_Handle_InvalidRoomID_ReturnsError(t *testing.T) {
	handler := NewMessageHandler(nil)
	client := ws.NewClient(nil, nil, "1")

	payload, _ := json.Marshal(SendPayload{RoomID: "not-a-number", Content: "hello"})
	err := handler.Handle(client, payload)

	assert.Error(t, err)
}

func TestChatMessageHandler_Handle_InvalidUserID_ReturnsError(t *testing.T) {
	handler := NewMessageHandler(nil)
	client := ws.NewClient(nil, nil, "not-a-number")

	payload, _ := json.Marshal(SendPayload{RoomID: "1", Content: "hello"})
	err := handler.Handle(client, payload)

	assert.Error(t, err)
}

func TestChatMessageHandler_Handle_AnonymousClient_ReturnsError(t *testing.T) {
	handler := NewMessageHandler(nil)
	client := ws.NewClient(nil, nil, "") // no user ID

	payload, _ := json.Marshal(SendPayload{RoomID: "1", Content: "hi"})
	err := handler.Handle(client, payload)

	assert.Error(t, err)
}
