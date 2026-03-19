package membersenderkey

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

type Repository interface {
	FindLatest(ctx context.Context, chatMemberID chatmember.ID) (*MemberSenderKey, error)
	FindAllByMembers(ctx context.Context, chatMemberIDs []chatmember.ID) ([]*MemberSenderKey, error)
	Add(ctx context.Context, sk *MemberSenderKey) error
}
