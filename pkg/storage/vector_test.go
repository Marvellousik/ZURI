package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"zuri-daemon/pkg/storage"
)

func TestVectorStore_WorkloadModes(t *testing.T) {
	modes := []string{"local", "team", "enterprise"}

	for _, mode := range modes {
		t.Run("Mode_"+mode, func(t *testing.T) {
			tmpDir := t.TempDir()
			cfg := storage.DefaultConfig()
			cfg.Mode = mode
			cfg.SQLite.Path = filepath.Join(tmpDir, "test.db")

			store, err := storage.NewVectorStore(cfg)
			if err != nil {
				t.Fatalf("failed initializing vector store for mode '%s': %v", mode, err)
			}
			defer store.Close()

			ctx := context.Background()
			indexName := "code_memory_test"

			// 1. Create Index
			if err := store.CreateOrOpenIndex(ctx, indexName, 4, storage.MetricCosine); err != nil {
				t.Fatalf("failed creating index in mode '%s': %v", mode, err)
			}

			// 2. Insert Vector
			vecRecord := storage.VectorRecord{
				ID:     "mem-001",
				Vector: []float32{0.1, 0.2, 0.3, 0.4},
				Payload: map[string]interface{}{
					"repo_id":      "repo-auth",
					"boundary":     "security",
					"decision_key": "boundary:security/concern:auth/decision_type:jwt",
				},
				CreatedAt: time.Now(),
			}

			if err := store.Insert(ctx, indexName, vecRecord); err != nil {
				t.Fatalf("failed inserting record in mode '%s': %v", mode, err)
			}

			// 3. Similarity Search with Payload Filter
			results, err := store.SimilaritySearch(ctx, indexName, []float32{0.1, 0.2, 0.3, 0.4}, storage.SearchFilter{
				RepoID: "repo-auth",
			}, 10)

			if err != nil {
				t.Fatalf("failed performing similarity search in mode '%s': %v", mode, err)
			}

			if len(results) != 1 {
				t.Fatalf("expected 1 result in mode '%s', got %d", mode, len(results))
			}

			if results[0].ID != "mem-001" {
				t.Errorf("expected result ID 'mem-001', got '%s'", results[0].ID)
			}

			// 4. Stats Verification
			stats, err := store.GetStats(ctx, indexName)
			if err != nil {
				t.Fatalf("failed getting stats in mode '%s': %v", mode, err)
			}

			if stats.WorkloadMode != mode {
				t.Errorf("expected stats workload mode '%s', got '%s'", mode, stats.WorkloadMode)
			}

			// 5. Capability Discovery
			caps := store.Capabilities()
			if !caps.SupportsBatchUpsert {
				t.Errorf("expected SupportsBatchUpsert to be true in mode '%s'", mode)
			}
		})
	}
}

func TestStorage_MigrationUtility(t *testing.T) {
	ctx := context.Background()
	indexName := "migration_test_index"

	// Source: Local Store
	tmpDir := t.TempDir()
	srcCfg := storage.DefaultConfig()
	srcCfg.Mode = "local"
	srcCfg.SQLite.Path = filepath.Join(tmpDir, "source.db")

	srcStore, err := storage.NewVectorStore(srcCfg)
	if err != nil {
		t.Fatalf("failed creating source store: %v", err)
	}
	defer srcStore.Close()

	// Seed source store with records
	_ = srcStore.CreateOrOpenIndex(ctx, indexName, 4, storage.MetricCosine)
	_ = srcStore.BatchInsert(ctx, indexName, []storage.VectorRecord{
		{
			ID:      "vec-1",
			Vector:  []float32{0.5, 0.5, 0.0, 0.0},
			Payload: map[string]interface{}{"repo": "repo-a"},
		},
		{
			ID:      "vec-2",
			Vector:  []float32{0.0, 0.0, 0.8, 0.8},
			Payload: map[string]interface{}{"repo": "repo-b"},
		},
	})

	// Target: Team Store
	targetCfg := storage.DefaultConfig()
	targetCfg.Mode = "team"

	targetStore, err := storage.NewVectorStore(targetCfg)
	if err != nil {
		t.Fatalf("failed creating target store: %v", err)
	}
	defer targetStore.Close()

	// Execute migration
	migrator := storage.NewMigrator()
	res, err := migrator.Migrate(ctx, indexName, srcStore, targetStore)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if !res.IsSuccess {
		t.Fatalf("expected migration success, got failure: %s", res.ErrorMessage)
	}

	if res.MigratedCount != 2 {
		t.Errorf("expected 2 migrated records, got %d", res.MigratedCount)
	}

	// Verify target store has records
	targetResults, err := targetStore.SimilaritySearch(ctx, indexName, []float32{0.5, 0.5, 0.0, 0.0}, storage.SearchFilter{}, 10)
	if err != nil {
		t.Fatalf("failed querying target store after migration: %v", err)
	}

	if len(targetResults) != 2 {
		t.Errorf("expected 2 records in target store post-migration, got %d", len(targetResults))
	}
}
