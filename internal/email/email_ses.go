package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/raditzlawliet/test-mass-email/internal/config"
)

// bulkEntry holds a single recipient's data for batched template sending.
type bulkEntry struct {
	To   string
	Data map[string]string
}

// sesAPI is the subset of SES client methods used by sesSender.
// This interface exists for testability; production code passes *ses.Client.
type sesAPI interface {
	SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
	SendBulkTemplatedEmail(ctx context.Context, params *ses.SendBulkTemplatedEmailInput, optFns ...func(*ses.Options)) (*ses.SendBulkTemplatedEmailOutput, error)
}

// sesSender implements EmailSender via AWS SES.
type sesSender struct {
	from         string
	client       sesAPI
	useTemplate  bool
	templateName string
	batchSize    int
	batch        []bulkEntry
}

// NewSESSender creates a new SES-based EmailSender.
func NewSESSender(from string, cfg config.SESConfig) (EmailSender, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("SES region is required")
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 50 // SES SendBulkTemplatedEmail limit
	}

	ctx := context.Background()

	var loadOpts []func(*awsconfig.LoadOptions) error

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		loadOpts = append(loadOpts,
			awsconfig.WithRegion(cfg.Region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
			),
		)
	} else {
		loadOpts = append(loadOpts, awsconfig.WithRegion(cfg.Region))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := ses.NewFromConfig(awsCfg)

	return &sesSender{
		from:         from,
		client:       client,
		useTemplate:  cfg.UseTemplate,
		templateName: cfg.TemplateName,
		batchSize:    batchSize,
		batch:        make([]bulkEntry, 0, batchSize),
	}, nil
}

// Send delivers an email via AWS SES.
// In template mode, subject and body are ignored — the SES template
// defines its own content. recipient data is sent as TemplateData.
func (s *sesSender) Send(to string, subject string, body string, data map[string]string) error {
	if s.useTemplate {
		return s.sendTemplate(to, data)
	}
	return s.sendRaw(to, subject, body)
}

func (s *sesSender) sendRaw(to string, subject string, body string) error {
	input := &ses.SendEmailInput{
		Source: aws.String(s.from),
		Destination: &sestypes.Destination{
			ToAddresses: []string{to},
		},
		Message: &sestypes.Message{
			Subject: &sestypes.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
			Body: &sestypes.Body{
				Text: &sestypes.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
		},
	}

	_, err := s.client.SendEmail(context.Background(), input)
	if err != nil {
		return fmt.Errorf("SES send email: %w", err)
	}

	return nil
}

func (s *sesSender) sendTemplate(to string, data map[string]string) error {
	s.batch = append(s.batch, bulkEntry{To: to, Data: data})

	if len(s.batch) >= s.batchSize {
		return s.flushTemplate()
	}
	return nil
}

func (s *sesSender) flushTemplate() error {
	if len(s.batch) == 0 {
		return nil
	}

	destinations := make([]sestypes.BulkEmailDestination, 0, len(s.batch))
	for _, entry := range s.batch {
		templateData, err := json.Marshal(entry.Data)
		if err != nil {
			return fmt.Errorf("marshalling template data for %s: %w", entry.To, err)
		}
		destinations = append(destinations, sestypes.BulkEmailDestination{
			Destination: &sestypes.Destination{
				ToAddresses: []string{entry.To},
			},
			ReplacementTemplateData: aws.String(string(templateData)),
		})
	}

	input := &ses.SendBulkTemplatedEmailInput{
		Source:              aws.String(s.from),
		Template:            aws.String(s.templateName),
		Destinations:        destinations,
		DefaultTemplateData: aws.String("{}"),
	}

	batchSize := len(s.batch)
	s.batch = make([]bulkEntry, 0, s.batchSize)

	_, err := s.client.SendBulkTemplatedEmail(context.Background(), input)
	if err != nil {
		return fmt.Errorf("SES send bulk templated email (batch of %d): %w", batchSize, err)
	}

	return nil
}

// Flush sends any remaining buffered template emails. No-op for raw mode.
func (s *sesSender) Flush() error {
	if s.useTemplate {
		return s.flushTemplate()
	}
	return nil
}
