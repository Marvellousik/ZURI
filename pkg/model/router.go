package model

import (
	"context"
	"fmt"
	"sync"
)

// ProviderType specifies the provider backends.
type ProviderType string

const (
	ProviderOllama    ProviderType = "ollama"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderOpenAI    ProviderType = "openai"
	ProviderLocalONNX ProviderType = "local_onnx"
	ProviderLocalGGUF ProviderType = "local_gguf"
)

// ModelSpec encapsulates metadata, provider details, and capabilities for a model.
type ModelSpec struct {
	ID           string       `json:"id"`
	Alias        string       `json:"alias"`
	Provider     ProviderType `json:"provider"`
	ModelName    string       `json:"model_name"`
	Capabilities Capabilities `json:"capabilities"`
	VRAMRequired int          `json:"vram_required_mb"`
	IsDefault    bool         `json:"is_default"`
}

// Registry manages registered models and profiles.
type Registry struct {
	mu     sync.RWMutex
	models map[string]ModelSpec // alias -> spec
}

// NewRegistry initializes a new Registry instance.
func NewRegistry() *Registry {
	r := &Registry{
		models: make(map[string]ModelSpec),
	}
	r.seedDefaultModels()
	return r
}

func (r *Registry) seedDefaultModels() {
	r.models["qwen-coder-local"] = ModelSpec{
		ID:        "qwen2.5-coder-7b",
		Alias:     "qwen-coder-local",
		Provider:  ProviderLocalGGUF,
		ModelName: "qwen2.5-coder-7b-instruct-q4_k_m.gguf",
		Capabilities: Capabilities{
			Streaming:        true,
			ToolCalling:      true,
			StructuredOutput: true,
			Reasoning:        true,
			MaxContextWindow: 16384,
		},
		VRAMRequired: 5800,
		IsDefault:    true,
	}

	r.models["bge-large-local"] = ModelSpec{
		ID:        "bge-large-en-v1.5",
		Alias:     "bge-large-local",
		Provider:  ProviderLocalONNX,
		ModelName: "bge-large-en-v1.5",
		Capabilities: Capabilities{
			Embedding:        true,
			MaxContextWindow: 512,
		},
		VRAMRequired: 450,
		IsDefault:    true,
	}

	r.models["claude-sonnet"] = ModelSpec{
		ID:        "claude-3-5-sonnet",
		Alias:     "claude-sonnet",
		Provider:  ProviderAnthropic,
		ModelName: "claude-3-5-sonnet-20241022",
		Capabilities: Capabilities{
			Streaming:        true,
			ToolCalling:      true,
			StructuredOutput: true,
			Reasoning:        true,
			Vision:           true,
			MaxContextWindow: 200000,
		},
		VRAMRequired: 0,
		IsDefault:    false,
	}
}

// Register adds or updates a model specification in the registry.
func (r *Registry) Register(spec ModelSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[spec.Alias] = spec
}

// ListModels returns all registered model specifications.
func (r *Registry) ListModels() []ModelSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []ModelSpec
	for _, spec := range r.models {
		list = append(list, spec)
	}
	return list
}

// Router routes tasks to the optimal registered model satisfying requirements.
type Router struct {
	registry *Registry
}

// NewRouter creates a new Router instance.
func NewRouter(registry *Registry) *Router {
	return &Router{registry: registry}
}

// SelectModel finds the best matching model for the given task requirement.
func (rt *Router) SelectModel(ctx context.Context, req ModelRequirement) (*ModelSpec, error) {
	rt.registry.mu.RLock()
	defer rt.registry.mu.RUnlock()

	var bestMatch *ModelSpec
	for _, spec := range rt.registry.models {
		if spec.Capabilities.Satisfies(req) {
			if bestMatch == nil || (spec.IsDefault && !bestMatch.IsDefault) {
				s := spec
				bestMatch = &s
			}
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("router: no registered model satisfies requirement (min_ctx: %d, tool: %v, reasoning: %v)",
			req.MinContextWindow, req.NeedToolCalling, req.NeedReasoning)
	}

	return bestMatch, nil
}
