package backfill

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockExtractor struct {
	processed []int
	err       error
}

func (m *mockExtractor) Process(ctx context.Context, pr HistoricalPR) error {
	if m.err != nil {
		return m.err
	}
	m.processed = append(m.processed, pr.Number)
	return nil
}

type mockLogger struct {
	skipped map[int]string
}

func newMockLogger() *mockLogger {
	return &mockLogger{skipped: make(map[int]string)}
}

func (m *mockLogger) LogSkipped(prNumber int, reason string) {
	m.skipped[prNumber] = reason
}

func intPtr(v int) *int {
	return &v
}

func TestService_Run(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	prs := []HistoricalPR{
		{Number: 1, MergedAt: now.Add(-10 * 24 * time.Hour), Tokens: 100}, // oldest
		{Number: 2, MergedAt: now.Add(-5 * 24 * time.Hour), Tokens: 200},
		{Number: 3, MergedAt: now.Add(-1 * 24 * time.Hour), Tokens: 300}, // newest
	}

	t.Run("recency ordering and complete processing", func(t *testing.T) {
		extractor := &mockExtractor{}
		logger := newMockLogger()
		service := NewService(extractor, logger)

		err := service.Run(ctx, append([]HistoricalPR{}, prs...), Budget{}) // No limits
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(extractor.processed) != 3 {
			t.Fatalf("expected 3 PRs processed, got %d", len(extractor.processed))
		}

		// Verify recency ordering (newest first)
		if extractor.processed[0] != 3 || extractor.processed[1] != 2 || extractor.processed[2] != 1 {
			t.Errorf("expected ordering [3, 2, 1], got %v", extractor.processed)
		}

		if len(logger.skipped) != 0 {
			t.Errorf("expected 0 skips, got %d", len(logger.skipped))
		}
	})

	t.Run("budget cutoff by PR count", func(t *testing.T) {
		extractor := &mockExtractor{}
		logger := newMockLogger()
		service := NewService(extractor, logger)

		budget := Budget{MaxPRs: intPtr(2)}
		err := service.Run(ctx, append([]HistoricalPR{}, prs...), budget)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(extractor.processed) != 2 {
			t.Fatalf("expected 2 PRs processed, got %d", len(extractor.processed))
		}

		if extractor.processed[0] != 3 || extractor.processed[1] != 2 {
			t.Errorf("expected processed [3, 2], got %v", extractor.processed)
		}

		if len(logger.skipped) != 1 {
			t.Fatalf("expected 1 skip, got %d", len(logger.skipped))
		}

		if reason, ok := logger.skipped[1]; !ok || reason != "budget cutoff (MaxPRs reached)" {
			t.Errorf("expected PR 1 to be skipped due to PR cap, got %v", reason)
		}
	})

	t.Run("budget cutoff by token limit", func(t *testing.T) {
		extractor := &mockExtractor{}
		logger := newMockLogger()
		service := NewService(extractor, logger)

		budget := Budget{MaxTokens: intPtr(450)} // 300 (PR 3) + 200 (PR 2) = 500, which exceeds 450. Only PR 3 fits.
		err := service.Run(ctx, append([]HistoricalPR{}, prs...), budget)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(extractor.processed) != 1 {
			t.Fatalf("expected 1 PR processed, got %d", len(extractor.processed))
		}

		if extractor.processed[0] != 3 {
			t.Errorf("expected processed [3], got %v", extractor.processed)
		}

		if len(logger.skipped) != 2 {
			t.Fatalf("expected 2 skips, got %d", len(logger.skipped))
		}
	})

	t.Run("extractor failure halts backfill", func(t *testing.T) {
		extractor := &mockExtractor{err: errors.New("llm timeout")}
		logger := newMockLogger()
		service := NewService(extractor, logger)

		err := service.Run(ctx, append([]HistoricalPR{}, prs...), Budget{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if len(extractor.processed) != 0 {
			t.Errorf("expected 0 PRs fully processed, got %d", len(extractor.processed))
		}
	})
}
