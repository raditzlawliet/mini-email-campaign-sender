package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the full application configuration.
type Config struct {
	Server ServerConfig `yaml:"server"`
	Email  EmailConfig  `yaml:"email"`
	Worker WorkerConfig `yaml:"worker"`
	Log    LogConfig    `yaml:"log"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Campaign CampaignLogConfig `yaml:"campaign"`
}

// CampaignLogConfig holds campaign-specific logging settings.
type CampaignLogConfig struct {
	LogToFile bool `yaml:"log_to_file"`
	Verbose   bool `yaml:"verbose"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `yaml:"port"`
}

// EmailConfig holds the default email provider configuration.
type EmailConfig struct {
	Provider string     `yaml:"provider"`
	From     string     `yaml:"from"`
	SMTP     SMTPConfig `yaml:"smtp"`
	SES      SESConfig  `yaml:"ses"`
}

// SMTPConfig holds SMTP server connection details.
type SMTPConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	TLS       bool   `yaml:"tls"`
	BatchSize int    `yaml:"batch_size"`
}

// SESConfig holds AWS SES connection details.
type SESConfig struct {
	Region          string `yaml:"region"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	UseTemplate     bool   `yaml:"use_template"`
	TemplateName    string `yaml:"template_name"`
	BatchSize       int    `yaml:"batch_size"`
}

// WorkerConfig holds worker pool and retry settings.
type WorkerConfig struct {
	Concurrency      int           `yaml:"concurrency"`
	MaxRetries       int           `yaml:"max_retries"`
	RetryBackoffBase time.Duration `yaml:"retry_backoff_base"`
	RetryBackoffMax  time.Duration `yaml:"retry_backoff_max"`
}

// rawWorkerConfig is used for YAML unmarshalling to handle duration strings.
type rawWorkerConfig struct {
	Concurrency      int    `yaml:"concurrency"`
	MaxRetries       int    `yaml:"max_retries"`
	RetryBackoffBase string `yaml:"retry_backoff_base"`
	RetryBackoffMax  string `yaml:"retry_backoff_max"`
}

// rawConfig mirrors Config but uses rawWorkerConfig for duration parsing.
type rawConfig struct {
	Server ServerConfig    `yaml:"server"`
	Email  EmailConfig     `yaml:"email"`
	Worker rawWorkerConfig `yaml:"worker"`
	Log    LogConfig       `yaml:"log"`
}

// Load reads and parses a YAML configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	backoffBase := 1 * time.Second
	if raw.Worker.RetryBackoffBase != "" {
		backoffBase, err = time.ParseDuration(raw.Worker.RetryBackoffBase)
		if err != nil {
			return nil, fmt.Errorf("invalid retry_backoff_base %q: %w", raw.Worker.RetryBackoffBase, err)
		}
	}

	backoffMax := 30 * time.Second
	if raw.Worker.RetryBackoffMax != "" {
		backoffMax, err = time.ParseDuration(raw.Worker.RetryBackoffMax)
		if err != nil {
			return nil, fmt.Errorf("invalid retry_backoff_max %q: %w", raw.Worker.RetryBackoffMax, err)
		}
	}

	cfg := &Config{
		Server: raw.Server,
		Email:  raw.Email,
		Worker: WorkerConfig{
			Concurrency:      raw.Worker.Concurrency,
			MaxRetries:       raw.Worker.MaxRetries,
			RetryBackoffBase: backoffBase,
			RetryBackoffMax:  backoffMax,
		},
		Log: raw.Log,
	}

	if cfg.Email.Provider == "" {
		cfg.Email.Provider = "smtp"
	}
	if cfg.Worker.Concurrency == 0 {
		cfg.Worker.Concurrency = 10
	}
	if cfg.Worker.MaxRetries == 0 {
		cfg.Worker.MaxRetries = 3
	}

	return cfg, nil
}
