package store

import (
	"context"
	"log/slog"
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
	StatePaused    CampaignState = "paused"
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

// LogEntry is a single campaign log event.
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"` // info, warn, error
	Message string `json:"message"`
}

// Store is a thread-safe in-memory store for campaign data.
type Store struct {
	mu sync.RWMutex

	recipients []Recipient
	statuses   []RecipientStatus
	template   Template
	config     CampaignConfig
	state      CampaignState
	events     []LogEntry
	cancelFn   context.CancelFunc
	verbose    bool
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
			events:     []LogEntry{},
			state:      StateIdle,
		}
	})
}

// SetVerbose enables or disables verbose (debug-level) event logging.
func (s *Store) SetVerbose(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verbose = v
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

// SetCancelFn stores the cancel function for the current campaign context.
func (s *Store) SetCancelFn(fn context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelFn = fn
}

// Pause cancels the current campaign context and sets state to paused.
func (s *Store) Pause() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
	s.state = StatePaused
}

// IsRunning returns true if campaign is currently running.
func (s *Store) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == StateRunning
}

// IsPaused returns true if campaign is paused.
func (s *Store) IsPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state == StatePaused
}

// AddEvent appends a log event (in-memory only, for frontend).
// Prefer LogAndEvent for events that should also appear in console.
func (s *Store) AddEvent(level, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addEventLocked(level, message)
}

// LogAndEvent logs to both slog (console) and in-memory events (frontend).
func (s *Store) LogAndEvent(level, message string) {
	switch level {
	case "error":
		slog.Error(message)
	case "warn":
		slog.Warn(message)
	case "debug":
		slog.Debug(message)
		if !s.verbose {
			return
		}
	default:
		slog.Info(message)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addEventLocked(level, message)
}

func (s *Store) addEventLocked(level, message string) {
	s.events = append(s.events, LogEntry{
		Time:    time.Now().Format(time.RFC3339),
		Level:   level,
		Message: message,
	})
	if len(s.events) > 500 {
		s.events = s.events[len(s.events)-500:]
	}
}

// ClearEvents removes all in-memory log events.
func (s *Store) ClearEvents() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = []LogEntry{}
}

// GetEvents returns a copy of all log events.
func (s *Store) GetEvents() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]LogEntry, len(s.events))
	copy(result, s.events)
	return result
}

// Reset clears all stored data and resets state to idle.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recipients = []Recipient{}
	s.statuses = []RecipientStatus{}
	s.template = Template{}
	s.config = CampaignConfig{}
	s.events = []LogEntry{}
	s.state = StateIdle
	if s.cancelFn != nil {
		s.cancelFn()
		s.cancelFn = nil
	}
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
