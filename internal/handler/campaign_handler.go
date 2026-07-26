package handler

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/raditzlawliet/test-mass-email/internal/campaign"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
)

// Handler holds shared dependencies for campaign HTTP handlers.
type Handler struct {
	DefaultConfig *config.Config
	Store         *store.Store
}

// NewHandler creates a new Handler with the given dependencies.
func NewHandler(defaultCfg *config.Config, st *store.Store) *Handler {
	return &Handler{
		DefaultConfig: defaultCfg,
		Store:         st,
	}
}

// HandleGetConfig returns the default configuration.
func (h *Handler) HandleGetConfig(c fiber.Ctx) error {
	return c.JSON(map[string]any{
		"server": h.DefaultConfig.Server,
		"email": map[string]any{
			"provider": h.DefaultConfig.Email.Provider,
			"from":     h.DefaultConfig.Email.From,
			"smtp":     h.DefaultConfig.Email.SMTP,
			"ses":      h.DefaultConfig.Email.SES,
		},
		"worker": h.DefaultConfig.Worker,
	})
}

// HandlePreview parses CSV and renders sample email previews.
func (h *Handler) HandlePreview(c fiber.Ctx) error {
	var req struct {
		campaign.CampaignRequest
		Count int `json:"count"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "invalid request body: " + err.Error(),
		})
	}

	if req.CSV == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "csv field is required",
		})
	}

	if req.Count <= 0 {
		req.Count = 5
	}

	results, err := campaign.Preview(req.CampaignRequest, req.Count)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"previews": results,
	})
}

// HandleStart launches the campaign in a background goroutine.
func (h *Handler) HandleStart(c fiber.Ctx) error {
	var req campaign.CampaignRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "invalid request body: " + err.Error(),
		})
	}

	if req.CSV == "" {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "csv field is required",
		})
	}

	if h.Store.IsRunning() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "campaign is already running",
		})
	}

	logger, err := campaign.NewCampaignLogger(".")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"error": "failed to create campaign logger: " + err.Error(),
		})
	}

	ctx := context.Background()
	if err := campaign.StartCampaign(ctx, h.DefaultConfig, h.Store, req, logger); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"status": "started",
	})
}

// HandlePause pauses the running campaign.
func (h *Handler) HandlePause(c fiber.Ctx) error {
	if !h.Store.IsRunning() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "campaign is not running",
		})
	}

	h.Store.Pause()
	h.Store.AddEvent("info", "Campaign paused")

	return c.JSON(map[string]any{
		"status": "paused",
	})
}

// HandleResume resumes a paused campaign.
func (h *Handler) HandleResume(c fiber.Ctx) error {
	if !h.Store.IsPaused() {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"error": "campaign is not paused",
		})
	}

	logger, err := campaign.NewCampaignLogger(".")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"error": "failed to create campaign logger: " + err.Error(),
		})
	}

	ctx := context.Background()

	// Build a minimal request from stored data — we use the default config + current state
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

	if err := campaign.ResumeCampaign(ctx, h.DefaultConfig, h.Store, req, logger); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"error": err.Error(),
		})
	}

	return c.JSON(map[string]any{
		"status": "resumed",
	})
}

// HandleProgress returns current campaign progress.
func (h *Handler) HandleProgress(c fiber.Ctx) error {
	progress := h.Store.GetProgress()
	return c.JSON(progress)
}

// HandleLog returns campaign log events.
func (h *Handler) HandleLog(c fiber.Ctx) error {
	events := h.Store.GetEvents()
	if events == nil {
		events = []store.LogEntry{}
	}
	return c.JSON(map[string]any{
		"events": events,
	})
}

// HandleReset clears all campaign data.
func (h *Handler) HandleReset(c fiber.Ctx) error {
	h.Store.Reset()
	return c.JSON(map[string]any{
		"status": "reset",
	})
}
