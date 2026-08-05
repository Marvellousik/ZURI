-- Migration 003: Spec v1.1 & RFC §7.4 Updates
-- Adds decision classification taxonomy columns (concern, decision_type, boundary),
-- dual confidence columns, model_calibration table, knowledge_gap table, and audit log extensions.

-- 1. Classification & confidence columns on memory_record
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS decision_key TEXT;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS concern TEXT;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS decision_type TEXT;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS boundary TEXT;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS model_id TEXT;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS extraction_confidence_raw DOUBLE PRECISION;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS extraction_confidence DOUBLE PRECISION;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS evidence_strength DOUBLE PRECISION NOT NULL DEFAULT 0.5;
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS evidence_strength_formula_version INT NOT NULL DEFAULT 1;

-- 2. Indexing for taxonomy lookup & gap detection
CREATE INDEX IF NOT EXISTS idx_memory_record_decision_key ON memory_record(decision_key);
CREATE INDEX IF NOT EXISTS idx_memory_record_concern ON memory_record(concern);

-- 3. Model Calibration table (keyed by model_id and concern per RFC §7.4)
CREATE TABLE IF NOT EXISTS model_calibration (
    model_id TEXT NOT NULL,
    concern TEXT NOT NULL,
    calibration_curve JSONB NOT NULL DEFAULT '{}'::jsonb,
    sample_size INT NOT NULL DEFAULT 0,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (model_id, concern)
);

-- 4. Knowledge Gap enums and table
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'knowledge_gap_type') THEN
        CREATE TYPE knowledge_gap_type AS ENUM (
            'conflicting_conventions',
            'insufficient_evidence',
            'unowned_decision',
            'stale_unreinforced'
        );
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'knowledge_gap_status') THEN
        CREATE TYPE knowledge_gap_status AS ENUM (
            'open',
            'surfaced',
            'answered',
            'acknowledged_unknown',
            'stale'
        );
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS knowledge_gap (
    gap_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_key TEXT NOT NULL,
    scope TEXT NOT NULL,
    gap_type knowledge_gap_type NOT NULL,
    candidate_hypotheses JSONB NOT NULL DEFAULT '[]'::jsonb,
    affected_memory_ids UUID[] NOT NULL DEFAULT '{}',
    status knowledge_gap_status NOT NULL DEFAULT 'open',
    routed_to TEXT[] NOT NULL DEFAULT '{}',
    detected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_surfaced_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_knowledge_gap_decision_key ON knowledge_gap(decision_key);
CREATE INDEX IF NOT EXISTS idx_knowledge_gap_status ON knowledge_gap(status);

-- 5. Audit log column for gap linking
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS gap_id UUID REFERENCES knowledge_gap(gap_id) ON DELETE SET NULL;
