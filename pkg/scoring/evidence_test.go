package scoring

import (
	"testing"
	"time"
)

func TestCalculateEvidenceStrength(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		tier          string
		status        string
		sourceType    string
		citationCount int
		lastCitedAt   *time.Time
		createdAt     time.Time
		minScore      float64
		maxScore      float64
	}{
		{
			name:          "Canonical Confirmed High Citations",
			tier:          "canonical",
			status:        "confirmed",
			sourceType:    "pr_merge",
			citationCount: 10,
			lastCitedAt:   &now,
			createdAt:     now.Add(-24 * time.Hour),
			minScore:      0.85,
			maxScore:      1.0,
		},
		{
			name:          "Onboarding Survey Baseline",
			tier:          "canonical",
			status:        "confirmed",
			sourceType:    "onboarding_survey",
			citationCount: 0,
			lastCitedAt:   nil,
			createdAt:     now,
			minScore:      0.64,
			maxScore:      0.66,
		},
		{
			name:          "Probabilistic Lapsed Memory",
			tier:          "probabilistic",
			status:        "lapsed",
			sourceType:    "pr_merge",
			citationCount: 0,
			lastCitedAt:   nil,
			createdAt:     now.Add(-90 * 24 * time.Hour),
			minScore:      0.0,
			maxScore:      0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateEvidenceStrength(tt.tier, tt.status, tt.sourceType, tt.citationCount, tt.lastCitedAt, tt.createdAt)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected evidence_strength score in range [%f, %f], got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}
