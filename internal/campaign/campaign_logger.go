package campaign

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// CampaignLogger writes campaign events to a log file with simple JSON lines.
type CampaignLogger struct {
	file    *os.File
	path    string
	verbose bool
}

// NewCampaignLogger creates a new CampaignLogger.
// If logToFile is false, no file is created and Log is a no-op.
func NewCampaignLogger(baseDir string, logToFile bool, verbose bool) (*CampaignLogger, error) {
	cl := &CampaignLogger{verbose: verbose}

	if !logToFile {
		return cl, nil
	}

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

	cl.file = f
	cl.path = path
	return cl, nil
}

// Log writes a log entry to the file.
// Debug-level messages are suppressed unless verbose mode is enabled.
func (cl *CampaignLogger) Log(level, msg string) {
	if cl.file == nil {
		return
	}
	if level == "debug" && !cl.verbose {
		return
	}
	t := time.Now().Format(time.RFC3339)
	fmt.Fprintf(cl.file, `{"time":"%s","level":"%s","msg":"%s"}`+"\n", t, level, msg)
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
