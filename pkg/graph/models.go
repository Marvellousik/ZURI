package graph

import (
	"encoding/json"
	"time"
)

type NodeKind string

const (
	NodeKindRepository  NodeKind = "Repository"
	NodeKindService     NodeKind = "Service"
	NodeKindModule      NodeKind = "Module"
	NodeKindFunction    NodeKind = "Function"
	NodeKindAPIEndpoint NodeKind = "APIEndpoint"
)

type EdgeKind string

const (
	EdgeKindContains   EdgeKind = "CONTAINS"
	EdgeKindCalls      EdgeKind = "CALLS"
	EdgeKindImports    EdgeKind = "IMPORTS"
	EdgeKindInvokesAPI EdgeKind = "INVOKES_API"
	EdgeKindDefinesAPI EdgeKind = "DEFINES_API"
	EdgeKindDependsOn  EdgeKind = "DEPENDS_ON"
)

type GraphNode struct {
	ID         string         `json:"node_id"`
	RepoID     string         `json:"repo_id"`
	Kind       NodeKind       `json:"kind"`
	Name       string         `json:"name"`
	FilePath   string         `json:"file_path"`
	StartLine  int            `json:"start_line"`
	EndLine    int            `json:"end_line"`
	Language   string         `json:"language"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type GraphEdge struct {
	ID         string         `json:"edge_id,omitempty"`
	RepoID     string         `json:"repo_id"`
	SourceID   string         `json:"source_id"`
	TargetID   string         `json:"target_id"`
	EdgeKind   EdgeKind       `json:"edge_kind"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
}

type StructuralPathQuery struct {
	RepoID      string   `json:"repo_id"`
	TargetFiles []string `json:"target_files"`
	MaxDepth    int      `json:"max_depth"`
}

type StructuralDistanceMap map[string]float64 // file_path -> distance

type CypherQueryPayload struct {
	CypherQuery string         `json:"cypher_query"`
	Parameters  map[string]any `json:"parameters"`
}

func (n *GraphNode) PropertiesJSON() string {
	if n.Properties == nil {
		return "{}"
	}
	b, err := json.Marshal(n.Properties)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (e *GraphEdge) PropertiesJSON() string {
	if e.Properties == nil {
		return "{}"
	}
	b, err := json.Marshal(e.Properties)
	if err != nil {
		return "{}"
	}
	return string(b)
}
