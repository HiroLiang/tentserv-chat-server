package builder

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/domain/shared"
)

func TestRegisterMailBuilder_BuildEmailHasStructuredLog(t *testing.T) {
	start := time.Now()
	sender := shared.EmailSender{
		Address: shared.EmailAddress("noreply@example.com"),
		Name:    "Tentserv",
	}
	builder := NewRegisterMailBuilder(
		sender,
		"new@example.com",
		`New <Display>`,
		`https://api.example.com/api/auth/verify-email?token=redacted-token&next=<home>`,
	)

	t.Log("Given: register mail builder with sender, recipient, display name, and verify URL")
	t.Log("Input: recipient_email=new@example.com recipient_name_contains_html=true verify_url_present=true")
	t.Log("Action: build register verification email")

	mail, err := builder.BuildEmail(context.Background())

	t.Logf("Output: subject=%q recipients=%d html_len=%d text_len=%d err=%v", mail.Subject, len(mail.Recipients), len(mail.Body.HTML), len(mail.Body.Text), err)
	t.Log("Mutation: none; builder is pure")
	t.Logf("Duration: %s", time.Since(start))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mail.Sender != sender {
		t.Fatalf("expected sender %+v, got %+v", sender, mail.Sender)
	}
	if len(mail.Recipients) != 1 || mail.Recipients[0] != "new@example.com" {
		t.Fatalf("expected one recipient new@example.com, got %v", mail.Recipients)
	}
	if mail.Subject != "Verify your email" {
		t.Fatalf("expected verify subject, got %q", mail.Subject)
	}
	if !strings.Contains(mail.Body.HTML, "New &lt;Display&gt;") || strings.Contains(mail.Body.HTML, "New <Display>") {
		t.Fatalf("expected display name to be escaped in HTML body, got %s", mail.Body.HTML)
	}
	if !strings.Contains(mail.Body.HTML, "&lt;home&gt;") || strings.Contains(mail.Body.HTML, "<home>") {
		t.Fatalf("expected verify URL to be escaped in HTML body, got %s", mail.Body.HTML)
	}
	if !strings.Contains(mail.Body.Text, "New <Display>") {
		t.Fatalf("expected text body to keep readable display name, got %s", mail.Body.Text)
	}
	if !strings.Contains(mail.Body.Text, "https://api.example.com/api/auth/verify-email?token=redacted-token") {
		t.Fatalf("expected text body to include verify URL, got %s", mail.Body.Text)
	}
}
