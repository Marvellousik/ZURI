package gaps

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

type GapDetector struct {
	db *sql.DB
}

func NewGapDetector(db *sql.DB) *GapDetector {
	return &GapDetector{db: db}
}

// DetectConflictingConventions scans memory_record for decision_keys shared by multiple records with different decisions (§10.7).
func (d *GapDetector) DetectConflictingConventions(ctx context.Context, repoID string) (int, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT decision_key, ARRAY_AGG(memory_id::text) AS affected_ids, ARRAY_AGG(decision) AS decisions
		FROM memory_record
		WHERE repo_id = $1 AND decision_key IS NOT NULL AND status IN ('confirmed', 'proposed')
		GROUP BY decision_key
		HAVING COUNT(DISTINCT decision) > 1;
	`, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed querying conflicting conventions: %w", err)
	}
	defer rows.Close()

	detectedCount := 0
	for rows.Next() {
		var decisionKey string
		var affectedIDs, decisions []string

		if err := rows.Scan(&decisionKey, pq.Array(&affectedIDs), pq.Array(&decisions)); err != nil {
			continue
		}

		hypothesesJSON, _ := json.Marshal(decisions)
		var gapID string
		err = d.db.QueryRowContext(ctx, `
			INSERT INTO knowledge_gap (decision_key, scope, gap_type, candidate_hypotheses, affected_memory_ids, status)
			VALUES ($1, $2, 'conflicting_conventions', $3, $4::uuid[], 'open')
			RETURNING gap_id;
		`, decisionKey, repoID, string(hypothesesJSON), pq.Array(affectedIDs)).Scan(&gapID)

		if err == nil && gapID != "" {
			detectedCount++
			auditCtx, _ := json.Marshal(map[string]any{
				"gap_type":     "conflicting_conventions",
				"decision_key": decisionKey,
			})
			_, _ = d.db.ExecContext(ctx, `
				INSERT INTO audit_log (gap_id, event_type, actor, context)
				VALUES ($1, 'gap_detected', 'system_detector', $2);
			`, gapID, auditCtx)
		}
	}

	return detectedCount, nil
}

// DetectInsufficientEvidence scans memory_record for records with low evidence_strength (< 0.4) and proposed status (§10.7).
func (d *GapDetector) DetectInsufficientEvidence(ctx context.Context, repoID string) (int, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT memory_id, decision_key, decision
		FROM memory_record
		WHERE repo_id = $1 AND evidence_strength < 0.4 AND status = 'proposed';
	`, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed querying insufficient evidence records: %w", err)
	}
	defer rows.Close()

	detectedCount := 0
	for rows.Next() {
		var memoryID string
		var decisionKey, decision sql.NullString

		if err := rows.Scan(&memoryID, &decisionKey, &decision); err != nil {
			continue
		}

		dKey := "boundary:repo/concern:architecture/decision_type:convention"
		if decisionKey.Valid && decisionKey.String != "" {
			dKey = decisionKey.String
		}

		hypothesesJSON, _ := json.Marshal([]string{decision.String})
		var gapID string
		err = d.db.QueryRowContext(ctx, `
			INSERT INTO knowledge_gap (decision_key, scope, gap_type, candidate_hypotheses, affected_memory_ids, status)
			VALUES ($1, $2, 'insufficient_evidence', $3, ARRAY[$4]::uuid[], 'open')
			RETURNING gap_id;
		`, dKey, repoID, string(hypothesesJSON), memoryID).Scan(&gapID)

		if err == nil && gapID != "" {
			detectedCount++
			auditCtx, _ := json.Marshal(map[string]any{
				"gap_type":     "insufficient_evidence",
				"decision_key": dKey,
				"memory_id":    memoryID,
			})
			_, _ = d.db.ExecContext(ctx, `
				INSERT INTO audit_log (gap_id, memory_id, event_type, actor, context)
				VALUES ($1, $2, 'gap_detected', 'system_detector', $3);
			`, gapID, memoryID, auditCtx)
		}
	}

	return detectedCount, nil
}

// RunAllDetectors executes all gap detection queries for a given repository.
func (d *GapDetector) RunAllDetectors(ctx context.Context, repoID string) (int, error) {
	conflicts, err1 := d.DetectConflictingConventions(ctx, repoID)
	insufficient, err2 := d.DetectInsufficientEvidence(ctx, repoID)

	total := conflicts + insufficient
	if err1 != nil {
		return total, err1
	}
	if err2 != nil {
		return total, err2
	}
	return total, nil
}
