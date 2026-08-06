// Package app exposes the MECS application to the Wails desktop frontend.
package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/campaign"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound application struct.
type App struct {
	ctx           context.Context
	defaultConfig *config.Config
	store         *store.Store
	configPath    string
	version       string
}

// NewApp creates the Wails application instance.
func NewApp(defaultCfg *config.Config, st *store.Store, configPath string, version string) *App {
	return &App{
		defaultConfig: defaultCfg,
		store:         st,
		configPath:    configPath,
		version:       version,
	}
}

// Startup is the Wails OnStartup callback; stores the app context.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	go a.emitProgressLoop()
}

// emitProgressLoop pushes progress + log events to the frontend every 1s,
// mirroring the old SSE stream.
func (a *App) emitProgressLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if a.ctx == nil {
			continue
		}
		events := a.store.GetEvents()
		if events == nil {
			events = []store.LogEntry{}
		}
		runtime.EventsEmit(a.ctx, "campaign:progress", map[string]any{
			"progress": a.store.GetProgress(),
			"events":   events,
		})
	}
}

// GetVersion returns the application version.
func (a *App) GetVersion() string {
	return a.version
}

// GetCampaignConfig returns default config merged with current campaign state.
func (a *App) GetCampaignConfig() map[string]any {
	st := a.store
	tmpl := st.GetTemplate()
	cfg := st.GetConfig()

	return map[string]any{
		"app": map[string]any{
			"theme":    a.defaultConfig.App.Theme,
			"language": a.defaultConfig.App.Language,
		},
		"server": a.defaultConfig.Server,
		"email": map[string]any{
			"provider": a.defaultConfig.Email.Provider,
			"from":     a.defaultConfig.Email.From,
			"smtp":     a.defaultConfig.Email.SMTP,
			"ses":      a.defaultConfig.Email.SES,
		},
		"worker": map[string]any{
			"concurrency":        a.defaultConfig.Worker.Concurrency,
			"max_retries":        a.defaultConfig.Worker.MaxRetries,
			"retry_backoff_base": a.defaultConfig.Worker.RetryBackoffBase.String(),
			"retry_backoff_max":  a.defaultConfig.Worker.RetryBackoffMax.String(),
		},
		"log": map[string]any{
			"campaign": map[string]any{
				"log_to_file": a.defaultConfig.Log.Campaign.LogToFile,
				"verbose":     a.defaultConfig.Log.Campaign.Verbose,
			},
		},
		"campaign": map[string]any{
			"state":    st.GetState(),
			"progress": st.GetProgress(),
			"events":   st.GetEvents(),
			"template": tmpl,
			"config":   cfg,
		},
	}
}

// CampaignInput carries all campaign form fields from the frontend.
type CampaignInput struct {
	CSVText     string
	CSVFilePath string

	Subject  string
	Body     string
	To       string
	From     string
	Provider string

	SMTPHost        string
	SMTPPort        int
	SMTPUsername    string
	SMTPPassword    string
	SMTPTLS         bool
	SMTPBatchSize   int
	SESRegion       string
	SESAccessKeyID  string
	SESSecretKey    string
	SESUseTemplate  bool
	SESTemplateName string
	SESBatchSize    int

	Concurrency int
	MaxRetries  int
	BackoffBase string
	BackoffMax  string
	LogToFile   bool
	Verbose     bool
	Count       int
}

// buildRequest assembles a CampaignRequest from frontend input.
func buildRequest(in CampaignInput, csv string) campaign.CampaignRequest {
	return campaign.CampaignRequest{
		CSV:      csv,
		Subject:  in.Subject,
		Body:     in.Body,
		To:       in.To,
		From:     in.From,
		Provider: in.Provider,
		SMTP: config.SMTPConfig{
			Host:      in.SMTPHost,
			Port:      in.SMTPPort,
			Username:  in.SMTPUsername,
			Password:  in.SMTPPassword,
			TLS:       in.SMTPTLS,
			BatchSize: in.SMTPBatchSize,
		},
		SES: config.SESConfig{
			Region:          in.SESRegion,
			AccessKeyID:     in.SESAccessKeyID,
			SecretAccessKey: in.SESSecretKey,
			UseTemplate:     in.SESUseTemplate,
			TemplateName:    in.SESTemplateName,
			BatchSize:       in.SESBatchSize,
		},
		SmtpBatchSize:  in.SMTPBatchSize,
		BackoffBase:    in.BackoffBase,
		BackoffMax:     in.BackoffMax,
		LogToFileValue: strconv.FormatBool(in.LogToFile),
		VerboseValue:   strconv.FormatBool(in.Verbose),
		Concurrency:    in.Concurrency,
		MaxRetries:     in.MaxRetries,
	}
}

// resolveCSV returns CSV text from either manual input or a picked file path.
func resolveCSV(in CampaignInput) (string, error) {
	if in.CSVText != "" {
		return in.CSVText, nil
	}
	if in.CSVFilePath != "" {
		data, err := os.ReadFile(in.CSVFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read CSV file: %w", err)
		}
		return string(data), nil
	}
	return "", nil
}

// Preview parses CSV and renders sample emails without sending.
func (a *App) Preview(in CampaignInput) ([]campaign.PreviewResult, error) {
	csv, err := resolveCSV(in)
	if err != nil {
		return nil, err
	}
	if csv == "" {
		return nil, fmt.Errorf("csv data is required")
	}
	count := in.Count
	if count <= 0 {
		count = 5
	}
	return campaign.Preview(buildRequest(in, csv), count)
}

// StartCampaign parses CSV and starts the worker pool.
func (a *App) StartCampaign(in CampaignInput) error {
	csv, err := resolveCSV(in)
	if err != nil {
		return err
	}
	if csv == "" {
		return fmt.Errorf("csv data is required")
	}
	if a.store.IsRunning() {
		return fmt.Errorf("campaign is already running")
	}

	logCfg := a.defaultConfig.Log.Campaign
	if in.LogToFile {
		logCfg.LogToFile = true
	}
	if in.Verbose {
		logCfg.Verbose = true
	}
	a.store.SetVerbose(logCfg.Verbose)

	logger, err := campaign.NewCampaignLogger(".", logCfg.LogToFile, logCfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to create campaign logger: %w", err)
	}

	ctx := context.Background()
	if err := campaign.StartCampaign(ctx, a.defaultConfig, a.store, buildRequest(in, csv), logger); err != nil {
		return err
	}
	return nil
}

// PauseCampaign gracefully pauses the running campaign.
func (a *App) PauseCampaign() error {
	if !a.store.IsRunning() {
		return fmt.Errorf("campaign is not running")
	}
	a.store.Pause()
	a.store.LogAndEvent("info", "Campaign paused")
	return nil
}

// ResumeCampaign resumes processing pending recipients only.
func (a *App) ResumeCampaign() error {
	if !a.store.IsPaused() {
		return fmt.Errorf("campaign is not paused")
	}
	ctx := context.Background()
	tmpl := a.store.GetTemplate()
	cfg := a.store.GetConfig()

	req := campaign.CampaignRequest{
		Subject:       tmpl.Subject,
		Body:          tmpl.Body,
		To:            tmpl.To,
		From:          cfg.From,
		Provider:      cfg.Provider,
		SMTP:          cfg.SMTP,
		SES:           cfg.SES,
		Concurrency:   cfg.Worker.Concurrency,
		MaxRetries:    cfg.Worker.MaxRetries,
		SmtpBatchSize: cfg.SmtpBatchSize,
	}
	if cfg.Worker.RetryBackoffBase > 0 {
		req.BackoffBase = cfg.Worker.RetryBackoffBase.String()
	}
	if cfg.Worker.RetryBackoffMax > 0 {
		req.BackoffMax = cfg.Worker.RetryBackoffMax.String()
	}

	logCfg := a.defaultConfig.Log.Campaign
	if cfg.LogToFile {
		logCfg.LogToFile = true
	}
	if cfg.Verbose {
		logCfg.Verbose = true
	}
	a.store.SetVerbose(logCfg.Verbose)

	logger, err := campaign.NewCampaignLogger(".", logCfg.LogToFile, logCfg.Verbose)
	if err != nil {
		return fmt.Errorf("failed to create campaign logger: %w", err)
	}
	if err := campaign.ResumeCampaign(ctx, a.defaultConfig, a.store, req, logger); err != nil {
		return err
	}
	return nil
}

// ResetCampaign clears all campaign state.
func (a *App) ResetCampaign() error {
	a.store.Reset()
	return nil
}

// SaveConfig deep-merges partial JSON into config.yaml and reloads defaults.
func (a *App) SaveConfig(partialJSON string) error {
	if partialJSON == "" {
		return fmt.Errorf("empty body")
	}
	if err := config.SavePartial(a.configPath, []byte(partialJSON)); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	cfg, err := config.Load(a.configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	a.defaultConfig = cfg
	return nil
}

// PickCSVFile opens a native file dialog and returns the selected CSV path.
func (a *App) PickCSVFile() string {
	if a.ctx == nil {
		return ""
	}
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select CSV file",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV files (*.csv)", Pattern: "*.csv"},
			{DisplayName: "All files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return ""
	}
	return file
}

// ParseCSVFile parses a picked CSV file and returns its headers and recipient count.
func (a *App) ParseCSVFile(csvFilePath string) (map[string]any, error) {
	if csvFilePath == "" {
		return nil, fmt.Errorf("no CSV file selected")
	}
	data, err := os.ReadFile(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file: %w", err)
	}
	recipients, err := campaign.ParseCSV(string(data))
	if err != nil {
		return nil, err
	}
	headers := make([]string, 0)
	if len(recipients) > 0 {
		for k := range recipients[0].Data {
			headers = append(headers, k)
		}
	}
	return map[string]any{
		"headers": headers,
		"count":   len(recipients),
	}, nil
}
