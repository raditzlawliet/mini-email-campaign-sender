package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	t.Run("missing file", func(t *testing.T) {
		_, err := Load("/nonexistent/config.yaml")
		assert.Error(t, err)
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

		// Simulate saving theme — adds app key
		partial, _ := json.Marshal(map[string]any{
			"app": map[string]any{"theme": "dark"},
		})
		err = SavePartial(path, partial)
		require.NoError(err)

		data, err := os.ReadFile(path)
		require.NoError(err)
		content := string(data)

		t.Logf("output:\n%s", content)

		// Verify canonical order: app → server → email → worker
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

	t.Run("missing file", func(t *testing.T) {
		partial, _ := json.Marshal(map[string]any{"app": map[string]any{"theme": "dark"}})
		err := SavePartial("/nonexistent/config.yaml", partial)
		assert.Error(t, err)
	})
}
