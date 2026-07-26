package email

import (
	"fmt"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	mail "github.com/wneessen/go-mail"
)

// smtpSender implements EmailSender via SMTP using go-mail.
type smtpSender struct {
	from string
	cfg  config.SMTPConfig
}

// NewSMTPSender creates a new SMTP-based EmailSender.
func NewSMTPSender(from string, cfg config.SMTPConfig) (EmailSender, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 25
	}
	return &smtpSender{from: from, cfg: cfg}, nil
}

// Send delivers an email via SMTP.
func (s *smtpSender) Send(to string, subject string, body string) error {
	msg := mail.NewMsg()
	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("setting from address: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("setting to address: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)

	opts := []mail.Option{
		mail.WithPort(s.cfg.Port),
	}

	if s.cfg.TLS {
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	} else {
		opts = append(opts, mail.WithTLSPortPolicy(mail.NoTLS))
	}

	if s.cfg.Username != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
			mail.WithUsername(s.cfg.Username),
			mail.WithPassword(s.cfg.Password),
		)
	} else {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthCustom))
	}

	client, err := mail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return fmt.Errorf("creating SMTP client: %w", err)
	}

	if err := client.DialAndSend(msg); err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}
