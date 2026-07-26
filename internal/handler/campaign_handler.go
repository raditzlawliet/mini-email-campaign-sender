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

	state := h.Store.GetState()
	if state == store.StateRunning {
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

// HandleProgress returns current campaign progress.
func (h *Handler) HandleProgress(c fiber.Ctx) error {
	progress := h.Store.GetProgress()
	return c.JSON(progress)
}

// HandleReset clears all campaign data.
func (h *Handler) HandleReset(c fiber.Ctx) error {
	h.Store.Reset()
	return c.JSON(map[string]any{
		"status": "reset",
	})
}
