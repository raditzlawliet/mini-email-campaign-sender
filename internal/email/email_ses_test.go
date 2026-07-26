package email

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raditzlawliet/test-mass-email/internal/config"
)

// mockSESClient records calls for test assertions.
type mockSESClient struct {
	sendEmailCalls             []*ses.SendEmailInput
	sendBulkTemplatedCalls     []*ses.SendBulkTemplatedEmailInput
	sendBulkTemplatedDestCount int
	sendEmailFn                func(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
	bulkSendFn                 func(ctx context.Context, params *ses.SendBulkTemplatedEmailInput, optFns ...func(*ses.Options)) (*ses.SendBulkTemplatedEmailOutput, error)
}

func (m *mockSESClient) SendEmail(ctx context.Context, params *ses.SendEmailInput, optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error) {
	m.sendEmailCalls = append(m.sendEmailCalls, params)
	if m.sendEmailFn != nil {
		return m.sendEmailFn(ctx, params, optFns...)
	}
	return &ses.SendEmailOutput{}, nil
}

func (m *mockSESClient) SendBulkTemplatedEmail(ctx context.Context, params *ses.SendBulkTemplatedEmailInput, optFns ...func(*ses.Options)) (*ses.SendBulkTemplatedEmailOutput, error) {
	m.sendBulkTemplatedCalls = append(m.sendBulkTemplatedCalls, params)
	m.sendBulkTemplatedDestCount += len(params.Destinations)
	if m.bulkSendFn != nil {
		return m.bulkSendFn(ctx, params, optFns...)
	}
	return &ses.SendBulkTemplatedEmailOutput{}, nil
}

// newSESSenderWithClient creates a sesSender with a mock SES client for testing.
func newSESSenderWithClient(from string, cfg config.SESConfig, client sesAPI) *sesSender {
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}
	return &sesSender{
		from:         from,
		client:       client,
		useTemplate:  cfg.UseTemplate,
		templateName: cfg.TemplateName,
		batchSize:    batchSize,
		batch:        make([]bulkEntry, 0, batchSize),
	}
}

func TestSESSender_RawMode(t *testing.T) {
	t.Run("sends single email via SendEmail", func(t *testing.T) {
		assert := assert.New(t)

		mock := &mockSESClient{}
		sender := newSESSenderWithClient("sender@example.com", config.SESConfig{
			Region:      "us-east-1",
			AccessKeyID: "key",
		}, mock)

		err := sender.Send("to@example.com", "Hello", "<p>Body</p>", nil)
		assert.NoError(err)
		assert.Len(mock.sendEmailCalls, 1)
		assert.Equal("sender@example.com", *mock.sendEmailCalls[0].Source)
		assert.Equal([]string{"to@example.com"}, mock.sendEmailCalls[0].Destination.ToAddresses)
		assert.Equal("Hello", *mock.sendEmailCalls[0].Message.Subject.Data)
	})
}

func TestSESSender_TemplateMode(t *testing.T) {
	t.Run("buffers and flushes bulk templated email", func(t *testing.T) {
		assert := assert.New(t)

		mock := &mockSESClient{}
		sender := newSESSenderWithClient("sender@example.com", config.SESConfig{
			Region:       "us-east-1",
			UseTemplate:  true,
			TemplateName: "marketing-v2",
			BatchSize:    3,
		}, mock)

		// Add 2 emails — below batch size, should not flush
		err := sender.Send("a@example.com", "ignored", "ignored", map[string]string{"name": "A"})
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 0, "should buffer, not flush yet")

		err = sender.Send("b@example.com", "ignored", "ignored", map[string]string{"name": "B"})
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 0, "still below batch size")

		// 3rd email triggers auto-flush
		err = sender.Send("c@example.com", "ignored", "ignored", map[string]string{"name": "C"})
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 1)
		assert.Equal(3, mock.sendBulkTemplatedDestCount)

		call := mock.sendBulkTemplatedCalls[0]
		assert.Equal("sender@example.com", *call.Source)
		assert.Equal("marketing-v2", *call.Template)
		assert.Len(call.Destinations, 3)

		// Verify template data for first destination
		var td map[string]string
		err = json.Unmarshal([]byte(*call.Destinations[0].ReplacementTemplateData), &td)
		require.NoError(t, err)
		assert.Equal("A", td["name"])
	})

	t.Run("Flush sends remaining batch", func(t *testing.T) {
		assert := assert.New(t)

		mock := &mockSESClient{}
		sender := newSESSenderWithClient("sender@example.com", config.SESConfig{
			Region:       "us-east-1",
			UseTemplate:  true,
			TemplateName: "tpl",
			BatchSize:    50, // large batch — won't auto-flush
		}, mock)

		err := sender.Send("x@example.com", "", "", map[string]string{"key": "val"})
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 0)

		err = sender.Flush()
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 1)
		assert.Equal(1, mock.sendBulkTemplatedDestCount)
	})

	t.Run("Flush empty batch is no-op", func(t *testing.T) {
		assert := assert.New(t)

		mock := &mockSESClient{}
		sender := newSESSenderWithClient("sender@example.com", config.SESConfig{
			Region:       "us-east-1",
			UseTemplate:  true,
			TemplateName: "tpl",
		}, mock)

		err := sender.Flush()
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 0)
	})
}

func TestSESSender_FlushNoopForRaw(t *testing.T) {
	t.Run("Flush is no-op in raw mode", func(t *testing.T) {
		assert := assert.New(t)

		mock := &mockSESClient{}
		sender := newSESSenderWithClient("sender@example.com", config.SESConfig{
			Region: "us-east-1",
		}, mock)

		err := sender.Flush()
		assert.NoError(err)
		assert.Len(mock.sendBulkTemplatedCalls, 0)
	})
}
