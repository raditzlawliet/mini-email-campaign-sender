package worker

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raditzlawliet/test-mass-email/internal/store"
)

type mockSender struct {
	mu        sync.Mutex
	sentTo    []string
	callCount map[string]int
	failsFor  map[string]int // number of times to fail per recipient before succeeding
	delay     time.Duration  // optional per-call delay
}

func newMockSender() *mockSender {
	return &mockSender{
		callCount: map[string]int{},
		failsFor:  map[string]int{},
	}
}

func (m *mockSender) setFailsFor(email string, count int) {
	m.failsFor[email] = count
}

func (m *mockSender) Send(to string, subject string, body string) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount[to]++
	if m.callCount[to] <= m.failsFor[to] {
		return fmt.Errorf("simulated failure for %s (attempt %d)", to, m.callCount[to])
	}
	m.sentTo = append(m.sentTo, to)
	return nil
}

type mockLogger struct{}

func (m *mockLogger) Log(level, msg string) {}
func (m *mockLogger) Close()                {}

func TestWorkerPool_Run(t *testing.T) {
	t.Run("sends all emails successfully", func(t *testing.T) {
		assert := assert.New(t)

		store.InitStore()
		st := store.GetStore()

		recipients := []store.Recipient{
			{Index: 0, Data: map[string]string{"name": "A"}, Email: "a@example.com"},
			{Index: 1, Data: map[string]string{"name": "B"}, Email: "b@example.com"},
			{Index: 2, Data: map[string]string{"name": "C"}, Email: "c@example.com"},
		}
		st.SetCSV(recipients)
		st.SetTemplate(store.Template{Subject: "Hi {name}", Body: "Hello {name}"})

		sender := newMockSender()
		wp := &WorkerPool{
			Concurrency: 2,
			MaxRetries:  2,
			BackoffBase: 10 * time.Millisecond,
			BackoffMax:  100 * time.Millisecond,
		}

		result := wp.Run(context.Background(), sender, st, &mockLogger{})

		assert.Equal(3, result.Sent)
		assert.Equal(0, result.Failed)
		assert.Len(sender.sentTo, 3)

		prog := st.GetProgress()
		assert.Equal(3, prog.Sent)
		assert.Equal(0, prog.Failed)
	})

	t.Run("retries on failure then succeeds", func(t *testing.T) {
		assert := assert.New(t)

		store.InitStore()
		st := store.GetStore()

		recipients := []store.Recipient{
			{Index: 0, Data: map[string]string{"name": "A"}, Email: "a@example.com"},
		}
		st.SetCSV(recipients)
		st.SetTemplate(store.Template{Subject: "Hi", Body: "Body"})

		sender := newMockSender()
		sender.setFailsFor("a@example.com", 1) // fails first attempt, succeeds on retry
		wp := &WorkerPool{
			Concurrency: 1,
			MaxRetries:  3,
			BackoffBase: 5 * time.Millisecond,
			BackoffMax:  50 * time.Millisecond,
		}

		result := wp.Run(context.Background(), sender, st, &mockLogger{})

		assert.Equal(1, result.Sent)
		assert.Equal(0, result.Failed)
		assert.Equal(2, sender.callCount["a@example.com"]) // 1 fail + 1 success

		statuses := st.GetAllStatuses()
		assert.Equal("sent", statuses[0].Status)
		assert.Equal(2, statuses[0].Attempts)
	})

	t.Run("marks failed after exhausting retries", func(t *testing.T) {
		assert := assert.New(t)

		store.InitStore()
		st := store.GetStore()

		recipients := []store.Recipient{
			{Index: 0, Data: map[string]string{"name": "A"}, Email: "a@example.com"},
		}
		st.SetCSV(recipients)
		st.SetTemplate(store.Template{Subject: "Hi", Body: "Body"})

		sender := newMockSender()
		sender.setFailsFor("a@example.com", 999) // always fails
		wp := &WorkerPool{
			Concurrency: 1,
			MaxRetries:  2,
			BackoffBase: 5 * time.Millisecond,
			BackoffMax:  20 * time.Millisecond,
		}

		result := wp.Run(context.Background(), sender, st, &mockLogger{})

		assert.Equal(0, result.Sent)
		assert.Equal(1, result.Failed)

		statuses := st.GetAllStatuses()
		assert.Equal("failed", statuses[0].Status)
		assert.Equal(2, statuses[0].Attempts)
	})

	t.Run("context cancellation stops processing", func(t *testing.T) {
		require := require.New(t)

		store.InitStore()
		st := store.GetStore()

		recipients := make([]store.Recipient, 100)
		for i := range recipients {
			recipients[i] = store.Recipient{
				Index: i,
				Data:  map[string]string{"name": fmt.Sprintf("User%d", i)},
				Email: fmt.Sprintf("user%d@example.com", i),
			}
		}
		st.SetCSV(recipients)
		st.SetTemplate(store.Template{Subject: "Hi", Body: "Body"})

		ctx, cancel := context.WithCancel(context.Background())

		sender := newMockSender()
		sender.delay = 10 * time.Millisecond // slow enough to be cancelled
		wp := &WorkerPool{
			Concurrency: 1,
			MaxRetries:  1,
			BackoffBase: 100 * time.Millisecond,
			BackoffMax:  200 * time.Millisecond,
		}

		go func() {
			time.Sleep(30 * time.Millisecond)
			cancel()
		}()

		result := wp.Run(ctx, sender, st, &mockLogger{})

		// Some emails processed, but not all 100
		require.Less(result.Sent+result.Failed, 100)
	})
}
