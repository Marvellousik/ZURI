package orchestrator_test

import (
	"context"
	"path/filepath"
	"testing"

	"zuri-daemon/pkg/contextengine"
	"zuri-daemon/pkg/model"
	"zuri-daemon/pkg/orchestrator"
	"zuri-daemon/pkg/session"
	"zuri-daemon/pkg/storage"
)

func TestOrchestrator_ExecuteTask(t *testing.T) {
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

	synthesizer := contextengine.NewSynthesizer(vecStore)
	sessMgr, err := session.NewManager(filepath.Join(tmpDir, "sessions"))
	if err != nil {
		t.Fatalf("failed creating session manager: %v", err)
	}

	reg := model.NewRegistry()
	router := model.NewRouter(reg)

	orch := orchestrator.NewOrchestrator(synthesizer, sessMgr, router)

	sess, err := sessMgr.CreateSession(ctx, "task-1", "ws-core", "Implement key rotation")
	if err != nil {
		t.Fatalf("failed creating session: %v", err)
	}

	resp, err := orch.ExecuteTask(ctx, sess.ID, "Implement key rotation")
	if err != nil {
		t.Fatalf("failed executing task: %v", err)
	}

	if resp == "" {
		t.Error("expected non-empty response from orchestrator, got empty")
	}
}
