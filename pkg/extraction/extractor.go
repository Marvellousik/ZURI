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

// ArchitecturalDecision is the structured result inferred by the LLM (Spec v1.1 & RFC §7.4).
type ArchitecturalDecision struct {
	Decision                string  `json:"decision"`
	Reasoning               string  `json:"reasoning"`
	ExtractionConfidenceRaw float64 `json:"extraction_confidence_raw"`
	Concern                 string  `json:"concern"`
	DecisionType            string  `json:"decision_type"`
	Boundary                string  `json:"boundary"`
	DecisionKey             string  `json:"decision_key"`
	ExtractionConfidence    float64 `json:"extraction_confidence"`
	IsCalibrated            bool    `json:"is_calibrated"`
}

// ExtractionResult represents the validated output of the pipeline.
type ExtractionResult struct {
	Decisions []ArchitecturalDecision
}

// LLMProvider defines the abstract interface for invoking an LLM.
type LLMProvider interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type PRExtractor struct {
	llm        LLMProvider
	calibrator Calibrator
	modelID    string
}

// NewPRExtractor initializes a new extraction pipeline with optional calibrator.
func NewPRExtractor(llm LLMProvider, calibrator Calibrator, modelID string) *PRExtractor {
	if modelID == "" {
		modelID = "default-extractor-v1"
	}
	return &PRExtractor{
		llm:        llm,
		calibrator: calibrator,
		modelID:    modelID,
	}
}

// buildPrompt formats the PRContext into a normalized extraction input for the LLM.
func (e *PRExtractor) buildPrompt(pr PRContext) string {
	var sb strings.Builder
	sb.WriteString("Extract architectural and engineering decisions from the following Pull Request per Zuri classification taxonomy.\n")
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

	sb.WriteString(`Return the extracted decisions as a JSON object with a single key 'decisions', which is an array of objects having the following fields:
- 'decision' (string): 1-2 sentence description of the decision.
- 'reasoning' (string): supporting rationale.
- 'extraction_confidence_raw' (float 0.0 to 1.0): raw confidence in this extraction.
- 'concern' (string): one of ['reliability', 'security', 'performance', 'data', 'architecture', 'deployment', 'observability'].
- 'decision_type' (string): specific decision kind (e.g. 'retry-policy', 'schema-design', 'storage-selection', 'fallback-strategy').
- 'boundary' (string): concrete affected system boundary (e.g. 'payments', 'checkout', 'auth').

If no architectural decisions are present, return an empty array {"decisions": []}.`)
	return sb.String()
}

// Extract processes the PR context, invokes the LLM, validates the output, calibrates confidence, and returns the result.
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

	// Validate, construct decision keys, and calibrate results
	var validDecisions []ArchitecturalDecision
	for _, d := range raw.Decisions {
		decisionStr := strings.TrimSpace(d.Decision)
		reasoningStr := strings.TrimSpace(d.Reasoning)

		// Both fields must be present and non-empty
		if decisionStr != "" && reasoningStr != "" {
			concern := strings.ToLower(strings.TrimSpace(d.Concern))
			if concern == "" {
				concern = "architecture"
			}
			decisionType := strings.ToLower(strings.TrimSpace(d.DecisionType))
			if decisionType == "" {
				decisionType = "convention"
			}
			boundary := strings.ToLower(strings.TrimSpace(d.Boundary))
			if boundary == "" {
				boundary = strings.ToLower(pr.RepoFullName)
			}

			// Construct decision key per RFC §7.4: boundary:<boundary>/concern:<concern>/decision_type:<decision_type>
			decisionKey := fmt.Sprintf("boundary:%s/concern:%s/decision_type:%s", boundary, concern, decisionType)

			rawConf := d.ExtractionConfidenceRaw
			if rawConf <= 0.0 {
				rawConf = 0.8
			}

			calibratedConf := rawConf
			isCalibrated := false

			if e.calibrator != nil {
				cConf, isCal, err := e.calibrator.Calibrate(ctx, e.modelID, concern, rawConf)
				if err == nil {
					calibratedConf = cConf
					isCalibrated = isCal
				}
			}

			validDecisions = append(validDecisions, ArchitecturalDecision{
				Decision:                decisionStr,
				Reasoning:               reasoningStr,
				ExtractionConfidenceRaw: rawConf,
				Concern:                 concern,
				DecisionType:            decisionType,
				Boundary:                boundary,
				DecisionKey:             decisionKey,
				ExtractionConfidence:    calibratedConf,
				IsCalibrated:            isCalibrated,
			})
		}
	}

	return &ExtractionResult{Decisions: validDecisions}, nil
}
