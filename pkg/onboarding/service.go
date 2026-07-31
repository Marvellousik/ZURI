package onboarding

import (
	"context"
	"fmt"
	"time"

	"zuri-daemon/pkg/db"
)

// SynthesizedMemory represents a domain tradeoff or architectural decision synthesized during onboarding.
type SynthesizedMemory struct {
	Decision  string
	Reasoning string
}

// MemoryStore defines the interface for persisting memory records to the database.
type MemoryStore interface {
	InsertMemory(ctx context.Context, record *db.MemoryRecord) error
}

// Service orchestrates the onboarding ingestion pipeline.
type Service struct {
	store MemoryStore
}

// NewService creates a new onboarding ingestion service.
func NewService(store MemoryStore) *Service {
	return &Service{store: store}
}

// Ingest processes a list of founder-confirmed synthesized memories and writes them directly as canonical records.
func (s *Service) Ingest(ctx context.Context, repoID string, founder string, confirmedMemories []SynthesizedMemory) error {
	for _, mem := range confirmedMemories {
		now := time.Now()

		record := &db.MemoryRecord{
			RepoID:            repoID,
			Tier:              db.TierCanonical,
			Status:            db.StatusConfirmed,
			SourceType:        db.SourceOnboardingSurvey,
			Decision:          mem.Decision,
			Reasoning:         mem.Reasoning,
			CreatedBy:         founder,
			ResolvedBy:        &founder,
			CreatedAt:         now,
			ResolvedAt:        &now,
		}

		if err := s.store.InsertMemory(ctx, record); err != nil {
			return fmt.Errorf("failed to write onboarding memory (Decision: %s): %w", mem.Decision, err)
		}
	}
	return nil
}
