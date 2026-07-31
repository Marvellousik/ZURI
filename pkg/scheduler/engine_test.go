package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockProvider struct {
	memories []PendingMemory
	err      error
}

func (m *mockProvider) GetPendingMemories(ctx context.Context) ([]PendingMemory, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.memories, nil
}

func (m *mockProvider) MarkReminded(ctx context.Context, memoryID string, t time.Time) error {
	for i, mem := range m.memories {
		if mem.MemoryID == memoryID {
			m.memories[i].LastRemindedAt = &t
			return nil
		}
	}
	return nil
}

type mockSink struct {
	sentCount int
	err       error
}

func (m *mockSink) SendReminder(ctx context.Context, mem PendingMemory) error {
	if m.err != nil {
		return m.err
	}
	m.sentCount++
	return nil
}

func TestEngine_CadenceAndOverlap(t *testing.T) {
	sink := &mockSink{}
	engine := NewEngine(nil, sink, time.Second)

	mem := PendingMemory{
		MemoryID: "mem-1",
		Cadence:  24 * time.Hour,
	}

	ctx := context.Background()

	// 1. First trigger should succeed
	engine.TriggerReminder(ctx, mem, false)
	if sink.sentCount != 1 {
		t.Fatalf("Expected 1 sent reminder, got %d", sink.sentCount)
	}

	// 2. Immediate scheduled trigger should be blocked by cadence (Overlap Detection)
	engine.TriggerReminder(ctx, mem, false)
	if sink.sentCount != 1 {
		t.Fatalf("Expected scheduled reminder to be blocked by cadence, got %d", sink.sentCount)
	}

	// 3. Immediate event trigger should be blocked by 1-hour debounce (Overlap Detection)
	engine.TriggerReminder(ctx, mem, true)
	if sink.sentCount != 1 {
		t.Fatalf("Expected event reminder to be blocked by debounce, got %d", sink.sentCount)
	}

	// 4. Override time to simulate past cadence
	engine.OverrideLastReminded("mem-1", time.Now().Add(-25*time.Hour))
	
	// 5. Next scheduled trigger should succeed
	engine.TriggerReminder(ctx, mem, false)
	if sink.sentCount != 2 {
		t.Fatalf("Expected reminder after cadence to succeed, got %d", sink.sentCount)
	}

	// 6. Override time to simulate inside cadence but past debounce
	engine.OverrideLastReminded("mem-1", time.Now().Add(-2*time.Hour))

	// Scheduled trigger should fail (2h < 24h)
	engine.TriggerReminder(ctx, mem, false)
	if sink.sentCount != 2 {
		t.Fatalf("Expected scheduled reminder to fail, got %d", sink.sentCount)
	}

	// Event trigger should succeed (2h > 1h)
	engine.TriggerReminder(ctx, mem, true)
	if sink.sentCount != 3 {
		t.Fatalf("Expected event reminder to succeed past debounce, got %d", sink.sentCount)
	}
}

func TestEngine_RunScheduledJob(t *testing.T) {
	provider := &mockProvider{
		memories: []PendingMemory{
			{MemoryID: "mem-1", Cadence: 24 * time.Hour},
			{MemoryID: "mem-2", Cadence: 24 * time.Hour},
		},
	}
	sink := &mockSink{}
	engine := NewEngine(provider, sink, time.Second)

	err := engine.RunScheduledJob(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if sink.sentCount != 2 {
		t.Fatalf("Expected 2 reminders sent, got %d", sink.sentCount)
	}

	// Running it again immediately should result in 0 new sends due to cadence
	err = engine.RunScheduledJob(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if sink.sentCount != 2 {
		t.Fatalf("Expected no new reminders, got %d", sink.sentCount)
	}
}

func TestEngine_RunScheduledJob_ProviderError(t *testing.T) {
	provider := &mockProvider{err: errors.New("db down")}
	engine := NewEngine(provider, &mockSink{}, time.Second)

	err := engine.RunScheduledJob(context.Background())
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}
