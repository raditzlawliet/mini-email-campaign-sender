package email

import (
	"fmt"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	mail "github.com/wneessen/go-mail"
)

// smtpSender implements EmailSender via SMTP using go-mail.
// It accumulates messages into batches and flushes them via a single
// DialAndSend call, avoiding per-message connection overhead.
type smtpSender struct {
	from      string
	cfg       config.SMTPConfig
	batchSize int
	client    *mail.Client
	batch     []*mail.Msg
}

// NewSMTPSender creates a new SMTP-based EmailSender.
func NewSMTPSender(from string, cfg config.SMTPConfig) (EmailSender, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if cfg.Port == 0 {
		cfg.Port = 25
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	opts := []mail.Option{
		mail.WithPort(cfg.Port),
	}

	if cfg.TLS {
		opts = append(opts, mail.WithTLSPortPolicy(mail.TLSMandatory))
	} else {
		opts = append(opts, mail.WithTLSPortPolicy(mail.NoTLS))
	}

	if cfg.Username != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthLogin),
			mail.WithUsername(cfg.Username),
			mail.WithPassword(cfg.Password),
		)
	} else {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthCustom))
	}

	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating SMTP client: %w", err)
	}

	return &smtpSender{
		from:      from,
		cfg:       cfg,
		batchSize: batchSize,
		client:    client,
		batch:     make([]*mail.Msg, 0, batchSize),
	}, nil
}

// Send builds a message and appends it to the batch.
// It auto-flushes when the batch reaches the configured size.
func (s *smtpSender) Send(to string, subject string, body string, data map[string]string) error {
	msg := mail.NewMsg()
	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("setting from address: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("setting to address: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)

	s.batch = append(s.batch, msg)

	if len(s.batch) >= s.batchSize {
		return s.Flush()
	}
	return nil
}

// PendingCount returns the number of messages buffered in the batch.
func (s *smtpSender) PendingCount() int {
	return len(s.batch)
}

// Flush sends all accumulated messages in a single DialAndSend call.
func (s *smtpSender) Flush() error {
	if len(s.batch) == 0 {
		return nil
	}

	batch := s.batch
	s.batch = make([]*mail.Msg, 0, s.batchSize)

	if err := s.client.DialAndSend(batch...); err != nil {
		return fmt.Errorf("sending batch of %d emails: %w", len(batch), err)
	}
	return nil
}
