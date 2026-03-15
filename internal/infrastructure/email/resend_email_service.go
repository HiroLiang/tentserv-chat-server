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

func NewResendEmailService(key string, recorder email.EmailRecorder) *ResendEmailService {
	client := resend.NewClient(key)
	logger.Log.Info("Resend email service initialized" + key)
	return &ResendEmailService{
		client:   client,
		recorder: recorder,
	}
}

func (s *ResendEmailService) Send(ctx context.Context, builder email.EmailBuilder) error {
	mail, err := builder.BuildEmail(ctx)
	if err != nil {
		return shared.ErrSendingEmail
	}

	if s.recorder != nil {
		s.recorder.RecordSending(ctx, mail)
	}

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
		if s.recorder != nil {
			s.recorder.RecordFailed(ctx, mail)
		}
		logger.Log.Error(err.Error())
		return shared.ErrSendingEmail
	}

	if s.recorder != nil {
		s.recorder.RecordSuccess(ctx, mail, sent.Id)
	}
	return nil
}

var _ email.EmailService = (*ResendEmailService)(nil)
