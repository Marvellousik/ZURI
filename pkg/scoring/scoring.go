package scoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"time"
)

type StatusWeightKey struct {
	Tier   string
	Status string
}

// StatusWeightMap defines the configurable status_weight lookup table per section 9.3 of spec.
var StatusWeightMap = map[StatusWeightKey]float64{
	{Tier: "canonical", Status: "confirmed"}:     1.0,
	{Tier: "canonical", Status: "proposed"}:      1.0, // Canonical implicitly confirmed
	{Tier: "probabilistic", Status: "confirmed"}: 0.8,
	{Tier: "probabilistic", Status: "proposed"}:  0.6,
	{Tier: "working", Status: "proposed"}:        0.4,
	{Tier: "working", Status: "confirmed"}:       0.4,
	{Tier: "probabilistic", Status: "lapsed"}:    0.1,
	{Tier: "canonical", Status: "lapsed"}:        0.1,
	{Tier: "working", Status: "lapsed"}:          0.1,
	{Tier: "probabilistic", Status: "rejected"}:  0.0,
	{Tier: "canonical", Status: "rejected"}:      0.0,
	{Tier: "working", Status: "rejected"}:        0.0,
}

// DefaultHalfLifeDays specifies the 30-day half-life for exponential recency decay.
// 30 days aligns with monthly engineering sprint cycles and release cadences.
const DefaultHalfLifeDays = 30.0

// MaxTrendMultiplier caps the maximum trend boost multiplier to 3.0x.
// This allows sleeper decisions with new citations to receive a strong positive boost
// without letting high citation spikes overwhelm relevance and tier weights.
const MaxTrendMultiplier = 3.0

func GetStatusWeight(tier, status string) float64 {
	key := StatusWeightKey{Tier: tier, Status: status}
	if weight, exists := StatusWeightMap[key]; exists {
		return weight
	}
	return 0.5
}

func CalculateRelevance(stage1Similarity float64) float64 {
	if stage1Similarity < 0.0 {
		return 0.0
	}
	if stage1Similarity > 1.0 {
		return 1.0
	}
	return stage1Similarity
}

func CalculateTrend(recentCitations, priorCitations int) float64 {
	recent := float64(recentCitations)
	prior := float64(priorCitations)

	// Add Laplace smoothing (+1.0 to denominator) to prevent division by zero when prior_citations is 0.
	// For example: prior=0, recent=5 -> (5+1)/(0+1) = 6.0, capped at MaxTrendMultiplier (3.0).
	ratio := (recent + 1.0) / (prior + 1.0)

	if ratio > MaxTrendMultiplier {
		return MaxTrendMultiplier
	}
	return ratio
}

func CalculateRecency(lastCitedAt *time.Time, createdAt time.Time, halfLifeDays float64) float64 {
	refTime := createdAt
	if lastCitedAt != nil && !lastCitedAt.IsZero() {
		refTime = *lastCitedAt
	}

	daysElapsed := time.Since(refTime).Hours() / 24.0
	if daysElapsed < 0.0 {
		daysElapsed = 0.0
	}

	// Exponential decay formula: N(t) = exp(-lambda * t) where lambda = ln(2) / half_life
	lambda := math.Ln2 / halfLifeDays
	decay := math.Exp(-lambda * daysElapsed)
	return decay
}

func CalculateFinalScore(relevance, trend, statusWeight, recency float64) float64 {
	score := relevance * trend * statusWeight * recency
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0.0
	}
	return score
}

// CheckAndFlagRevival inspects if a lapsed record's trend inverts to rising (> 1.0)
// and writes an audit_log entry with event_type = 'revival_flagged' per section 9.5 of spec.
func CheckAndFlagRevival(ctx context.Context, db *sql.DB, memoryID string, status string, trend float64) error {
	if status != "lapsed" || trend <= 1.0 {
		return nil
	}

	// Verify if already flagged recently to prevent log spam
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM audit_log
			WHERE memory_id = $1 AND event_type = 'revival_flagged'
			  AND occurred_at > now() - INTERVAL '7 days'
		);
	`, memoryID).Scan(&exists)

	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	auditCtx, _ := json.Marshal(map[string]any{
		"message": "Abandoned decision is accumulating new citations and rising in trend",
		"trend":   trend,
		"status":  status,
	})

	_, err = db.ExecContext(ctx, `
		INSERT INTO audit_log (memory_id, event_type, actor, context, occurred_at)
		VALUES ($1, 'revival_flagged', 'system', $2, now());
	`, memoryID, auditCtx)

	return err
}
