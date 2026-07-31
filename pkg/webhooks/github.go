package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type GitHubWebhookHandler struct {
	Secret string
}

func NewGitHubWebhookHandler(secret string) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		Secret: secret,
	}
}

func (h *GitHubWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if h.Secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			http.Error(w, "Missing X-Hub-Signature-256", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(signature, "sha256=") {
			http.Error(w, "Invalid signature format", http.StatusUnauthorized)
			return
		}

		mac := hmac.New(sha256.New, []byte(h.Secret))
		mac.Write(body)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))
		
		if !hmac.Equal([]byte(signature[7:]), []byte(expectedMAC)) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		http.Error(w, "Missing X-GitHub-Event", http.StatusBadRequest)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Malformed JSON payload", http.StatusBadRequest)
		return
	}

	action, _ := payload["action"].(string)

	switch eventType {
	case "pull_request":
		if action == "opened" || action == "synchronize" || action == "closed" {
			h.handlePullRequest(w, payload, action)
			return
		}
		http.Error(w, "Unsupported pull_request action", http.StatusNotImplemented)
	case "pull_request_review_comment":
		if action == "created" {
			h.handlePullRequestReviewComment(w, payload)
			return
		}
		http.Error(w, "Unsupported pull_request_review_comment action", http.StatusNotImplemented)
	case "issue_comment":
		if action == "created" {
			h.handleIssueComment(w, payload)
			return
		}
		http.Error(w, "Unsupported issue_comment action", http.StatusNotImplemented)
	default:
		http.Error(w, "Unsupported event type", http.StatusNotImplemented)
	}
}

func (h *GitHubWebhookHandler) handlePullRequest(w http.ResponseWriter, payload map[string]interface{}, action string) {
	// Stub for Stage 5
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"pull_request ` + action + ` processed"}`))
}

func (h *GitHubWebhookHandler) handlePullRequestReviewComment(w http.ResponseWriter, payload map[string]interface{}) {
	// Stub for Stage 5
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"pull_request_review_comment created processed"}`))
}

func (h *GitHubWebhookHandler) handleIssueComment(w http.ResponseWriter, payload map[string]interface{}) {
	// Stub for Stage 5
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success","message":"issue_comment created processed"}`))
}
