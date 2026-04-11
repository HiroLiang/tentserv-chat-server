package builder

import (
	"context"
	"fmt"
	"html"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type RegisterMailBuilder struct {
	sender         shared.EmailSender
	recipientEmail shared.EmailAddress
	recipientName  string
	verificationCode string
}

func NewRegisterMailBuilder(sender shared.EmailSender, recipientEmail, recipientName, verificationCode string) *RegisterMailBuilder {
	return &RegisterMailBuilder{
		sender:           sender,
		recipientEmail:   shared.EmailAddress(recipientEmail),
		recipientName:    recipientName,
		verificationCode: verificationCode,
	}
}

func (b *RegisterMailBuilder) BuildEmail(_ context.Context) (*shared.Email, error) {
	htmlRecipientName := html.EscapeString(b.recipientName)
	htmlVerificationCode := html.EscapeString(b.verificationCode)
	htmlBody := fmt.Sprintf(
		`<p>Hi %s,</p><p>Use the verification code below to finish creating your Tentserv Chat account:</p><p style="font-size: 28px; font-weight: 700; letter-spacing: 0.4rem;">%s</p><p>This code will expire in 3 minutes.</p>`,
		htmlRecipientName, htmlVerificationCode,
	)
	textBody := fmt.Sprintf(
		"Hi %s,\n\nUse this verification code to finish creating your Tentserv Chat account:\n%s\n\nThis code will expire in 3 minutes.",
		b.recipientName, b.verificationCode,
	)

	return &shared.Email{
		Sender:     b.sender,
		Recipients: []shared.EmailAddress{b.recipientEmail},
		Subject:    "Your Tentserv verification code",
		Body: shared.EmailBody{
			HTML: htmlBody,
			Text: textBody,
		},
	}, nil
}

var _ email.EmailBuilder = (*RegisterMailBuilder)(nil)
