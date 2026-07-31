package onboarding

import (
	"context"
	"errors"
	"testing"
	"zuri-daemon/pkg/db"
)

type mockMemoryStore struct {
	records []*db.MemoryRecord
	err     error
}

func (m *mockMemoryStore) InsertMemory(ctx context.Context, record *db.MemoryRecord) error {
	if m.err != nil {
		return m.err
	}
	m.records = append(m.records, record)
	return nil
}

func TestService_Ingest(t *testing.T) {
	ctx := context.Background()

	t.Run("successful ingestion", func(t *testing.T) {
		store := &mockMemoryStore{}
		service := NewService(store)

		memories := []SynthesizedMemory{
			{Decision: "Use Postgres", Reasoning: "Familiarity"},
			{Decision: "Use Go", Reasoning: "Performance"},
		}

		err := service.Ingest(ctx, "repo-123", "founder-alice", memories)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(store.records) != 2 {
			t.Fatalf("expected 2 records, got %d", len(store.records))
		}

		// Verify structural guarantees for the first record
		rec := store.records[0]
		if rec.RepoID != "repo-123" {
			t.Errorf("expected RepoID repo-123, got %s", rec.RepoID)
		}
		if rec.Tier != db.TierCanonical {
			t.Errorf("expected canonical tier, got %s", rec.Tier)
		}
		if rec.Status != db.StatusConfirmed {
			t.Errorf("expected confirmed status, got %s", rec.Status)
		}
		if rec.SourceType != db.SourceOnboardingSurvey {
			t.Errorf("expected onboarding_survey source type, got %s", rec.SourceType)
		}
		if rec.Decision != "Use Postgres" {
			t.Errorf("expected decision Use Postgres, got %s", rec.Decision)
		}
		if rec.OriginatingCommit != nil {
			t.Errorf("expected originating_commit to be nil, got %s", *rec.OriginatingCommit)
		}
		if rec.CreatedBy != "founder-alice" {
			t.Errorf("expected created_by founder-alice, got %s", rec.CreatedBy)
		}
		if rec.ResolvedBy == nil || *rec.ResolvedBy != "founder-alice" {
			t.Errorf("expected resolved_by founder-alice, got %v", rec.ResolvedBy)
		}
		if rec.ResolvedAt == nil {
			t.Error("expected resolved_at to be populated")
		}
	})

	t.Run("database error", func(t *testing.T) {
		store := &mockMemoryStore{err: errors.New("db disconnect")}
		service := NewService(store)

		memories := []SynthesizedMemory{
			{Decision: "Use Postgres", Reasoning: "Familiarity"},
		}

		err := service.Ingest(ctx, "repo-123", "founder-alice", memories)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		store := &mockMemoryStore{}
		service := NewService(store)

		err := service.Ingest(ctx, "repo-123", "founder-alice", nil)
		if err != nil {
			t.Fatalf("expected no error for empty list, got %v", err)
		}

		if len(store.records) != 0 {
			t.Errorf("expected 0 records, got %d", len(store.records))
		}
	})
}
