package membersenderkey

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindLatest(ctx context.Context, chatMemberID chatmember.ID, senderDeviceID shared.DeviceID) (*MemberSenderKey, error)
	FindLatestForMember(ctx context.Context, chatMemberID chatmember.ID) ([]*MemberSenderKey, error)
	FindAllByMembers(ctx context.Context, chatMemberIDs []chatmember.ID) ([]*MemberSenderKey, error)
	Add(ctx context.Context, sk *MemberSenderKey) error
	UpsertLatest(ctx context.Context, sk *MemberSenderKey) error
}
