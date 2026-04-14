package e2ee

import (
	"context"
	"fmt"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyrequest"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/jmoiron/sqlx"
)

var senderKeyRequestTable = postgres.Table{
	Name:    "public.sender_key_requests",
	Columns: []string{"id", "requester_member_id", "requester_device_id", "provider_member_id", "provider_device_id", "created_at", "fulfilled_at"},
}

type SenderKeyRequestRepository struct {
	postgres.BaseRepo
}

var _ senderkeyrequest.Repository = (*SenderKeyRequestRepository)(nil)

func NewSenderKeyRequestRepository(db *sqlx.DB) *SenderKeyRequestRepository {
	return &SenderKeyRequestRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SenderKeyRequestRepository) Upsert(ctx context.Context, req *senderkeyrequest.SenderKeyRequest) error {
	query, args, err := senderKeyRequestTable.Insert().
		Columns("requester_member_id", "requester_device_id", "provider_member_id", "provider_device_id").
		Values(int64(req.RequesterMemberID), req.RequesterDeviceID.String(), int64(req.ProviderMemberID), req.ProviderDeviceID.String()).
		Suffix(`ON CONFLICT (requester_member_id, requester_device_id, provider_member_id, provider_device_id)
			DO UPDATE SET fulfilled_at = NULL, created_at = now()
			RETURNING id, created_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build upsert sender key request: %w", err)
	}

	row := r.GetDB(ctx).QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&req.ID, &req.CreatedAt); err != nil {
		return fmt.Errorf("upsert sender key request: %w", err)
	}
	return nil
}

func (r *SenderKeyRequestRepository) FindPendingByProvider(
	ctx context.Context,
	providerMemberID chatmember.ID,
	providerDeviceID shared.DeviceID,
) ([]*senderkeyrequest.SenderKeyRequest, error) {
	const query = `
SELECT id, requester_member_id, requester_device_id, provider_member_id, provider_device_id, created_at, fulfilled_at
FROM public.sender_key_requests
WHERE provider_member_id = $1
  AND provider_device_id = $2
  AND fulfilled_at IS NULL`

	type row struct {
		ID                int64      `db:"id"`
		RequesterMemberID int64      `db:"requester_member_id"`
		RequesterDeviceID string     `db:"requester_device_id"`
		ProviderMemberID  int64      `db:"provider_member_id"`
		ProviderDeviceID  string     `db:"provider_device_id"`
		CreatedAt         time.Time  `db:"created_at"`
		FulfilledAt       *time.Time `db:"fulfilled_at"`
	}

	var rows []row
	if err := sqlx.SelectContext(ctx, r.GetDB(ctx), &rows, query, int64(providerMemberID), providerDeviceID.String()); err != nil {
		return nil, fmt.Errorf("find pending sender key requests: %w", err)
	}

	result := make([]*senderkeyrequest.SenderKeyRequest, len(rows))
	for i, r := range rows {
		result[i] = &senderkeyrequest.SenderKeyRequest{
			ID:                senderkeyrequest.ID(r.ID),
			RequesterMemberID: chatmember.ID(r.RequesterMemberID),
			RequesterDeviceID: shared.DeviceID(parseUUIDOrNil(r.RequesterDeviceID)),
			ProviderMemberID:  chatmember.ID(r.ProviderMemberID),
			ProviderDeviceID:  shared.DeviceID(parseUUIDOrNil(r.ProviderDeviceID)),
			CreatedAt:         r.CreatedAt,
			FulfilledAt:       r.FulfilledAt,
		}
	}
	return result, nil
}

func (r *SenderKeyRequestRepository) MarkFulfilled(
	ctx context.Context,
	requesterMemberID chatmember.ID,
	requesterDeviceID shared.DeviceID,
	providerMemberID chatmember.ID,
	providerDeviceID shared.DeviceID,
) error {
	const query = `
UPDATE public.sender_key_requests
SET fulfilled_at = now()
WHERE requester_member_id = $1
  AND provider_member_id = $2
  AND requester_device_id = $3
  AND provider_device_id = $4
  AND fulfilled_at IS NULL`

	if err := postgres.Exec(ctx, r.GetDB(ctx), query, int64(requesterMemberID), int64(providerMemberID), requesterDeviceID.String(), providerDeviceID.String()); err != nil {
		return fmt.Errorf("mark sender key request fulfilled: %w", err)
	}
	return nil
}
