package campaign

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/email"
)

// CampaignLogger writes campaign events to a log file with structured logging.
type CampaignLogger struct {
	file *os.File
	path string
}

// NewCampaignLogger creates a new CampaignLogger that writes to a timestamped
// file in the given base directory.
func NewCampaignLogger(baseDir string) (*CampaignLogger, error) {
	logsDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("creating logs directory %s: %w", logsDir, err)
	}

	filename := fmt.Sprintf("campaign_%s.log", time.Now().Format("20060102_150405"))
	path := filepath.Join(logsDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating log file %s: %w", path, err)
	}

	slog.Info("campaign logger created", "path", path)

	return &CampaignLogger{
		file: f,
		path: path,
	}, nil
}

// LogConfig writes the campaign configuration to the log.
func (cl *CampaignLogger) LogConfig(senderCfg email.SenderConfig, workerCfg config.WorkerConfig) {
	cl.write(slog.LevelInfo, "campaign configuration",
		"provider", senderCfg.Provider,
		"from", senderCfg.From,
		"smtp_host", senderCfg.SMTP.Host,
		"smtp_port", senderCfg.SMTP.Port,
		"concurrency", workerCfg.Concurrency,
		"max_retries", workerCfg.MaxRetries,
		"retry_backoff_base", workerCfg.RetryBackoffBase.String(),
		"retry_backoff_max", workerCfg.RetryBackoffMax.String(),
	)
}

// LogStatus writes a general status message to the log.
func (cl *CampaignLogger) LogStatus(msg string, args ...any) {
	cl.write(slog.LevelInfo, msg, args...)
}

// LogRecipient writes a recipient result to the log.
func (cl *CampaignLogger) LogRecipient(index int, status string, errMsg string, attempts int) {
	args := []any{
		"index", index,
		"status", status,
		"attempts", attempts,
	}
	if errMsg != "" {
		args = append(args, "error", errMsg)
	}
	level := slog.LevelInfo
	if status == "failed" {
		level = slog.LevelError
	}
	cl.write(level, "recipient result", args...)
}

// LogRetry writes a retry attempt to the log.
func (cl *CampaignLogger) LogRetry(index int, attempt int, maxRetries int, err error) {
	cl.write(slog.LevelWarn, "retry attempt",
		"index", index,
		"attempt", attempt,
		"max_retries", maxRetries,
		"error", err.Error(),
	)
}

// Path returns the full path to the log file.
func (cl *CampaignLogger) Path() string {
	return cl.path
}

// Close flushes and closes the log file.
func (cl *CampaignLogger) Close() {
	if cl.file != nil {
		slog.Info("campaign log closed", "path", cl.path)
		cl.file.Close()
		cl.file = nil
	}
}

func (cl *CampaignLogger) write(level slog.Level, msg string, args ...any) {
	if cl.file == nil {
		return
	}

	handler := slog.NewJSONHandler(cl.file, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	switch level {
	case slog.LevelDebug:
		logger.Debug(msg, args...)
	case slog.LevelInfo:
		logger.Info(msg, args...)
	case slog.LevelWarn:
		logger.Warn(msg, args...)
	case slog.LevelError:
		logger.Error(msg, args...)
	default:
		logger.Info(msg, args...)
	}
}
