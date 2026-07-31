package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func signPayload(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestGitHubWebhookHandler(t *testing.T) {
	secret := "test-secret"
	handler := NewGitHubWebhookHandler(secret)

	tests := []struct {
		name           string
		method         string
		event          string
		payload        string
		signature      string
		expectedStatus int
	}{
		{
			name:           "Method Not Allowed (GET)",
			method:         http.MethodGet,
			event:          "pull_request",
			payload:        `{"action":"opened"}`,
			signature:      "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Signature",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"opened"}`,
			signature:      "", // Missing
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Signature Format",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"opened"}`,
			signature:      "md5=invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Signature Value",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"opened"}`,
			signature:      "sha256=abcdef123456",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing GitHub Event Header",
			method:         http.MethodPost,
			event:          "",
			payload:        `{"action":"opened"}`,
			signature:      signPayload(secret, `{"action":"opened"}`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Malformed JSON Payload",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"opened"`, // Malformed
			signature:      signPayload(secret, `{"action":"opened"`),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unsupported Event Type",
			method:         http.MethodPost,
			event:          "push",
			payload:        `{"ref":"refs/heads/main"}`,
			signature:      signPayload(secret, `{"ref":"refs/heads/main"}`),
			expectedStatus: http.StatusNotImplemented,
		},
		{
			name:           "Unsupported Pull Request Action",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"assigned"}`,
			signature:      signPayload(secret, `{"action":"assigned"}`),
			expectedStatus: http.StatusNotImplemented,
		},
		{
			name:           "Valid Pull Request Opened",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"opened"}`,
			signature:      signPayload(secret, `{"action":"opened"}`),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid Pull Request Synchronize",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"synchronize"}`,
			signature:      signPayload(secret, `{"action":"synchronize"}`),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid Pull Request Closed",
			method:         http.MethodPost,
			event:          "pull_request",
			payload:        `{"action":"closed"}`,
			signature:      signPayload(secret, `{"action":"closed"}`),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid Pull Request Review Comment Created",
			method:         http.MethodPost,
			event:          "pull_request_review_comment",
			payload:        `{"action":"created"}`,
			signature:      signPayload(secret, `{"action":"created"}`),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unsupported Pull Request Review Comment Action",
			method:         http.MethodPost,
			event:          "pull_request_review_comment",
			payload:        `{"action":"edited"}`,
			signature:      signPayload(secret, `{"action":"edited"}`),
			expectedStatus: http.StatusNotImplemented,
		},
		{
			name:           "Valid Issue Comment Created",
			method:         http.MethodPost,
			event:          "issue_comment",
			payload:        `{"action":"created"}`,
			signature:      signPayload(secret, `{"action":"created"}`),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unsupported Issue Comment Action",
			method:         http.MethodPost,
			event:          "issue_comment",
			payload:        `{"action":"deleted"}`,
			signature:      signPayload(secret, `{"action":"deleted"}`),
			expectedStatus: http.StatusNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/webhooks/github", bytes.NewBuffer([]byte(tt.payload)))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.event != "" {
				req.Header.Set("X-GitHub-Event", tt.event)
			}
			if tt.signature != "" {
				req.Header.Set("X-Hub-Signature-256", tt.signature)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
