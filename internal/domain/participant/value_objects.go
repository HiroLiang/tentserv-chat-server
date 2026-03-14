package participant

type ID int64

type ParticipantType string

const (
	UserType   ParticipantType = "user"
	AgentType  ParticipantType = "agent"
	SystemType ParticipantType = "system"
)
