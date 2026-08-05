package gaps

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// KnowledgeGapDigestItem represents a single gap formatted for periodic team digest notification (§10.7).
type KnowledgeGapDigestItem struct {
	GapID               string   `json:"gap_id"`
	DecisionKey         string   `json:"decision_key"`
	Scope               string   `json:"scope"`
	GapType             string   `json:"gap_type"`
	CandidateHypotheses []string `json:"candidate_hypotheses"`
	RoutedTo            []string `json:"routed_to"`
	DetectedAt          string   `json:"detected_at"`
}

// GenerateGapDigest collects open/surfaced gaps for a repository, updates status to 'surfaced', and emits audit log events.
func GenerateGapDigest(ctx context.Context, db *sql.DB, repoID string) ([]KnowledgeGapDigestItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT gap_id, decision_key, scope, gap_type, candidate_hypotheses::text, routed_to, detected_at
		FROM knowledge_gap
		WHERE scope = $1 AND status IN ('open', 'surfaced')
		ORDER BY detected_at DESC;
	`, repoID)

	if err != nil {
		return nil, fmt.Errorf("failed querying gaps for digest: %w", err)
	}
	defer rows.Close()

	var digest []KnowledgeGapDigestItem
	var surfacedIDs []string

	for rows.Next() {
		var item KnowledgeGapDigestItem
		var hypJSON string
		var detectedAt time.Time
		var routed []string

		if err := rows.Scan(&item.GapID, &item.DecisionKey, &item.Scope, &item.GapType, &hypJSON, pq.Array(&routed), &detectedAt); err != nil {
			continue
		}

		var hypotheses []string
		_ = json.Unmarshal([]byte(hypJSON), &hypotheses)

		item.CandidateHypotheses = hypotheses
		item.RoutedTo = routed
		item.DetectedAt = detectedAt.UTC().Format(time.RFC3339)

		digest = append(digest, item)
		surfacedIDs = append(surfacedIDs, item.GapID)
	}

	if len(surfacedIDs) > 0 {
		now := time.Now().UTC()
		_, err = db.ExecContext(ctx, `
			UPDATE knowledge_gap
			SET status = 'surfaced', last_surfaced_at = $1
			WHERE gap_id = ANY($2::uuid[]);
		`, now, pq.Array(surfacedIDs))

		if err == nil {
			for _, id := range surfacedIDs {
				auditCtx, _ := json.Marshal(map[string]any{
					"action": "digest_surfaced",
				})
				_, _ = db.ExecContext(ctx, `
					INSERT INTO audit_log (gap_id, event_type, actor, context, occurred_at)
					VALUES ($1, 'gap_surfaced', 'digest_engine', $2, $3);
				`, id, auditCtx, now)
			}
		}
	}

	return digest, nil
}
