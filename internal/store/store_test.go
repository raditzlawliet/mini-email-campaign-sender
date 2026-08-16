package store

import (
	"testing"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		// CSV text round-trip
		st.SetCSVText("email\nalice@example.com\n")
		assert.Equal("email\nalice@example.com\n", st.GetCSVText())

		// Reset
		st.Reset()
		assert.Equal(StateIdle, st.GetState())
		assert.Empty(st.GetRecipients())
		assert.Empty(st.GetCSVText())
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

	t.Run("revision bumps on mutations", func(t *testing.T) {
		assert := assert.New(t)

		InitStore()
		st := GetStore()
		rev := st.GetRevision()

		st.SetCSV([]Recipient{{Index: 0, Data: map[string]string{"email": "a@b.c"}, Email: "a@b.c"}})
		assert.Greater(st.GetRevision(), rev)
		rev = st.GetRevision()

		st.SetCSVText("email\na@b.c\n")
		assert.Greater(st.GetRevision(), rev)
		rev = st.GetRevision()

		st.SetTemplate(Template{Subject: "S"})
		assert.Greater(st.GetRevision(), rev)
		rev = st.GetRevision()

		st.SetConfig(CampaignConfig{Provider: "smtp"})
		assert.Greater(st.GetRevision(), rev)
		rev = st.GetRevision()

		st.Reset()
		assert.Greater(st.GetRevision(), rev)
	})

	t.Run("StageCampaign stages atomically with one revision", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		InitStore()
		st := GetStore()
		st.Reset()
		rev := st.GetRevision()

		recipients := []Recipient{{Index: 0, Data: map[string]string{"email": "a@b.c"}, Email: "a@b.c"}}
		csvText := "email\na@b.c\n"
		tmpl := Template{Subject: "S", Body: "B", To: "T"}
		cfg := CampaignConfig{Provider: "smtp", SmtpBatchSize: 1}
		require.NoError(st.StageCampaign(&recipients, &csvText, &tmpl, &cfg))

		assert.Equal(StateReady, st.GetState())
		assert.Len(st.GetRecipients(), 1)
		assert.Equal(csvText, st.GetCSVText())
		assert.Equal("S", st.GetTemplate().Subject)
		assert.Equal(1, st.GetConfig().SmtpBatchSize)
		assert.Equal(rev+1, st.GetRevision())

		// Nil recipients keep existing staging untouched (config-only update),
		// and the revision advances by exactly one again.
		rev = st.GetRevision()
		require.NoError(st.StageCampaign(nil, nil, &tmpl, &cfg))
		assert.Len(st.GetRecipients(), 1)
		assert.Equal(StateReady, st.GetState())
		assert.Equal(rev+1, st.GetRevision())
	})

	t.Run("StageCampaign rejects while running", func(t *testing.T) {
		assert := assert.New(t)

		InitStore()
		st := GetStore()
		st.Reset()
		st.StartCampaign()

		err := st.StageCampaign(nil, nil, nil, nil)
		assert.ErrorIs(err, ErrCampaignRunning)
	})
}

func mustTimeParse(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
