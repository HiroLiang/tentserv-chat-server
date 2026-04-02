package port

// Broadcaster fans a serialized message out to a user's active WS connections.
// Implemented by *ws.Hub (structural match via SendToUser method).
type Broadcaster interface {
	SendToUser(userID string, msg []byte)
}
