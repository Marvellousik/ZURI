package gaps

import (
	"context"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"zuri-daemon/pkg/db"
)

func TestGapDetectorAndRouting(t *testing.T) {
	os.Setenv("ZURI_DB_PORT", "5440")
	tmpDir, err := os.MkdirTemp("", "zuri_gap_test_*")
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

	_, err = db.RunMigrations(sqlDB)
	if err != nil {
		t.Logf("RunMigrations result: %v", err)
	}

	var repoID string
	err = sqlDB.QueryRow(`
		INSERT INTO repo (github_installation_id, github_repo_full_name)
		VALUES (998877, 'org/gap-test-repo')
RETURNING repo_id;
	`).Scan(&repoID)
	if err != nil {
		t.Fatalf("Failed to seed repo: %v", err)
	}

	ctx := context.Background()

	t.Run("CODEOWNERS Parsing and Matching", func(t *testing.T) {
		codeownersContent := `
# CODEOWNERS File
* @default-team
/pkg/db/* @db-team @alice
/pkg/gaps/* @gap-team
`
		rules := ParseCodeowners(codeownersContent)
		if len(rules) != 3 {
			t.Fatalf("Expected 3 rules parsed, got %d", len(rules))
		}

		owners := ResolveOwnersForFile("pkg/db/embedded.go", rules)
		if len(owners) != 2 || owners[0] != "db-team" || owners[1] != "alice" {
			t.Errorf("Expected owners [db-team, alice], got %v", owners)
		}

		gapOwners := ResolveOwnersForFile("pkg/gaps/detector.go", rules)
		if len(gapOwners) != 1 || gapOwners[0] != "gap-team" {
			t.Errorf("Expected owner [gap-team], got %v", gapOwners)
		}
	})

	t.Run("Detect Conflicting Conventions (§10.7)", func(t *testing.T) {
		dKey := "boundary:payments/concern:reliability/decision_type:retry-policy"

		// Insert two conflicting records with same decision_key
		_, err = sqlDB.Exec(`
			INSERT INTO memory_record (repo_id, tier, status, decision_key, decision, reasoning, originating_commit, created_by)
			VALUES 
			($1, 'canonical', 'confirmed', $2, 'Retry payment up to 3 times', 'Reliability', 'c1', 'alice'),
			($1, 'canonical', 'confirmed', $2, 'Do not retry payment requests', 'Idempotency', 'c2', 'bob');
		`, repoID, dKey)
		if err != nil {
			t.Fatalf("Failed seeding conflicting records: %v", err)
		}

		detector := NewGapDetector(sqlDB)
		count, err := detector.DetectConflictingConventions(ctx, repoID)
		if err != nil {
			t.Fatalf("DetectConflictingConventions failed: %v", err)
		}
		if count == 0 {
			t.Fatalf("Expected at least 1 conflicting convention gap detected, got 0")
		}

		// Verify row in knowledge_gap table
		var gapID, status string
		err = sqlDB.QueryRow("SELECT gap_id, status FROM knowledge_gap WHERE decision_key = $1 AND gap_type = 'conflicting_conventions';", dKey).Scan(&gapID, &status)
		if err != nil || gapID == "" {
			t.Fatalf("Knowledge gap record missing: %v", err)
		}
		if status != "open" {
			t.Fatalf("Expected gap status 'open', got %s", status)
		}
	})
}
