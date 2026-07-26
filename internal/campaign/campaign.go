package campaign

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/email"
	"github.com/raditzlawliet/test-mass-email/internal/store"
	"github.com/raditzlawliet/test-mass-email/internal/worker"
)

// CampaignRequest is the payload sent by the frontend for preview and start.
type CampaignRequest struct {
	CSV         string            `json:"csv"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	To          string            `json:"to"`
	From        string            `json:"from"`
	Provider    string            `json:"provider"`
	SMTP        config.SMTPConfig `json:"smtp"`
	SES         config.SESConfig  `json:"ses"`
	Concurrency int               `json:"concurrency"`
	MaxRetries  int               `json:"max_retries"`
	BackoffBase string            `json:"retry_backoff_base"`
	BackoffMax  string            `json:"retry_backoff_max"`
}

// ParseCSV parses CSV text and returns recipients.
func ParseCSV(text string) ([]store.Recipient, error) {
	reader := csv.NewReader(strings.NewReader(text))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	headers := records[0]
	emailColIdx := -1
	for i, h := range headers {
		if strings.EqualFold(strings.TrimSpace(h), "email") {
			emailColIdx = i
			break
		}
	}
	if emailColIdx == -1 {
		return nil, fmt.Errorf("CSV must contain an 'email' column")
	}

	recipients := make([]store.Recipient, 0, len(records)-1)
	for rowIdx, row := range records[1:] {
		if len(row) < emailColIdx+1 {
			slog.Warn("skipping row with insufficient columns", "row", rowIdx+1)
			continue
		}

		data := make(map[string]string, len(headers))
		for colIdx, header := range headers {
			if colIdx < len(row) {
				data[strings.TrimSpace(header)] = row[colIdx]
			} else {
				data[strings.TrimSpace(header)] = ""
			}
		}

		emailAddr := strings.TrimSpace(row[emailColIdx])
		if emailAddr == "" {
			slog.Warn("skipping row with empty email", "row", rowIdx+1)
			continue
		}

		recipients = append(recipients, store.Recipient{
			Index: len(recipients),
			Data:  data,
			Email: emailAddr,
		})
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no valid recipients found in CSV")
	}

	return recipients, nil
}

// PreviewResult is a pre-rendered email preview.
type PreviewResult struct {
	Index   int               `json:"index"`
	To      string            `json:"to"`
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	Data    map[string]string `json:"data"`
}

// Preview parses CSV text and renders sample email previews.
func Preview(req CampaignRequest, count int) ([]PreviewResult, error) {
	recipients, err := ParseCSV(req.CSV)
	if err != nil {
		return nil, err
	}

	if count <= 0 || count > len(recipients) {
		count = len(recipients)
	}
	if count > 5 {
		count = 5
	}

	results := make([]PreviewResult, 0, count)
	for i := 0; i < count; i++ {
		r := recipients[i]
		results = append(results, PreviewResult{
			Index:   r.Index,
			To:      email.Render(req.To, r.Data),
			Subject: email.Render(req.Subject, r.Data),
			Body:    email.Render(req.Body, r.Data),
			Data:    r.Data,
		})
	}

	return results, nil
}

// StartCampaign parses CSV, stores it, then runs the campaign with worker pool.
func StartCampaign(parentCtx context.Context, defaultCfg *config.Config, st *store.Store, req CampaignRequest, logger *CampaignLogger) error {
	recipients, err := ParseCSV(req.CSV)
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients to send")
	}

	st.SetCSV(recipients)
	st.SetTemplate(store.Template{
		Subject: req.Subject,
		Body:    req.Body,
		To:      req.To,
	})

	sender, wp, err := buildSenderAndPool(defaultCfg, req)
	if err != nil {
		return err
	}

	logger.LogConfig(email.SenderConfig{}, config.WorkerConfig{})
	st.StartCampaign()
	st.AddEvent("info", fmt.Sprintf("Campaign started — %d recipients", len(recipients)))

	ctx, cancel := context.WithCancel(parentCtx)
	st.SetCancelFn(cancel)

	go func() {
		wp.Run(ctx, sender, st, logger)
		if st.GetState() == store.StateRunning {
			st.FinishCampaign()
			st.AddEvent("info", "Campaign completed")
		}
		logger.Close()
		slog.Info("campaign completed")
	}()

	return nil
}

// ResumeCampaign restarts the worker pool for only pending recipients.
func ResumeCampaign(parentCtx context.Context, defaultCfg *config.Config, st *store.Store, req CampaignRequest, logger *CampaignLogger) error {
	sender, wp, err := buildSenderAndPool(defaultCfg, req)
	if err != nil {
		return err
	}

	st.StartCampaign()
	st.AddEvent("info", "Campaign resumed — processing remaining pending recipients")

	ctx, cancel := context.WithCancel(parentCtx)
	st.SetCancelFn(cancel)

	go func() {
		wp.RunPending(ctx, sender, st, logger)
		if st.GetState() == store.StateRunning {
			st.FinishCampaign()
			st.AddEvent("info", "Campaign completed")
		}
		logger.Close()
		slog.Info("campaign completed after resume")
	}()

	return nil
}

func buildSenderAndPool(defaultCfg *config.Config, req CampaignRequest) (email.EmailSender, *worker.WorkerPool, error) {
	senderCfg := email.SenderConfig{
		Provider: defaultCfg.Email.Provider,
		From:     defaultCfg.Email.From,
		SMTP:     defaultCfg.Email.SMTP,
		SES:      defaultCfg.Email.SES,
	}
	if req.Provider != "" {
		senderCfg.Provider = req.Provider
	}
	if req.From != "" {
		senderCfg.From = req.From
	}
	if req.SMTP.Host != "" {
		senderCfg.SMTP = req.SMTP
	}
	if req.SES.Region != "" {
		senderCfg.SES = req.SES
	}

	sender, err := email.NewSender(senderCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("creating email sender: %w", err)
	}

	workerCfg := defaultCfg.Worker
	if req.Concurrency > 0 {
		workerCfg.Concurrency = req.Concurrency
	}
	if req.MaxRetries > 0 {
		workerCfg.MaxRetries = req.MaxRetries
	}
	if req.BackoffBase != "" {
		if d, err := parseDuration(req.BackoffBase); err == nil {
			workerCfg.RetryBackoffBase = d
		}
	}
	if req.BackoffMax != "" {
		if d, err := parseDuration(req.BackoffMax); err == nil {
			workerCfg.RetryBackoffMax = d
		}
	}

	wp := &worker.WorkerPool{
		Concurrency: workerCfg.Concurrency,
		MaxRetries:  workerCfg.MaxRetries,
		BackoffBase: workerCfg.RetryBackoffBase,
		BackoffMax:  workerCfg.RetryBackoffMax,
	}

	return sender, wp, nil
}

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
