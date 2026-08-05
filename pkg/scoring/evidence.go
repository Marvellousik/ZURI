package scoring

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// CurrentEvidenceFormulaVersion is the version identifier for evidence_strength calculation (§7.2, §1.3).
const CurrentEvidenceFormulaVersion = 1

// OnboardingBaselineEvidenceStrength is the fixed moderate baseline for founder-derived onboarding memory (§10.5).
const OnboardingBaselineEvidenceStrength = 0.65

// CalculateEvidenceStrength calculates the evidence strength score (0.0 to 1.0)
// combining status_weight, citation volume, recency, and source_type per §1.3 & §9.3.
func CalculateEvidenceStrength(tier, status, sourceType string, citationCount int, lastCitedAt *time.Time, createdAt time.Time) float64 {
	if sourceType == "onboarding_survey" && citationCount == 0 {
		return OnboardingBaselineEvidenceStrength
	}

	baseWeight := GetStatusWeight(tier, status)
	
	// Citation volume boost (logarithmic curve, max +0.2)
	citationBoost := 0.0
	if citationCount > 0 {
		citationBoost = 0.2 * (1.0 - math.Exp(-0.2*float64(citationCount)))
	}

	// Recency decay factor (30-day half life)
	recencyFactor := CalculateRecency(lastCitedAt, createdAt, DefaultHalfLifeDays)

	raw := baseWeight + citationBoost
	score := raw * (0.7 + (0.3 * recencyFactor))

	// Clamp between 0.0 and 1.0
	return math.Max(0.0, math.Min(1.0, score))
}

// RecomputeEvidenceStrength recalculates evidence_strength for a single memory record
// and updates the database row per §7.2.
func RecomputeEvidenceStrength(ctx context.Context, db *sql.DB, memoryID string) (float64, error) {
	var tier, status, sourceType string
	var citationCount int
	var lastCitedAt *time.Time
	var createdAt time.Time

	err := db.QueryRowContext(ctx, `
		SELECT tier, status, source_type, citation_count, last_cited_at, created_at
		FROM memory_record
		WHERE memory_id = $1;
	`, memoryID).Scan(&tier, &status, &sourceType, &citationCount, &lastCitedAt, &createdAt)

	if err != nil {
		return 0.0, fmt.Errorf("failed fetching memory record for evidence recomputation: %w", err)
	}

	score := CalculateEvidenceStrength(tier, status, sourceType, citationCount, lastCitedAt, createdAt)

	_, err = db.ExecContext(ctx, `
		UPDATE memory_record
		SET evidence_strength = $1,
		    evidence_strength_formula_version = $2
		WHERE memory_id = $3;
	`, score, CurrentEvidenceFormulaVersion, memoryID)

	if err != nil {
		return score, fmt.Errorf("failed updating evidence_strength: %w", err)
	}

	return score, nil
}
