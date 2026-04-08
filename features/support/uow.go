package support

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/transaction"
)

type Tx struct{}

func (Tx) Commit() error   { return nil }
func (Tx) Rollback() error { return nil }

type UOW struct{}

func (UOW) Begin(ctx context.Context) (context.Context, transaction.Transaction, error) {
	return ctx, Tx{}, nil
}

var (
	_ transaction.UnitOfWork  = (*UOW)(nil)
	_ transaction.Transaction = (*Tx)(nil)
)
