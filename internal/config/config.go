package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the full application configuration.
type Config struct {
	App    AppConfig    `yaml:"app"`
	Server ServerConfig `yaml:"server"`
	Email  EmailConfig  `yaml:"email"`
	Worker WorkerConfig `yaml:"worker"`
	Log    LogConfig    `yaml:"log"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Theme string `yaml:"theme"`
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
	App    AppConfig       `yaml:"app"`
	Server ServerConfig    `yaml:"server"`
	Email  EmailConfig     `yaml:"email"`
	Worker rawWorkerConfig `yaml:"worker"`
	Log    LogConfig       `yaml:"log"`
}

//go:embed config.example.yaml
var defaultConfigYAML []byte

// canonicalOrder defines the expected key order for each config section
// by full path (dot-separated), matching the struct field declarations.
// Root uses empty string. Keys not listed appear last.
var canonicalOrder = map[string][]string{
	"":             {"app", "server", "email", "worker", "log"},
	"app":          {"theme"},
	"server":       {"port"},
	"email":        {"provider", "from", "smtp", "ses"},
	"email.smtp":   {"host", "port", "username", "password", "tls", "batch_size"},
	"email.ses":    {"region", "access_key_id", "secret_access_key", "use_template", "template_name", "batch_size"},
	"worker":       {"concurrency", "max_retries", "retry_backoff_base", "retry_backoff_max"},
	"log":          {"campaign"},
	"log.campaign": {"log_to_file", "verbose"},
}

// SavePartial deep-merges a partial JSON object into the YAML file at the given path.
// Uses yaml.Node to preserve key order of the existing file.
func SavePartial(path string, partialJSON []byte) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var current yaml.Node
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &current); err != nil {
			return fmt.Errorf("failed to parse config file %s: %w", path, err)
		}
	}
	if current.Kind == 0 {
		current.Kind = yaml.DocumentNode
		current.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}

	partialNode := jsonToYAMLNode(partialJSON)
	if partialNode == nil {
		return fmt.Errorf("failed to parse partial JSON")
	}

	mergeNode(current.Content[0], partialNode, "")

	out, err := yaml.Marshal(current.Content[0])
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}

// jsonToYAMLNode parses JSON bytes and converts them to a clean yaml.Node
// with proper YAML style and canonical key ordering.
func jsonToYAMLNode(raw []byte) *yaml.Node {
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return mapToNode(m, "")
}

// mapToNode recursively converts a map to a yaml.Node with clean YAML style.
// parent is the full dot-separated path used to look up canonical key ordering.
func mapToNode(m map[string]any, parent string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range orderedKeys(m, parent) {
		v := m[k]
		key := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		var val *yaml.Node
		switch vv := v.(type) {
		case map[string]any:
			path := k
			if parent != "" {
				path = parent + "." + k
			}
			val = mapToNode(vv, path)
		case string:
			val = &yaml.Node{Kind: yaml.ScalarNode, Value: vv}
		case bool:
			val = &yaml.Node{Kind: yaml.ScalarNode, Value: fmtBool(vv)}
		case float64:
			if vv == float64(int64(vv)) {
				val = &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", int64(vv))}
			} else {
				val = &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", vv)}
			}
		default:
			val = &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", vv)}
		}
		node.Content = append(node.Content, key, val)
	}
	return node
}

// orderedKeys returns map keys sorted by canonicalOrder[parent], then remaining keys.
func orderedKeys(m map[string]any, parent string) []string {
	order, _ := canonicalOrder[parent]
	seen := make(map[string]bool)
	var result []string
	for _, k := range order {
		if _, exists := m[k]; exists {
			result = append(result, k)
			seen[k] = true
		}
	}
	for k := range m {
		if !seen[k] {
			result = append(result, k)
		}
	}
	return result
}

func fmtBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// mergeNode recursively merges src into dst, preserving dst's key order.
// parent is the full dot-separated path used for canonical key ordering of new keys.
func mergeNode(dst, src *yaml.Node, parent string) {
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return
	}

	for si := 0; si < len(src.Content)-1; si += 2 {
		key := src.Content[si]
		val := src.Content[si+1]

		found := false
		for di := 0; di < len(dst.Content)-1; di += 2 {
			if dst.Content[di].Value == key.Value {
				found = true
				childPath := key.Value
				if parent != "" {
					childPath = parent + "." + key.Value
				}
				if val.Kind == yaml.MappingNode && dst.Content[di+1].Kind == yaml.MappingNode {
					mergeNode(dst.Content[di+1], val, childPath)
				} else {
					dst.Content[di+1] = val
				}
				break
			}
		}

		if !found {
			idx := insertionIndex(dst.Content, key.Value, parent)
			dst.Content = append(dst.Content, nil, nil)
			copy(dst.Content[idx+2:], dst.Content[idx:])
			dst.Content[idx] = key
			dst.Content[idx+1] = val
		}
	}
}

// insertionIndex finds the position to insert a new key into a mapping node's
// Content slice, based on canonicalOrder for the given parent path.
func insertionIndex(content []*yaml.Node, key, parent string) int {
	order, ok := canonicalOrder[parent]
	if !ok {
		return len(content) // append at end if no canonical order defined
	}

	newRank := keyRank(key, order)

	// Walk existing key-value pairs, insert before first key with higher rank
	for i := 0; i < len(content)-1; i += 2 {
		existingRank := keyRank(content[i].Value, order)
		if existingRank > newRank { // existing key comes AFTER new key, insert here
			return i
		}
	}
	return len(content)
}

// keyRank returns the position of a key in the order slice (0-based).
// Keys not in the order list get a very high rank (sorted alphabetically as tiebreaker).
func keyRank(key string, order []string) int {
	for i, k := range order {
		if k == key {
			return i
		}
	}
	return len(order)
}

// Load reads and parses a YAML configuration file at the given path.
// If the file does not exist, it writes the default configuration and loads it.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if writeErr := os.WriteFile(path, defaultConfigYAML, 0o644); writeErr != nil {
				return nil, fmt.Errorf("failed to create default config file %s: %w", path, writeErr)
			}
			data = defaultConfigYAML
		} else {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}
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
		App:    raw.App,
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

	if cfg.App.Theme == "" {
		cfg.App.Theme = "dark"
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
