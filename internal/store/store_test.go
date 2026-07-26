package store

import (
	"testing"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	t.Run("lifecycle", func(t *testing.T) {
		assert := assert.New(t)

		InitStore()
		st := GetStore()
		assert.NotNil(st)
		assert.Equal(StateIdle, st.GetState())

		// Set CSV
		recipients := []Recipient{
			{Index: 0, Data: map[string]string{"name": "Alice", "email": "alice@example.com"}, Email: "alice@example.com"},
			{Index: 1, Data: map[string]string{"name": "Bob", "email": "bob@example.com"}, Email: "bob@example.com"},
		}
		st.SetCSV(recipients)
		assert.Equal(StateReady, st.GetState())

		got := st.GetRecipients()
		assert.Len(got, 2)

		// Set template
		st.SetTemplate(Template{Subject: "Hello {name}", Body: "Hi {name}"})
		tmpl := st.GetTemplate()
		assert.Equal("Hello {name}", tmpl.Subject)

		// Update status
		now := mustTimeParse("2024-01-01T00:00:00Z")
		st.UpdateStatus(0, RecipientStatus{Status: "sent", Attempts: 1, SentAt: &now})
		statuses := st.GetAllStatuses()
		assert.Equal("sent", statuses[0].Status)
		assert.Equal("pending", statuses[1].Status)

		// Progress
		prog := st.GetProgress()
		assert.Equal(2, prog.Total)
		assert.Equal(1, prog.Sent)
		assert.Equal(1, prog.Pending)

		// Reset
		st.Reset()
		assert.Equal(StateIdle, st.GetState())
		assert.Empty(st.GetRecipients())
	})

	t.Run("SetConfig merges overrides", func(t *testing.T) {
		assert := assert.New(t)

		InitStore()
		st := GetStore()
		st.SetConfig(CampaignConfig{
			Provider: "ses",
			Worker:   config.WorkerConfig{Concurrency: 20},
		})

		cfg := st.GetConfig()
		assert.Equal("ses", cfg.Provider)
		assert.Equal(20, cfg.Worker.Concurrency)
	})
}

func mustTimeParse(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
