package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/raditzlawliet/test-mass-email/internal/config"
)

// sesSender implements EmailSender via AWS SES.
type sesSender struct {
	from   string
	client *ses.Client
}

// NewSESSender creates a new SES-based EmailSender.
func NewSESSender(from string, cfg config.SESConfig) (EmailSender, error) {
	if cfg.Region == "" {
		return nil, fmt.Errorf("SES region is required")
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
		from:   from,
		client: client,
	}, nil
}

// Send delivers an email via AWS SES.
func (s *sesSender) Send(to string, subject string, body string) error {
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
