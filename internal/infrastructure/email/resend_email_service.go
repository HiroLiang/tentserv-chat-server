package email

import (
	"context"

	"github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/resend/resend-go/v3"
)

type ResendEmailService struct {
	client   *resend.Client
	recorder email.EmailRecorder
}

func NewResendEmailService(key string) *ResendEmailService {
	client := resend.NewClient(key)
	return &ResendEmailService{
		client: client,
	}
}

func (s *ResendEmailService) Send(ctx context.Context, builder email.EmailBuilder) error {
	mail, err := builder.BuildEmail(ctx)
	if err != nil {
		return shared.ErrSendingEmail
	}

	s.recorder.RecordSending(ctx, mail)

	params := &resend.SendEmailRequest{
		From:    mail.Sender.String(),
		To:      shared.ToStringSlice(mail.Recipients),
		Html:    mail.Body.HTML,
		Text:    mail.Body.Text,
		Subject: mail.Subject,
		Cc:      shared.ToStringSlice(mail.Cc),
		Bcc:     shared.ToStringSlice(mail.Bcc),
		ReplyTo: string(mail.ReplyTo),
	}

	sent, err := s.client.Emails.Send(params)
	if err != nil {
		s.recorder.RecordFailed(ctx, mail)
		logger.Log.Error(err.Error())
	}

	s.recorder.RecordSuccess(ctx, mail, sent.Id)
	return nil
}

var _ email.EmailService = (*ResendEmailService)(nil)
