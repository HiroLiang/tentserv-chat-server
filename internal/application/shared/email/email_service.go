package email

import (
	"context"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type EmailBuilder interface {
	BuildEmail(ctx context.Context) (*shared.Email, error)
}

type EmailRecorder interface {
	RecordSending(ctx context.Context, email *shared.Email)
	RecordFailed(ctx context.Context, email *shared.Email)
	RecordSuccess(ctx context.Context, email *shared.Email, id string)
}

type EmailService interface {
	Send(ctx context.Context, builder EmailBuilder) error
}
