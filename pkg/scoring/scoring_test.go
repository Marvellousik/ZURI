package scoring

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"zuri-daemon/pkg/db"
)

func TestStatusWeightLookup(t *testing.T) {
	tests := []struct {
		tier     string
		status   string
		expected float64
	}{
		{tier: "canonical", status: "confirmed", expected: 1.0},
		{tier: "probabilistic", status: "confirmed", expected: 0.8},
		{tier: "probabilistic", status: "proposed", expected: 0.6},
		{tier: "working", status: "proposed", expected: 0.4},
		{tier: "probabilistic", status: "lapsed", expected: 0.1},
		{tier: "probabilistic", status: "rejected", expected: 0.0},
	}

	for _, tt := range tests {
		weight := GetStatusWeight(tt.tier, tt.status)
		if weight != tt.expected {
			t.Errorf("GetStatusWeight(%s, %s) = %f, expected %f", tt.tier, tt.status, weight, tt.expected)
		}
	}
}

func TestRecencyExponentialDecay(t *testing.T) {
	now := time.Now()

	// 0 days ago -> decay factor 1.0
	r0 := CalculateRecency(&now, now, DefaultHalfLifeDays)
	if math.Abs(r0-1.0) > 0.001 {
		t.Errorf("Expected recency 1.0 for t=0, got %f", r0)
	}

	// 30 days ago (exact half life) -> decay factor ~0.5
	t30 := now.AddDate(0, 0, -30)
	r30 := CalculateRecency(&t30, t30, DefaultHalfLifeDays)
	if math.Abs(r30-0.5) > 0.01 {
		t.Errorf("Expected recency ~0.5 for t=30 days, got %f", r30)
	}

	// 60 days ago (two half lives) -> decay factor ~0.25
	t60 := now.AddDate(0, 0, -60)
	r60 := CalculateRecency(&t60, t60, DefaultHalfLifeDays)
	if math.Abs(r60-0.25) > 0.01 {
		t.Errorf("Expected recency ~0.25 for t=60 days, got %f", r60)
	}
}

func TestTrendCalculationWithZeroPrior(t *testing.T) {
	// Brand new decision with 0 prior citations and 5 recent citations
	// Formula: (5 + 1) / (0 + 1) = 6.0, capped at MaxTrendMultiplier (3.0)
	tZeroPrior := CalculateTrend(5, 0)
	if tZeroPrior != 3.0 {
		t.Errorf("Expected capped trend multiplier 3.0 for prior=0, recent=5, got %f", tZeroPrior)
	}

	// Single recent citation, zero prior citations -> (1 + 1) / (0 + 1) = 2.0
	tSingleNew := CalculateTrend(1, 0)
	if tSingleNew != 2.0 {
		t.Errorf("Expected trend multiplier 2.0 for prior=0, recent=1, got %f", tSingleNew)
	}

	// Equal citations in recent vs prior -> (5 + 1) / (5 + 1) = 1.0 (flat)
	tFlat := CalculateTrend(5, 5)
	if math.Abs(tFlat-1.0) > 0.001 {
		t.Errorf("Expected trend 1.0 for equal citations, got %f", tFlat)
	}
}

func TestRevivalFlagging(t *testing.T) {
	os.Setenv("ZURI_DB_PORT", "5439")
	tmpDir, err := os.MkdirTemp("", "zuri_scoring_test_*")
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
	_, _ = db.RunMigrations(sqlDB)

	ctx := context.Background()

	// Seed repo
	var repoID string
	_ = sqlDB.QueryRow("INSERT INTO repo (github_installation_id, github_repo_full_name) VALUES (1, 'org/revival-test') RETURNING repo_id;").Scan(&repoID)

	// Seed a lapsed memory record
	var memID string
	_ = sqlDB.QueryRow(`
		INSERT INTO memory_record (repo_id, tier, status, decision, reasoning, originating_commit, created_by)
		VALUES ($1, 'probabilistic', 'lapsed', 'Abandoned cache strategy', 'Never confirmed', 'comm123', 'dev')
		RETURNING memory_id;
	`, repoID).Scan(&memID)

	// Check revival flagging when trend > 1.0
	err = CheckAndFlagRevival(ctx, sqlDB, memID, "lapsed", 2.5)
	if err != nil {
		t.Fatalf("CheckAndFlagRevival failed: %v", err)
	}

	// Verify audit_log entry written for revival_flagged
	var eventType string
	err = sqlDB.QueryRow("SELECT event_type FROM audit_log WHERE memory_id = $1;", memID).Scan(&eventType)
	if err != nil || eventType != "revival_flagged" {
		t.Fatalf("Expected audit_log entry with event_type 'revival_flagged', got eventType=%s, err=%v", eventType, err)
	}
}

func TestRetrievalLatencyBenchmark5000Records(t *testing.T) {
	os.Setenv("ZURI_DB_PORT", "5440")
	tmpDir, err := os.MkdirTemp("", "zuri_bench_test_*")
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
	_, _ = db.RunMigrations(sqlDB)

	// Seed 5,000 memory records matching full spec section 9.2 NFR scale target
	var repoID string
	_ = sqlDB.QueryRow("INSERT INTO repo (github_installation_id, github_repo_full_name) VALUES (999, 'org/bench-repo-5k') RETURNING repo_id;").Scan(&repoID)

	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatalf("Failed to begin seed tx: %v", err)
	}

	batchSize := 500
	for batch := 0; batch < 10; batch++ {
		for i := 0; i < batchSize; i++ {
			idx := batch*batchSize + i
			tier := "canonical"
			status := "confirmed"
			if idx%3 == 1 {
				tier = "probabilistic"
				status = "proposed"
			} else if idx%3 == 2 {
				tier = "probabilistic"
				status = "lapsed"
			}

			var memID string
			err := tx.QueryRow(`
				INSERT INTO memory_record (
					repo_id, tier, status, decision, reasoning, originating_commit, created_by
				) VALUES (
					$1, $2, $3, $4, $5, 'commit_hash', 'benchmark_user'
				) RETURNING memory_id;
			`, repoID, tier, status, fmt.Sprintf("Architectural decision #%d regarding database schema performance", idx), fmt.Sprintf("Detailed context and rationale for decision %d", idx)).Scan(&memID)

			if err != nil {
				tx.Rollback()
				t.Fatalf("Failed seeding benchmark memory %d: %v", idx, err)
			}

			_, _ = tx.Exec("INSERT INTO memory_touches_file (memory_id, file_path) VALUES ($1, $2);", memID, fmt.Sprintf("src/module_%d.go", idx%50))
		}
	}
	_ = tx.Commit()

	// Measure latency over 20 query runs against 5,000 records
	var latencies []time.Duration
	ctx := context.Background()

	for r := 0; r < 20; r++ {
		start := time.Now()

		rows, err := sqlDB.QueryContext(ctx, `
			SELECT DISTINCT m.memory_id, m.tier, m.status, m.decision, m.reasoning, m.created_at, m.last_cited_at
			FROM memory_record m
			LEFT JOIN memory_touches_file f ON m.memory_id = f.memory_id
			WHERE m.status != 'rejected'
			  AND (f.file_path = 'src/module_5.go' OR m.decision ILIKE '%database%')
			LIMIT 100;
		`)
		if err != nil {
			t.Fatalf("Benchmark query failed: %v", err)
		}

		for rows.Next() {
			var id, tier, status, dec, reas string
			var created time.Time
			var lastCited sql.NullTime
			_ = rows.Scan(&id, &tier, &status, &dec, &reas, &created, &lastCited)
		}
		rows.Close()

		latencies = append(latencies, time.Since(start))
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p95Index := int(float64(len(latencies))*0.95) - 1
	if p95Index < 0 {
		p95Index = 0
	}
	p95Latency := latencies[p95Index]

	t.Logf("Verified p95 retrieval latency over 5,000 records (Spec §9.2 scale): %v (Spec NFR target: < 2000ms)", p95Latency)

	if p95Latency > 2*time.Second {
		t.Fatalf("NFR Latency violation at 5,000 records: p95 latency %v exceeded 2 second target", p95Latency)
	}
}
