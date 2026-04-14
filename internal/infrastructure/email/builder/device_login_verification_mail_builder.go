package builder

import (
	"context"
	"fmt"
	"html"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/application/shared/email"
	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

type DeviceLoginVerificationMailBuilder struct {
	sender           shared.EmailSender
	recipientEmail   shared.EmailAddress
	recipientName    string
	deviceName       string
	deviceID         string
	ip               string
	verificationCode string
	loginTime        time.Time
}

func NewDeviceLoginVerificationMailBuilder(
	sender shared.EmailSender,
	recipientEmail, recipientName, deviceName, deviceID, ip, verificationCode string,
	loginTime time.Time,
) *DeviceLoginVerificationMailBuilder {
	return &DeviceLoginVerificationMailBuilder{
		sender:           sender,
		recipientEmail:   shared.EmailAddress(recipientEmail),
		recipientName:    recipientName,
		deviceName:       deviceName,
		deviceID:         deviceID,
		ip:               ip,
		verificationCode: verificationCode,
		loginTime:        loginTime,
	}
}

func (b *DeviceLoginVerificationMailBuilder) BuildEmail(_ context.Context) (*shared.Email, error) {
	htmlBody := fmt.Sprintf(
		`<p>Hi %s,</p><p>We detected a login attempt from a new device.</p><ul><li><b>Device:</b> %s (%s)</li><li><b>IP:</b> %s</li><li><b>Time:</b> %s</li></ul><p>Use this verification code to approve the login:</p><p style="font-size: 28px; font-weight: 700; letter-spacing: 0.4rem;">%s</p><p>If this wasn't you, please change your password immediately.</p>`,
		html.EscapeString(b.recipientName),
		html.EscapeString(b.deviceName),
		html.EscapeString(b.deviceID),
		html.EscapeString(b.ip),
		html.EscapeString(b.loginTime.Format(time.RFC3339)),
		html.EscapeString(b.verificationCode),
	)
	textBody := fmt.Sprintf(
		"Hi %s,\n\nWe detected a login attempt from a new device.\n\nDevice: %s (%s)\nIP: %s\nTime: %s\n\nUse this verification code to approve the login:\n%s\n\nIf this wasn't you, please change your password immediately.",
		b.recipientName,
		b.deviceName,
		b.deviceID,
		b.ip,
		b.loginTime.Format(time.RFC3339),
		b.verificationCode,
	)

	return &shared.Email{
		Sender:     b.sender,
		Recipients: []shared.EmailAddress{b.recipientEmail},
		Subject:    "Verify your new device login",
		Body: shared.EmailBody{
			HTML: htmlBody,
			Text: textBody,
		},
	}, nil
}

var _ email.EmailBuilder = (*DeviceLoginVerificationMailBuilder)(nil)
