package extraction

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
)

// MinSampleSizeForCalibration is the minimum number of resolved outcomes required before applying a calibration curve (§1.4).
const MinSampleSizeForCalibration = 10

// Calibrator defines the interface for correcting raw extraction confidence against per-model outcome curves.
type Calibrator interface {
	Calibrate(ctx context.Context, modelID string, concern string, rawConfidence float64) (float64, bool, error)
}

// DBCalibrator implements Calibrator against the `model_calibration` table (RFC §7.4).
type DBCalibrator struct {
	db *sql.DB
}

// NewDBCalibrator creates a new DBCalibrator.
func NewDBCalibrator(db *sql.DB) *DBCalibrator {
	return &DBCalibrator{db: db}
}

// Calibrate looks up calibration curves in `model_calibration` by (model_id, concern)
// and corrects rawConfidence against historical outcome performance per §1.4.
func (c *DBCalibrator) Calibrate(ctx context.Context, modelID string, concern string, rawConfidence float64) (float64, bool, error) {
	if c.db == nil {
		return rawConfidence, false, nil
	}

	var curveJSON string
	var sampleSize int
	err := c.db.QueryRowContext(ctx, `
		SELECT calibration_curve::text, sample_size
		FROM model_calibration
		WHERE model_id = $1 AND concern = $2;
	`, modelID, concern).Scan(&curveJSON, &sampleSize)

	if err == sql.ErrNoRows || sampleSize < MinSampleSizeForCalibration {
		return rawConfidence, false, nil
	}
	if err != nil {
		return rawConfidence, false, fmt.Errorf("failed querying model_calibration: %w", err)
	}

	var buckets map[string]float64
	if err := json.Unmarshal([]byte(curveJSON), &buckets); err != nil {
		return rawConfidence, false, nil
	}

	corrected := rawConfidence
	if val, ok := buckets[getBucketKey(rawConfidence)]; ok {
		corrected = val
	}

	corrected = math.Max(0.0, math.Min(1.0, corrected))
	return corrected, true, nil
}

func getBucketKey(raw float64) string {
	switch {
	case raw < 0.2:
		return "0.0-0.2"
	case raw < 0.4:
		return "0.2-0.4"
	case raw < 0.6:
		return "0.4-0.6"
	case raw < 0.8:
		return "0.6-0.8"
	default:
		return "0.8-1.0"
	}
}
