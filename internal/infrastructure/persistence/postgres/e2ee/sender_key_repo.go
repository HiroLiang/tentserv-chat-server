package e2ee

import (
	"context"
	"errors"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/chatmember"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/membersenderkey"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/postgres"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var senderKeyTable = postgres.Table{
	Name: "public.member_sender_keys",
	Columns: []string{
		"id", "chat_member_id", "chain_id", "sender_key_public", "distribution_message", "created_at",
	},
}

type SenderKeyRepository struct {
	postgres.BaseRepo
}

var _ membersenderkey.Repository = (*SenderKeyRepository)(nil)

func NewSenderKeyRepository(db *sqlx.DB) *SenderKeyRepository {
	return &SenderKeyRepository{BaseRepo: postgres.NewBaseRepo(db)}
}

func (r *SenderKeyRepository) FindLatest(
	ctx context.Context,
	chatMemberID chatmember.ID,
) (*membersenderkey.MemberSenderKey, error) {
	query, args, err := senderKeyTable.Select(senderKeyTable.Columns...).
		Where(squirrel.Eq{"chat_member_id": chatMemberID}).
		OrderBy("chain_id DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sender key query: %w", err)
	}

	rec, err := postgres.ScanOne[SenderKeyRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, membersenderkey.ErrNotFound
		}
		return nil, fmt.Errorf("find latest sender key: %w", err)
	}

	return toSenderKeyDomain(rec)
}

func (r *SenderKeyRepository) FindAllByMembers(
	ctx context.Context,
	chatMemberIDs []chatmember.ID,
) ([]*membersenderkey.MemberSenderKey, error) {
	if len(chatMemberIDs) == 0 {
		return nil, nil
	}

	query, args, err := senderKeyTable.Select(senderKeyTable.Columns...).
		Where(squirrel.Eq{"chat_member_id": chatMemberIDs}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sender keys query: %w", err)
	}

	records, err := postgres.ScanAll[SenderKeyRecord](ctx, r.GetDB(ctx), query, args...)
	if err != nil {
		return nil, fmt.Errorf("scan sender keys: %w", err)
	}

	keys := make([]*membersenderkey.MemberSenderKey, 0, len(records))
	for _, rec := range records {
		k, err := toSenderKeyDomain(&rec)
		if err != nil {
			return nil, fmt.Errorf("convert sender key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *SenderKeyRepository) Add(ctx context.Context, sk *membersenderkey.MemberSenderKey) error {
	rec := toSenderKeyRecord(sk)

	query, args, err := senderKeyTable.Insert().
		Columns("chat_member_id", "chain_id", "sender_key_public", "distribution_message").
		Values(rec.ChatMemberID, rec.ChainID, rec.SenderKeyPublic, rec.DistributionMessage).
		Suffix("RETURNING id, created_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert sender key: %w", err)
	}

	db := r.GetDB(ctx)
	row := db.QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&sk.ID, &sk.CreatedAt); err != nil {
		return fmt.Errorf("insert sender key: %w", err)
	}
	return nil
}
