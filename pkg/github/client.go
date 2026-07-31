package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client defines the interface for interacting with GitHub's API.
type Client interface {
	PostIssueComment(ctx context.Context, repoFullName string, prNumber int, body string) error
}

type APIClient struct {
	BaseURL    string
	token      string
	httpClient *http.Client
}

func NewAPIClient(token string) *APIClient {
	return &APIClient{
		BaseURL:    "https://api.github.com",
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) PostIssueComment(ctx context.Context, repoFullName string, prNumber int, body string) error {
	if c.token == "" {
		return fmt.Errorf("github token is not configured")
	}

	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", c.BaseURL, repoFullName, prNumber)

	payload := map[string]string{"body": body}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github api returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}
