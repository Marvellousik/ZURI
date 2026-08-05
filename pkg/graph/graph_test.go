package graph_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"zuri-daemon/pkg/db"
	"zuri-daemon/pkg/graph"
)

func TestCodeParserMultiLanguage(t *testing.T) {
	parser := graph.NewCodeParser()

	// 1. Test Go Parsing
	goCode := `package server

import "net/http"

func HandleUserRoute(w http.ResponseWriter, r *http.Request) {
	http.HandleFunc("/api/users", HandleUserRoute)
}
`
	nodes, edges, err := parser.ParseFile("repo-1", "pkg/server/main.go", goCode)
	if err != nil {
		t.Fatalf("Failed parsing Go file: %v", err)
	}

	if len(nodes) == 0 {
		t.Errorf("Expected extracted nodes from Go file, got 0")
	}
	if len(edges) == 0 {
		t.Errorf("Expected extracted edges from Go file, got 0")
	}

	foundGoFunc := false
	for _, n := range nodes {
		if n.Kind == graph.NodeKindFunction && n.Name == "HandleUserRoute" {
			foundGoFunc = true
		}
	}
	if !foundGoFunc {
		t.Errorf("Expected to find Go function 'HandleUserRoute'")
	}

	// 2. Test TypeScript Parsing
	tsCode := `import { fetchUser } from './api';

export async function getUser(id: string) {
	const res = await fetch("/api/users/" + id);
	return res.json();
}

app.get('/api/users', getUser);
`
	nodes, _, err = parser.ParseFile("repo-1", "src/user.ts", tsCode)
	if err != nil {
		t.Fatalf("Failed parsing TypeScript file: %v", err)
	}

	foundTSFunc := false
	foundTSAPI := false
	for _, n := range nodes {
		if n.Kind == graph.NodeKindFunction && n.Name == "getUser" {
			foundTSFunc = true
		}
		if n.Kind == graph.NodeKindAPIEndpoint && (n.Name == "GET /api/users" || n.Name == "/api/users/") {
			foundTSAPI = true
		}
	}
	if !foundTSFunc {
		t.Errorf("Expected to find TS function 'getUser'")
	}
	if !foundTSAPI {
		t.Errorf("Expected to find TS API endpoint")
	}

	// 3. Test Python Parsing
	pyCode := `import requests

@app.get('/api/health')
def health_check():
    return {"status": "ok"}
`
	nodes, _, err = parser.ParseFile("repo-1", "service/app.py", pyCode)
	if err != nil {
		t.Fatalf("Failed parsing Python file: %v", err)
	}

	foundPyFunc := false
	for _, n := range nodes {
		if n.Kind == graph.NodeKindFunction && n.Name == "health_check" {
			foundPyFunc = true
		}
	}
	if !foundPyFunc {
		t.Errorf("Expected to find Python function 'health_check'")
	}
}

func TestGraphStoreAndProximityBoosting(t *testing.T) {
	os.Setenv("ZURI_DISABLE_PGVECTOR_VALIDATION_FOR_TESTS", "1")
	os.Setenv("ZURI_DB_PORT", "5499")

	tmpDir, err := os.MkdirTemp("", "zuri_graph_test_*")
	if err != nil {
		t.Fatalf("Failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	os.Setenv("ZURI_DB_PATH", filepath.Join(tmpDir, "db"))

	dbMgr := db.NewDBManager()
	if err := dbMgr.Init(); err != nil {
		t.Fatalf("Failed starting db: %v", err)
	}
	defer dbMgr.Close()

	if _, err := db.RunMigrations(dbMgr.GetDB()); err != nil {
		t.Fatalf("Failed running migrations: %v", err)
	}

	store := graph.NewPostgresGraphStore(dbMgr.GetDB())
	booster := graph.NewProximityBooster(store, dbMgr.GetDB())
	ctx := context.Background()

	repoID := "test-repo-graph"

	// Create sample nodes
	nodes := []graph.GraphNode{
		{
			ID:       "node1",
			RepoID:   repoID,
			Kind:     graph.NodeKindModule,
			Name:     "main.go",
			FilePath: "cmd/main.go",
			Language: "go",
		},
		{
			ID:       "node2",
			RepoID:   repoID,
			Kind:     graph.NodeKindModule,
			Name:     "service.go",
			FilePath: "pkg/service/service.go",
			Language: "go",
		},
		{
			ID:       "node3",
			RepoID:   repoID,
			Kind:     graph.NodeKindModule,
			Name:     "db.go",
			FilePath: "pkg/db/db.go",
			Language: "go",
		},
	}

	// Create sample edges: main.go -> service.go -> db.go
	edges := []graph.GraphEdge{
		{
			RepoID:   repoID,
			SourceID: "node1",
			TargetID: "node2",
			EdgeKind: graph.EdgeKindCalls,
		},
		{
			RepoID:   repoID,
			SourceID: "node2",
			TargetID: "node3",
			EdgeKind: graph.EdgeKindCalls,
		},
	}

	if err := store.SaveNodesAndEdges(ctx, nodes, edges); err != nil {
		t.Fatalf("Failed saving nodes and edges: %v", err)
	}

	// Verify Cypher payload generator
	cypherPayload := store.GenerateCypherQuery(repoID, []string{"cmd/main.go"}, 3)
	if cypherPayload.CypherQuery == "" {
		t.Errorf("Expected non-empty Cypher query string")
	}

	// 1. Test distance to self (0)
	dist0, err := store.CalculateStructuralDistance(ctx, repoID, []string{"cmd/main.go"}, "cmd/main.go")
	if err != nil || dist0 != 0.0 {
		t.Errorf("Expected distance 0.0, got %f, err: %v", dist0, err)
	}

	// 2. Test 1-hop distance
	dist1, err := store.CalculateStructuralDistance(ctx, repoID, []string{"cmd/main.go"}, "pkg/service/service.go")
	if err != nil || dist1 != 1.0 {
		t.Errorf("Expected distance 1.0, got %f, err: %v", dist1, err)
	}

	// 3. Test 2-hop distance
	dist2, err := store.CalculateStructuralDistance(ctx, repoID, []string{"cmd/main.go"}, "pkg/db/db.go")
	if err != nil || dist2 != 2.0 {
		t.Errorf("Expected distance 2.0, got %f, err: %v", dist2, err)
	}

	// 4. Test Proximity Boosting
	baseScore := 0.80

	// Exact match boost (1.5x)
	boosted0, mult0, _ := booster.ApplyProximityBoost(ctx, repoID, "", "cmd/main.go", baseScore, []string{"cmd/main.go"})
	if math.Abs(mult0-1.50) > 0.001 || math.Abs(boosted0-1.20) > 0.001 {
		t.Errorf("Expected multiplier 1.50 and boosted score 1.20, got %f and %f", mult0, boosted0)
	}

	// 1-hop boost (1.25x)
	boosted1, mult1, _ := booster.ApplyProximityBoost(ctx, repoID, "", "pkg/service/service.go", baseScore, []string{"cmd/main.go"})
	if mult1 != 1.25 || math.Abs(boosted1-1.0) > 0.001 {
		t.Errorf("Expected multiplier 1.25 and boosted score 1.0, got %f and %f", mult1, boosted1)
	}

	// Unreachable file boost (1.0x)
	boostedUnreach, multUnreach, _ := booster.ApplyProximityBoost(ctx, repoID, "", "unreachable.go", baseScore, []string{"cmd/main.go"})
	if multUnreach != 1.0 || boostedUnreach != 0.80 {
		t.Errorf("Expected multiplier 1.0 and boosted score 0.80, got %f and %f", multUnreach, boostedUnreach)
	}
}
