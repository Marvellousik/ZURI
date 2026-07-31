package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"zuri-daemon/pkg/extraction"
)

func TestFormatProposalComment(t *testing.T) {
	// Empty case
	if got := FormatProposalComment(nil); got != "" {
		t.Errorf("expected empty string for nil result, got %q", got)
	}

	res := &extraction.ExtractionResult{
		Decisions: []extraction.ArchitecturalDecision{
			{Decision: "Use Postgres", Reasoning: "Better JSON support"},
		},
	}

	formatted := FormatProposalComment(res)
	if !strings.Contains(formatted, "🤖 **Zuri Memory Proposal**") {
		t.Errorf("missing header in formatting: %q", formatted)
	}
	if !strings.Contains(formatted, "- **Decision:** Use Postgres") {
		t.Errorf("missing decision in formatting: %q", formatted)
	}
	if !strings.Contains(formatted, "  **Reasoning:** Better JSON support") {
		t.Errorf("missing reasoning in formatting: %q", formatted)
	}
	if !strings.Contains(formatted, "`confirm`") {
		t.Errorf("missing instructions in formatting: %q", formatted)
	}
}

func TestParseReply(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		action  ReplyAction
		content string
	}{
		{"Confirm lowercase", "confirm", ActionConfirm, ""},
		{"Confirm uppercase", "CONFIRM", ActionConfirm, ""},
		{"Confirm with whitespace", "  confirm  ", ActionConfirm, ""},
		{"Reject", "reject", ActionReject, ""},
		{"Edit with new reasoning", "edit Use MySQL instead", ActionEdit, "Use MySQL instead"},
		{"Edit with messy casing", "eDiT Use MySQL instead", ActionEdit, "Use MySQL instead"},
		{"Irrelevant comment", "Looks good to me", ActionNone, ""},
		{"Almost match", "confirmation", ActionConfirm, ""}, // Because of HasPrefix("confirm")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseReply(tt.body)
			if got.Action != tt.action {
				t.Errorf("ParseReply(%q) Action = %v, want %v", tt.body, got.Action, tt.action)
			}
			if got.Content != tt.content {
				t.Errorf("ParseReply(%q) Content = %q, want %q", tt.body, got.Content, tt.content)
			}
		})
	}
}

func TestAPIClient_PostIssueComment(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			t.Errorf("Expected valid token, got %s", r.Header.Get("Authorization"))
		}
		if !strings.Contains(r.URL.Path, "/repos/org/repo/issues/42/comments") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	client := NewAPIClient("valid-token")
	client.BaseURL = ts.URL
	
	err := client.PostIssueComment(context.Background(), "org/repo", 42, "hello world")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test missing token
	clientNoToken := NewAPIClient("")
	err = clientNoToken.PostIssueComment(context.Background(), "org/repo", 42, "hello")
	if err == nil {
		t.Fatalf("Expected error for missing token, got nil")
	}

	// Test API error
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer tsErr.Close()

	clientErr := NewAPIClient("valid-token")
	clientErr.BaseURL = tsErr.URL
	err = clientErr.PostIssueComment(context.Background(), "org/repo", 42, "hello")
	if err == nil {
		t.Fatalf("Expected error for 403 status, got nil")
	}
}
