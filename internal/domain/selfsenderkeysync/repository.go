package selfsenderkeysync

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/participant"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	FindByParticipantID(ctx context.Context, participantID participant.ID) (*SelfSenderKeySync, error)
	UpsertPending(ctx context.Context, participantID participant.ID, requesterDeviceID shared.DeviceID) (*SelfSenderKeySync, error)
	ClaimProvider(ctx context.Context, id ID, participantID participant.ID, requesterDeviceID, providerDeviceID shared.DeviceID) (bool, error)
	MarkUploaded(ctx context.Context, id ID, participantID participant.ID, requesterDeviceID, providerDeviceID shared.DeviceID) error
	MarkCompleted(ctx context.Context, id ID, participantID participant.ID, requesterDeviceID shared.DeviceID) error
	MarkFailed(ctx context.Context, id ID, participantID participant.ID, requesterDeviceID, providerDeviceID shared.DeviceID, lastError string, retryable bool) error
}
