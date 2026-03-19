package system

import (
	"encoding/json"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/push"
	"github.com/HiroLiang/tentserv-chat-server/internal/interface/ws"
)

// AckPayload is the payload for a "system.ack" message.
type AckPayload struct {
	DeliveryID int64 `json:"delivery_id"`
}

// AckHandler handles "system.ack" messages from clients.
type AckHandler struct {
	hub push.DirectPusher
}

func NewAckHandler(hub push.DirectPusher) *AckHandler {
	return &AckHandler{hub: hub}
}

func (h *AckHandler) Handle(client *ws.Client, payload json.RawMessage) error {
	var p AckPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	h.hub.ResolveAck(p.DeliveryID)
	return nil
}
