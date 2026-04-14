package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/senderkeyreceipt"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var senderKeyReceiptTable = postgres.Table{
	Name: "public.sender_key_receipts",
	Columns: []string{
		"id",
		"sender_member_id",
		"sender_device_id",
		"receiver_member_id",
		"receiver_device_id",
		"sender_key_version",
		"source",
		"updated_at",
	},
}

type SenderKeyReceiptRepository struct {
	postgres.BaseRepo
}

var _ senderkeyreceipt.Repository = (*SenderKeyReceiptRepository)(nil)

func NewSenderKeyReceiptRepository(db *sqlx.DB) *SenderKeyReceiptRepository {
	return &SenderKeyReceiptRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SenderKeyReceiptRepository) FindLatest(
	ctx context.Context,
	senderMemberID chatmember.ID,
	receiverDeviceID shared.DeviceID,
) (*senderkeyreceipt.SenderKeyReceipt, error) {
	query, args, err := senderKeyReceiptTable.Select(senderKeyReceiptTable.Columns...).
		Where(squirrel.Eq{
			"sender_member_id":   int64(senderMemberID),
			"receiver_device_id": receiverDeviceID.String(),
		}).
		OrderBy("sender_key_version DESC", "updated_at DESC", "id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sender key receipt query: %w", err)
	}

	record, err := postgres.ScanOne[SenderKeyReceiptRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, senderkeyreceipt.ErrNotFound
		}
		return nil, fmt.Errorf("find sender key receipt: %w", err)
	}
	return toSenderKeyReceiptDomain(record)
}

func (r *SenderKeyReceiptRepository) Upsert(
	ctx context.Context,
	receipt *senderkeyreceipt.SenderKeyReceipt,
) error {
	query := `
INSERT INTO public.sender_key_receipts
    (sender_member_id, sender_device_id, receiver_member_id, receiver_device_id, sender_key_version, source, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (sender_member_id, receiver_device_id)
DO UPDATE SET sender_key_version = GREATEST(EXCLUDED.sender_key_version, sender_key_receipts.sender_key_version),
              sender_device_id = CASE
                  WHEN EXCLUDED.sender_key_version >= sender_key_receipts.sender_key_version THEN EXCLUDED.sender_device_id
                  ELSE sender_key_receipts.sender_device_id
              END,
              receiver_member_id = CASE
                  WHEN EXCLUDED.sender_key_version >= sender_key_receipts.sender_key_version THEN EXCLUDED.receiver_member_id
                  ELSE sender_key_receipts.receiver_member_id
              END,
              source = CASE
                  WHEN EXCLUDED.sender_key_version >= sender_key_receipts.sender_key_version THEN EXCLUDED.source
                  ELSE sender_key_receipts.source
              END,
              updated_at = now()`

	return postgres.Exec(
		ctx,
		r.GetDB(ctx),
		query,
		int64(receipt.SenderMemberID),
		receipt.SenderDeviceID.String(),
		int64(receipt.ReceiverMemberID),
		receipt.ReceiverDeviceID.String(),
		receipt.SenderKeyVersion,
		string(receipt.Source),
	)
}

func toSenderKeyReceiptDomain(record *SenderKeyReceiptRecord) (*senderkeyreceipt.SenderKeyReceipt, error) {
	if record == nil {
		return nil, senderkeyreceipt.ErrNotFound
	}

	senderDeviceID, err := shared.ParseDeviceID(record.SenderDeviceID)
	if err != nil {
		return nil, fmt.Errorf("parse sender device id: %w", err)
	}
	receiverDeviceID, err := shared.ParseDeviceID(record.ReceiverDeviceID)
	if err != nil {
		return nil, fmt.Errorf("parse receiver device id: %w", err)
	}

	return &senderkeyreceipt.SenderKeyReceipt{
		ID:               senderkeyreceipt.ID(record.ID),
		SenderMemberID:   chatmember.ID(record.SenderMemberID),
		SenderDeviceID:   senderDeviceID,
		ReceiverMemberID: chatmember.ID(record.ReceiverMemberID),
		ReceiverDeviceID: receiverDeviceID,
		SenderKeyVersion: record.SenderKeyVersion,
		Source:           senderkeyreceipt.Source(record.Source),
		UpdatedAt:        record.UpdatedAt,
	}, nil
}
