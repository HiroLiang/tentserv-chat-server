package senderkeyrequest

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type Repository interface {
	// Upsert creates or updates a request. Returns the request ID.
	Upsert(ctx context.Context, req *SenderKeyRequest) error
	// FindPendingByProvider returns all unfulfilled requests where the given member is the provider.
	FindPendingByProvider(ctx context.Context, providerMemberID chatmember.ID, providerDeviceID shared.DeviceID) ([]*SenderKeyRequest, error)
	// MarkFulfilled marks a request as fulfilled by setting fulfilled_at to now.
	MarkFulfilled(ctx context.Context, requesterMemberID chatmember.ID, requesterDeviceID shared.DeviceID, providerMemberID chatmember.ID, providerDeviceID shared.DeviceID) error
}
