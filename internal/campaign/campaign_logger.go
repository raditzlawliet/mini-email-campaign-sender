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
	file *os.File
	path string
}

// NewCampaignLogger creates a new CampaignLogger that writes to a timestamped file.
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

	return &CampaignLogger{file: f, path: path}, nil
}

// Log writes a log entry to the file with the same message as console/frontend.
func (cl *CampaignLogger) Log(level, msg string) {
	if cl.file == nil {
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
