package builder

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type RegisterMailBuilder struct {
}

func NewRegisterMailBuilder() *RegisterMailBuilder {
	return &RegisterMailBuilder{}
}

func (b *RegisterMailBuilder) BuildEmail(ctx context.Context) (*shared.Email, error) {
	//TODO implement me
	panic("implement me")
}

var _ email.EmailBuilder = (*RegisterMailBuilder)(nil)
