package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/raditzlawliet/test-mass-email/internal/campaign"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
)

type Handler struct {
	DefaultConfig *config.Config
	Store         *store.Store
}

func NewHandler(defaultCfg *config.Config, st *store.Store) *Handler {
	return &Handler{DefaultConfig: defaultCfg, Store: st}
}

// HandleGetConfig returns default config merged with current campaign state.
func (h *Handler) HandleGetConfig(c fiber.Ctx) error {
	st := h.Store
	tmpl := st.GetTemplate()
	cfg := st.GetConfig()

	return c.JSON(map[string]any{
		"server": h.DefaultConfig.Server,
		"email": map[string]any{
			"provider": h.DefaultConfig.Email.Provider,
			"from":     h.DefaultConfig.Email.From,
			"smtp":     h.DefaultConfig.Email.SMTP,
			"ses":      h.DefaultConfig.Email.SES,
		},
		"worker": map[string]any{
			"concurrency":        h.DefaultConfig.Worker.Concurrency,
			"max_retries":        h.DefaultConfig.Worker.MaxRetries,
			"retry_backoff_base": h.DefaultConfig.Worker.RetryBackoffBase.String(),
			"retry_backoff_max":  h.DefaultConfig.Worker.RetryBackoffMax.String(),
		},
		"campaign": map[string]any{
			"state":    st.GetState(),
			"progress": st.GetProgress(),
			"events":   st.GetEvents(),
			"template": tmpl,
			"config":   cfg,
		},
	})
}

func (h *Handler) HandlePreview(c fiber.Ctx) error {
	req, csvText, err := parseMultipart(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": err.Error()})
	}
	if csvText == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "csv data is required"})
	}
	req.CSV = csvText

	countStr := c.FormValue("count")
	count := 5
	if countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
			count = n
		}
	}

	results, err := campaign.Preview(req, count)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": err.Error()})
	}
	return c.JSON(map[string]any{"previews": results})
}

func (h *Handler) HandleStart(c fiber.Ctx) error {
	req, csvText, err := parseMultipart(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": err.Error()})
	}
	if csvText == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "csv data is required"})
	}
	if h.Store.IsRunning() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "campaign is already running"})
	}
	req.CSV = csvText

	logCfg := h.DefaultConfig.Log.Campaign
	h.Store.SetVerbose(logCfg.Verbose)

	logger, err := campaign.NewCampaignLogger(".", logCfg.LogToFile, logCfg.Verbose)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"error": "failed to create campaign logger: " + err.Error()})
	}
	ctx := context.Background()
	if err := campaign.StartCampaign(ctx, h.DefaultConfig, h.Store, req, logger); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"error": err.Error()})
	}
	return c.JSON(map[string]any{"status": "started"})
}

func (h *Handler) HandlePause(c fiber.Ctx) error {
	if !h.Store.IsRunning() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "campaign is not running"})
	}
	h.Store.Pause()
	h.Store.LogAndEvent("info", "Campaign paused")
	return c.JSON(map[string]any{"status": "paused"})
}

func (h *Handler) HandleResume(c fiber.Ctx) error {
	if !h.Store.IsPaused() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "campaign is not paused"})
	}
	logCfg := h.DefaultConfig.Log.Campaign
	h.Store.SetVerbose(logCfg.Verbose)

	logger, err := campaign.NewCampaignLogger(".", logCfg.LogToFile, logCfg.Verbose)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"error": "failed to create campaign logger: " + err.Error()})
	}
	ctx := context.Background()
	var req campaign.CampaignRequest
	tmpl := h.Store.GetTemplate()
	cfg := h.Store.GetConfig()
	req.Subject = tmpl.Subject
	req.Body = tmpl.Body
	req.To = tmpl.To
	req.From = cfg.From
	req.Provider = cfg.Provider
	req.SMTP = cfg.SMTP
	req.SES = cfg.SES
	req.Concurrency = cfg.Worker.Concurrency
	req.MaxRetries = cfg.Worker.MaxRetries
	if cfg.Worker.RetryBackoffBase > 0 {
		req.BackoffBase = cfg.Worker.RetryBackoffBase.String()
	}
	if cfg.Worker.RetryBackoffMax > 0 {
		req.BackoffMax = cfg.Worker.RetryBackoffMax.String()
	}
	if err := campaign.ResumeCampaign(ctx, h.DefaultConfig, h.Store, req, logger); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{"error": err.Error()})
	}
	return c.JSON(map[string]any{"status": "resumed"})
}

// HandleEvents streams progress and log via SSE.
func (h *Handler) HandleEvents(c fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			progress := h.Store.GetProgress()
			events := h.Store.GetEvents()
			if events == nil {
				events = []store.LogEntry{}
			}
			data, _ := json.Marshal(map[string]any{
				"progress": progress,
				"events":   events,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			if err := w.Flush(); err != nil {
				return // client disconnected
			}
		}
	})
}

func (h *Handler) HandleReset(c fiber.Ctx) error {
	h.Store.Reset()
	return c.JSON(map[string]any{"status": "reset"})
}

// parseMultipart reads multipart form data and builds a CampaignRequest + CSV text.
func parseMultipart(c fiber.Ctx) (campaign.CampaignRequest, string, error) {
	var csvText string

	// Try file upload first
	file, err := c.FormFile("csv_file")
	if err == nil {
		fh, openErr := file.Open()
		if openErr != nil {
			return campaign.CampaignRequest{}, "", fmt.Errorf("failed to open uploaded file: %w", openErr)
		}
		defer fh.Close()
		data, readErr := io.ReadAll(fh)
		if readErr != nil {
			return campaign.CampaignRequest{}, "", fmt.Errorf("failed to read uploaded file: %w", readErr)
		}
		csvText = string(data)
	} else {
		// Fallback to text field
		csvText = c.FormValue("csv_text")
	}

	smtpPort, _ := strconv.Atoi(c.FormValue("smtp_port"))

	req := campaign.CampaignRequest{
		Subject:  c.FormValue("subject"),
		Body:     c.FormValue("body"),
		To:       c.FormValue("to"),
		From:     c.FormValue("from"),
		Provider: c.FormValue("provider"),
		SMTP: config.SMTPConfig{
			Host:     c.FormValue("smtp_host"),
			Port:     smtpPort,
			Username: c.FormValue("smtp_username"),
			Password: c.FormValue("smtp_password"),
			TLS:      c.FormValue("smtp_tls") == "true",
		},
		SES: config.SESConfig{
			Region:          c.FormValue("ses_region"),
			AccessKeyID:     c.FormValue("ses_access_key_id"),
			SecretAccessKey: c.FormValue("ses_secret_access_key"),
		},
		BackoffBase: c.FormValue("retry_backoff_base"),
		BackoffMax:  c.FormValue("retry_backoff_max"),
	}

	if v := c.FormValue("concurrency"); v != "" {
		req.Concurrency, _ = strconv.Atoi(v)
	}
	if v := c.FormValue("max_retries"); v != "" {
		req.MaxRetries, _ = strconv.Atoi(v)
	}

	return req, csvText, nil
}
