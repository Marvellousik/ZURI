package model_test

import (
	"context"
	"testing"

	"zuri-daemon/pkg/model"
)

func TestModelRegistry_Router(t *testing.T) {
	reg := model.NewRegistry()
	router := model.NewRouter(reg)

	ctx := context.Background()

	// 1. Select Reasoning & Tool Calling LLM
	spec, err := router.SelectModel(ctx, model.ModelRequirement{
		NeedReasoning:   true,
		NeedToolCalling: true,
	})
	if err != nil {
		t.Fatalf("failed selecting reasoning model: %v", err)
	}

	if spec.Alias != "qwen-coder-local" {
		t.Errorf("expected default local model 'qwen-coder-local', got '%s'", spec.Alias)
	}

	// 2. Select Embedding Model
	embedSpec, err := router.SelectModel(ctx, model.ModelRequirement{
		NeedEmbedding: true,
	})
	if err != nil {
		t.Fatalf("failed selecting embedding model: %v", err)
	}

	if embedSpec.Alias != "bge-large-local" {
		t.Errorf("expected embedding model 'bge-large-local', got '%s'", embedSpec.Alias)
	}

	// 3. Hardware Profile Detection
	hw := model.DetectHardwareProfile()
	if hw.PhysicalCores <= 0 {
		t.Errorf("expected physical cores > 0, got %d", hw.PhysicalCores)
	}
}
