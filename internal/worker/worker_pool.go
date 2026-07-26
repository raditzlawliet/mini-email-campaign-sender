package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raditzlawliet/test-mass-email/internal/email"
	"github.com/raditzlawliet/test-mass-email/internal/store"
)

// CampaignLogWriter is the interface for campaign logging.
type CampaignLogWriter interface {
	Log(level, msg string)
	Close()
}

// WorkerPool manages concurrent email sending with retry logic.
type WorkerPool struct {
	Concurrency int
	MaxRetries  int
	BackoffBase time.Duration
	BackoffMax  time.Duration
}

// RunResult holds the outcome of a campaign run.
type RunResult struct {
	Sent   int
	Failed int
}

// Run starts the worker pool to process all recipients.
// factory is called once per worker goroutine to create a dedicated sender instance.
func (wp *WorkerPool) Run(ctx context.Context, factory email.SenderFactory, st *store.Store, logger CampaignLogWriter) RunResult {
	recipients := st.GetRecipients()
	return wp.processRecipients(ctx, factory, st, logger, recipients)
}

// RunPending starts the worker pool to process only pending recipients.
func (wp *WorkerPool) RunPending(ctx context.Context, factory email.SenderFactory, st *store.Store, logger CampaignLogWriter) RunResult {
	all := st.GetRecipients()
	statuses := st.GetAllStatuses()
	var pending []store.Recipient
	for i, r := range all {
		if i < len(statuses) && statuses[i].Status == "pending" {
			pending = append(pending, r)
		}
	}
	return wp.processRecipients(ctx, factory, st, logger, pending)
}

func (wp *WorkerPool) processRecipients(ctx context.Context, factory email.SenderFactory, st *store.Store, logger CampaignLogWriter, recipients []store.Recipient) RunResult {
	if len(recipients) == 0 {
		return RunResult{}
	}

	queue := make(chan store.Recipient, len(recipients))
	for _, r := range recipients {
		queue <- r
	}
	close(queue)

	var wg sync.WaitGroup
	var sentCount, failedCount int64

	for i := 0; i < wp.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			sender, err := factory()
			if err != nil {
				msg := fmt.Sprintf("Worker %d: failed to create sender: %v", workerID, err)
				logger.Log("error", msg)
				st.LogAndEvent("error", msg)
				// Drain queue to prevent deadlock
				for range queue {
					atomic.AddInt64(&failedCount, 1)
				}
				return
			}
			wp.worker(ctx, workerID, queue, sender, st, logger, &sentCount, &failedCount)
		}(i)
	}

	wg.Wait()

	result := RunResult{Sent: int(sentCount), Failed: int(failedCount)}
	msg := fmt.Sprintf("Batch done — %d sent, %d failed", sentCount, failedCount)
	logger.Log("info", msg)
	st.LogAndEvent("info", msg)

	return result
}

func (wp *WorkerPool) worker(
	ctx context.Context, workerID int, queue <-chan store.Recipient,
	sender email.EmailSender, st *store.Store, logger CampaignLogWriter,
	sentCount *int64, failedCount *int64,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case recipient, ok := <-queue:
			if !ok {
				return
			}
			wp.processRecipient(ctx, workerID, recipient, sender, st, logger, sentCount, failedCount)
		}
	}
}

func (wp *WorkerPool) processRecipient(
	ctx context.Context, workerID int, recipient store.Recipient,
	sender email.EmailSender, st *store.Store, logger CampaignLogWriter,
	sentCount *int64, failedCount *int64,
) {
	tmpl := st.GetTemplate()
	toAddr := email.Render(tmpl.To, recipient.Data)
	if toAddr == "" {
		toAddr = recipient.Email
	}
	subject := email.Render(tmpl.Subject, recipient.Data)
	body := email.Render(tmpl.Body, recipient.Data)

	var lastErr error
	attempts := 0

	for attempts < wp.MaxRetries {
		attempts++

		select {
		case <-ctx.Done():
			st.UpdateStatus(recipient.Index, store.RecipientStatus{
				Status:   "pending",
				Attempts: attempts,
			})
			return
		default:
		}

		lastErr = sender.Send(toAddr, subject, body)
		if lastErr == nil {
			now := time.Now()
			st.UpdateStatus(recipient.Index, store.RecipientStatus{
				Status:   "sent",
				Attempts: attempts,
				SentAt:   &now,
			})
			msg := fmt.Sprintf("Sent to %s (%d attempt(s))", recipient.Email, attempts)
			logger.Log("debug", msg)
			st.LogAndEvent("debug", msg)

			atomic.AddInt64(sentCount, 1)
			return
		}

		msg := fmt.Sprintf("Retry %d/%d for %s: %v", attempts, wp.MaxRetries, recipient.Email, lastErr)
		logger.Log("warn", msg)
		st.LogAndEvent("warn", msg)

		if attempts < wp.MaxRetries {
			backoff := wp.BackoffBase * time.Duration(1<<(attempts-1))
			if backoff > wp.BackoffMax {
				backoff = wp.BackoffMax
			}

			select {
			case <-ctx.Done():
				st.UpdateStatus(recipient.Index, store.RecipientStatus{
					Status:   "pending",
					Attempts: attempts,
				})
				return
			case <-time.After(backoff):
			}
		}
	}

	errMsg := fmt.Sprintf("all %d attempts failed: %v", wp.MaxRetries, lastErr)
	st.UpdateStatus(recipient.Index, store.RecipientStatus{
		Status:   "failed",
		Error:    errMsg,
		Attempts: attempts,
	})
	msg := fmt.Sprintf("Failed: %s — %s", recipient.Email, errMsg)
	logger.Log("error", msg)
	st.LogAndEvent("error", msg)

	atomic.AddInt64(failedCount, 1)
}
