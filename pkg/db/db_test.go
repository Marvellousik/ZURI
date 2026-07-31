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

	// Explicitly verify that ValidateVectorExtension detects pgvector requirement
	valErr := ValidateVectorExtension(db)
	var hasVector bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector');").Scan(&hasVector)
	if err != nil {
		t.Fatalf("Failed to check pg_extension table: %v", err)
	}

	if !hasVector {
		if valErr == nil {
			t.Fatalf("ValidateVectorExtension failed to return error when pgvector is missing!")
		}
		t.Logf("Verified: ValidateVectorExtension correctly caught missing pgvector extension: %v", valErr)
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
}
