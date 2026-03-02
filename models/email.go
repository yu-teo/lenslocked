package models

import (
	"fmt"

	"github.com/go-mail/mail/v2"
)

const (
	DefaultSender = "support@optcgtest.com"
)

type Email struct {
	From      string
	To        string
	Subject   string
	Plaintext string
	Htmltext  string
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type EmailService struct {
	// DefaultSender is used to as the default sender when one isn't provided for an email.
	// This is also used in functions where the email is predeternined, e.g. forgotten pass email.
	DefaultSender string

	// unexported fields
	dialer *mail.Dialer
}

func NewEmailService(config SMTPConfig) *EmailService {
	es := EmailService{
		dialer: mail.NewDialer(config.Host, config.Port, config.Username, config.Password),
	}
	return &es
}
func (es *EmailService) Send(email Email) error {
	msg := mail.NewMessage()
	msg.SetHeader("To", email.To)
	// added default sender via helper function
	es.setFrom(msg, email)
	msg.SetHeader("Subject", email.Subject)
	switch {
	case email.Plaintext != "" && email.Htmltext != "":
		msg.SetBody("text/plain", email.Plaintext)
		msg.AddAlternative("text/html", email.Htmltext)
	case email.Plaintext != "":
		msg.SetBody("text/plain", email.Plaintext)
	case email.Htmltext != "":
		msg.SetBody("text/html", email.Htmltext)
	}

	err := es.dialer.DialAndSend(msg)
	if err != nil {
		return fmt.Errorf("sending the email failed with: %w", err)
	}
	return nil
}

func (es *EmailService) ForgotPassword(to, resetURL string) error {
	email := Email{
		Subject:   "Reset your password",
		To:        to,
		Plaintext: "To reset your password, please visit the following link: " + resetURL,
		Htmltext:  `<p>To reset your password, please visit the following link: <a href="` + resetURL + `">"` + resetURL + `</a></p>`,
	}
	err := es.Send(email)
	if err != nil {
		return fmt.Errorf("forgot password email failed with: %w", err)
	}
	return nil
}

func (es *EmailService) setFrom(msg *mail.Message, email Email) {
	var from string
	switch {
	case email.From != "":
		from = email.From
	case es.DefaultSender != "":
		from = es.DefaultSender
	default:
		from = DefaultSender
	}
	msg.SetHeader("From", from)
}
