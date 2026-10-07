package campaign

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/email"
	"github.com/raditzlawliet/test-mass-email/internal/store"
	"github.com/raditzlawliet/test-mass-email/internal/worker"
)

// csvBufferSize is the read buffer used when streaming CSV files from disk.
const csvBufferSize = 1 << 20

// CampaignRequest is the payload sent by the frontend for preview and start.
type CampaignRequest struct {
	CSV            string            `json:"csv"`
	CSVPath        string            `json:"csv_path"` // streamed from disk when CSV is empty
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	To             string            `json:"to"`
	From           string            `json:"from"`
	Provider       string            `json:"provider"`
	SMTP           config.SMTPConfig `json:"smtp"`
	SES            config.SESConfig  `json:"ses"`
	Concurrency    int               `json:"concurrency"`
	MaxRetries     int               `json:"max_retries"`
	SmtpBatchSize  int               `json:"smtp_batch_size"`
	BackoffBase    string            `json:"retry_backoff_base"`
	BackoffMax     string            `json:"retry_backoff_max"`
	LogToFileValue string            `json:"log_to_file"` // "true" or "false"
	VerboseValue   string            `json:"verbose"`     // "true" or "false"
}

// ParseCSV parses CSV text and returns recipients.
func ParseCSV(text string) ([]store.Recipient, error) {
	return ParseCSVReader(strings.NewReader(text), 0)
}

// ParseCSVFile streams a CSV file from disk and returns up to limit recipients
// (limit <= 0 means all). The file is never loaded into memory as a whole.
func ParseCSVFile(path string, limit int) ([]store.Recipient, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}
	defer f.Close()
	return ParseCSVReader(bufio.NewReaderSize(f, csvBufferSize), limit)
}

// ParseCSVReader streams CSV rows from r and returns up to limit recipients
// (limit <= 0 means all). Only the recipients that are kept stay in memory.
func ParseCSVReader(r io.Reader, limit int) ([]store.Recipient, error) {
	var recipients []store.Recipient
	_, err := scanCSV(r, limit, func(rec store.Recipient) {
		recipients = append(recipients, rec)
	})
	if err != nil {
		return nil, err
	}
	if recipients == nil {
		recipients = []store.Recipient{}
	}
	return recipients, nil
}

// ScanCSVFile streams a CSV file and returns its headers and valid recipient
// count without keeping any recipient in memory.
func ScanCSVFile(path string) (headers []string, count int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read CSV file: %w", err)
	}
	defer f.Close()
	headers, err = scanCSV(bufio.NewReaderSize(f, csvBufferSize), 0, func(store.Recipient) { count++ })
	if err != nil {
		return nil, 0, err
	}
	return headers, count, nil
}

// scanCSV reads CSV rows one at a time and calls emit for each valid
// recipient, stopping after limit recipients (limit <= 0 means no limit).
// It returns the trimmed header names in column order.
func scanCSV(r io.Reader, limit int, emit func(store.Recipient)) ([]string, error) {
	reader := csv.NewReader(r)
	reader.ReuseRecord = true

	first, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}
	if err != nil {
		return nil, fmt.Errorf("parsing CSV: %w", err)
	}

	headers := make([]string, len(first))
	emailColIdx := -1
	for i, h := range first {
		headers[i] = strings.TrimSpace(h)
		if emailColIdx == -1 && strings.EqualFold(headers[i], "email") {
			emailColIdx = i
		}
	}
	if emailColIdx == -1 {
		return nil, fmt.Errorf("CSV must contain an 'email' column")
	}

	rows, valid := 0, 0
	for limit <= 0 || valid < limit {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing CSV: %w", err)
		}
		rows++

		if len(row) < emailColIdx+1 {
			slog.Warn("skipping row with insufficient columns", "row", rows)
			continue
		}
		emailAddr := strings.TrimSpace(row[emailColIdx])
		if emailAddr == "" {
			slog.Warn("skipping row with empty email", "row", rows)
			continue
		}

		data := make(map[string]string, len(headers))
		for colIdx, header := range headers {
			if colIdx < len(row) {
				data[header] = row[colIdx]
			} else {
				data[header] = ""
			}
		}

		emit(store.Recipient{Index: valid, Data: data, Email: emailAddr})
		valid++
	}

	if rows == 0 {
		return nil, fmt.Errorf("CSV must have a header row and at least one data row")
	}
	if valid == 0 {
		return nil, fmt.Errorf("no valid recipients found in CSV")
	}
	return headers, nil
}

// loadRecipients parses recipients from the request: inline CSV text first,
// otherwise the CSV file path (streamed from disk). limit <= 0 means all.
func loadRecipients(req CampaignRequest, limit int) ([]store.Recipient, error) {
	if req.CSV == "" && req.CSVPath != "" {
		return ParseCSVFile(req.CSVPath, limit)
	}
	return ParseCSVReader(strings.NewReader(req.CSV), limit)
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
	if count <= 0 || count > 5 {
		count = 5
	}
	// Only the first rows are needed, so stop reading after count recipients.
	recipients, err := loadRecipients(req, count)
	if err != nil {
		return nil, err
	}
	if count > len(recipients) {
		count = len(recipients)
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
	recipients, err := loadRecipients(req, 0)
	if err != nil {
		return fmt.Errorf("parsing CSV: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients to send")
	}

	st.ClearEvents()
	st.SetCSV(recipients)
	st.SetTemplate(store.Template{
		Subject: req.Subject,
		Body:    req.Body,
		To:      req.To,
	})
	st.SetConfig(store.CampaignConfig{
		From:     req.From,
		Provider: req.Provider,
		SMTP:     req.SMTP,
		SES:      req.SES,
		Worker: config.WorkerConfig{
			Concurrency:      req.Concurrency,
			MaxRetries:       req.MaxRetries,
			RetryBackoffBase: parseDurationOrZero(req.BackoffBase),
			RetryBackoffMax:  parseDurationOrZero(req.BackoffMax),
		},
		SmtpBatchSize: req.SmtpBatchSize,
		LogToFile:     req.LogToFileValue == "true",
		Verbose:       req.VerboseValue == "true",
	})

	factory, wp, err := buildSenderFactory(defaultCfg, req)
	if err != nil {
		return err
	}

	provider := req.Provider
	if provider == "" {
		provider = defaultCfg.Email.Provider
	}
	logger.Log("info", fmt.Sprintf("Campaign started — %d recipients, provider=%s", len(recipients), provider))
	st.StartCampaign()
	st.LogAndEvent("info", fmt.Sprintf("Campaign started — %d recipients", len(recipients)))

	ctx, cancel := context.WithCancel(parentCtx)
	st.SetCancelFn(cancel)

	go func() {
		wp.Run(ctx, factory, st, logger)
		if st.GetState() == store.StateRunning {
			st.FinishCampaign()
			st.LogAndEvent("info", "Campaign completed")
		}
		logger.Close()
	}()

	return nil
}

// ResumeCampaign restarts the worker pool for only pending recipients.
func ResumeCampaign(parentCtx context.Context, defaultCfg *config.Config, st *store.Store, req CampaignRequest, logger *CampaignLogger) error {
	factory, wp, err := buildSenderFactory(defaultCfg, req)
	if err != nil {
		return err
	}

	st.StartCampaign()
	st.LogAndEvent("info", "Campaign resumed — processing remaining pending recipients")

	ctx, cancel := context.WithCancel(parentCtx)
	st.SetCancelFn(cancel)

	go func() {
		wp.RunPending(ctx, factory, st, logger)
		if st.GetState() == store.StateRunning {
			st.FinishCampaign()
			st.LogAndEvent("info", "Campaign completed")
		}
		logger.Close()
	}()

	return nil
}

func buildSenderFactory(defaultCfg *config.Config, req CampaignRequest) (email.SenderFactory, *worker.WorkerPool, error) {
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
	if req.SmtpBatchSize > 0 {
		senderCfg.SMTP.BatchSize = req.SmtpBatchSize
	}
	if req.SES.Region != "" {
		senderCfg.SES = req.SES
	}

	// Factory creates a fresh sender per worker goroutine.
	factory := func() (email.EmailSender, error) {
		return email.NewSender(senderCfg)
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

	return factory, wp, nil
}

func parseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

func parseDurationOrZero(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}
