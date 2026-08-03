package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigPath(t *testing.T) {
	t.Run("CONFIG_PATH env", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "/custom/path/config.yaml")
		assert.Equal(t, "/custom/path/config.yaml", ResolveConfigPath())
	})

	t.Run("local config.yaml exists", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "")
		tmp := t.TempDir()
		// Create config.yaml in temp dir, then run from there.
		oldDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmp))
		defer os.Chdir(oldDir)

		require.NoError(t, os.WriteFile("config.yaml", []byte("server:\n  port: 8080\n"), 0644))
		assert.Equal(t, "config.yaml", ResolveConfigPath())
	})

	t.Run("falls back to ~/.mecs", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "")
		// No ./config.yaml, expect ~/.mecs/config.yaml.
		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".mecs", "config.yaml")
		assert.Equal(t, expected, ResolveConfigPath())
	})
}

func TestLoad(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		err := os.WriteFile(path, []byte(`
server:
  port: 9090
email:
  provider: ses
  from: test@example.com
  smtp:
    host: smtp.test.com
    port: 587
    username: user
    password: pass
    tls: true
  ses:
    region: eu-west-1
    access_key_id: AKIATEST
    secret_access_key: secret
    use_template: true
    template_name: my-template
    batch_size: 40
worker:
  concurrency: 5
  max_retries: 2
  retry_backoff_base: 2s
  retry_backoff_max: 10s
`), 0644)
		require.NoError(err)

		cfg, err := Load(path)
		require.NoError(err)

		assert.Equal(9090, cfg.Server.Port)
		assert.Equal("ses", cfg.Email.Provider)
		assert.Equal("test@example.com", cfg.Email.From)
		assert.Equal("smtp.test.com", cfg.Email.SMTP.Host)
		assert.Equal(587, cfg.Email.SMTP.Port)
		assert.Equal("user", cfg.Email.SMTP.Username)
		assert.Equal("pass", cfg.Email.SMTP.Password)
		assert.True(cfg.Email.SMTP.TLS)
		assert.Equal("eu-west-1", cfg.Email.SES.Region)
		assert.Equal("AKIATEST", cfg.Email.SES.AccessKeyID)
		assert.Equal("secret", cfg.Email.SES.SecretAccessKey)
		assert.True(cfg.Email.SES.UseTemplate)
		assert.Equal("my-template", cfg.Email.SES.TemplateName)
		assert.Equal(40, cfg.Email.SES.BatchSize)
		assert.Equal(5, cfg.Worker.Concurrency)
		assert.Equal(2, cfg.Worker.MaxRetries)
		assert.Equal(2*time.Second, cfg.Worker.RetryBackoffBase)
		assert.Equal(10*time.Second, cfg.Worker.RetryBackoffMax)
	})

	t.Run("defaults when fields missing", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		err := os.WriteFile(path, []byte(`
server:
  port: 8080
`), 0644)
		require.NoError(err)

		cfg, err := Load(path)
		require.NoError(err)

		assert.Equal(8080, cfg.Server.Port)
		assert.Equal("smtp", cfg.Email.Provider)
		assert.Equal(10, cfg.Worker.Concurrency)
		assert.Equal(3, cfg.Worker.MaxRetries)
	})

	t.Run("creates default config when file missing", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := filepath.Join(tmp, "subdir", "config.yaml")

		cfg, err := Load(path)
		require.NoError(err)

		assert.Equal("dark", cfg.App.Theme)
		assert.Equal("en", cfg.App.Language)
		assert.Equal(8080, cfg.Server.Port)
		assert.Equal("smtp", cfg.Email.Provider)
		assert.Equal(10, cfg.Worker.Concurrency)

		// File should exist.
		_, err = os.Stat(path)
		assert.NoError(err)
	})

	t.Run("loads secrets from keyring when provider type is keyring", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		err := os.WriteFile(path, []byte(`
email:
  provider: smtp
  from: test@example.com
  smtp:
    host: localhost
    port: 1025
    username: ""
    password: ""
    password_provider:
      type: keyring
      account: ""
      service: raditzlawliet.mecs
    tls: false
    batch_size: 50
  ses:
    region: us-east-1
    access_key_id: ""
    access_key_id_provider:
      type: keyring
      account: ""
      service: raditzlawliet.mecs
    secret_access_key: ""
    secret_access_key_provider:
      type: keyring
      account: ""
      service: raditzlawliet.mecs
    use_template: false
    template_name: ""
    batch_size: 50
`), 0644)
		require.NoError(err)

		cfg, err := Load(path)
		require.NoError(err)

		// Keyring may or may not be available — don't assert exact values.
		// Just verify the config loaded successfully with correct provider metadata.
		assert.Equal("keyring", cfg.Email.SMTP.PasswordProvider.Type)
		assert.Equal("keyring", cfg.Email.SES.AccessKeyIDProvider.Type)
		assert.Equal("keyring", cfg.Email.SES.SecretAccessKeyProvider.Type)
	})
}

func TestSavePartial(t *testing.T) {
	t.Run("merge theme only", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		err := os.WriteFile(path, []byte(`
app:
  theme: dark
server:
  port: 8080
email:
  provider: smtp
worker:
  concurrency: 10
  max_retries: 3
`), 0644)
		require.NoError(err)

		partial, _ := json.Marshal(map[string]any{
			"app": map[string]any{"theme": "cupcake"},
		})
		err = SavePartial(path, partial)
		require.NoError(err)

		cfg, err := Load(path)
		require.NoError(err)
		assert.Equal("cupcake", cfg.App.Theme)
		assert.Equal(8080, cfg.Server.Port)
		assert.Equal("smtp", cfg.Email.Provider)
	})

	t.Run("merge nested config", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		err := os.WriteFile(path, []byte(`
app:
  theme: dark
email:
  provider: smtp
  from: old@test.com
  smtp:
    host: old.example.com
    port: 587
`), 0644)
		require.NoError(err)

		partial, _ := json.Marshal(map[string]any{
			"email": map[string]any{
				"from": "new@test.com",
				"smtp": map[string]any{
					"host": "new.example.com",
				},
			},
		})
		err = SavePartial(path, partial)
		require.NoError(err)

		cfg, err := Load(path)
		require.NoError(err)
		assert.Equal("new@test.com", cfg.Email.From)
		assert.Equal("new.example.com", cfg.Email.SMTP.Host)
		assert.Equal(587, cfg.Email.SMTP.Port)
		assert.Equal("smtp", cfg.Email.Provider)
		assert.Equal("dark", cfg.App.Theme)
	})

	t.Run("preserves key order with new key insertion", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		original := []byte(`server:
  port: 8080

email:
  provider: smtp
  from: sender@example.com
  smtp:
    host: localhost
    port: 1025

worker:
  concurrency: 10
  max_retries: 3
`)
		err := os.WriteFile(path, original, 0644)
		require.NoError(err)

		partial, _ := json.Marshal(map[string]any{
			"app": map[string]any{"theme": "dark"},
		})
		err = SavePartial(path, partial)
		require.NoError(err)

		data, err := os.ReadFile(path)
		require.NoError(err)
		content := string(data)

		t.Logf("output:\n%s", content)

		appIdx := strings.Index(content, "app:")
		serverIdx := strings.Index(content, "server:")
		emailIdx := strings.Index(content, "email:")
		workerIdx := strings.Index(content, "worker:")

		assert.True(appIdx >= 0, "app key should exist")
		assert.True(serverIdx >= 0, "server key should exist")
		assert.True(emailIdx >= 0, "email key should exist")
		assert.True(workerIdx >= 0, "worker key should exist")
		assert.True(appIdx < serverIdx, "app should come before server")
		assert.True(serverIdx < emailIdx, "server should come before email")
		assert.True(emailIdx < workerIdx, "email should come before worker")
	})

	t.Run("missing file returns error", func(t *testing.T) {
		assert := assert.New(t)

		tmp := t.TempDir()
		path := filepath.Join(tmp, "nonexistent", "config.yaml")

		partial, _ := json.Marshal(map[string]any{"app": map[string]any{"theme": "dark"}})
		err := SavePartial(path, partial)
		assert.Error(err)
	})
}

func TestSavePartialWithSecrets(t *testing.T) {
	t.Run("routes SMTP password to keyring and sets provider metadata", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		original := []byte(`
app:
  theme: dark
email:
  provider: smtp
  from: sender@example.com
  smtp:
    host: localhost
    port: 1025
    username: user
    password: ""
    tls: false
    batch_size: 50
`)
		require.NoError(os.WriteFile(path, original, 0644))

		partial, _ := json.Marshal(map[string]any{
			"email": map[string]any{
				"smtp": map[string]any{
					"password": "my-secret-password",
				},
			},
		})

		// This may fail if keyring is unavailable (e.g., CI). That's OK.
		err := SavePartial(path, partial)
		if err != nil {
			t.Logf("keyring may be unavailable, skipping: %v", err)
			return
		}

		// YAML must not contain the secret.
		raw, _ := os.ReadFile(path)
		assert.NotContains(string(raw), "my-secret-password")
		// Provider metadata must exist in YAML.
		assert.Contains(string(raw), "password_provider:")
		assert.Contains(string(raw), "type: keyring")

		cfg, err := Load(path)
		require.NoError(err)

		// Keyring may or may not be available — value comes from keyring if it is.
		assert.Equal("keyring", cfg.Email.SMTP.PasswordProvider.Type)
		assert.Equal("raditzlawliet.mecs", cfg.Email.SMTP.PasswordProvider.Service)
		assert.Equal("email.smtp.password", cfg.Email.SMTP.PasswordProvider.Account)
	})

	t.Run("routes SES secrets to keyring", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		original := []byte(`
email:
  provider: ses
  from: sender@example.com
  smtp:
    host: localhost
    port: 1025
    username: ""
    password: ""
    tls: false
    batch_size: 50
  ses:
    region: us-east-1
    access_key_id: ""
    secret_access_key: ""
    use_template: false
    template_name: ""
    batch_size: 50
`)
		require.NoError(os.WriteFile(path, original, 0644))

		partial, _ := json.Marshal(map[string]any{
			"email": map[string]any{
				"ses": map[string]any{
					"access_key_id":     "AKIATEST123",
					"secret_access_key": "super-secret",
				},
			},
		})

		err := SavePartial(path, partial)
		if err != nil {
			t.Logf("keyring may be unavailable, skipping: %v", err)
			return
		}

		cfg, err := Load(path)
		require.NoError(err)

		// Keyring may or may not be available.
		assert.Equal("keyring", cfg.Email.SES.AccessKeyIDProvider.Type)
		assert.Equal("raditzlawliet.mecs", cfg.Email.SES.AccessKeyIDProvider.Service)
		assert.Equal("email.ses.access_key_id", cfg.Email.SES.AccessKeyIDProvider.Account)
		assert.Equal("keyring", cfg.Email.SES.SecretAccessKeyProvider.Type)
		assert.Equal("raditzlawliet.mecs", cfg.Email.SES.SecretAccessKeyProvider.Service)
		assert.Equal("email.ses.secret_access_key", cfg.Email.SES.SecretAccessKeyProvider.Account)
	})

	t.Run("empty password value does not overwrite", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		tmp := t.TempDir()
		path := tmp + "/config.yaml"
		original := []byte(`
email:
  provider: smtp
  from: sender@example.com
  smtp:
    host: localhost
    port: 1025
    username: ""
    password: ""
    password_provider:
      type: keyring
      account: ""
      service: raditzlawliet.mecs
    tls: false
    batch_size: 50
`)
		require.NoError(os.WriteFile(path, original, 0644))

		// Send partial with empty password — should not touch keyring or provider.
		partial, _ := json.Marshal(map[string]any{
			"email": map[string]any{
				"smtp": map[string]any{
					"password": "",
				},
			},
		})

		err := SavePartial(path, partial)
		require.NoError(err)

		cfg, err := Load(path)
		require.NoError(err)

		// Password value depends on keyring state; just verify provider preserved.
		assert.Equal("keyring", cfg.Email.SMTP.PasswordProvider.Type)
	})
}
