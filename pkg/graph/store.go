package graph

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/lib/pq"
)

type GraphStore interface {
	SaveNodesAndEdges(ctx context.Context, nodes []GraphNode, edges []GraphEdge) error
	CalculateStructuralDistance(ctx context.Context, repoID string, targetFiles []string, candidateFile string) (float64, error)
	GenerateCypherQuery(repoID string, targetFiles []string, maxDepth int) CypherQueryPayload
	QueryConnectedNodes(ctx context.Context, repoID string, filePath string) ([]GraphNode, error)
}

type PostgresGraphStore struct {
	db *sql.DB
}

func NewPostgresGraphStore(db *sql.DB) *PostgresGraphStore {
	return &PostgresGraphStore{db: db}
}

// SaveNodesAndEdges inserts or updates structural graph nodes and edges in PostgreSQL / Apache AGE (§17.2).
func (s *PostgresGraphStore) SaveNodesAndEdges(ctx context.Context, nodes []GraphNode, edges []GraphEdge) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed starting transaction: %w", err)
	}
	defer tx.Rollback()

	for _, n := range nodes {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO graph_node (node_id, repo_id, kind, name, file_path, start_line, end_line, language, properties, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, now())
			ON CONFLICT (node_id) DO UPDATE SET
				name = EXCLUDED.name,
				file_path = EXCLUDED.file_path,
				start_line = EXCLUDED.start_line,
				end_line = EXCLUDED.end_line,
				properties = EXCLUDED.properties,
				updated_at = now();
		`, n.ID, n.RepoID, string(n.Kind), n.Name, n.FilePath, n.StartLine, n.EndLine, n.Language, n.PropertiesJSON())
		if err != nil {
			return fmt.Errorf("failed upserting graph node %s: %w", n.ID, err)
		}
	}

	for _, e := range edges {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO graph_edge (repo_id, source_id, target_id, edge_kind, properties)
			VALUES ($1, $2, $3, $4, $5::jsonb);
		`, e.RepoID, e.SourceID, e.TargetID, string(e.EdgeKind), e.PropertiesJSON())
		if err != nil {
			// Ignore foreign key or duplicate errors on edge insertion for external references
			continue
		}
	}

	return tx.Commit()
}

// GenerateCypherQuery produces an Apache AGE Cypher graph query payload for graph traversal (§17.2, §17.7).
func (s *PostgresGraphStore) GenerateCypherQuery(repoID string, targetFiles []string, maxDepth int) CypherQueryPayload {
	if maxDepth <= 0 {
		maxDepth = 3
	}

	cypher := fmt.Sprintf(`
		MATCH (source:Node {repo_id: $repo_id})-[r:CALLS|IMPORTS|CONTAINS|INVOKES_API*1..%d]-(target:Node)
		WHERE source.file_path IN $target_files
		RETURN source.file_path AS source_file, target.file_path AS target_file, length(r) AS distance
		ORDER BY distance ASC
	`, maxDepth)

	return CypherQueryPayload{
		CypherQuery: cypher,
		Parameters: map[string]any{
			"repo_id":      repoID,
			"target_files": targetFiles,
			"max_depth":    maxDepth,
		},
	}
}

// CalculateStructuralDistance calculates the shortest graph traversal distance between candidate file and query target files (§17.7).
func (s *PostgresGraphStore) CalculateStructuralDistance(ctx context.Context, repoID string, targetFiles []string, candidateFile string) (float64, error) {
	if len(targetFiles) == 0 {
		return math.Inf(1), nil
	}

	// Exact file match = distance 0
	for _, tf := range targetFiles {
		if tf == candidateFile {
			return 0.0, nil
		}
	}

	// BFS traversal via SQL graph edge table (simulating Apache AGE graph traversal)
	row := s.db.QueryRowContext(ctx, `
		WITH RECURSIVE graph_path AS (
			-- Base case: direct nodes matching candidate file
			SELECT n.node_id, n.file_path, 0 AS depth
			FROM graph_node n
			WHERE n.repo_id = $1 AND n.file_path = $2

			UNION ALL

			-- Recursive case: traversals across graph_edge
			SELECT target.node_id, target.file_path, gp.depth + 1
			FROM graph_path gp
			JOIN graph_edge e ON (e.source_id = gp.node_id OR e.target_id = gp.node_id)
			JOIN graph_node target ON (
				CASE WHEN e.source_id = gp.node_id THEN e.target_id ELSE e.source_id END = target.node_id
			)
			WHERE gp.depth < 5 AND target.repo_id = $1
		)
		SELECT MIN(depth)
		FROM graph_path
		WHERE file_path = ANY($3::text[]);
	`, repoID, candidateFile, pq.Array(targetFiles))

	var minDistance sql.NullInt64
	if err := row.Scan(&minDistance); err != nil {
		if err == sql.ErrNoRows {
			return math.Inf(1), nil
		}
		return math.Inf(1), nil
	}

	if !minDistance.Valid {
		return math.Inf(1), nil
	}

	return float64(minDistance.Int64), nil
}

// QueryConnectedNodes returns all structural graph nodes connected to a given file (§17.7).
func (s *PostgresGraphStore) QueryConnectedNodes(ctx context.Context, repoID string, filePath string) ([]GraphNode, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT target.node_id, target.repo_id, target.kind, target.name, target.file_path, target.start_line, target.end_line, target.language, target.properties
		FROM graph_node src
		JOIN graph_edge e ON (e.source_id = src.node_id OR e.target_id = src.node_id)
		JOIN graph_node target ON (
			CASE WHEN e.source_id = src.node_id THEN e.target_id ELSE e.source_id END = target.node_id
		)
		WHERE src.repo_id = $1 AND src.file_path = $2;
	`, repoID, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed querying connected nodes: %w", err)
	}
	defer rows.Close()

	var nodes []GraphNode
	for rows.Next() {
		var n GraphNode
		var kindStr string
		var propsJSON []byte
		if err := rows.Scan(&n.ID, &n.RepoID, &kindStr, &n.Name, &n.FilePath, &n.StartLine, &n.EndLine, &n.Language, &propsJSON); err != nil {
			continue
		}
		n.Kind = NodeKind(kindStr)
		nodes = append(nodes, n)
	}

	return nodes, nil
}
