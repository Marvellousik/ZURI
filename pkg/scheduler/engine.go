package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PendingMemory represents a memory that hasn't been resolved yet and may need a reminder.
type PendingMemory struct {
	MemoryID       string
	RepoFullName   string
	PRNumber       int
	Cadence        time.Duration
	LastRemindedAt *time.Time
}

// MemoryProvider fetches memories that are currently in the 'proposed' state.
type MemoryProvider interface {
	GetPendingMemories(ctx context.Context) ([]PendingMemory, error)
	MarkReminded(ctx context.Context, memoryID string, t time.Time) error
}

// NotificationSink sends the actual reminder (e.g., posting a GitHub PR comment).
type NotificationSink interface {
	SendReminder(ctx context.Context, mem PendingMemory) error
}

// Engine handles scheduled reminders, event-triggered resurfacing, and overlap detection.
type Engine struct {
	mu            sync.Mutex
	lastReminded  map[string]time.Time // Local cache for debouncing concurrent hits
	provider      MemoryProvider
	sink          NotificationSink
	checkInterval time.Duration
	stopCh        chan struct{}
}

// NewEngine creates a new scheduler engine.
func NewEngine(provider MemoryProvider, sink NotificationSink, interval time.Duration) *Engine {
	return &Engine{
		lastReminded:  make(map[string]time.Time),
		provider:      provider,
		sink:          sink,
		checkInterval: interval,
		stopCh:        make(chan struct{}),
	}
}

// Start boots the background scheduled reminder job.
func (e *Engine) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(e.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-e.stopCh:
				return
			case <-ticker.C:
				_ = e.RunScheduledJob(ctx) // Errors are intentionally swallowed in the background ticker per standard practices unless a logger is injected
			}
		}
	}()
}

// Stop halts the background job.
func (e *Engine) Stop() {
	close(e.stopCh)
}

// RunScheduledJob executes a single tick of the scheduled reminder job.
func (e *Engine) RunScheduledJob(ctx context.Context) error {
	memories, err := e.provider.GetPendingMemories(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch pending memories: %w", err)
	}

	for _, mem := range memories {
		e.TriggerReminder(ctx, mem, false)
	}
	return nil
}

// TriggerReminder processes a reminder request, enforcing cadence and overlap detection.
// It can be invoked explicitly for event-triggered resurfacing.
func (e *Engine) TriggerReminder(ctx context.Context, mem PendingMemory, isEventTriggered bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var dbLast time.Time
	if mem.LastRemindedAt != nil {
		dbLast = *mem.LastRemindedAt
	}

	memCache, cacheExists := e.lastReminded[mem.MemoryID]
	
	last := dbLast
	if cacheExists && memCache.After(dbLast) {
		last = memCache
	}

	if !last.IsZero() {
		// Reminder Cadence: For scheduled jobs, strictly respect the configured cadence.
		if !isEventTriggered && time.Since(last) < mem.Cadence {
			return
		}

		// Overlap Detection: For event-triggered resurfacing, prevent spamming within a short window.
		if isEventTriggered && time.Since(last) < time.Hour {
			return
		}
	}

	err := e.sink.SendReminder(ctx, mem)
	if err == nil {
		now := time.Now()
		e.lastReminded[mem.MemoryID] = now
		if e.provider != nil {
			_ = e.provider.MarkReminded(ctx, mem.MemoryID, now)
		}
	}
}

// MockTime allows injecting historical times for testing.
func (e *Engine) OverrideLastReminded(memoryID string, t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastReminded[memoryID] = t
}
