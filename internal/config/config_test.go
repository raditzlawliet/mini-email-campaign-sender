package config

import (
	"os"
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
