package senderkeyrequest

import (
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

type SenderKeyRequest struct {
	ID                 ID
	RequesterMemberID  chatmember.ID
	ProviderMemberID   chatmember.ID
	CreatedAt          time.Time
	FulfilledAt        *time.Time
}
