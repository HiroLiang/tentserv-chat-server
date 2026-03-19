package builder

import (
	"context"
	"fmt"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type RegisterMailBuilder struct {
	sender         shared.EmailSender
	recipientEmail shared.EmailAddress
	recipientName  string
	verifyURL      string
}

func NewRegisterMailBuilder(sender shared.EmailSender, recipientEmail, recipientName, verifyURL string) *RegisterMailBuilder {
	return &RegisterMailBuilder{
		sender:         sender,
		recipientEmail: shared.EmailAddress(recipientEmail),
		recipientName:  recipientName,
		verifyURL:      verifyURL,
	}
}

func (b *RegisterMailBuilder) BuildEmail(_ context.Context) (*shared.Email, error) {
	htmlBody := fmt.Sprintf(
		`<p>Hi %s,</p><p>Please verify your email address by clicking the link below:</p><p><a href="%s">Verify Email</a></p><p>This link will expire in 24 hours.</p>`,
		b.recipientName, b.verifyURL,
	)
	textBody := fmt.Sprintf(
		"Hi %s,\n\nPlease verify your email address by visiting:\n%s\n\nThis link will expire in 24 hours.",
		b.recipientName, b.verifyURL,
	)

	return &shared.Email{
		Sender:     b.sender,
		Recipients: []shared.EmailAddress{b.recipientEmail},
		Subject:    "Verify your email",
		Body: shared.EmailBody{
			HTML: htmlBody,
			Text: textBody,
		},
	}, nil
}

var _ email.EmailBuilder = (*RegisterMailBuilder)(nil)
