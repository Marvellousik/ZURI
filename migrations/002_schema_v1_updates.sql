-- Migration 002: Schema v1 Updates
-- Adds repo.local_path, memory_source_type enum, memory_record.source_type column, and memory_applies_to_repo join table.

-- Add local_path column to repo table
ALTER TABLE repo ADD COLUMN IF NOT EXISTS local_path TEXT NOT NULL DEFAULT '';

-- Add enum for memory source type
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'memory_source_type') THEN
        CREATE TYPE memory_source_type AS ENUM ('pr_merge', 'onboarding_survey', 'agent_session');
    END IF;
END $$;

-- Add source_type column to memory_record table
ALTER TABLE memory_record ADD COLUMN IF NOT EXISTS source_type memory_source_type NOT NULL DEFAULT 'pr_merge';

-- Join table for cross-repo integration decisions
CREATE TABLE IF NOT EXISTS memory_applies_to_repo (
    memory_id UUID NOT NULL REFERENCES memory_record(memory_id) ON DELETE CASCADE,
    repo_id UUID NOT NULL REFERENCES repo(repo_id) ON DELETE CASCADE,
    PRIMARY KEY (memory_id, repo_id)
);

CREATE INDEX IF NOT EXISTS idx_memory_applies_to_repo_repo_id ON memory_applies_to_repo(repo_id);
