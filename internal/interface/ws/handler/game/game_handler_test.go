package game

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

func TestGameMoveHandler_Handle_ValidPayload(t *testing.T) {
	handler := NewMoveHandler()
	client := ws.NewClient(nil, nil, "user1")

	payload, _ := json.Marshal(MovePayload{GameID: "game1", X: 3, Y: 5})
	err := handler.Handle(client, payload)

	assert.NoError(t, err)
}

func TestGameMoveHandler_Handle_ZeroCoordinates(t *testing.T) {
	handler := NewMoveHandler()
	client := ws.NewClient(nil, nil, "user1")

	payload, _ := json.Marshal(MovePayload{GameID: "game1", X: 0, Y: 0})
	err := handler.Handle(client, payload)

	assert.NoError(t, err)
}

func TestGameMoveHandler_Handle_InvalidJSON_ReturnsError(t *testing.T) {
	handler := NewMoveHandler()
	client := ws.NewClient(nil, nil, "user1")

	err := handler.Handle(client, json.RawMessage(`not-valid-json`))

	assert.Error(t, err)
}

func TestGameMoveHandler_Handle_AnonymousClient(t *testing.T) {
	handler := NewMoveHandler()
	client := ws.NewClient(nil, nil, "") // no user ID

	payload, _ := json.Marshal(MovePayload{GameID: "game1", X: 1, Y: 2})
	err := handler.Handle(client, payload)

	assert.NoError(t, err)
}
