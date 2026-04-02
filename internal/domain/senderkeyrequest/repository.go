package senderkeyrequest

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
)

type Repository interface {
	// Upsert creates or updates a request. Returns the request ID.
	Upsert(ctx context.Context, req *SenderKeyRequest) error
	// FindPendingByProvider returns all unfulfilled requests where the given member is the provider.
	FindPendingByProvider(ctx context.Context, providerMemberID chatmember.ID) ([]*SenderKeyRequest, error)
	// MarkFulfilled marks a request as fulfilled by setting fulfilled_at to now.
	MarkFulfilled(ctx context.Context, requesterMemberID, providerMemberID chatmember.ID) error
}
