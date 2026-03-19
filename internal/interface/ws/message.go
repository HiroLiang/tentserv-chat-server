package ws

import "encoding/json"

// Message is the envelope for all WebSocket messages.
// Clients send and receive JSON in the format:
//
//	{"type": "module.action", "payload": {...}}
type Message struct {
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	DeliveryID *int64          `json:"delivery_id,omitempty"`
}
