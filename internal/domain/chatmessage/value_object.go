package chatmessage

import (
	"errors"
	"fmt"
	"strings"
)

type ID int64

type MessageType string

const (
	Text   MessageType = "text"
	Image  MessageType = "image"
	File   MessageType = "file"
	System MessageType = "system"
)

var ErrInvalidMessageType = errors.New("invalid message type")

func ParseMessageType(s string) (MessageType, error) {
	t := MessageType(strings.ToLower(strings.TrimSpace(s)))
	switch t {
	case Text, Image, File, System:
		return t, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidMessageType, s)
	}
}
