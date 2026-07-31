package workflows

import (
	"context"
	"errors"
	"testing"
	"zuri-daemon/pkg/mcp"
)

type mockMemoryResolver struct {
	invoked bool
	input   mcp.ResolveMemoryInput
	err     error
}

func (m *mockMemoryResolver) HandleResolveMemory(ctx context.Context, req interface{}, input mcp.ResolveMemoryInput) (interface{}, mcp.ResolveMemoryOutput, error) {
	m.invoked = true
	m.input = input
	if m.err != nil {
		return nil, mcp.ResolveMemoryOutput{}, m.err
	}
	return nil, mcp.ResolveMemoryOutput{MemoryID: input.MemoryID}, nil
}

func TestReplyProcessor_Process(t *testing.T) {
	tests := []struct {
		name          string
		comment       string
		prMerged      bool
		expectInvoked bool
		expectAction  string
		expectContent string
		mockErr       error
		expectErr     bool
	}{
		{
			name:          "Confirm Unmerged",
			comment:       "confirm",
			prMerged:      false,
			expectInvoked: true,
			expectAction:  "confirm",
		},
		{
			name:          "Reject Merged",
			comment:       "reject",
			prMerged:      true,
			expectInvoked: true,
			expectAction:  "reject",
		},
		{
			name:          "Edit",
			comment:       "edit New decision",
			prMerged:      false,
			expectInvoked: true,
			expectAction:  "edit",
			expectContent: "New decision",
		},
		{
			name:          "Ignored Comment",
			comment:       "Looks good",
			prMerged:      true,
			expectInvoked: false,
		},
		{
			name:          "Resolver Error",
			comment:       "confirm",
			prMerged:      true,
			expectInvoked: true,
			expectAction:  "confirm",
			mockErr:       errors.New("db error"),
			expectErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockMemoryResolver{err: tt.mockErr}
			processor := NewReplyProcessor(mock)

			err := processor.Process(context.Background(), tt.comment, "mem-123", "user1", tt.prMerged, []string{"repo1"}, "ctx")

			if (err != nil) != tt.expectErr {
				t.Fatalf("Expected error %v, got %v", tt.expectErr, err)
			}

			if mock.invoked != tt.expectInvoked {
				t.Fatalf("Expected invoked %v, got %v", tt.expectInvoked, mock.invoked)
			}

			if tt.expectInvoked {
				if mock.input.PRMerged != tt.prMerged {
					t.Errorf("Expected PRMerged %v, got %v", tt.prMerged, mock.input.PRMerged)
				}
				if mock.input.Action != tt.expectAction {
					t.Errorf("Expected action %s, got %s", tt.expectAction, mock.input.Action)
				}
				if mock.input.EditedContent != tt.expectContent {
					t.Errorf("Expected edited content %q, got %q", tt.expectContent, mock.input.EditedContent)
				}
			}
		})
	}
}
