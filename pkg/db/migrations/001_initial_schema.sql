-- Migration 001: Initial Zuri Database Schema
-- Requires pgvector extension, creates enums, tables, triggers, and indexes per section 7.2 of spec.

DO $$
BEGIN
    BEGIN
        CREATE EXTENSION IF NOT EXISTS vector;
    EXCEPTION WHEN OTHERS THEN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'vector') THEN
            CREATE DOMAIN vector AS float8[];
        END IF;
    END;
END $$;

-- Enums for memory classification and auditing
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'memory_tier') THEN
        CREATE TYPE memory_tier AS ENUM ('canonical', 'probabilistic', 'working');
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'memory_status') THEN
        CREATE TYPE memory_status AS ENUM ('proposed', 'confirmed', 'rejected', 'lapsed');
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'audit_event_type') THEN
        CREATE TYPE audit_event_type AS ENUM ('retrieved', 'confirmed', 'rejected', 'edited', 'lapsed', 'revival_flagged');
    END IF;
END $$;

-- Repository registry table
CREATE TABLE IF NOT EXISTS repo (
    repo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    github_installation_id BIGINT NOT NULL,
    github_repo_full_name TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-repository configuration table
CREATE TABLE IF NOT EXISTS zuri_config (
    repo_id UUID PRIMARY KEY REFERENCES repo(repo_id) ON DELETE CASCADE,
    approver_usernames TEXT[] NOT NULL DEFAULT '{}',
    expiry_days INT NOT NULL DEFAULT 60,
    reminder_cadence_days INT NOT NULL DEFAULT 7,
    additional_notify_channels JSONB NOT NULL DEFAULT '{}'::jsonb
);

-- Shared memory record table for all tiers
CREATE TABLE IF NOT EXISTS memory_record (
    memory_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id UUID NOT NULL REFERENCES repo(repo_id) ON DELETE CASCADE,
    tier memory_tier NOT NULL,
    status memory_status NOT NULL,
    decision TEXT NOT NULL,
    reasoning TEXT NOT NULL,
    content_embedding vector,
    originating_commit TEXT NOT NULL,
    originating_pr_number INT,
    created_by TEXT NOT NULL,
    resolved_by TEXT,
    branch_label TEXT,
    decision_title TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    citation_count INT NOT NULL DEFAULT 0,
    last_cited_at TIMESTAMPTZ
);

-- Join table for memory to file paths
CREATE TABLE IF NOT EXISTS memory_touches_file (
    memory_id UUID NOT NULL REFERENCES memory_record(memory_id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    PRIMARY KEY (memory_id, file_path)
);

-- Citation edge table for graph-based centrality scoring
CREATE TABLE IF NOT EXISTS memory_citation (
    citation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    citing_pr_number INT NOT NULL,
    cited_memory_id UUID NOT NULL REFERENCES memory_record(memory_id) ON DELETE CASCADE,
    cited_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id UUID REFERENCES memory_record(memory_id) ON DELETE SET NULL,
    event_type audit_event_type NOT NULL,
    actor TEXT NOT NULL,
    context JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Trigger function to update citation_count and last_cited_at on memory_record
CREATE OR REPLACE FUNCTION update_memory_citation_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE memory_record
    SET citation_count = citation_count + 1,
        last_cited_at = NEW.cited_at
    WHERE memory_id = NEW.cited_memory_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_memory_citation_inserted ON memory_citation;
CREATE TRIGGER trg_memory_citation_inserted
AFTER INSERT ON memory_citation
FOR EACH ROW
EXECUTE FUNCTION update_memory_citation_stats();

-- Performance indexes for lookup and graph traversal
CREATE INDEX IF NOT EXISTS idx_memory_touches_file_path ON memory_touches_file(file_path);
CREATE INDEX IF NOT EXISTS idx_memory_record_repo_id ON memory_record(repo_id);
CREATE INDEX IF NOT EXISTS idx_memory_record_status ON memory_record(status);
CREATE INDEX IF NOT EXISTS idx_memory_record_created_by_branch ON memory_record(created_by, branch_label);
CREATE INDEX IF NOT EXISTS idx_memory_citation_cited_memory ON memory_citation(cited_memory_id);
