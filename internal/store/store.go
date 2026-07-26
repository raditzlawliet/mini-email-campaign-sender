package store

import (
	"sync"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/config"
)

// CampaignState represents the lifecycle state of a campaign.
type CampaignState string

const (
	StateIdle      CampaignState = "idle"
	StateReady     CampaignState = "ready"
	StateRunning   CampaignState = "running"
	StateCompleted CampaignState = "completed"
)

// Recipient represents a single recipient parsed from the CSV.
type Recipient struct {
	Index int               `json:"index"`
	Data  map[string]string `json:"data"`
	Email string            `json:"email"`
}

// RecipientStatus tracks the delivery status of a single recipient.
type RecipientStatus struct {
	Status   string     `json:"status"` // pending, sent, failed
	Error    string     `json:"error,omitempty"`
	Attempts int        `json:"attempts"`
	SentAt   *time.Time `json:"sent_at,omitempty"`
}

// Template holds the email subject, body, and to-address templates.
type Template struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	To      string `json:"to"`
}

// CampaignConfig holds per-campaign overrides.
type CampaignConfig struct {
	From     string              `json:"from"`
	Provider string              `json:"provider"`
	SMTP     config.SMTPConfig   `json:"smtp"`
	SES      config.SESConfig    `json:"ses"`
	Worker   config.WorkerConfig `json:"worker"`
}

// Store is a thread-safe in-memory store for campaign data.
type Store struct {
	mu sync.RWMutex

	recipients []Recipient
	statuses   []RecipientStatus
	template   Template
	config     CampaignConfig
	state      CampaignState
}

var (
	globalStore *Store
	storeOnce   sync.Once
)

// GetStore returns the global singleton store instance.
func GetStore() *Store {
	return globalStore
}

// InitStore initializes the global singleton store.
func InitStore() {
	storeOnce.Do(func() {
		globalStore = &Store{
			recipients: []Recipient{},
			statuses:   []RecipientStatus{},
			state:      StateIdle,
		}
	})
}

// SetCSV replaces the recipient list from parsed CSV data.
func (s *Store) SetCSV(recipients []Recipient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recipients = recipients
	s.statuses = make([]RecipientStatus, len(recipients))
	for i := range s.statuses {
		s.statuses[i] = RecipientStatus{Status: "pending"}
	}
	s.state = StateReady
}

// SetTemplate stores the email template.
func (s *Store) SetTemplate(t Template) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.template = t
}

// SetConfig stores campaign-level overrides.
func (s *Store) SetConfig(c CampaignConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = c
}

// SetProviderOverride sets only the provider and provider-specific config.
func (s *Store) SetProviderOverride(provider string, smtp config.SMTPConfig, ses config.SESConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Provider = provider
	s.config.SMTP = smtp
	s.config.SES = ses
}

// SetWorkerOverride sets only the worker config override.
func (s *Store) SetWorkerOverride(w config.WorkerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.Worker = w
}

// StartCampaign transitions state to running.
func (s *Store) StartCampaign() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateRunning
}

// FinishCampaign transitions state to completed.
func (s *Store) FinishCampaign() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateCompleted
}

// UpdateStatus updates the status of a recipient by index.
func (s *Store) UpdateStatus(index int, status RecipientStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= 0 && index < len(s.statuses) {
		s.statuses[index] = status
	}
}

// GetAllStatuses returns a copy of all recipient statuses.
func (s *Store) GetAllStatuses() []RecipientStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RecipientStatus, len(s.statuses))
	copy(result, s.statuses)
	return result
}

// GetRecipients returns a copy of all recipients.
func (s *Store) GetRecipients() []Recipient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Recipient, len(s.recipients))
	copy(result, s.recipients)
	return result
}

// GetTemplate returns the stored template.
func (s *Store) GetTemplate() Template {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.template
}

// GetConfig returns the stored campaign config.
func (s *Store) GetConfig() CampaignConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetState returns the current campaign state.
func (s *Store) GetState() CampaignState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Reset clears all stored data and resets state to idle.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recipients = []Recipient{}
	s.statuses = []RecipientStatus{}
	s.template = Template{}
	s.config = CampaignConfig{}
	s.state = StateIdle
}

// Progress holds campaign progress counters.
type Progress struct {
	Total   int           `json:"total"`
	Sent    int           `json:"sent"`
	Failed  int           `json:"failed"`
	Pending int           `json:"pending"`
	State   CampaignState `json:"state"`
}

// GetProgress computes current campaign progress.
func (s *Store) GetProgress() Progress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p := Progress{
		Total: len(s.statuses),
		State: s.state,
	}
	for _, st := range s.statuses {
		switch st.Status {
		case "sent":
			p.Sent++
		case "failed":
			p.Failed++
		case "pending":
			p.Pending++
		}
	}
	return p
}
