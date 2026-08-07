package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

// Config holds the full application configuration.
type Config struct {
	App    AppConfig    `yaml:"app"`
	Email  EmailConfig  `yaml:"email"`
	Worker WorkerConfig `yaml:"worker"`
	Log    LogConfig    `yaml:"log"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Theme    string `yaml:"theme"`
	Language string `yaml:"language"`
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

// EmailConfig holds the default email provider configuration.
type EmailConfig struct {
	Provider string     `yaml:"provider"`
	From     string     `yaml:"from"`
	SMTP     SMTPConfig `yaml:"smtp"`
	SES      SESConfig  `yaml:"ses"`
}

// SecretProvider tracks where a sensitive value is stored.
// type: "keyring" (OS keyring) or "plaintext" (inline in YAML).
type SecretProvider struct {
	Type    string `yaml:"type"`
	Account string `yaml:"account"`
	Service string `yaml:"service"`
}

// SMTPConfig holds SMTP server connection details.
type SMTPConfig struct {
	Host             string         `yaml:"host"`
	Port             int            `yaml:"port"`
	Username         string         `yaml:"username"`
	Password         string         `yaml:"password"`
	PasswordProvider SecretProvider `yaml:"password_provider"`
	TLS              bool           `yaml:"tls"`
	BatchSize        int            `yaml:"batch_size"`
}

// SESConfig holds AWS SES connection details.
type SESConfig struct {
	Region                  string         `yaml:"region"`
	AccessKeyID             string         `yaml:"access_key_id"`
	AccessKeyIDProvider     SecretProvider `yaml:"access_key_id_provider"`
	SecretAccessKey         string         `yaml:"secret_access_key"`
	SecretAccessKeyProvider SecretProvider `yaml:"secret_access_key_provider"`
	UseTemplate             bool           `yaml:"use_template"`
	TemplateName            string         `yaml:"template_name"`
	BatchSize               int            `yaml:"batch_size"`
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
	Email  EmailConfig     `yaml:"email"`
	Worker rawWorkerConfig `yaml:"worker"`
	Log    LogConfig       `yaml:"log"`
}

//go:embed config.example.yaml
var defaultConfigYAML []byte

const keyringService = "raditzlawliet.mecs"

// sensitiveKeyPaths lists YAML dot-path keys that may be stored in keyring.
var sensitiveKeyPaths = map[string]bool{
	"email.smtp.password":         true,
	"email.ses.access_key_id":     true,
	"email.ses.secret_access_key": true,
}

// canonicalOrder defines the expected key order for each config section
// by full path (dot-separated), matching the struct field declarations.
// Root uses empty string. Keys not listed appear last.
var canonicalOrder = map[string][]string{
	"":             {"app", "email", "worker", "log"},
	"app":          {"theme", "language"},
	"email":        {"provider", "from", "smtp", "ses"},
	"email.smtp":   {"host", "port", "username", "password", "password_provider", "tls", "batch_size"},
	"email.ses":    {"region", "access_key_id", "access_key_id_provider", "secret_access_key", "secret_access_key_provider", "use_template", "template_name", "batch_size"},
	"worker":       {"concurrency", "max_retries", "retry_backoff_base", "retry_backoff_max"},
	"log":          {"campaign"},
	"log.campaign": {"log_to_file", "verbose"},
}

// ResolveConfigPath determines the config file path.
// Priority: CONFIG_PATH env → ./config.yaml (if exists) → ~/.mecs/config.yaml
func ResolveConfigPath() string {
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}
	return filepath.Join(defaultConfigDir(), "config.yaml")
}

// defaultConfigDir returns the platform-specific config directory.
func defaultConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".mecs"
	}
	return filepath.Join(home, ".mecs")
}

// SavePartial deep-merges a partial JSON object into the YAML file at the given path.
// Uses yaml.Node to preserve key order of the existing file.
// Sensitive fields are stored in the OS keyring when available, falling back to
// plaintext YAML if the keyring is unreachable. Existing _provider metadata
// (type, account, service) is preserved.
func SavePartial(path string, partialJSON []byte) error {
	// Parse partial JSON to detect sensitive keys.
	var partialMap map[string]any
	if err := json.Unmarshal(partialJSON, &partialMap); err != nil {
		return fmt.Errorf("failed to parse partial JSON: %w", err)
	}

	// Read existing config to extract current provider metadata.
	existingProviders := loadExistingProviders(path)

	// Extract and store sensitive values, falling back to plaintext if keyring fails.
	strippedJSON, err := routeSecretsToKeyring(partialJSON, partialMap, "", existingProviders)
	if err != nil {
		return fmt.Errorf("failed to process secrets: %w", err)
	}

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

	partialNode := jsonToYAMLNode(strippedJSON)
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

// routeSecretsToKeyring walks a parsed JSON partial, detects sensitive keys,
// stores their values in the OS keyring (with fallback to plaintext),
// and adds _provider metadata blocks.
// existingProviders maps full YAML path -> SecretProvider from the current config.
// Returns the modified JSON bytes.
func routeSecretsToKeyring(originalJSON []byte, partialMap map[string]any, parent string, existingProviders map[string]SecretProvider) ([]byte, error) {
	modified := make(map[string]any)
	hasChanges := false

	for k, v := range partialMap {
		fullPath := k
		if parent != "" {
			fullPath = parent + "." + k
		}

		if sensitiveKeyPaths[fullPath] {
			strVal, isStr := v.(string)
			existing, hasExisting := existingProviders[fullPath]

			// Try keyring first; fall back to plaintext.
			providerType := "keyring"
			account := fullPath
			service := keyringService

			if hasExisting {
				if existing.Account != "" {
					account = existing.Account
				}
				if existing.Service != "" {
					service = existing.Service
				}
			}

			if isStr {
				if err := keyring.Set(service, account, strVal); err != nil {
					// Keyring unavailable - keep value in YAML as plaintext.
					slog.Warn("keyring unavailable, storing secret as plaintext in config", "key", fullPath, "error", err)
					providerType = "plaintext"
					modified[k] = strVal
				} else {
					slog.Debug("stored secret in keyring", "key", fullPath)
					modified[k] = ""
				}
			} else {
				// Non-string value - keep existing provider type or default to keyring.
				if hasExisting && existing.Type != "" {
					providerType = existing.Type
				}
				modified[k] = ""
			}

			modified[k+"_provider"] = map[string]any{
				"type":    providerType,
				"account": account,
				"service": service,
			}
			hasChanges = true
			continue
		}

		// Recurse into nested maps.
		if nested, ok := v.(map[string]any); ok {
			nestedJSON, err := json.Marshal(nested)
			if err != nil {
				modified[k] = v
				continue
			}
			result, err := routeSecretsToKeyring(nestedJSON, nested, fullPath, existingProviders)
			if err != nil {
				return nil, err
			}
			var resultMap map[string]any
			if err := json.Unmarshal(result, &resultMap); err != nil {
				modified[k] = v
				continue
			}
			modified[k] = resultMap
			hasChanges = true
			continue
		}

		modified[k] = v
	}

	if !hasChanges {
		return originalJSON, nil
	}

	out, err := json.Marshal(modified)
	if err != nil {
		return originalJSON, nil
	}
	return out, nil
}

// loadExistingProviders reads the current config file and extracts
// SecretProvider metadata for all sensitive keys.
func loadExistingProviders(path string) map[string]SecretProvider {
	providers := make(map[string]SecretProvider)

	data, err := os.ReadFile(path)
	if err != nil {
		return providers
	}

	var cfg rawConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return providers
	}

	providers["email.smtp.password"] = cfg.Email.SMTP.PasswordProvider
	providers["email.ses.access_key_id"] = cfg.Email.SES.AccessKeyIDProvider
	providers["email.ses.secret_access_key"] = cfg.Email.SES.SecretAccessKeyProvider

	return providers
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
// Sensitive fields with _provider.type == "keyring" are loaded from the OS keyring.
func Load(path string) (*Config, error) {
	// Ensure parent directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

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
		App:   raw.App,
		Email: raw.Email,
		Worker: WorkerConfig{
			Concurrency:      raw.Worker.Concurrency,
			MaxRetries:       raw.Worker.MaxRetries,
			RetryBackoffBase: backoffBase,
			RetryBackoffMax:  backoffMax,
		},
		Log: raw.Log,
	}

	// Load secrets from keyring where provider.type == "keyring".
	if err := loadSecretsFromKeyring(cfg); err != nil {
		slog.Warn("failed to load some secrets from keyring", "error", err)
	}

	// Try to migrate plaintext secrets to keyring (e.g., after keyring becomes available).
	tryMigrateSecretsToKeyring(cfg, path)

	// Apply defaults.
	if cfg.App.Theme == "" {
		cfg.App.Theme = "dark"
	}
	if cfg.App.Language == "" {
		cfg.App.Language = "en"
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

// loadSecretsFromKeyring checks each sensitive field's _provider and loads
// the actual value from the OS keyring when type == "keyring".
func loadSecretsFromKeyring(cfg *Config) error {
	// SMTP password.
	if cfg.Email.SMTP.PasswordProvider.Type == "keyring" {
		p := &cfg.Email.SMTP.PasswordProvider
		service, account := resolveProvider(p.Service, p.Account, "email.smtp.password")
		val, err := keyring.Get(service, account)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("keyring get email.smtp.password: %w", err)
		}
		if err == nil {
			cfg.Email.SMTP.Password = val
		}
		p.Service, p.Account = service, account
	}

	// SES access key ID.
	if cfg.Email.SES.AccessKeyIDProvider.Type == "keyring" {
		p := &cfg.Email.SES.AccessKeyIDProvider
		service, account := resolveProvider(p.Service, p.Account, "email.ses.access_key_id")
		val, err := keyring.Get(service, account)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("keyring get email.ses.access_key_id: %w", err)
		}
		if err == nil {
			cfg.Email.SES.AccessKeyID = val
		}
		p.Service, p.Account = service, account
	}

	// SES secret access key.
	if cfg.Email.SES.SecretAccessKeyProvider.Type == "keyring" {
		p := &cfg.Email.SES.SecretAccessKeyProvider
		service, account := resolveProvider(p.Service, p.Account, "email.ses.secret_access_key")
		val, err := keyring.Get(service, account)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("keyring get email.ses.secret_access_key: %w", err)
		}
		if err == nil {
			cfg.Email.SES.SecretAccessKey = val
		}
		p.Service, p.Account = service, account
	}

	return nil
}

// resolveProvider returns the service and account to use for a keyring operation.
// Values from config take precedence; empty ones fall back to defaults.
func resolveProvider(service, account, defaultAccount string) (string, string) {
	if service == "" {
		service = keyringService
	}
	if account == "" {
		account = defaultAccount
	}
	return service, account
}

// tryMigrateSecretsToKeyring attempts to move plaintext secrets into the OS keyring.
// Called at startup - if keyring was previously unavailable but is now working,
// secrets stored as plaintext in YAML are migrated to keyring and stripped from the file.
func tryMigrateSecretsToKeyring(cfg *Config, path string) {
	type secretField struct {
		value    string
		provider *SecretProvider
		key      string
	}

	fields := []secretField{
		{cfg.Email.SMTP.Password, &cfg.Email.SMTP.PasswordProvider, "email.smtp.password"},
		{cfg.Email.SES.AccessKeyID, &cfg.Email.SES.AccessKeyIDProvider, "email.ses.access_key_id"},
		{cfg.Email.SES.SecretAccessKey, &cfg.Email.SES.SecretAccessKeyProvider, "email.ses.secret_access_key"},
	}

	needsRewrite := false
	for _, f := range fields {
		if f.value == "" || f.provider.Type == "keyring" {
			continue
		}
		service, account := resolveProvider(f.provider.Service, f.provider.Account, f.key)
		if err := keyring.Set(service, account, f.value); err != nil {
			slog.Debug("keyring migration not available, keeping plaintext", "key", f.key)
			continue
		}
		f.provider.Type = "keyring"
		f.provider.Account = account
		f.provider.Service = service
		slog.Info("migrated secret from plaintext to keyring", "key", f.key)
		needsRewrite = true
	}

	if !needsRewrite {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("failed to read config for migration rewrite", "error", err)
		return
	}
	var current yaml.Node
	if err := yaml.Unmarshal(data, &current); err != nil {
		slog.Warn("failed to parse config for migration rewrite", "error", err)
		return
	}
	if current.Kind == 0 || len(current.Content) == 0 {
		return
	}

	partial := make(map[string]any)
	for _, f := range fields {
		if f.provider.Type != "keyring" || f.value == "" {
			continue
		}
		parts := splitKey(f.key)
		buildNestedPartial(partial, parts, f.provider)
	}

	if len(partial) == 0 {
		return
	}

	partialJSON, err := json.Marshal(partial)
	if err != nil {
		return
	}
	partialNode := jsonToYAMLNode(partialJSON)
	if partialNode == nil {
		return
	}
	mergeNode(current.Content[0], partialNode, "")

	out, err := yaml.Marshal(current.Content[0])
	if err != nil {
		return
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		slog.Warn("failed to write migrated config", "error", err)
		return
	}
	slog.Info("config file updated after keyring migration")
}

// splitKey splits "email.smtp.password" into ["email", "smtp", "password"].
func splitKey(key string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}

// buildNestedPartial builds a nested map from dot-separated key parts.
func buildNestedPartial(m map[string]any, parts []string, provider *SecretProvider) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		m[parts[0]] = ""
		m[parts[0]+"_provider"] = map[string]any{
			"type":    provider.Type,
			"account": provider.Account,
			"service": provider.Service,
		}
		return
	}
	child, ok := m[parts[0]]
	if !ok {
		child = make(map[string]any)
		m[parts[0]] = child
	}
	childMap, ok := child.(map[string]any)
	if !ok {
		childMap = make(map[string]any)
		m[parts[0]] = childMap
	}
	buildNestedPartial(childMap, parts[1:], provider)
}
