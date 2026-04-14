package senderkeyreceipt

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindLatest(
		ctx context.Context,
		senderMemberID chatmember.ID,
		receiverDeviceID shared.DeviceID,
	) (*SenderKeyReceipt, error)
	Upsert(ctx context.Context, receipt *SenderKeyReceipt) error
}
