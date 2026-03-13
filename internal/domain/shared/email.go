package shared

import (
	"fmt"
	"net/mail"
)

type EmailAddress string

func ParseEmail(s string) (EmailAddress, error) {
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return "", ErrInvalidEmail
	}
	return EmailAddress(addr.Address), nil
}

type EmailSender struct {
	Address EmailAddress
	Name    string
}

func (s EmailSender) String() string {
	if s.Name == "" {
		return string(s.Address)
	}
	return fmt.Sprintf("%s <%s>", s.Name, s.Address)
}

type EmailBody struct {
	HTML string
	Text string
}

type Email struct {
	Sender     EmailSender
	Recipients []EmailAddress
	Subject    string
	Body       EmailBody
	Cc         []EmailAddress
	Bcc        []EmailAddress
	ReplyTo    EmailAddress
}

func ToStringSlice(addresses []EmailAddress) []string {
	result := make([]string, len(addresses))
	for i, addr := range addresses {
		result[i] = string(addr)
	}
	return result
}
