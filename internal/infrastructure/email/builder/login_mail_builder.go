package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/HiroLiang/goat-server/internal/application/shared/email"
	"github.com/HiroLiang/goat-server/internal/domain/shared"
)

type LoginMailBuilder struct {
	sender         shared.EmailSender
	recipientEmail shared.EmailAddress
	recipientName  string
	deviceID       string
	ip             string
	loginTime      time.Time
}

func NewLoginMailBuilder(
	sender shared.EmailSender,
	recipientEmail, recipientName, deviceID, ip string,
	loginTime time.Time,
) *LoginMailBuilder {
	return &LoginMailBuilder{
		sender:         sender,
		recipientEmail: shared.EmailAddress(recipientEmail),
		recipientName:  recipientName,
		deviceID:       deviceID,
		ip:             ip,
		loginTime:      loginTime,
	}
}

func (b *LoginMailBuilder) BuildEmail(_ context.Context) (*shared.Email, error) {
	htmlBody := fmt.Sprintf(
		`<p>Hi %s,</p><p>A new login was detected on your account.</p><ul><li><b>Device:</b> %s</li><li><b>IP:</b> %s</li><li><b>Time:</b> %s</li></ul><p>If this wasn't you, please contact support.</p>`,
		b.recipientName, b.deviceID, b.ip, b.loginTime.Format(time.RFC3339),
	)
	textBody := fmt.Sprintf(
		"Hi %s,\n\nA new login was detected on your account.\n\nDevice: %s\nIP: %s\nTime: %s\n\nIf this wasn't you, please contact support.",
		b.recipientName, b.deviceID, b.ip, b.loginTime.Format(time.RFC3339),
	)

	return &shared.Email{
		Sender:     b.sender,
		Recipients: []shared.EmailAddress{b.recipientEmail},
		Subject:    "New login to your account",
		Body: shared.EmailBody{
			HTML: htmlBody,
			Text: textBody,
		},
	}, nil
}

var _ email.EmailBuilder = (*LoginMailBuilder)(nil)
