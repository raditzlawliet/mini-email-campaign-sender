package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestApp(t *testing.T, cfgYAML string) (*App, string) {
	t.Helper()
	store.InitStore()
	st := store.GetStore()
	st.Reset()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(cfgYAML), 0644))

	cfg, err := config.Load(path)
	require.NoError(t, err)

	return NewApp(cfg, st, path, "test-version"), path
}

func TestGetVersion(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	assert.Equal(t, "test-version", a.GetVersion())
}

func TestGetCampaignConfig(t *testing.T) {
	a, _ := newTestApp(t, `
app:
  theme: cupcake
  language: id
email:
  provider: smtp
  from: test@example.com
`)
	data := a.GetCampaignConfig()

	assert.Equal(t, "cupcake", data["app"].(map[string]any)["theme"])
	assert.Equal(t, "id", data["app"].(map[string]any)["language"])
	assert.Equal(t, "smtp", data["email"].(map[string]any)["provider"])

	camp := data["campaign"].(map[string]any)
	assert.Equal(t, store.StateIdle, camp["state"])
}

const testCSV = "name,email\nAlice,alice@example.com\nBob,bob@example.com\n"

func TestPreviewText(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")

	results, err := a.Preview(CampaignInput{
		CSVText: testCSV,
		Subject: "Hello {name}",
		Body:    "Hi {name}",
		To:      "{name} <{email}>",
		Count:   5,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "Alice <alice@example.com>", results[0].To)
	assert.Equal(t, "Hello Alice", results[0].Subject)
	assert.Equal(t, "Hi Bob", results[1].Body)
}

func TestPreviewFilePath(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")

	csvPath := filepath.Join(t.TempDir(), "recipients.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(testCSV), 0644))

	results, err := a.Preview(CampaignInput{
		CSVFilePath: csvPath,
		Subject:     "Hello {name}",
		Body:        "Hi {name}",
		To:          "{email}",
		Count:       5,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "alice@example.com", results[0].To)
}

func TestPreviewRequiresCSV(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	_, err := a.Preview(CampaignInput{Subject: "x"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "csv data is required")
}

func TestPreviewMissingEmailColumn(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	_, err := a.Preview(CampaignInput{
		CSVText: "name\nAlice\n",
		Count:   5,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email")
}

func TestPreviewInvalidFilePath(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	_, err := a.Preview(CampaignInput{
		CSVFilePath: filepath.Join(t.TempDir(), "missing.csv"),
	})
	assert.Error(t, err)
}

func TestStartCampaignRequiresCSV(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	err := a.StartCampaign(CampaignInput{Subject: "x"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "csv data is required")
}

func TestStartCampaignTwiceRejected(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	st := a.store

	// Simulate an already running campaign.
	st.SetCSV([]store.Recipient{{Index: 0, Data: map[string]string{"name": "A"}, Email: "a@x.com"}})
	st.StartCampaign()

	err := a.StartCampaign(CampaignInput{
		CSVText: testCSV,
		Subject: "Hi",
		Body:    "Hello",
		To:      "{email}",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestPauseResumeErrorsWhenNotRunning(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	assert.Error(t, a.PauseCampaign())
	assert.Error(t, a.ResumeCampaign())
}

func TestResetCampaign(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	a.store.SetCSV([]store.Recipient{{Index: 0, Data: map[string]string{"name": "A"}, Email: "a@x.com"}})
	a.store.SetTemplate(store.Template{Subject: "x"})
	assert.NotEqual(t, store.StateIdle, a.store.GetState())

	require.NoError(t, a.ResetCampaign())
	assert.Equal(t, store.StateIdle, a.store.GetState())
	assert.Empty(t, a.store.GetTemplate().Subject)
}

func TestSaveConfigPartial(t *testing.T) {
	a, path := newTestApp(t, `
app:
  theme: dark
  language: en
server:
  port: 8080
`)
	require.NoError(t, a.SaveConfig(`{"app":{"theme":"cupcake"}}`))

	// In-memory default config reloaded.
	assert.Equal(t, "cupcake", a.defaultConfig.App.Theme)
	assert.Equal(t, "en", a.defaultConfig.App.Language)

	// Persisted to disk with language untouched.
	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "cupcake", cfg.App.Theme)
	assert.Equal(t, "en", cfg.App.Language)
}

func TestSaveConfigEmpty(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	assert.Error(t, a.SaveConfig(""))
}

func TestParseCSVFile(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	csvPath := filepath.Join(t.TempDir(), "recipients.csv")
	require.NoError(t, os.WriteFile(csvPath, []byte(testCSV), 0644))

	data, err := a.ParseCSVFile(csvPath)
	require.NoError(t, err)
	assert.Equal(t, 2, data["count"])
	headers := data["headers"].([]string)
	assert.Contains(t, headers, "name")
	assert.Contains(t, headers, "email")
}

func TestParseCSVFileEmpty(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	_, err := a.ParseCSVFile("")
	assert.Error(t, err)
}

func TestStartupStoresContext(t *testing.T) {
	a, _ := newTestApp(t, "server:\n  port: 8080\n")
	ctx := context.Background()
	a.Startup(ctx)
	assert.Equal(t, ctx, a.ctx)
}
