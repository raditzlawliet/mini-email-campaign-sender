package email

import (
	"fmt"

	"github.com/raditzlawliet/test-mass-email/internal/config"
)

// EmailSender defines the interface for sending emails.
type EmailSender interface {
	Send(to string, subject string, body string) error
	// Flush sends any batched/buffered messages. No-op for non-batched senders.
	Flush() error
}

// SenderConfig holds the configuration needed to create an email sender.
type SenderConfig struct {
	Provider string            `json:"provider"`
	From     string            `json:"from"`
	SMTP     config.SMTPConfig `json:"smtp"`
	SES      config.SESConfig  `json:"ses"`
}

// SenderFactory creates a new EmailSender instance.
// Each worker goroutine calls this to get its own dedicated sender,
// avoiding contention on a shared instance.
type SenderFactory func() (EmailSender, error)

// NewSender creates an EmailSender based on the provider in the configuration.
func NewSender(cfg SenderConfig) (EmailSender, error) {
	switch cfg.Provider {
	case "smtp":
		return NewSMTPSender(cfg.From, cfg.SMTP)
	case "ses":
		return NewSESSender(cfg.From, cfg.SES)
	default:
		return nil, fmt.Errorf("unknown email provider: %s", cfg.Provider)
	}
}
