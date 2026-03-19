package game

import (
	"encoding/json"

	"github.com/HiroLiang/tentserv-chat-server/internal/interface/ws"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"go.uber.org/zap"
)

// MovePayload is the payload for a "game.move" message.
type MovePayload struct {
	GameID string `json:"game_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// MoveHandler handles "game.move" messages.
type MoveHandler struct{}

func NewMoveHandler() *MoveHandler {
	return &MoveHandler{}
}

func (h *MoveHandler) Handle(client *ws.Client, payload json.RawMessage) error {
	var p MovePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}

	logger.Log.Info("game.move",
		zap.String("user_id", client.UserID),
		zap.String("game_id", p.GameID),
		zap.Int("x", p.X),
		zap.Int("y", p.Y),
	)

	// TODO: validate move and broadcast to game participants via hub.SendToUser
	return nil
}
