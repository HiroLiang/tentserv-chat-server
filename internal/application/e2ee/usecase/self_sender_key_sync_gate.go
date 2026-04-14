package usecase

import (
	"context"
	"errors"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/selfsenderkeysync"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func ensureDeviceNotBlockedByActiveSelfSync(
	ctx context.Context,
	selfSyncRepo selfsenderkeysync.Repository,
	participantID participant.ID,
	deviceID shared.DeviceID,
) error {
	syncState, err := selfSyncRepo.FindByParticipantID(ctx, participantID)
	if err != nil {
		if errors.Is(err, selfsenderkeysync.ErrNotFound) {
			return nil
		}
		return err
	}
	if syncState.IsActive() && syncState.RequesterDeviceID == deviceID {
		return ErrSelfSenderKeySyncInProgress
	}
	return nil
}
