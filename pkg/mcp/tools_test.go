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

	t.Run("stage 2 multi-repo scope resolution and cross-repo retrieval (§9.6)", func(t *testing.T) {
		// Seed Repo 1 and Repo 2 with local_path
		var repo1ID, repo2ID string
		err = sqlDB.QueryRow(`
			INSERT INTO repo (github_installation_id, github_repo_full_name, local_path)
			VALUES (111, 'org/backend-service', '/workspace/backend')
			RETURNING repo_id;
		`).Scan(&repo1ID)
		if err != nil {
			t.Fatalf("Failed seeding repo1: %v", err)
		}

		err = sqlDB.QueryRow(`
			INSERT INTO repo (github_installation_id, github_repo_full_name, local_path)
			VALUES (222, 'org/frontend-app', '/workspace/frontend')
			RETURNING repo_id;
		`).Scan(&repo2ID)
		if err != nil {
			t.Fatalf("Failed seeding repo2: %v", err)
		}

		// Seed memory record for backend repo
		var memBackendID string
		err = sqlDB.QueryRow(`
			INSERT INTO memory_record (repo_id, tier, status, decision, reasoning, originating_commit, created_by)
			VALUES ($1, 'canonical', 'confirmed', 'gRPC API contract v2 for auth', 'High throughput binary protocol', 'c1', 'alice')
			RETURNING memory_id;
		`, repo1ID).Scan(&memBackendID)
		if err != nil {
			t.Fatalf("Failed seeding backend memory: %v", err)
		}

		// Seed memory record for frontend repo (isolated)
		var memFrontendID string
		err = sqlDB.QueryRow(`
			INSERT INTO memory_record (repo_id, tier, status, decision, reasoning, originating_commit, created_by)
			VALUES ($1, 'canonical', 'confirmed', 'React Query state management', 'Caching client side state', 'c2', 'bob')
			RETURNING memory_id;
		`, repo2ID).Scan(&memFrontendID)
		if err != nil {
			t.Fatalf("Failed seeding frontend memory: %v", err)
		}

		// Seed cross-repo decision originating in backend repo, but applied to frontend repo via resolve_memory / memory_applies_to_repo
		var memCrossID string
		err = sqlDB.QueryRow(`
			INSERT INTO memory_record (repo_id, tier, status, decision, reasoning, originating_commit, created_by)
			VALUES ($1, 'probabilistic', 'proposed', 'Shared OAuth Token Schema', 'Common token format between frontend and backend', 'c3', 'charlie')
			RETURNING memory_id;
		`, repo1ID).Scan(&memCrossID)
		if err != nil {
			t.Fatalf("Failed seeding cross memory: %v", err)
		}

		// Confirm cross memory and mark as applying to frontend repo (repo2ID)
		_, _, err = svc.HandleResolveMemory(ctx, nil, ResolveMemoryInput{
			MemoryID:         memCrossID,
			Action:           "confirm",
			ResolvedBy:       "charlie",
			PRMerged:         true,
			AppliesToRepoIDs: []string{repo2ID},
		})
		if err != nil {
			t.Fatalf("Failed resolving cross-repo memory: %v", err)
		}

		// Add memory_touches_file entries for test records
		_, err = sqlDB.Exec(`
			INSERT INTO memory_touches_file (memory_id, file_path)
			VALUES ($1, '/workspace/backend/src/main.go'), ($2, '/workspace/frontend/src/App.tsx'), ($3, '/workspace/frontend/src/App.tsx');
		`, memBackendID, memFrontendID, memCrossID)
		if err != nil {
			t.Fatalf("Failed seeding memory_touches_file for multi-repo test: %v", err)
		}

		// 1. Query with files_in_scope targeting frontend (/workspace/frontend/src/App.tsx)
		_, outputFrontend, err := svc.HandleGetRelevantMemory(ctx, nil, GetRelevantMemoryInput{
			PromptText:   "OAuth state management",
			FilesInScope: []string{"/workspace/frontend/src/App.tsx"},
			TokenBudget:  4000,
		})
		if err != nil {
			t.Fatalf("Querying frontend scope failed: %v", err)
		}

		// Should include memFrontendID and memCrossID (applies to frontend), but NOT memBackendID
		foundFrontend := false
		foundCross := false
		foundBackend := false

		for _, m := range outputFrontend.Memories {
			if m.MemoryID == memFrontendID {
				foundFrontend = true
			}
			if m.MemoryID == memCrossID {
				foundCross = true
			}
			if m.MemoryID == memBackendID {
				foundBackend = true
			}
		}

		if !foundFrontend || !foundCross {
			t.Fatalf("Expected frontend and cross-repo memories returned for frontend scope query. Got memories: %+v", outputFrontend.Memories)
		}
		if foundBackend {
			t.Fatalf("Backend-only memory improperly returned for frontend scope query!")
		}

		// 2. Query with empty files_in_scope (requirement 5 fallback to all registered repos)
		_, outputAll, err := svc.HandleGetRelevantMemory(ctx, nil, GetRelevantMemoryInput{
			PromptText:   "contract",
			FilesInScope: []string{},
			TokenBudget:  4000,
		})
		if err != nil {
			t.Fatalf("Querying with unsupplied files_in_scope failed: %v", err)
		}

		if len(outputAll.Memories) == 0 {
			t.Fatalf("Expected memories returned when files_in_scope is empty, got 0")
		}
	})

	t.Run("resolve_memory transaction rollback on failure", func(t *testing.T) {
		// We'll reuse repoID from earlier in the test for seeding
		var txMemID string
		err = sqlDB.QueryRow(`
			INSERT INTO memory_record (repo_id, tier, status, decision, reasoning, originating_commit, created_by)
			VALUES ($1, 'probabilistic', 'proposed', 'Test rollback', 'Reasoning', 'comm00112233', 'alice')
			RETURNING memory_id;
		`, repoID).Scan(&txMemID)
		if err != nil {
			t.Fatalf("Failed seeding tx memory: %v", err)
		}

		// Try to resolve with an invalid repo ID in AppliesToRepoIDs to trigger a database constraint/type error
		_, _, err = svc.HandleResolveMemory(ctx, nil, ResolveMemoryInput{
			MemoryID:         txMemID,
			Action:           "confirm",
			ResolvedBy:       "alice",
			PRMerged:         true,
			AppliesToRepoIDs: []string{"invalid-uuid"},
		})
		
		if err == nil {
			t.Fatalf("Expected error when providing invalid repo ID, got nil")
		}

		// Verify that memory_record is NOT updated (status should still be 'proposed')
		var currentStatus string
		err = sqlDB.QueryRow("SELECT status FROM memory_record WHERE memory_id = $1;", txMemID).Scan(&currentStatus)
		if err != nil {
			t.Fatalf("Failed to query status: %v", err)
		}
		if currentStatus != "proposed" {
			t.Fatalf("Expected status to remain 'proposed', got '%s'", currentStatus)
		}

		// Verify no rows in memory_applies_to_repo for this memory
		var count int
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM memory_applies_to_repo WHERE memory_id = $1;", txMemID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query memory_applies_to_repo: %v", err)
		}
		if count > 0 {
			t.Fatalf("Expected 0 rows in memory_applies_to_repo, got %d", count)
		}
	})

	t.Run("resolve_memory reject does not insert applies_to_repo", func(t *testing.T) {
		var rejectMemID string
		err = sqlDB.QueryRow(`
			INSERT INTO memory_record (repo_id, tier, status, decision, reasoning, originating_commit, created_by)
			VALUES ($1, 'probabilistic', 'proposed', 'Test reject', 'Reasoning', 'comm00112233', 'alice')
			RETURNING memory_id;
		`, repoID).Scan(&rejectMemID)
		if err != nil {
			t.Fatalf("Failed seeding reject memory: %v", err)
		}

		_, _, err = svc.HandleResolveMemory(ctx, nil, ResolveMemoryInput{
			MemoryID:         rejectMemID,
			Action:           "reject",
			ResolvedBy:       "alice",
			PRMerged:         false,
			AppliesToRepoIDs: []string{repoID}, // Even with a valid ID, reject shouldn't insert
		})
		if err != nil {
			t.Fatalf("Expected success for reject, got error: %v", err)
		}

		// Verify no rows in memory_applies_to_repo for this memory
		var count int
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM memory_applies_to_repo WHERE memory_id = $1;", rejectMemID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query memory_applies_to_repo: %v", err)
		}
		if count > 0 {
			t.Fatalf("Expected 0 rows in memory_applies_to_repo for rejected memory, got %d", count)
		}
	})

	t.Run("resolve_knowledge_gap answer and acknowledge_unknown (§13.4)", func(t *testing.T) {
		// Seed open knowledge gap
		var gapID string
		err = sqlDB.QueryRow(`
			INSERT INTO knowledge_gap (decision_key, scope, gap_type, status)
			VALUES ('boundary:payments/concern:reliability/decision_type:retry-policy', 'org/mcp-test-repo', 'conflicting_conventions', 'open')
			RETURNING gap_id;
		`).Scan(&gapID)
		if err != nil {
			t.Fatalf("Failed seeding test knowledge gap: %v", err)
		}

		// Test answer action
		_, gapOutput, err := svc.HandleResolveKnowledgeGap(ctx, nil, ResolveKnowledgeGapInput{
			GapID:         gapID,
			Action:        "answer",
			AnswerContent: "All payment calls must retry up to 3 times with exponential backoff",
			ResolvedBy:    "alice_lead",
		})
		if err != nil {
			t.Fatalf("HandleResolveKnowledgeGap failed: %v", err)
		}

		if gapOutput.NewStatus != "answered" {
			t.Fatalf("Expected NewStatus 'answered', got %s", gapOutput.NewStatus)
		}
		if gapOutput.MemoryID == "" {
			t.Fatalf("Expected memory_id returned for answered gap, got empty")
		}

		// Verify audit log entry
		var gapAuditCount int
		err = sqlDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE gap_id = $1 AND event_type = 'gap_answered';", gapID).Scan(&gapAuditCount)
		if err != nil || gapAuditCount == 0 {
			t.Fatalf("Expected audit_log entry for gap_answered event, count=%d, err=%v", gapAuditCount, err)
		}
	})
}
