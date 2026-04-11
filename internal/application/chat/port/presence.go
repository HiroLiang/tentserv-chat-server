package port

import "time"

type PresenceStatus string

const (
	PresenceStatusOnline  PresenceStatus = "online"
	PresenceStatusOffline PresenceStatus = "offline"
)

type PresenceSnapshot struct {
	Status     PresenceStatus
	LastSeenAt *time.Time
}

type PresenceReader interface {
	GetPresence(userID string) PresenceSnapshot
}
