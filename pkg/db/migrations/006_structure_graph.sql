-- Migration 006: Structure Graph Schema & Apache AGE Nodes/Edges (§17)

CREATE TABLE IF NOT EXISTS graph_node (
    node_id TEXT PRIMARY KEY,
    repo_id TEXT NOT NULL,
    kind TEXT NOT NULL, -- Repository, Service, Module, Function, APIEndpoint
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    start_line INT DEFAULT 0,
    end_line INT DEFAULT 0,
    language TEXT NOT NULL DEFAULT 'unknown',
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_graph_node_repo_file ON graph_node(repo_id, file_path);
CREATE INDEX IF NOT EXISTS idx_graph_node_kind ON graph_node(kind);

CREATE TABLE IF NOT EXISTS graph_edge (
    edge_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id TEXT NOT NULL,
    source_id TEXT NOT NULL REFERENCES graph_node(node_id) ON DELETE CASCADE,
    target_id TEXT NOT NULL REFERENCES graph_node(node_id) ON DELETE CASCADE,
    edge_kind TEXT NOT NULL, -- CONTAINS, CALLS, IMPORTS, INVOKES_API, DEFINES_API, DEPENDS_ON
    properties JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_graph_edge_source ON graph_edge(source_id);
CREATE INDEX IF NOT EXISTS idx_graph_edge_target ON graph_edge(target_id);
CREATE INDEX IF NOT EXISTS idx_graph_edge_repo_kind ON graph_edge(repo_id, edge_kind);
