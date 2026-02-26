package chat

type GetMyGroupsInput struct{}

type GetGroupMessagesInput struct {
	GroupID  int64
	BeforeID *int64
	Limit    uint64
}

type CreateGroupInput struct {
	Type        string
	Name        string
	Description string
	MemberIDs   []int64 // participant IDs (excluding self)
	MaxMembers  *int    // optional, only used for a group type; defaults to 100
}
