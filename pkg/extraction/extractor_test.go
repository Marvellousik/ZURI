package extraction

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

type mockCalibrator struct {
	calibrated float64
	isCal      bool
	err        error
}

func (mc *mockCalibrator) Calibrate(ctx context.Context, modelID string, concern string, rawConfidence float64) (float64, bool, error) {
	if mc.err != nil {
		return rawConfidence, false, mc.err
	}
	return mc.calibrated, mc.isCal, nil
}

func TestPRExtractor_Extract(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		prCtx         PRContext
		mockResp      string
		mockErr       error
		calibrator    Calibrator
		expectCount   int
		expectErr     bool
		expectKey     string
		expectConcern string
	}{
		{
			name: "Successful Extraction with RFC 7.4 taxonomy",
			prCtx: PRContext{
				RepoFullName: "org/repo",
				Description:  "Migrated to Go",
			},
			mockResp:      `{"decisions":[{"decision":"Use Go","reasoning":"Performance","concern":"reliability","decision_type":"retry-policy","boundary":"payments","extraction_confidence_raw":0.9}]}`,
			expectCount:   1,
			expectErr:     false,
			expectKey:     "boundary:payments/concern:reliability/decision_type:retry-policy",
			expectConcern: "reliability",
		},
		{
			name: "Malformed LLM Response",
			prCtx: PRContext{
				RepoFullName: "org/repo",
			},
			mockResp:  `{"decisions":[{"decision":}`, // Invalid JSON
			expectErr: true,
		},
		{
			name: "Provider Failure",
			prCtx: PRContext{
				RepoFullName: "org/repo",
			},
			mockErr:   errors.New("API timeout"),
			expectErr: true,
		},
		{
			name: "Empty PR / No Decisions",
			prCtx: PRContext{
				RepoFullName: "org/repo",
			},
			mockResp:    `{"decisions":[]}`,
			expectCount: 0,
			expectErr:   false,
		},
		{
			name: "Response Validation Drops Incomplete Decisions",
			prCtx: PRContext{
				RepoFullName: "org/repo",
			},
			mockResp:    `{"decisions":[{"decision":"","reasoning":"No decision text"},{"decision":"Good","reasoning":"Valid"}]}`,
			expectCount: 1, // Only the fully populated one
			expectErr:   false,
		},
		{
			name: "JSON with Markdown Fences",
			prCtx: PRContext{
				RepoFullName: "org/repo",
			},
			mockResp:    "```json\n{\"decisions\":[{\"decision\":\"Clean JSON\",\"reasoning\":\"Because\"}]}\n```",
			expectCount: 1,
			expectErr:   false,
		},
		{
			name: "Calibrated Extraction",
			prCtx: PRContext{
				RepoFullName: "org/repo",
			},
			mockResp:    `{"decisions":[{"decision":"Use Retry","reasoning":"Reliability","concern":"reliability","decision_type":"retry-policy","boundary":"checkout","extraction_confidence_raw":0.85}]}`,
			calibrator:  &mockCalibrator{calibrated: 0.92, isCal: true},
			expectCount: 1,
			expectErr:   false,
			expectKey:   "boundary:checkout/concern:reliability/decision_type:retry-policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewPRExtractor(&mockLLM{response: tt.mockResp, err: tt.mockErr}, tt.calibrator, "test-model")
			res, err := extractor.Extract(ctx, tt.prCtx)

			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error state %v, got error: %v", tt.expectErr, err)
			}

			if !tt.expectErr && len(res.Decisions) != tt.expectCount {
				t.Errorf("Expected %d decisions, got %d", tt.expectCount, len(res.Decisions))
			}

			if !tt.expectErr && tt.expectKey != "" && len(res.Decisions) > 0 {
				if res.Decisions[0].DecisionKey != tt.expectKey {
					t.Errorf("Expected decision_key %s, got %s", tt.expectKey, res.Decisions[0].DecisionKey)
				}
			}
		})
	}
}

func TestPRExtractor_BuildPrompt(t *testing.T) {
	extractor := NewPRExtractor(nil, nil, "test-model")

	pr := PRContext{
		RepoFullName:     "org/repo",
		PRNumber:         42,
		Description:      "Fixed the bug",
		UnifiedDiff:      "+ new line",
		Commits:          []string{"Fix bug"},
		ReviewComments:   []string{"Looks good"},
		IssueComments:    []string{"Thanks"},
		LinkedIssuesBody: []string{"Fixes #1"},
	}

	prompt := extractor.buildPrompt(pr)

	sections := []string{
		"Repository: org/repo",
		"PR Number: 42",
		"### Description\nFixed the bug",
		"### Commits\n- Fix bug",
		"### Unified Diff\n+ new line",
		"### Review Comments\n- Looks good",
		"### Issue Comments\n- Thanks",
		"### Linked Issues\n- Fixes #1",
	}

	for _, s := range sections {
		if !strings.Contains(prompt, s) {
			t.Errorf("Expected prompt to contain '%s', but it did not. Prompt:\n%s", s, prompt)
		}
	}
}

func TestDBCalibrator_UncalibratedUnderMinSample(t *testing.T) {
	calibrator := NewDBCalibrator(nil)
	ctx := context.Background()

	conf, isCal, err := calibrator.Calibrate(ctx, "claude-sonnet", "reliability", 0.85)
	if err != nil {
		t.Fatalf("Expected no error from nil db calibrator fallback, got %v", err)
	}

	if !isCal && conf != 0.85 {
		t.Errorf("Expected raw confidence pass-through 0.85, got %f", conf)
	}
}
