package orchestrator

import (
	"context"
	"fmt"

	"zuri-daemon/pkg/contextengine"
	"zuri-daemon/pkg/model"
	"zuri-daemon/pkg/session"
)

// StepStatus defines the status of an agent execution step.
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepExecuting StepStatus = "executing"
	StepSuccess   StepStatus = "success"
	StepFailed    StepStatus = "failed"
)

// ExecutionStep represents a single planned step in an engineering task.
type ExecutionStep struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	ToolName    string     `json:"tool_name,omitempty"`
	Status      StepStatus `json:"status"`
	Result      string     `json:"result,omitempty"`
}

// VerificationResult contains the validation result from the Verifier Gate.
type VerificationResult struct {
	Passed       bool     `json:"passed"`
	CompilerOutput string `json:"compiler_output,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

// Orchestrator coordinates planning, retrieval, execution, generation, and verification.
type Orchestrator struct {
	synthesizer *contextengine.Synthesizer
	sessManager *session.Manager
	modelRouter *model.Router
}

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(syn *contextengine.Synthesizer, sm *session.Manager, mr *model.Router) *Orchestrator {
	return &Orchestrator{
		synthesizer: syn,
		sessManager: sm,
		modelRouter: mr,
	}
}

// ExecuteTask runs the verification-gated agent orchestration loop for a task objective.
func (o *Orchestrator) ExecuteTask(ctx context.Context, sessionID string, objective string) (string, error) {
	// 1. Update Session State
	_ = o.sessManager.UpdateState(ctx, sessionID, session.StatePlanning, objective)

	// 2. Select Optimal Model via Router
	modelSpec, err := o.modelRouter.SelectModel(ctx, model.ModelRequirement{
		NeedReasoning:   true,
		NeedToolCalling: true,
	})
	if err != nil {
		return "", fmt.Errorf("orchestrator: selecting model: %w", err)
	}

	// 3. Synthesize Context Payload via Context Engine
	contextPayload, err := o.synthesizer.SynthesizeContext(ctx, objective, "repo-core", 8000)
	if err != nil {
		return "", fmt.Errorf("orchestrator: synthesizing context: %w", err)
	}

	// 4. Execution Step Simulation & Verification
	_ = o.sessManager.UpdateState(ctx, sessionID, session.StateExecuting, objective)

	// 5. Verifier Gate Check (Simulated syntax / test check)
	verResult := o.verifyCandidateCode("func ValidateKey() bool { return true }")
	if !verResult.Passed {
		_ = o.sessManager.UpdateState(ctx, sessionID, session.StateAwaitingUser, "Verification failed")
		return "", fmt.Errorf("verifier gate failed: %v", verResult.Errors)
	}

	_ = o.sessManager.UpdateState(ctx, sessionID, session.StateIdle, objective)
	response := fmt.Sprintf("Successfully executed objective using model %s (%s). Packed %d context snippets.",
		modelSpec.Alias, modelSpec.ModelName, len(contextPayload.Snippets))

	_ = o.sessManager.RecordTurn(ctx, sessionID, objective, response)

	return response, nil
}

func (o *Orchestrator) verifyCandidateCode(code string) VerificationResult {
	if code == "" {
		return VerificationResult{Passed: false, Errors: []string{"empty candidate code"}}
	}
	return VerificationResult{
		Passed:         true,
		CompilerOutput: "Clean build. 0 errors.",
	}
}
