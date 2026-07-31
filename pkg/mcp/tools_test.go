package mcp

import (
	"context"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"zuri-daemon/pkg/db"
)

func TestMCPToolsEndToEnd(t *testing.T) {
	os.Setenv("ZURI_DB_PORT", "5438")
	tmpDir, err := os.MkdirTemp("", "zuri_mcp_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("ZURI_DB_PATH", tmpDir)

	dbMgr := db.NewDBManager()
	if err := dbMgr.Init(); err != nil {
		t.Fatalf("Failed to init DBManager: %v", err)
	}
	defer dbMgr.Close()

	sqlDB := dbMgr.GetDB()

	// Ensure schema exists for testing
	_, err = db.RunMigrations(sqlDB)
	if err != nil {
		t.Logf("Note: RunMigrations result: %v", err)
	}

	// Seed test fixtures per spec section 4 (no manual write path in daemon API; use SQL seed script for test setup)
	var repoID string
	err = sqlDB.QueryRow(`
		INSERT INTO repo (github_installation_id, github_repo_full_name)
		VALUES (112233, 'org/mcp-test-repo')
		RETURNING repo_id;
	`).Scan(&repoID)
	if err != nil {
		t.Fatalf("Failed to seed repo fixture: %v", err)
	}

	_, err = sqlDB.Exec(`
		INSERT INTO zuri_config (repo_id, approver_usernames, expiry_days)
		VALUES ($1, $2, 60);
	`, repoID, "{alice_lead}")
	if err != nil {
		t.Fatalf("Failed to seed zuri_config fixture: %v", err)
	}

	// Seed a proposed probabilistic record
	var proposedMemID string
	err = sqlDB.QueryRow(`
		INSERT INTO memory_record (
			repo_id, tier, status, decision, reasoning,
			originating_commit, created_by
		) VALUES (
			$1, 'probabilistic', 'proposed', 'Migrate to pgvector for memory index', 'Vector search improves semantic retrieval speed',
			'comm00112233', 'bob_dev'
		) RETURNING memory_id;
	`, repoID).Scan(&proposedMemID)
	if err != nil {
		t.Fatalf("Failed to seed proposed memory: %v", err)
	}

	// Seed a confirmed canonical record
	var canonicalMemID string
	err = sqlDB.QueryRow(`
		INSERT INTO memory_record (
			repo_id, tier, status, decision, reasoning,
			originating_commit, originating_pr_number, created_by
		) VALUES (
			$1, 'canonical', 'confirmed', 'Use Go for daemon binary', 'Go provides fast startup and single binary packaging',
			'comm44556677', 101, 'alice_lead'
		) RETURNING memory_id;
	`, repoID).Scan(&canonicalMemID)
	if err != nil {
		t.Fatalf("Failed to seed canonical memory: %v", err)
	}

	// Add memory_touches_file entries
	_, err = sqlDB.Exec(`
		INSERT INTO memory_touches_file (memory_id, file_path)
		VALUES ($1, 'pkg/db/embedded.go'), ($2, 'cmd/daemon/main.go');
	`, proposedMemID, canonicalMemID)
	if err != nil {
		t.Fatalf("Failed to seed memory_touches_file: %v", err)
	}

	svc := NewMemoryService(sqlDB)
	ctx := context.Background()

	t.Run("get_relevant_memory", func(t *testing.T) {
		input := GetRelevantMemoryInput{
			PromptText:   "pgvector search",
			FilesInScope: []string{"pkg/db/embedded.go"},
			TokenBudget:  4000,
		}

		_, output, err := svc.HandleGetRelevantMemory(ctx, nil, input)
		if err != nil {
			t.Fatalf("HandleGetRelevantMemory failed: %v", err)
		}

		if len(output.Memories) == 0 {
			t.Fatalf("Expected candidate memories returned, got 0")
		}

		// Verify audit log entry for retrieved event (§14)
		var auditCount int
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE event_type = 'retrieved';").Scan(&auditCount)
		if err != nil || auditCount == 0 {
			t.Fatalf("Expected audit_log entry for 'retrieved' event, found count=%d, err=%v", auditCount, err)
		}
	})

	t.Run("resolve_memory enforcement on confirmed record", func(t *testing.T) {
		// Mechanical enforcement (§4 & §13.3): resolve_memory MUST reject resolving non-proposed records
		input := ResolveMemoryInput{
			MemoryID:   canonicalMemID,
			Action:     "confirm",
			ResolvedBy: "charlie_lead",
		}

		_, _, err := svc.HandleResolveMemory(ctx, nil, input)
		if err == nil {
			t.Fatalf("Expected error attempting resolve_memory on confirmed canonical record, got nil")
		}
		t.Logf("Verified mechanical write gating enforcement: %v", err)
	})

	t.Run("resolve_memory edit and confirm merged proposed record", func(t *testing.T) {
		sourceCtx := "PR #601 merged to main"
		input := ResolveMemoryInput{
			MemoryID:      proposedMemID,
			Action:        "edit",
			EditedContent: "Migrate to native pgvector HNSW index for high performance memory retrieval",
			ResolvedBy:    "alice_lead",
			PRMerged:      true,
			SourceContext: &sourceCtx,
		}

		_, output, err := svc.HandleResolveMemory(ctx, nil, input)
		if err != nil {
			t.Fatalf("HandleResolveMemory failed: %v", err)
		}

		if output.NewStatus != "confirmed" {
			t.Fatalf("Expected NewStatus 'confirmed', got %s", output.NewStatus)
		}
		if output.NewTier != "canonical" {
			t.Fatalf("Expected NewTier 'canonical' when PRMerged=true, got %s", output.NewTier)
		}

		// Verify audit log entry for edited event (§14)
		var auditCount int
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE memory_id = $1 AND event_type = 'edited';", proposedMemID).Scan(&auditCount)
		if err != nil || auditCount == 0 {
			t.Fatalf("Expected audit_log entry for 'edited' event, found count=%d, err=%v", auditCount, err)
		}
	})
}
