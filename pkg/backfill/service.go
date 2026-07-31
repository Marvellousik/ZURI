package backfill

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// HistoricalPR represents a merged PR available for backfill.
type HistoricalPR struct {
	Number   int
	MergedAt time.Time
	Tokens   int // Estimated tokens this PR will consume during extraction
}

// Budget defines limits for the backfill operation.
type Budget struct {
	MaxPRs    *int
	MaxTokens *int
}

// Extractor processes a single historical PR.
type Extractor interface {
	Process(ctx context.Context, pr HistoricalPR) error
}

// Logger records skipped PRs so the gaps are visible and inspectable.
type Logger interface {
	LogSkipped(prNumber int, reason string)
}

// Service orchestrates the backfill prioritization and budget capping.
type Service struct {
	extractor Extractor
	logger    Logger
}

// NewService creates a new backfill service.
func NewService(extractor Extractor, logger Logger) *Service {
	return &Service{
		extractor: extractor,
		logger:    logger,
	}
}

// Run executes the backfill process against a set of historical PRs, adhering to recency ordering and budget caps.
func (s *Service) Run(ctx context.Context, prs []HistoricalPR, budget Budget) error {
	// Recency ordering: Sort by MergedAt descending (most recent first).
	// This acts as the fallback mechanism until Structure Graph centrality is available.
	sort.SliceStable(prs, func(i, j int) bool {
		return prs[i].MergedAt.After(prs[j].MergedAt)
	})

	prsProcessed := 0
	tokensProcessed := 0

	budgetExhausted := false

	for _, pr := range prs {
		if budgetExhausted {
			s.logger.LogSkipped(pr.Number, "budget cutoff")
			continue
		}

		// Budget capping by PR count
		if budget.MaxPRs != nil && prsProcessed >= *budget.MaxPRs {
			budgetExhausted = true
			s.logger.LogSkipped(pr.Number, "budget cutoff (MaxPRs reached)")
			continue
		}

		// Budget capping by Token limit
		if budget.MaxTokens != nil && (tokensProcessed+pr.Tokens) > *budget.MaxTokens {
			budgetExhausted = true
			s.logger.LogSkipped(pr.Number, "budget cutoff (MaxTokens reached)")
			continue
		}

		if err := s.extractor.Process(ctx, pr); err != nil {
			return fmt.Errorf("failed to process PR %d: %w", pr.Number, err)
		}

		prsProcessed++
		tokensProcessed += pr.Tokens
	}

	return nil
}
