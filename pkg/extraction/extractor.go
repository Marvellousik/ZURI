package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PRContext represents all collected PR data normalized into a single struct.
type PRContext struct {
	RepoFullName     string
	PRNumber         int
	Description      string
	UnifiedDiff      string
	Commits          []string
	ReviewComments   []string
	IssueComments    []string
	LinkedIssuesBody []string
}

// ArchitecturalDecision is the structured result inferred by the LLM.
type ArchitecturalDecision struct {
	Decision  string `json:"decision"`
	Reasoning string `json:"reasoning"`
}

// ExtractionResult represents the validated output of the pipeline.
type ExtractionResult struct {
	Decisions []ArchitecturalDecision
}

// LLMProvider defines the abstract interface for invoking an LLM.
// This prevents hardcoding a specific model or vendor (e.g., OpenAI, Anthropic, Gemini).
type LLMProvider interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type PRExtractor struct {
	llm LLMProvider
}

// NewPRExtractor initializes a new extraction pipeline.
func NewPRExtractor(llm LLMProvider) *PRExtractor {
	return &PRExtractor{
		llm: llm,
	}
}

// buildPrompt formats the PRContext into a normalized extraction input for the LLM.
func (e *PRExtractor) buildPrompt(pr PRContext) string {
	var sb strings.Builder
	sb.WriteString("Extract architectural and engineering decisions from the following Pull Request.\n")
	sb.WriteString(fmt.Sprintf("Repository: %s\nPR Number: %d\n\n", pr.RepoFullName, pr.PRNumber))

	if pr.Description != "" {
		sb.WriteString("### Description\n" + pr.Description + "\n\n")
	}

	if len(pr.Commits) > 0 {
		sb.WriteString("### Commits\n- " + strings.Join(pr.Commits, "\n- ") + "\n\n")
	}

	if pr.UnifiedDiff != "" {
		sb.WriteString("### Unified Diff\n" + pr.UnifiedDiff + "\n\n")
	}

	if len(pr.ReviewComments) > 0 {
		sb.WriteString("### Review Comments\n- " + strings.Join(pr.ReviewComments, "\n- ") + "\n\n")
	}

	if len(pr.IssueComments) > 0 {
		sb.WriteString("### Issue Comments\n- " + strings.Join(pr.IssueComments, "\n- ") + "\n\n")
	}

	if len(pr.LinkedIssuesBody) > 0 {
		sb.WriteString("### Linked Issues\n- " + strings.Join(pr.LinkedIssuesBody, "\n- ") + "\n\n")
	}

	sb.WriteString("Return the extracted decisions as a JSON object with a single key 'decisions', which is an array of objects having 'decision' and 'reasoning' string fields. If no architectural decisions are present, return an empty array.")
	return sb.String()
}

// Extract processes the PR context, invokes the LLM, validates the output, and returns the result.
func (e *PRExtractor) Extract(ctx context.Context, pr PRContext) (*ExtractionResult, error) {
	prompt := e.buildPrompt(pr)

	respJSON, err := e.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("llm provider error: %w", err)
	}

	// Normalize the response by stripping Markdown code fences if present.
	normalizedJSON := strings.TrimSpace(respJSON)
	if strings.HasPrefix(normalizedJSON, "```json") {
		normalizedJSON = strings.TrimPrefix(normalizedJSON, "```json")
		normalizedJSON = strings.TrimSuffix(strings.TrimSpace(normalizedJSON), "```")
		normalizedJSON = strings.TrimSpace(normalizedJSON)
	} else if strings.HasPrefix(normalizedJSON, "```") {
		normalizedJSON = strings.TrimPrefix(normalizedJSON, "```")
		normalizedJSON = strings.TrimSuffix(strings.TrimSpace(normalizedJSON), "```")
		normalizedJSON = strings.TrimSpace(normalizedJSON)
	}

	var raw struct {
		Decisions []ArchitecturalDecision `json:"decisions"`
	}

	if err := json.Unmarshal([]byte(normalizedJSON), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse llm response: %w", err)
	}

	// Validate and filter results
	var validDecisions []ArchitecturalDecision
	for _, d := range raw.Decisions {
		decisionStr := strings.TrimSpace(d.Decision)
		reasoningStr := strings.TrimSpace(d.Reasoning)

		// Both fields must be present and non-empty
		if decisionStr != "" && reasoningStr != "" {
			validDecisions = append(validDecisions, ArchitecturalDecision{
				Decision:  decisionStr,
				Reasoning: reasoningStr,
			})
		}
	}

	return &ExtractionResult{Decisions: validDecisions}, nil
}
