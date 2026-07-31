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

func TestPRExtractor_Extract(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		prCtx       PRContext
		mockResp    string
		mockErr     error
		expectCount int
		expectErr   bool
	}{
		{
			name: "Successful Extraction",
			prCtx: PRContext{
				RepoFullName: "org/repo",
				Description:  "Migrated to Go",
			},
			mockResp:    `{"decisions":[{"decision":"Use Go","reasoning":"Performance"}]}`,
			expectCount: 1,
			expectErr:   false,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewPRExtractor(&mockLLM{response: tt.mockResp, err: tt.mockErr})
			res, err := extractor.Extract(ctx, tt.prCtx)

			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error state %v, got error: %v", tt.expectErr, err)
			}

			if !tt.expectErr && len(res.Decisions) != tt.expectCount {
				t.Errorf("Expected %d decisions, got %d", tt.expectCount, len(res.Decisions))
			}
		})
	}
}

func TestPRExtractor_BuildPrompt(t *testing.T) {
	extractor := NewPRExtractor(nil)

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

	// Ensure all sections are present
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

	// Test missing sections
	emptyPr := PRContext{
		RepoFullName: "org/repo",
		PRNumber:     43,
	}

	emptyPrompt := extractor.buildPrompt(emptyPr)
	
	notExpectedSections := []string{
		"### Description",
		"### Commits",
		"### Unified Diff",
		"### Review Comments",
		"### Issue Comments",
		"### Linked Issues",
	}

	for _, s := range notExpectedSections {
		if strings.Contains(emptyPrompt, s) {
			t.Errorf("Expected prompt to omit '%s', but it was included. Prompt:\n%s", s, emptyPrompt)
		}
	}
}
