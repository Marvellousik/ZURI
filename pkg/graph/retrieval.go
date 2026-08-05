package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
)

// ProximityBooster handles combining semantic similarity scores with structural graph distance (§17.7, §9.3).
type ProximityBooster struct {
	store GraphStore
	db    *sql.DB
}

func NewProximityBooster(store GraphStore, db *sql.DB) *ProximityBooster {
	return &ProximityBooster{store: store, db: db}
}

// CalculateProximityMultiplier returns a score multiplier based on graph structural distance (§17.7).
// Distance = 0 (exact file match) -> 1.50
// Distance = 1 (1-hop call/import neighbor) -> 1.25
// Distance = 2 (2-hop structural neighbor) -> 1.17
// Distance = 3 -> 1.125
// Distance = ∞ / unreachable -> 1.00
func CalculateProximityMultiplier(distance float64) float64 {
	if math.IsInf(distance, 1) || distance < 0 {
		return 1.0
	}
	const maxBoost = 0.50
	return 1.0 + (maxBoost / (1.0 + distance))
}

// ApplyProximityBoost computes the structural proximity boost for a candidate memory record given scope files (§17.7).
func (b *ProximityBooster) ApplyProximityBoost(ctx context.Context, repoID string, memoryID string, candidateFile string, semanticScore float64, scopeFiles []string) (float64, float64, error) {
	if len(scopeFiles) == 0 || candidateFile == "" {
		return semanticScore, 1.0, nil
	}

	distance, err := b.store.CalculateStructuralDistance(ctx, repoID, scopeFiles, candidateFile)
	if err != nil {
		distance = math.Inf(1)
	}

	multiplier := CalculateProximityMultiplier(distance)
	boostedScore := semanticScore * multiplier

	// Log graph traversal decision in audit_log if boost was applied
	if multiplier > 1.0 && b.db != nil {
		auditCtx, _ := json.Marshal(map[string]any{
			"memory_id":       memoryID,
			"candidate_file":  candidateFile,
			"scope_files":     scopeFiles,
			"distance":        distance,
			"semantic_score":  semanticScore,
			"multiplier":      multiplier,
			"boosted_score":   boostedScore,
		})
		var memIDArg any = memoryID
		if memoryID == "" {
			memIDArg = nil
		}
		_, _ = b.db.ExecContext(ctx, `
			INSERT INTO audit_log (memory_id, event_type, actor, context)
			VALUES ($1, 'graph_traversal', 'proximity_booster', $2);
		`, memIDArg, auditCtx)
	}

	return boostedScore, multiplier, nil
}
