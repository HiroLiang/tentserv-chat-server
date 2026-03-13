package transaction

import (
	"context"
)

type UnitOfWork interface {
	Begin(ctx context.Context) (context.Context, Transaction, error)
}

type Transaction interface {
	Commit() error
	Rollback() error
}
