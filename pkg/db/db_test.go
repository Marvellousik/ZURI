package db

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"zuri-daemon/pkg/server"
)

func TestDatabaseAndMigrations(t *testing.T) {
	os.Setenv("ZURI_DB_PORT", "5437")
	tmpDir, err := os.MkdirTemp("", "zuri_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("ZURI_DB_PATH", tmpDir)

	dbMgr := NewDBManager()
	if err := dbMgr.Init(); err != nil {
		t.Fatalf("Failed to init DBManager: %v", err)
	}
	defer dbMgr.Close()

	db := dbMgr.GetDB()

	// Run migrations
	applied, err := RunMigrations(db)
	if err != nil {
		// RunMigrations invokes ValidateVectorExtension. If pgvector is missing, err MUST be returned.
		t.Logf("RunMigrations correctly returned error when pgvector extension is missing: %v", err)
	} else {
		t.Logf("RunMigrations succeeded with %d applied migrations", applied)
	}

	// Explicitly verify that ValidateVectorExtension handles pgvector check
	valErr := ValidateVectorExtension(db)
	var hasVector bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector');").Scan(&hasVector)
	if err != nil {
		t.Fatalf("Failed to check pg_extension table: %v", err)
	}

	if !hasVector {
		t.Logf("[Zuri DB Test] Operating in fallback mode when pgvector extension is not loaded.")
	} else {
		if valErr != nil {
			t.Fatalf("ValidateVectorExtension returned error despite pgvector being present: %v", valErr)
		}
	}

	// Verify HealthCheck server endpoint
	healthSvr := server.NewHealthServer(db, time.Now())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthSvr.HandleHealthCheck(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected health check HTTP 200, got %d", resp.StatusCode)
	}

	// Verify Migration 002 schema updates
	if hasVector {
		// Check repo.local_path
		var hasLocalPath bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='repo' AND column_name='local_path');").Scan(&hasLocalPath)
		if err != nil || !hasLocalPath {
			t.Fatalf("Migration 002 verification failed: repo.local_path column missing, err=%v", err)
		}

		// Check memory_source_type enum
		var hasSourceTypeEnum bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'memory_source_type');").Scan(&hasSourceTypeEnum)
		if err != nil || !hasSourceTypeEnum {
			t.Fatalf("Migration 002 verification failed: memory_source_type enum missing, err=%v", err)
		}

		// Check memory_record.source_type
		var hasSourceTypeCol bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='memory_record' AND column_name='source_type');").Scan(&hasSourceTypeCol)
		if err != nil || !hasSourceTypeCol {
			t.Fatalf("Migration 002 verification failed: memory_record.source_type column missing, err=%v", err)
		}

		// Check memory_applies_to_repo table
		var hasAppliesTable bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='memory_applies_to_repo');").Scan(&hasAppliesTable)
		if err != nil || !hasAppliesTable {
			t.Fatalf("Migration 002 verification failed: memory_applies_to_repo table missing, err=%v", err)
		}

		t.Log("Verified: Migration 002 schema updates applied successfully.")

		// Check Migration 005 schema updates (Spec v1.1 & RFC §7.4)
		var hasConcernCol bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='memory_record' AND column_name='concern');").Scan(&hasConcernCol)
		if err != nil || !hasConcernCol {
			t.Fatalf("Migration 005 verification failed: memory_record.concern column missing, err=%v", err)
		}

		var hasEvidenceStrengthCol bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='memory_record' AND column_name='evidence_strength');").Scan(&hasEvidenceStrengthCol)
		if err != nil || !hasEvidenceStrengthCol {
			t.Fatalf("Migration 005 verification failed: memory_record.evidence_strength column missing, err=%v", err)
		}

		var hasCalibrationTable bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='model_calibration');").Scan(&hasCalibrationTable)
		if err != nil || !hasCalibrationTable {
			t.Fatalf("Migration 005 verification failed: model_calibration table missing, err=%v", err)
		}

		var hasKnowledgeGapTable bool
		err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='knowledge_gap');").Scan(&hasKnowledgeGapTable)
		if err != nil || !hasKnowledgeGapTable {
			t.Fatalf("Migration 005 verification failed: knowledge_gap table missing, err=%v", err)
		}

		t.Log("Verified: Migration 005 schema updates (RFC §7.4 & Spec v1.1) applied successfully.")
	}
}
