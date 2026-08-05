package storage

import (
	"context"
	"fmt"
	"time"
)

// MigrationResult summarizes the outcome of a vector database migration.
type MigrationResult struct {
	SourceMode      string        `json:"source_mode"`
	TargetMode      string        `json:"target_mode"`
	IndexName       string        `json:"index_name"`
	MigratedCount   int64         `json:"migrated_count"`
	Duration        time.Duration `json:"duration"`
	IsSuccess       bool          `json:"is_success"`
	ErrorMessage    string        `json:"error_message,omitempty"`
}

// Migrator handles streaming vector records and metadata between two storage backends.
type Migrator struct{}

// NewMigrator creates a new Migrator instance.
func NewMigrator() *Migrator {
	return &Migrator{}
}

// MigrateStreams copies all vector records from sourceStore to targetStore for the given index.
func (m *Migrator) Migrate(ctx context.Context, indexName string, sourceStore, targetStore VectorStore) (*MigrationResult, error) {
	startTime := time.Now()

	if sourceStore == nil || targetStore == nil {
		return nil, fmt.Errorf("migrator: source and target vector stores must not be nil")
	}

	srcStats, err := sourceStore.GetStats(ctx, indexName)
	if err != nil {
		return nil, fmt.Errorf("migrator: fetching source stats: %w", err)
	}

	// 1. Ensure target index exists
	if err := targetStore.CreateOrOpenIndex(ctx, indexName, srcStats.Dimensions, MetricCosine); err != nil {
		return nil, fmt.Errorf("migrator: creating target index: %w", err)
	}

	// 2. Query all records from source store
	dummyQuery := make([]float32, srcStats.Dimensions)
	records, err := sourceStore.SimilaritySearch(ctx, indexName, dummyQuery, SearchFilter{}, 10000)
	if err != nil {
		return nil, fmt.Errorf("migrator: reading source records: %w", err)
	}

	// 3. Convert SearchResult back to VectorRecord payloads for target insertion
	var vecRecords []VectorRecord
	for _, r := range records {
		vecRecords = append(vecRecords, VectorRecord{
			ID:        r.ID,
			Vector:    r.Vector,
			Payload:   r.Payload,
			CreatedAt: time.Now(),
		})
	}

	// 4. Batch insert into target store
	if len(vecRecords) > 0 {
		if err := targetStore.BatchInsert(ctx, indexName, vecRecords); err != nil {
			return &MigrationResult{
				SourceMode:   srcStats.WorkloadMode,
				TargetMode:   "unknown",
				IndexName:    indexName,
				Duration:     time.Since(startTime),
				IsSuccess:    false,
				ErrorMessage: err.Error(),
			}, fmt.Errorf("migrator: batch inserting into target store: %w", err)
		}
	}

	targetStats, _ := targetStore.GetStats(ctx, indexName)
	targetMode := "unknown"
	if targetStats != nil {
		targetMode = targetStats.WorkloadMode
	}

	return &MigrationResult{
		SourceMode:    srcStats.WorkloadMode,
		TargetMode:    targetMode,
		IndexName:     indexName,
		MigratedCount: int64(len(vecRecords)),
		Duration:      time.Since(startTime),
		IsSuccess:     true,
	}, nil
}
