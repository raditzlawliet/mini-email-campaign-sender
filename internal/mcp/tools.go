package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/raditzlawliet/test-mass-email/internal/campaign"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
)

// RegisterTools registers all MECS tools on the given MCP server.
func (s *Server) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_prepare_campaign",
		Description: "Stage (or update) the current campaign without sending. Accepts partial parameters: csv_text or csv_path (validated and stored), template fields (subject, body, to), sender fields (from, provider), SMTP/SES settings, worker settings (concurrency, max_retries, retry_backoff_base, retry_backoff_max), and log flags (log_to_file, verbose). Empty values leave the current staging unchanged. Returns the staged state. Rejected while a campaign is running.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: false, OpenWorldHint: boolPtr(false)},
	}, s.PrepareCampaign)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_get_current_campaign",
		Description: "Return the current campaign state: lifecycle state (idle/ready/running/paused/completed), progress counters (total, sent, failed, pending), whether CSV is prepared, the staged template and config (secrets masked), and optionally the recent log events (include_events: true).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolPtr(false)},
	}, s.GetCurrentCampaign)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_set_config",
		Description: "Persist partial global configuration (config.yaml) by deep-merging a JSON object, e.g. {\"mcp\":{\"port\":19000}} or {\"email\":{\"from\":\"x@y.com\"}}. Only the provided keys change; existing keys and order are preserved. Sensitive values (smtp.password, ses.access_key_id, ses.secret_access_key) are routed to the OS keyring. Changing mcp settings restarts the MCP server with the new settings.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: false, OpenWorldHint: boolPtr(false)},
	}, s.SetConfig)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_get_config",
		Description: "Return the current global configuration from config.yaml: app (theme, language), email (provider, from, smtp, ses), worker, log, and mcp (enabled, host, port). Secrets are masked. Optionally pass section (app|email|worker|log|mcp) to return only that section.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolPtr(false)},
	}, s.GetConfig)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_dry_run_campaign",
		Description: "Render sample emails from the prepared campaign without sending anything. count (default 5, max 5) previews are returned with rendered subject, body, and recipient data. Requires a prepared campaign (see mecs_prepare_campaign).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolPtr(false)},
	}, s.DryRunCampaign)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_start_campaign",
		Description: "Start sending the prepared campaign using the staged CSV, template, and config (global defaults fill any gaps). Returns immediately; the campaign runs in the background. Rejected when no campaign is prepared or when a campaign is already running. Pause with mecs_pause_campaign, monitor with mecs_get_current_campaign.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: false, OpenWorldHint: boolPtr(true)},
	}, s.StartCampaign)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_pause_campaign",
		Description: "Gracefully pause the running campaign: the in-flight email finishes, then processing stops. Rejected when no campaign is running. Resume with mecs_resume_campaign.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: false, OpenWorldHint: boolPtr(false)},
	}, s.PauseCampaign)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_resume_campaign",
		Description: "Resume a paused campaign, processing only recipients still pending. Rejected when the campaign is not paused.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false), IdempotentHint: false, OpenWorldHint: boolPtr(false)},
	}, s.ResumeCampaign)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "mecs_clear_campaign",
		Description: "Clear all campaign state (recipients, template, config, events) and return to idle. Works on paused or completed campaigns. Rejected while running - pause first.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(true), IdempotentHint: true, OpenWorldHint: boolPtr(false)},
	}, s.ClearCampaign)
}

// --- Params ---

type PrepareCampaignParams struct {
	CSVText string `json:"csv_text,omitempty"`
	CSVPath string `json:"csv_path,omitempty"`

	Subject  string `json:"subject,omitempty"`
	Body     string `json:"body,omitempty"`
	To       string `json:"to,omitempty"`
	From     string `json:"from,omitempty"`
	Provider string `json:"provider,omitempty"`

	SMTPHost        string `json:"smtp_host,omitempty"`
	SMTPPort        int    `json:"smtp_port,omitempty"`
	SMTPUsername    string `json:"smtp_username,omitempty"`
	SMTPPassword    string `json:"smtp_password,omitempty"`
	SMTPTLS         *bool  `json:"smtp_tls,omitempty"`
	SMTPBatchSize   int    `json:"smtp_batch_size,omitempty"`
	SESRegion       string `json:"ses_region,omitempty"`
	SESAccessKeyID  string `json:"ses_access_key_id,omitempty"`
	SESSecretKey    string `json:"ses_secret_key,omitempty"`
	SESUseTemplate  *bool  `json:"ses_use_template,omitempty"`
	SESTemplateName string `json:"ses_template_name,omitempty"`
	SESBatchSize    int    `json:"ses_batch_size,omitempty"`

	Concurrency int    `json:"concurrency,omitempty"`
	MaxRetries  int    `json:"max_retries,omitempty"`
	BackoffBase string `json:"retry_backoff_base,omitempty"`
	BackoffMax  string `json:"retry_backoff_max,omitempty"`

	LogToFile *bool `json:"log_to_file,omitempty"`
	Verbose   *bool `json:"verbose,omitempty"`
}

type GetCampaignParams struct {
	IncludeEvents bool `json:"include_events,omitempty"`
}

type SetConfigParams struct {
	PartialJSON string `json:"partial_json"`
}

type GetConfigParams struct {
	Section string `json:"section,omitempty"`
}

type DryRunParams struct {
	Count int `json:"count,omitempty"`
}

// --- Tools ---

func (s *Server) PrepareCampaign(ctx context.Context, req *mcp.CallToolRequest, args PrepareCampaignParams) (*mcp.CallToolResult, any, error) {
	if s.store.IsRunning() {
		return nil, nil, errors.New("campaign is running: pause or wait for it to finish before preparing a new campaign")
	}

	// CSV: text, or file path, or keep current staging.
	csvText := ""
	if args.CSVText != "" {
		csvText = args.CSVText
	} else if args.CSVPath != "" {
		data, err := os.ReadFile(args.CSVPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read csv_path %q: %w", args.CSVPath, err)
		}
		csvText = string(data)
	}

	recipientCount := -1
	if csvText != "" {
		recipients, err := campaign.ParseCSV(csvText)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid CSV: %w", err)
		}
		s.store.SetCSV(recipients)
		s.store.SetCSVText(csvText)
		recipientCount = len(recipients)
	}

	// Merge template (empty values leave staging unchanged).
	tmpl := s.store.GetTemplate()
	if args.Subject != "" {
		tmpl.Subject = args.Subject
	}
	if args.Body != "" {
		tmpl.Body = args.Body
	}
	if args.To != "" {
		tmpl.To = args.To
	}
	s.store.SetTemplate(tmpl)

	// Merge campaign config.
	cfg := s.store.GetConfig()
	if args.From != "" {
		cfg.From = args.From
	}
	if args.Provider != "" {
		cfg.Provider = args.Provider
	}
	if args.SMTPHost != "" {
		cfg.SMTP.Host = args.SMTPHost
	}
	if args.SMTPPort != 0 {
		cfg.SMTP.Port = args.SMTPPort
	}
	if args.SMTPUsername != "" {
		cfg.SMTP.Username = args.SMTPUsername
	}
	if args.SMTPPassword != "" {
		cfg.SMTP.Password = args.SMTPPassword
	}
	if args.SMTPTLS != nil {
		cfg.SMTP.TLS = *args.SMTPTLS
	}
	if args.SMTPBatchSize != 0 {
		cfg.SMTP.BatchSize = args.SMTPBatchSize
		cfg.SmtpBatchSize = args.SMTPBatchSize
	}
	if args.SESRegion != "" {
		cfg.SES.Region = args.SESRegion
	}
	if args.SESAccessKeyID != "" {
		cfg.SES.AccessKeyID = args.SESAccessKeyID
	}
	if args.SESSecretKey != "" {
		cfg.SES.SecretAccessKey = args.SESSecretKey
	}
	if args.SESUseTemplate != nil {
		cfg.SES.UseTemplate = *args.SESUseTemplate
	}
	if args.SESTemplateName != "" {
		cfg.SES.TemplateName = args.SESTemplateName
	}
	if args.SESBatchSize != 0 {
		cfg.SES.BatchSize = args.SESBatchSize
	}
	if args.Concurrency != 0 {
		cfg.Worker.Concurrency = args.Concurrency
	}
	if args.MaxRetries != 0 {
		cfg.Worker.MaxRetries = args.MaxRetries
	}
	if args.BackoffBase != "" {
		cfg.Worker.RetryBackoffBase = parseDurationOrZero(args.BackoffBase)
	}
	if args.BackoffMax != "" {
		cfg.Worker.RetryBackoffMax = parseDurationOrZero(args.BackoffMax)
	}
	if args.LogToFile != nil {
		cfg.LogToFile = *args.LogToFile
	}
	if args.Verbose != nil {
		cfg.Verbose = *args.Verbose
	}
	s.store.SetConfig(cfg)

	s.store.LogAndEvent("info", "Campaign prepared via MCP")
	return resultJSON(map[string]any{
		"status":       "prepared",
		"state":        s.store.GetState(),
		"recipients":   s.recipientCount(recipientCount),
		"csv_prepared": s.store.GetCSVText() != "",
		"template":     s.store.GetTemplate(),
		"config":       campaignConfigToMap(s.store.GetConfig()),
	}), nil, nil
}

func (s *Server) GetCurrentCampaign(ctx context.Context, req *mcp.CallToolRequest, args GetCampaignParams) (*mcp.CallToolResult, any, error) {
	prog := s.store.GetProgress()
	out := map[string]any{
		"state": prog.State,
		"progress": map[string]any{
			"total":   prog.Total,
			"sent":    prog.Sent,
			"failed":  prog.Failed,
			"pending": prog.Pending,
		},
		"csv_prepared": s.store.GetCSVText() != "",
		"template":     s.store.GetTemplate(),
		"config":       campaignConfigToMap(s.store.GetConfig()),
	}
	if args.IncludeEvents {
		out["events"] = s.store.GetEvents()
	}
	return resultJSON(out), nil, nil
}

func (s *Server) SetConfig(ctx context.Context, req *mcp.CallToolRequest, args SetConfigParams) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(args.PartialJSON) == "" {
		return nil, nil, errors.New("partial_json is required: pass a JSON object with the config keys to change, e.g. {\"mcp\":{\"port\":19000}}")
	}
	if err := config.SavePartial(s.configPath, []byte(args.PartialJSON)); err != nil {
		return nil, nil, fmt.Errorf("failed to save config: %w", err)
	}
	if s.reload != nil {
		if err := s.reload(); err != nil {
			return nil, nil, fmt.Errorf("config saved but reload failed: %w", err)
		}
	}
	return resultJSON(map[string]any{"status": "saved"}), nil, nil
}

func (s *Server) GetConfig(ctx context.Context, req *mcp.CallToolRequest, args GetConfigParams) (*mcp.CallToolResult, any, error) {
	all := configToMap(s.cfgProvider())
	if args.Section == "" {
		return resultJSON(all), nil, nil
	}
	v, ok := all[args.Section]
	if !ok {
		return nil, nil, fmt.Errorf("unknown config section %q: valid sections are app, email, worker, log, mcp", args.Section)
	}
	return resultJSON(map[string]any{args.Section: v}), nil, nil
}

func (s *Server) DryRunCampaign(ctx context.Context, req *mcp.CallToolRequest, args DryRunParams) (*mcp.CallToolResult, any, error) {
	cReq, err := s.buildCampaignRequest(true)
	if err != nil {
		return nil, nil, err
	}
	count := args.Count
	if count <= 0 {
		count = 5
	}
	results, err := campaign.Preview(cReq, count)
	if err != nil {
		return nil, nil, fmt.Errorf("dry run failed: %w", err)
	}
	return resultJSON(map[string]any{"preview": results}), nil, nil
}

func (s *Server) StartCampaign(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	if s.store.IsRunning() {
		return nil, nil, errors.New("campaign is already running: pause or wait before starting another")
	}
	cReq, err := s.buildCampaignRequest(true)
	if err != nil {
		return nil, nil, err
	}
	if cReq.Subject == "" || cReq.Body == "" || cReq.To == "" {
		return nil, nil, errors.New("campaign is incomplete: subject, body, and to are required - prepare them with mecs_prepare_campaign first")
	}
	if cReq.From == "" {
		return nil, nil, errors.New("campaign sender (from) is not configured: set it via mecs_prepare_campaign or mecs_set_config")
	}

	logToFile, _ := strconv.ParseBool(cReq.LogToFileValue)
	verbose, _ := strconv.ParseBool(cReq.VerboseValue)
	s.store.SetVerbose(verbose)

	logger, err := campaign.NewCampaignLogger(".", logToFile, verbose)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create campaign logger: %w", err)
	}
	if err := campaign.StartCampaign(context.Background(), s.cfgProvider(), s.store, cReq, logger); err != nil {
		return nil, nil, err
	}
	return resultJSON(map[string]any{"status": "started", "state": s.store.GetState()}), nil, nil
}

func (s *Server) PauseCampaign(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	if !s.store.IsRunning() {
		return nil, nil, fmt.Errorf("campaign is not running (state: %s): nothing to pause", s.store.GetState())
	}
	s.store.Pause()
	s.store.LogAndEvent("info", "Campaign paused via MCP")
	return resultJSON(map[string]any{"status": "paused", "state": s.store.GetState()}), nil, nil
}

func (s *Server) ResumeCampaign(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	if !s.store.IsPaused() {
		return nil, nil, fmt.Errorf("campaign is not paused (state: %s): only a paused campaign can be resumed", s.store.GetState())
	}
	cReq, err := s.buildCampaignRequest(false)
	if err != nil {
		return nil, nil, err
	}

	logToFile, _ := strconv.ParseBool(cReq.LogToFileValue)
	verbose, _ := strconv.ParseBool(cReq.VerboseValue)
	s.store.SetVerbose(verbose)

	logger, err := campaign.NewCampaignLogger(".", logToFile, verbose)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create campaign logger: %w", err)
	}
	if err := campaign.ResumeCampaign(context.Background(), s.cfgProvider(), s.store, cReq, logger); err != nil {
		return nil, nil, err
	}
	return resultJSON(map[string]any{"status": "resumed", "state": s.store.GetState()}), nil, nil
}

func (s *Server) ClearCampaign(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	if s.store.IsRunning() {
		return nil, nil, errors.New("campaign is running: pause it before clearing")
	}
	s.store.Reset()
	return resultJSON(map[string]any{"status": "cleared", "state": s.store.GetState()}), nil, nil
}

// --- Helpers ---

// resultJSON marshals v as indented JSON text content.
func resultJSON(v any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "failed to serialize response"}},
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}
}

// mask replaces non-empty secrets with a fixed marker.
func mask(v string) string {
	if v == "" {
		return ""
	}
	return "****"
}

// configToMap renders the global config as a snake_case map with secrets masked.
func configToMap(cfg *config.Config) map[string]any {
	return map[string]any{
		"app": map[string]any{"theme": cfg.App.Theme, "language": cfg.App.Language},
		"email": map[string]any{
			"provider": cfg.Email.Provider,
			"from":     cfg.Email.From,
			"smtp": map[string]any{
				"host":       cfg.Email.SMTP.Host,
				"port":       cfg.Email.SMTP.Port,
				"username":   cfg.Email.SMTP.Username,
				"password":   mask(cfg.Email.SMTP.Password),
				"tls":        cfg.Email.SMTP.TLS,
				"batch_size": cfg.Email.SMTP.BatchSize,
			},
			"ses": map[string]any{
				"region":            cfg.Email.SES.Region,
				"access_key_id":     mask(cfg.Email.SES.AccessKeyID),
				"secret_access_key": mask(cfg.Email.SES.SecretAccessKey),
				"use_template":      cfg.Email.SES.UseTemplate,
				"template_name":     cfg.Email.SES.TemplateName,
				"batch_size":        cfg.Email.SES.BatchSize,
			},
		},
		"worker": map[string]any{
			"concurrency":        cfg.Worker.Concurrency,
			"max_retries":        cfg.Worker.MaxRetries,
			"retry_backoff_base": cfg.Worker.RetryBackoffBase.String(),
			"retry_backoff_max":  cfg.Worker.RetryBackoffMax.String(),
		},
		"log": map[string]any{
			"campaign": map[string]any{
				"log_to_file": cfg.Log.Campaign.LogToFile,
				"verbose":     cfg.Log.Campaign.Verbose,
			},
		},
		"mcp": map[string]any{
			"enabled": cfg.MCP.Enabled,
			"host":    cfg.MCP.Host,
			"port":    cfg.MCP.Port,
			"token":   mask(cfg.MCP.Token),
		},
	}
}

// campaignConfigToMap renders the staged campaign config with secrets masked.
func campaignConfigToMap(cfg store.CampaignConfig) map[string]any {
	return map[string]any{
		"from":     cfg.From,
		"provider": cfg.Provider,
		"smtp": map[string]any{
			"host":       cfg.SMTP.Host,
			"port":       cfg.SMTP.Port,
			"username":   cfg.SMTP.Username,
			"password":   mask(cfg.SMTP.Password),
			"tls":        cfg.SMTP.TLS,
			"batch_size": cfg.SMTP.BatchSize,
		},
		"ses": map[string]any{
			"region":            cfg.SES.Region,
			"access_key_id":     mask(cfg.SES.AccessKeyID),
			"secret_access_key": mask(cfg.SES.SecretAccessKey),
			"use_template":      cfg.SES.UseTemplate,
			"template_name":     cfg.SES.TemplateName,
			"batch_size":        cfg.SES.BatchSize,
		},
		"worker": map[string]any{
			"concurrency":        cfg.Worker.Concurrency,
			"max_retries":        cfg.Worker.MaxRetries,
			"retry_backoff_base": cfg.Worker.RetryBackoffBase.String(),
			"retry_backoff_max":  cfg.Worker.RetryBackoffMax.String(),
		},
		"smtp_batch_size": cfg.SmtpBatchSize,
		"log_to_file":     cfg.LogToFile,
		"verbose":         cfg.Verbose,
	}
}

// recipientCount returns the recipient count to report after prepare:
// the new count when CSV was provided, otherwise the existing count.
func (s *Server) recipientCount(newCount int) int {
	if newCount >= 0 {
		return newCount
	}
	return len(s.store.GetRecipients())
}

// buildCampaignRequest assembles a fully-resolved campaign request from the
// staged template + config, filling gaps with global defaults. When csvRequired
// is true it errors unless a CSV has been prepared.
func (s *Server) buildCampaignRequest(csvRequired bool) (campaign.CampaignRequest, error) {
	csvText := s.store.GetCSVText()
	if csvRequired && csvText == "" {
		return campaign.CampaignRequest{}, errors.New("no campaign prepared: call mecs_prepare_campaign with csv_text or csv_path first")
	}

	tmpl := s.store.GetTemplate()
	stored := s.store.GetConfig()
	dflt := s.cfgProvider()

	req := campaign.CampaignRequest{
		CSV:     csvText,
		Subject: tmpl.Subject,
		Body:    tmpl.Body,
		To:      tmpl.To,
	}
	req.From = firstNonEmpty(stored.From, dflt.Email.From)
	req.Provider = firstNonEmpty(stored.Provider, dflt.Email.Provider)

	if stored == (store.CampaignConfig{}) {
		// Nothing staged - use global defaults wholesale.
		req.SMTP = dflt.Email.SMTP
		req.SES = dflt.Email.SES
		req.Concurrency = dflt.Worker.Concurrency
		req.MaxRetries = dflt.Worker.MaxRetries
		req.BackoffBase = dflt.Worker.RetryBackoffBase.String()
		req.BackoffMax = dflt.Worker.RetryBackoffMax.String()
		req.SmtpBatchSize = dflt.Email.SMTP.BatchSize
		req.LogToFileValue = strconv.FormatBool(dflt.Log.Campaign.LogToFile)
		req.VerboseValue = strconv.FormatBool(dflt.Log.Campaign.Verbose)
		return req, nil
	}

	req.SMTP = fillSMTP(stored.SMTP, dflt.Email.SMTP)
	req.SES = fillSES(stored.SES, dflt.Email.SES)
	req.Concurrency = firstNonZero(stored.Worker.Concurrency, dflt.Worker.Concurrency)
	req.MaxRetries = firstNonZero(stored.Worker.MaxRetries, dflt.Worker.MaxRetries)
	req.BackoffBase = firstNonEmpty(durationString(stored.Worker.RetryBackoffBase), dflt.Worker.RetryBackoffBase.String())
	req.BackoffMax = firstNonEmpty(durationString(stored.Worker.RetryBackoffMax), dflt.Worker.RetryBackoffMax.String())
	req.SmtpBatchSize = firstNonZero(stored.SmtpBatchSize, dflt.Email.SMTP.BatchSize)
	req.LogToFileValue = strconv.FormatBool(stored.LogToFile || dflt.Log.Campaign.LogToFile)
	req.VerboseValue = strconv.FormatBool(stored.Verbose || dflt.Log.Campaign.Verbose)
	return req, nil
}

func fillSMTP(stored, dflt config.SMTPConfig) config.SMTPConfig {
	if stored.Host == "" {
		stored.Host = dflt.Host
	}
	if stored.Port == 0 {
		stored.Port = dflt.Port
	}
	if stored.Username == "" {
		stored.Username = dflt.Username
	}
	if stored.Password == "" {
		stored.Password = dflt.Password
	}
	if stored.BatchSize == 0 {
		stored.BatchSize = dflt.BatchSize
	}
	return stored
}

func fillSES(stored, dflt config.SESConfig) config.SESConfig {
	if stored.Region == "" {
		stored.Region = dflt.Region
	}
	if stored.AccessKeyID == "" {
		stored.AccessKeyID = dflt.AccessKeyID
	}
	if stored.SecretAccessKey == "" {
		stored.SecretAccessKey = dflt.SecretAccessKey
	}
	if stored.TemplateName == "" {
		stored.TemplateName = dflt.TemplateName
	}
	if stored.BatchSize == 0 {
		stored.BatchSize = dflt.BatchSize
	}
	return stored
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func firstNonZero(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}

func durationString(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

func parseDurationOrZero(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}
