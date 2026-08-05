package contextengine_test

import (
	"context"
	"path/filepath"
	"testing"

	"zuri-daemon/pkg/contextengine"
	"zuri-daemon/pkg/storage"
)

func TestContextEngine_SynthesizeContext(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := storage.DefaultConfig()
	cfg.Mode = "local"
	cfg.SQLite.Path = filepath.Join(tmpDir, "test.db")

	vecStore, err := storage.NewVectorStore(cfg)
	if err != nil {
		t.Fatalf("failed initializing vector store: %v", err)
	}
	defer vecStore.Close()

	ctx := context.Background()
	_ = vecStore.CreateOrOpenIndex(ctx, "code_memory", 4, storage.MetricCosine)
	_ = vecStore.Insert(ctx, "code_memory", storage.VectorRecord{
		ID:      "mem-001",
		Vector:  []float32{0.1, 0.2, 0.3, 0.4},
		Payload: map[string]interface{}{"repo_id": "repo-auth", "summary": "JWT auth architecture decision"},
	})

	synthesizer := contextengine.NewSynthesizer(vecStore)
	payload, err := synthesizer.SynthesizeContext(ctx, "Why did we choose JWT for architecture?", "repo-auth", 4000)
	if err != nil {
		t.Fatalf("failed synthesizing context: %v", err)
	}

	if payload.Intent != contextengine.IntentArchitecturalDecision {
		t.Errorf("expected intent 'architectural_decision', got '%s'", payload.Intent)
	}

	if len(payload.Snippets) != 1 {
		t.Fatalf("expected 1 snippet packed, got %d", len(payload.Snippets))
	}

	if payload.Snippets[0].ID != "mem-001" {
		t.Errorf("expected snippet ID 'mem-001', got '%s'", payload.Snippets[0].ID)
	}
}
