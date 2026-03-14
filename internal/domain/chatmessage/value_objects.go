package chatmessage

type ID int64

type MessageType string

const (
	Text   MessageType = "text"
	Image  MessageType = "image"
	File   MessageType = "file"
	Icon   MessageType = "icon"
	System MessageType = "system"
)
