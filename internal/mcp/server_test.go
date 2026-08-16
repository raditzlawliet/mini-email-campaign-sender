package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/raditzlawliet/test-mass-email/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCSV = "name,email\nalice,alice@example.com\nbob,bob@example.com\n"

func testConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{Theme: "dark", Language: "en"},
		Email: config.EmailConfig{
			Provider: "smtp",
			From:     "sender@example.com",
			SMTP:     config.SMTPConfig{Host: "localhost", Port: 1025, Username: "user", Password: "secret-pw", BatchSize: 50},
		},
		Worker: config.WorkerConfig{
			Concurrency:      10,
			MaxRetries:       3,
			RetryBackoffBase: time.Second,
			RetryBackoffMax:  30 * time.Second,
		},
		Log: config.LogConfig{Campaign: config.CampaignLogConfig{LogToFile: false, Verbose: false}},
		MCP: config.MCPConfig{Enabled: true, Host: "127.0.0.1", Port: 18799},
	}
}

func newTestServer(t *testing.T) (*Server, *store.Store) {
	t.Helper()
	store.InitStore()
	st := store.GetStore()
	st.Reset()
	cfg := testConfig()
	s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)
	return s, st
}

// text extracts the text content of a tool result.
func text(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, res)
	require.Len(t, res.Content, 1)
	tc, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected TextContent, got %T", res.Content[0])
	return tc.Text
}

func TestPrepareCampaign(t *testing.T) {
	t.Run("stages csv, template, and config", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)

		tls := true
		res, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText:    testCSV,
			Subject:    "Hello {name}",
			Body:       "Hi {name}",
			To:         "{email}",
			From:       "ai@example.com",
			Provider:   "smtp",
			SMTPHost:   "smtp.test.com",
			SMTPTLS:    &tls,
			MaxRetries: 5,
		})
		require.NoError(t, err)

		assert.Contains(text(t, res), `"recipients": 2`)
		assert.Equal("Hello {name}", st.GetTemplate().Subject)
		assert.Equal("ai@example.com", st.GetConfig().From)
		assert.Equal("smtp.test.com", st.GetConfig().SMTP.Host)
		assert.True(st.GetConfig().SMTP.TLS)
		assert.Equal(5, st.GetConfig().Worker.MaxRetries)
		assert.Equal(testCSV, st.GetCSVText())
		assert.Equal(store.StateReady, st.GetState())
	})

	t.Run("stages csv from file path", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)

		tmp := t.TempDir()
		path := filepath.Join(tmp, "recipients.csv")
		require.NoError(t, os.WriteFile(path, []byte(testCSV), 0644))

		res, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVPath: path,
		})
		require.NoError(t, err)
		assert.Contains(text(t, res), `"recipients": 2`)
		assert.Equal(testCSV, st.GetCSVText())
	})

	t.Run("rejects invalid csv", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText: "no-email-column\nfoo\n",
		})
		require.Error(t, err)
		assert.Contains(err.Error(), "invalid CSV")
	})

	t.Run("rejects while running", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)
		st.SetCSV([]store.Recipient{{Index: 0, Data: map[string]string{"email": "a@b.c"}, Email: "a@b.c"}})
		st.StartCampaign()

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText: testCSV,
		})
		require.Error(t, err)
		assert.Contains(err.Error(), "running")
	})

	t.Run("smtp_batch_size is a campaign-level override", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText:       testCSV,
			SMTPBatchSize: 1,
		})
		require.NoError(t, err)

		cfg := st.GetConfig()
		assert.Equal(1, cfg.SMTP.BatchSize)
		assert.Equal(1, cfg.SmtpBatchSize)

		// The resolved request used at send time must carry the override.
		req, err := s.buildCampaignRequest(true)
		require.NoError(t, err)
		assert.Equal(1, req.SmtpBatchSize)
		assert.Equal(1, req.SMTP.BatchSize)
	})

	t.Run("partial update keeps existing staging", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText: testCSV,
			Subject: "Hello {name}",
		})
		require.NoError(t, err)

		// Update only the body; subject and csv must survive.
		_, _, err = s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			Body: "New body",
		})
		require.NoError(t, err)

		assert.Equal("Hello {name}", st.GetTemplate().Subject)
		assert.Equal("New body", st.GetTemplate().Body)
		assert.Equal(testCSV, st.GetCSVText())
	})
}

func TestGetCurrentCampaign(t *testing.T) {
	t.Run("returns state, progress, and masked config", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)

		st.SetCSV([]store.Recipient{
			{Index: 0, Data: map[string]string{"email": "a@b.c"}, Email: "a@b.c"},
			{Index: 1, Data: map[string]string{"email": "d@e.f"}, Email: "d@e.f"},
		})
		st.SetTemplate(store.Template{Subject: "S", Body: "B", To: "T"})
		st.SetConfig(store.CampaignConfig{
			From:     "x@y.z",
			Provider: "smtp",
			SMTP:     config.SMTPConfig{Host: "h", Password: "pw"},
		})
		st.UpdateStatus(0, store.RecipientStatus{Status: "sent"})

		res, _, err := s.GetCurrentCampaign(context.Background(), &mcp.CallToolRequest{}, GetCampaignParams{})
		require.NoError(t, err)
		out := text(t, res)
		assert.Contains(out, `"sent": 1`)
		assert.Contains(out, `"password": "****"`)
		assert.NotContains(out, "pw")
	})

	t.Run("events only when requested", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)
		st.AddEvent("info", "hello")

		res, _, err := s.GetCurrentCampaign(context.Background(), &mcp.CallToolRequest{}, GetCampaignParams{})
		require.NoError(t, err)
		assert.NotContains(text(t, res), "hello")

		res, _, err = s.GetCurrentCampaign(context.Background(), &mcp.CallToolRequest{}, GetCampaignParams{IncludeEvents: true})
		require.NoError(t, err)
		assert.Contains(text(t, res), "hello")
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("returns masked global config", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		res, _, err := s.GetConfig(context.Background(), &mcp.CallToolRequest{}, GetConfigParams{})
		require.NoError(t, err)
		out := text(t, res)
		assert.Contains(out, `"provider": "smtp"`)
		assert.Contains(out, `"password": "****"`)
		assert.NotContains(out, "secret-pw")
	})

	t.Run("filters by section", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		res, _, err := s.GetConfig(context.Background(), &mcp.CallToolRequest{}, GetConfigParams{Section: "mcp"})
		require.NoError(t, err)
		out := text(t, res)
		assert.Contains(out, `"port": 18799`)
		assert.NotContains(out, "worker")

		_, _, err = s.GetConfig(context.Background(), &mcp.CallToolRequest{}, GetConfigParams{Section: "bogus"})
		require.Error(t, err)
	})
}

func TestSetConfig(t *testing.T) {
	t.Run("saves partial and triggers reload", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		reloaded := false

		tmp := t.TempDir()
		path := filepath.Join(tmp, "config.yaml")
		require.NoError(os.WriteFile(path, []byte("mcp:\n  enabled: true\n  port: 18799\n"), 0644))

		s := NewServer(st, func() *config.Config { return cfg }, path, "test", func() error {
			reloaded = true
			return nil
		})

		res, _, err := s.SetConfig(context.Background(), &mcp.CallToolRequest{}, SetConfigParams{
			PartialJSON: `{"mcp":{"port":19000}}`,
		})
		require.NoError(err)
		assert.Contains(text(t, res), "saved")
		assert.True(reloaded)
	})

	t.Run("requires partial_json", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.SetConfig(context.Background(), &mcp.CallToolRequest{}, SetConfigParams{})
		require.Error(t, err)
		assert.Contains(err.Error(), "partial_json")
	})

	t.Run("rejects non-loopback mcp host", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.SetConfig(context.Background(), &mcp.CallToolRequest{}, SetConfigParams{
			PartialJSON: `{"mcp":{"host":"0.0.0.0"}}`,
		})
		require.Error(t, err)
		assert.Contains(err.Error(), "loopback")

		// Loopback host changes pass the guard (then fail on the missing
		// config file - which is fine, it proves the host check let it through).
		_, _, err = s.SetConfig(context.Background(), &mcp.CallToolRequest{}, SetConfigParams{
			PartialJSON: `{"mcp":{"host":"127.0.0.1"}}`,
		})
		require.Error(t, err)
		assert.NotContains(err.Error(), "loopback")
	})
}

func TestDryRunCampaign(t *testing.T) {
	t.Run("renders previews from prepared csv", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText: testCSV,
			Subject: "Hi {name}",
			Body:    "Hello {name}",
			To:      "{email}",
		})
		require.NoError(t, err)

		res, _, err := s.DryRunCampaign(context.Background(), &mcp.CallToolRequest{}, DryRunParams{})
		require.NoError(t, err)
		out := text(t, res)
		assert.Contains(out, "Hi alice")
		assert.Contains(out, "alice@example.com")
	})

	t.Run("errors when nothing prepared", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.DryRunCampaign(context.Background(), &mcp.CallToolRequest{}, DryRunParams{})
		require.Error(t, err)
		assert.Contains(err.Error(), "prepare_campaign")
	})
}

func TestStartCampaignGuards(t *testing.T) {
	t.Run("errors when nothing prepared", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.StartCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
		assert.Contains(err.Error(), "prepare_campaign")
	})

	t.Run("errors when template incomplete", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText: testCSV,
		})
		require.NoError(t, err)

		_, _, err = s.StartCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
		assert.Contains(err.Error(), "subject, body, and to")
	})

	t.Run("errors when already running", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)
		st.StartCampaign()

		_, _, err := s.StartCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
		assert.Contains(err.Error(), "already running")
	})
}

func TestPauseResumeClear(t *testing.T) {
	t.Run("pause requires running", func(t *testing.T) {
		s, _ := newTestServer(t)

		_, _, err := s.PauseCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
	})

	t.Run("double pause is rejected", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)
		st.SetCSV([]store.Recipient{{Index: 0, Data: map[string]string{"email": "a@b.c"}, Email: "a@b.c"}})
		st.StartCampaign()

		res, _, err := s.PauseCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.NoError(t, err)
		assert.Contains(text(t, res), `"state": "paused"`)

		// Pausing an already-paused campaign is rejected.
		_, _, err = s.PauseCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
	})

	t.Run("resume requires paused", func(t *testing.T) {
		assert := assert.New(t)
		s, _ := newTestServer(t)

		_, _, err := s.ResumeCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
		assert.Contains(err.Error(), "not paused")
	})

	t.Run("clear resets state", func(t *testing.T) {
		assert := assert.New(t)
		s, st := newTestServer(t)

		_, _, err := s.PrepareCampaign(context.Background(), &mcp.CallToolRequest{}, PrepareCampaignParams{
			CSVText: testCSV,
		})
		require.NoError(t, err)

		res, _, err := s.ClearCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.NoError(t, err)
		assert.Contains(text(t, res), `"state": "idle"`)
		assert.Empty(st.GetRecipients())
		assert.Empty(st.GetCSVText())
	})

	t.Run("clear rejected while running", func(t *testing.T) {
		s, st := newTestServer(t)
		st.SetCSV([]store.Recipient{{Index: 0, Data: map[string]string{"email": "a@b.c"}, Email: "a@b.c"}})
		st.StartCampaign()

		_, _, err := s.ClearCampaign(context.Background(), &mcp.CallToolRequest{}, struct{}{})
		require.Error(t, err)
	})
}

func TestRedactionHelpers(t *testing.T) {
	t.Run("mask hides non-empty secrets", func(t *testing.T) {
		assert := assert.New(t)
		assert.Equal("****", mask("secret"))
		assert.Equal("", mask(""))
	})

	t.Run("campaignConfigToMap masks smtp password", func(t *testing.T) {
		assert := assert.New(t)
		out := campaignConfigToMap(store.CampaignConfig{
			SMTP: config.SMTPConfig{Password: "hunter2"},
			SES:  config.SESConfig{AccessKeyID: "AKIA", SecretAccessKey: "sk"},
		})
		b, _ := json.Marshal(out)
		assert.Contains(string(b), `"password":"****"`)
		assert.NotContains(string(b), "hunter2")
	})
}

func TestServerLifecycle(t *testing.T) {
	t.Run("start, health, reconcile stop", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		cfg.MCP.Port = 0 // random free port
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		require.NoError(t, s.Start())
		assert.True(s.Running())

		addr := s.ListenAddr()
		require.NotEmpty(t, addr)

		resp, err := http.Get("http://" + addr + "/mcp/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(http.StatusOK, resp.StatusCode)
		assert.Contains(string(body), "\"status\":\"ok\"")

		// Disable -> Reconcile stops the listener.
		cfg.MCP.Enabled = false
		s.Reconcile()
		require.Eventually(t, func() bool { return !s.Running() }, 3*time.Second, 20*time.Millisecond)

		s.Stop()
	})

	t.Run("rejects unauthorized when token set", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		cfg.MCP.Port = 0
		cfg.MCP.Token = "s3cret"
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		require.NoError(t, s.Start())
		defer s.Stop()

		addr := s.ListenAddr()
		require.NotEmpty(t, addr)

		resp, err := http.Get("http://" + addr + "/mcp")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(http.StatusUnauthorized, resp.StatusCode)

		req, _ := http.NewRequest(http.MethodGet, "http://"+addr+"/mcp", nil)
		req.Header.Set("Authorization", "Bearer s3cret")
		resp2, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.NotEqual(http.StatusUnauthorized, resp2.StatusCode)
	})

	t.Run("serves tools over json-rpc", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		cfg.MCP.Port = 0
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		require.NoError(t, s.Start())
		defer s.Stop()
		base := "http://" + s.ListenAddr() + "/mcp"

		// initialize + tools/list in a single batch (each POST is a fresh
		// ephemeral session, so both must arrive together)
		batch := []map[string]any{
			{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"params": map[string]any{
					"protocolVersion": "2025-06-18",
					"capabilities":    map[string]any{},
					"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
				},
			},
			{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "tools/list",
				"params":  map[string]any{},
			},
		}
		respBody := rpcBatch(t, base, batch)
		listResp, ok := respBody["2"].(map[string]any)
		require.True(t, ok, "second batch element should be tools/list result, got %v", respBody)
		assert.Empty(listResp["error"])

		result, ok := listResp["result"].(map[string]any)
		require.True(t, ok)
		tools, ok := result["tools"].([]any)
		require.True(t, ok)

		names := map[string]bool{}
		for _, tl := range tools {
			name, _ := tl.(map[string]any)["name"].(string)
			names[name] = true
		}
		for _, want := range []string{
			"mecs_prepare_campaign", "mecs_get_current_campaign", "mecs_set_config",
			"mecs_get_config", "mecs_dry_run_campaign", "mecs_start_campaign",
			"mecs_pause_campaign", "mecs_resume_campaign", "mecs_clear_campaign",
		} {
			assert.True(names[want], "missing tool %s (got %v)", want, names)
		}
	})
}

func TestServerStatus(t *testing.T) {
	t.Run("stopped when disabled and not started", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		cfg.MCP.Enabled = false
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		status, addr, errMsg := s.Status()
		assert.Equal("stopped", status)
		assert.Empty(addr)
		assert.Empty(errMsg)
	})

	t.Run("starting when enabled but not bound yet", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		cfg.MCP.Port = 0
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		status, _, _ := s.Status()
		assert.Equal("starting", status)
	})

	t.Run("running reports the bound address", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()
		cfg := testConfig()
		cfg.MCP.Port = 0
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		require.NoError(t, s.Start())
		defer s.Stop()

		status, addr, errMsg := s.Status()
		assert.Equal("running", status)
		assert.Contains(addr, "127.0.0.1:")
		assert.Empty(errMsg)
	})

	t.Run("error when the port is taken", func(t *testing.T) {
		assert := assert.New(t)
		store.InitStore()
		st := store.GetStore()
		st.Reset()

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer ln.Close()
		port := ln.Addr().(*net.TCPAddr).Port

		cfg := testConfig()
		cfg.MCP.Port = port
		s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)

		require.Error(t, s.Start())
		status, addr, errMsg := s.Status()
		assert.Equal("error", status)
		assert.Empty(addr)
		assert.NotEmpty(errMsg)
	})
}

// TestCampaignWorkflowOverHTTP drives a realistic agent session over the real
// HTTP endpoint: get_config -> prepare -> dry_run -> inspect -> start -> wait
// for completion -> clear, plus a guard rejection after clearing.
func TestCampaignWorkflowOverHTTP(t *testing.T) {
	store.InitStore()
	st := store.GetStore()
	st.Reset()
	cfg := testConfig()
	cfg.MCP.Port = 0
	s := NewServer(st, func() *config.Config { return cfg }, "", "test", nil)
	require.NoError(t, s.Start())
	defer s.Stop()

	// Reserve an ephemeral loopback port with no listener (sends will fail
	// fast with connection refused, like TestServerStatus).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	smtpPort := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	base := "http://" + s.ListenAddr() + "/mcp"
	t.Logf("connected to MCP server at %s/mcp", s.ListenAddr())

	// 1. get_config - global config, secrets masked
	txt := callText(t, base, 1, 2, "mecs_get_config", map[string]any{})
	t.Logf("get_config -> %s", txt)
	assert.Contains(t, txt, `"provider": "smtp"`)
	assert.Contains(t, txt, `"password": "****"`)
	assert.NotContains(t, txt, "secret-pw")

	// 2. prepare_campaign - partial staging, validated CSV
	txt = callText(t, base, 3, 4, "mecs_prepare_campaign", map[string]any{
		"csv_text":           testCSV,
		"subject":            "Hi {name}",
		"body":               "Hello {name}",
		"to":                 "{email}",
		"from":               "ai@example.com",
		"provider":           "smtp",
		"smtp_host":          "127.0.0.1",
		"smtp_port":          smtpPort,
		"max_retries":        1,
		"retry_backoff_base": "50ms",
		"retry_backoff_max":  "200ms",
	})
	t.Logf("prepare_campaign -> %s", txt)
	assert.Contains(t, txt, `"recipients": 2`)
	assert.Contains(t, txt, `"state": "ready"`)

	// 3. dry_run_campaign - renders sample emails without sending
	txt = callText(t, base, 5, 6, "mecs_dry_run_campaign", map[string]any{"count": 2})
	t.Logf("dry_run_campaign -> %s", txt)
	assert.Contains(t, txt, "Hi alice")
	assert.Contains(t, txt, "alice@example.com")

	// 4. get_current_campaign - staged state + progress
	txt = callText(t, base, 7, 8, "mecs_get_current_campaign", map[string]any{})
	t.Logf("get_current_campaign -> %s", txt)
	assert.Contains(t, txt, `"total": 2`)
	assert.Contains(t, txt, `"from": "ai@example.com"`)

	// 5. start_campaign - launches the worker pool
	txt = callText(t, base, 9, 10, "mecs_start_campaign", map[string]any{})
	t.Logf("start_campaign -> %s", txt)
	assert.Contains(t, txt, `"status": "started"`)

	// 6. campaign completes (sends fail: no SMTP server on the reserved port)
	require.Eventually(t, func() bool {
		return strings.Contains(
			callText(t, base, 11, 12, "mecs_get_current_campaign", map[string]any{}),
			`"state": "completed"`,
		)
	}, 15*time.Second, 250*time.Millisecond)

	// 7. clear_campaign - back to idle
	txt = callText(t, base, 13, 14, "mecs_clear_campaign", map[string]any{})
	t.Logf("clear_campaign -> %s", txt)
	assert.Contains(t, txt, `"state": "idle"`)

	// 8. start after clear - guard rejects with actionable message
	txt = callText(t, base, 15, 16, "mecs_start_campaign", map[string]any{})
	t.Logf("start_campaign (after clear) -> %s", txt)
	assert.Contains(t, txt, "prepare_campaign")
}

// rpcBatch performs a JSON-RPC batch request against the MCP endpoint and
// returns the decoded responses keyed by request id (as strings). Each value
// is a map[string]any JSON-RPC response object.
func rpcBatch(t *testing.T, base string, batch []map[string]any) map[string]any {
	t.Helper()
	body, err := json.Marshal(batch)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, base, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// The handler may respond with JSON or SSE (event: message / data: ...).
	// Collect every data line when it is SSE.
	s := string(data)
	if strings.HasPrefix(s, "event:") || strings.Contains(s, "\ndata:") {
		var lines []string
		for _, line := range strings.Split(s, "\n") {
			if strings.HasPrefix(line, "data:") {
				lines = append(lines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		out := map[string]any{}
		for _, l := range lines {
			var v map[string]any
			require.NoError(t, json.Unmarshal([]byte(l), &v))
			out[fmt.Sprintf("%v", v["id"])] = v
		}
		return out
	}

	var arr []map[string]any
	require.NoError(t, json.Unmarshal([]byte(s), &arr))
	out := map[string]any{}
	for _, v := range arr {
		out[fmt.Sprintf("%v", v["id"])] = v
	}
	return out
}

// mcpBatch builds a JSON-RPC batch that initializes a fresh ephemeral session
// and then calls one tool in the same request.
func mcpBatch(initID, callID int, tool string, args map[string]any) []map[string]any {
	return []map[string]any{
		{
			"jsonrpc": "2.0",
			"id":      initID,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": "2025-06-18",
				"capabilities":    map[string]any{},
				"clientInfo":      map[string]any{"name": "mecs-test-client", "version": "0.0.1"},
			},
		},
		{
			"jsonrpc": "2.0",
			"id":      callID,
			"method":  "tools/call",
			"params":  map[string]any{"name": tool, "arguments": args},
		},
	}
}

// callText runs a batch and returns the text content of the tool call response.
func callText(t *testing.T, base string, initID, callID int, tool string, args map[string]any) string {
	t.Helper()
	resp := rpcBatch(t, base, mcpBatch(initID, callID, tool, args))
	callResp, ok := resp[fmt.Sprintf("%d", callID)].(map[string]any)
	require.True(t, ok, "no response for call id %d: %v", callID, resp)
	require.Empty(t, callResp["error"]) // no JSON-RPC protocol-level error
	result, ok := callResp["result"].(map[string]any)
	require.True(t, ok)
	content, ok := result["content"].([]any)
	require.True(t, ok)
	txt, ok := content[0].(map[string]any)["text"].(string)
	require.True(t, ok)
	return txt
}
