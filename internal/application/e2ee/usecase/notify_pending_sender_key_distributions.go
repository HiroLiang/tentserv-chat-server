package usecase

import (
	"context"
	"strconv"

	e2eePort "github.com/HiroLiang/tentserv-chat-server/internal/application/e2ee/port"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeydistribution"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

// NotifyPendingSenderKeyDistributionsUseCase is called when a user connects via WebSocket.
// It replays still-available sender key distributions addressed to the user so missed realtime
// notifications can be consumed immediately after reconnect.
type NotifyPendingSenderKeyDistributionsUseCase struct {
	participantRepo  participant.Repository
	chatMemberRepo   chatmember.Repository
	distributionRepo senderkeydistribution.Repository
	broadcaster      e2eePort.Broadcaster
}

func NewNotifyPendingSenderKeyDistributionsUseCase(
	participantRepo participant.Repository,
	chatMemberRepo chatmember.Repository,
	distributionRepo senderkeydistribution.Repository,
	broadcaster e2eePort.Broadcaster,
) *NotifyPendingSenderKeyDistributionsUseCase {
	return &NotifyPendingSenderKeyDistributionsUseCase{
		participantRepo:  participantRepo,
		chatMemberRepo:   chatMemberRepo,
		distributionRepo: distributionRepo,
		broadcaster:      broadcaster,
	}
}

func (u *NotifyPendingSenderKeyDistributionsUseCase) Execute(ctx context.Context, userIDStr string) {
	userIDInt, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return
	}
	userID := shared.UserID(userIDInt)

	p, err := u.participantRepo.FindByUserID(ctx, userID)
	if err != nil {
		return
	}

	members, err := u.chatMemberRepo.FindByParticipant(ctx, p.ID)
	if err != nil {
		return
	}

	for _, member := range members {
		if member.IsDeleted {
			continue
		}
		distributions, err := u.distributionRepo.FindAvailableByRoomAndReceiver(ctx, member.RoomID, member.ID)
		if err != nil {
			continue
		}
		for _, dist := range distributions {
			if dist.RoomID <= 0 {
				continue
			}
			distCopy := *dist
			go notifySenderKeyDistributionAvailable(
				context.Background(),
				u.broadcaster,
				u.participantRepo,
				u.chatMemberRepo,
				&distCopy,
				distCopy.RoomID,
			)
		}
	}
}
