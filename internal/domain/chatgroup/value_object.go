package chatgroup

import (
	"errors"
	"fmt"
	"strings"
)

type ID int64

type GroupType string

const (
	Direct  GroupType = "direct"
	Group   GroupType = "group"
	Channel GroupType = "channel"
	Bot     GroupType = "bot"
)

var ErrInvalidGroupType = errors.New("invalid group type")

func ParseGroupType(s string) (GroupType, error) {
	t := GroupType(strings.ToLower(strings.TrimSpace(s)))
	switch t {
	case Direct, Group, Channel, Bot:
		return t, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidGroupType, s)
	}
}
