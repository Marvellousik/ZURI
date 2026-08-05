-- Migration 007: Connected Repositories Table
CREATE TABLE IF NOT EXISTS connected_repository (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    local_path TEXT NOT NULL UNIQUE,
    github_repo_full_name VARCHAR(255),
    default_branch VARCHAR(100) DEFAULT 'main',
    github_status VARCHAR(50) DEFAULT 'connected',
    indexing_status VARCHAR(50) DEFAULT 'indexed',
    health VARCHAR(50) DEFAULT 'healthy',
    last_synced_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
