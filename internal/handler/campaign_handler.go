package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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
	var req struct {
		campaign.CampaignRequest
		Count int `json:"count"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "invalid request body: " + err.Error()})
	}
	if req.CSV == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "csv field is required"})
	}
	if req.Count <= 0 {
		req.Count = 5
	}
	results, err := campaign.Preview(req.CampaignRequest, req.Count)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": err.Error()})
	}
	return c.JSON(map[string]any{"previews": results})
}

func (h *Handler) HandleStart(c fiber.Ctx) error {
	var req campaign.CampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "invalid request body: " + err.Error()})
	}
	if req.CSV == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "csv field is required"})
	}
	if h.Store.IsRunning() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{"error": "campaign is already running"})
	}
	logger, err := campaign.NewCampaignLogger(".")
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
	logger, err := campaign.NewCampaignLogger(".")
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
