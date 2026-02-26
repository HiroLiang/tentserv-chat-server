package participant

import (
	"errors"
	"fmt"
	"strings"
)

type ID int64

type ParticipantType string

const (
	UserType   ParticipantType = "user"
	AgentType  ParticipantType = "agent"
	SystemType ParticipantType = "system"
)

var ErrInvalidParticipantType = errors.New("invalid participant type")

func ParseParticipantType(s string) (ParticipantType, error) {
	t := ParticipantType(strings.ToLower(strings.TrimSpace(s)))
	switch t {
	case UserType, AgentType, SystemType:
		return t, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidParticipantType, s)
	}
}
