package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DaemonClient defines the contract for communicating with the zuri-daemon HTTP API.
type DaemonClient interface {
	GetHealth(ctx context.Context) (*HealthResponse, error)
	QueryMemory(ctx context.Context, req QueryMemoryRequest) ([]MemoryRecordDTO, error)
	ListGaps(ctx context.Context, status string) ([]KnowledgeGapDTO, error)
	ResolveGap(ctx context.Context, gapID string, req ResolveGapRequest) error
	GetAuditLogs(ctx context.Context, limit int) ([]AuditLogDTO, error)
	ListRepositories(ctx context.Context) ([]ConnectedRepositoryDTO, error)
	AddRepository(ctx context.Context, name, localPath, ghFullName string) (*ConnectedRepositoryDTO, error)
	RemoveRepository(ctx context.Context, repoID string) error
	OnboardProject(ctx context.Context, req OnboardRequestDTO) error
	ShutdownDaemon(ctx context.Context) error
}

// Client implements DaemonClient using standard Go http.Client.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// HealthResponse represents the health payload returned by zuri-daemon.
type HealthResponse struct {
	Status      string    `json:"status"`
	Uptime      string    `json:"uptime"`
	Database    string    `json:"database"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
}

// ConnectedRepositoryDTO represents a connected codebase repository.
type ConnectedRepositoryDTO struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	LocalPath          string    `json:"localPath"`
	GithubRepoFullName string    `json:"githubRepoFullName"`
	DefaultBranch      string    `json:"defaultBranch"`
	GithubStatus       string    `json:"githubStatus"`
	IndexingStatus     string    `json:"indexingStatus"`
	Health             string    `json:"health"`
	LastSyncedAt       time.Time `json:"lastSyncedAt"`
	CreatedAt          time.Time `json:"createdAt"`
}

// OnboardMemoryDTO represents an architectural decision synthesized during project onboarding.
type OnboardMemoryDTO struct {
	Decision  string `json:"decision"`
	Reasoning string `json:"reasoning"`
}

// OnboardRequestDTO represents the onboarding ingestion payload.
type OnboardRequestDTO struct {
	RepoID   string             `json:"repoId"`
	Founder  string             `json:"founder"`
	Memories []OnboardMemoryDTO `json:"memories"`
}

// QueryMemoryRequest defines the parameters for querying memory records.
type QueryMemoryRequest struct {
	Query         string   `json:"query"`
	Boundary      string   `json:"boundary,omitempty"`
	Concern       string   `json:"concern,omitempty"`
	MinConfidence float64  `json:"min_confidence,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}

// MemoryRecordDTO represents a memory record returned over transport.
type MemoryRecordDTO struct {
	ID                   string    `json:"id"`
	DecisionKey          string    `json:"decision_key"`
	Concern              string    `json:"concern"`
	DecisionType         string    `json:"decision_type"`
	Boundary             string    `json:"boundary"`
	Summary              string    `json:"content"`
	MemoryTier           string    `json:"memoryTier"`
	FilePath             string    `json:"filePath"`
	ExtractionConfidence float64   `json:"relevanceScore"`
	EvidenceStrength     float64   `json:"finalScore"`
	CreatedAt            time.Time `json:"created_at"`
}

type queryMemoryResponse struct {
	Results []MemoryRecordDTO `json:"results"`
	Total   int               `json:"total"`
}

// KnowledgeGapDTO represents a knowledge gap returned over transport.
type KnowledgeGapDTO struct {
	ID           string    `json:"gap_id"`
	DecisionKey  string    `json:"decision_key"`
	Scope        string    `json:"scope"`
	GapType      string    `json:"gap_type"`
	Status       string    `json:"status"`
	RoutedTo     string    `json:"routed_to,omitempty"`
	DetectedAt   time.Time `json:"detected_at"`
}

// ResolveGapRequest defines the request body for gap resolution.
type ResolveGapRequest struct {
	Action string `json:"action"` // "answer" or "acknowledge_unknown"
	Answer string `json:"answer,omitempty"`
}

// AuditLogDTO represents an audit log entry returned over transport.
type AuditLogDTO struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"eventType"`
	Actor     string                 `json:"actorId"`
	Details   map[string]interface{} `json:"details"`
	Timestamp time.Time              `json:"createdAt"`
}

// NewClient constructs a new Client for the specified base URL.
// If baseURL is empty, it defaults to "http://127.0.0.1:7331".
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:7331"
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetHealth fetches system health status from the daemon.
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	reqURL := fmt.Sprintf("%s/api/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing health request to daemon at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("daemon health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding health response: %w", err)
	}

	return &res, nil
}

// ListRepositories lists all connected codebase repositories.
func (c *Client) ListRepositories(ctx context.Context) ([]ConnectedRepositoryDTO, error) {
	reqURL := fmt.Sprintf("%s/api/repositories", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating repositories request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing repositories request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listing repositories failed with status %d: %s", resp.StatusCode, string(body))
	}

	var repos []ConnectedRepositoryDTO
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("decoding repositories response: %w", err)
	}

	return repos, nil
}

// AddRepository connects a local codebase directory or GitHub repo to Zuri memory.
func (c *Client) AddRepository(ctx context.Context, name, localPath, ghFullName string) (*ConnectedRepositoryDTO, error) {
	reqURL := fmt.Sprintf("%s/api/repositories", c.baseURL)
	payload, err := json.Marshal(ConnectedRepositoryDTO{
		Name:               name,
		LocalPath:          localPath,
		GithubRepoFullName: ghFullName,
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling repository payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating add repository request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing add repository request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("adding repository failed with status %d: %s", resp.StatusCode, string(body))
	}

	var repo ConnectedRepositoryDTO
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, fmt.Errorf("decoding add repository response: %w", err)
	}

	return &repo, nil
}

// RemoveRepository disconnects a codebase repository.
func (c *Client) RemoveRepository(ctx context.Context, repoID string) error {
	reqURL := fmt.Sprintf("%s/api/repositories?id=%s", c.baseURL, url.QueryEscape(repoID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating delete repository request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing delete repository request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deleting repository failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// OnboardProject ingests architectural survey memories directly into the database.
func (c *Client) OnboardProject(ctx context.Context, onboardReq OnboardRequestDTO) error {
	reqURL := fmt.Sprintf("%s/api/onboarding", c.baseURL)
	payload, err := json.Marshal(onboardReq)
	if err != nil {
		return fmt.Errorf("marshaling onboarding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating onboarding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing onboarding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("onboarding failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// QueryMemory queries the zuri-daemon for memory records matching criteria.
func (c *Client) QueryMemory(ctx context.Context, queryReq QueryMemoryRequest) ([]MemoryRecordDTO, error) {
	reqURL := fmt.Sprintf("%s/api/memory/query", c.baseURL)
	payload, err := json.Marshal(queryReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling query request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating query request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing memory query request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("memory query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res queryMemoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decoding query response: %w", err)
	}

	return res.Results, nil
}

// ListGaps retrieves knowledge gaps from the daemon.
func (c *Client) ListGaps(ctx context.Context, status string) ([]KnowledgeGapDTO, error) {
	reqURL := fmt.Sprintf("%s/api/gaps", c.baseURL)
	if status != "" {
		reqURL = fmt.Sprintf("%s?status=%s", reqURL, url.QueryEscape(status))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating gaps list request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing gaps request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listing gaps failed with status %d: %s", resp.StatusCode, string(body))
	}

	var gaps []KnowledgeGapDTO
	if err := json.NewDecoder(resp.Body).Decode(&gaps); err != nil {
		return nil, fmt.Errorf("decoding gaps response: %w", err)
	}

	return gaps, nil
}

// ResolveGap sends a gap resolution command to the daemon.
func (c *Client) ResolveGap(ctx context.Context, gapID string, resolveReq ResolveGapRequest) error {
	reqURL := fmt.Sprintf("%s/api/gaps/resolve?id=%s", c.baseURL, url.QueryEscape(gapID))
	payload, err := json.Marshal(resolveReq)
	if err != nil {
		return fmt.Errorf("marshaling resolve request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating resolve request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing resolve request for gap %s: %w", gapID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resolving gap failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAuditLogs fetches recent audit log entries from the daemon.
func (c *Client) GetAuditLogs(ctx context.Context, limit int) ([]AuditLogDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	reqURL := fmt.Sprintf("%s/api/audit-log?limit=%d", c.baseURL, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating audit log request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing audit log request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetching audit log failed with status %d: %s", resp.StatusCode, string(body))
	}

	var logs []AuditLogDTO
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("decoding audit log response: %w", err)
	}

	return logs, nil
}

// ShutdownDaemon sends a shutdown request to the zuri-daemon process.
func (c *Client) ShutdownDaemon(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/api/shutdown", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, nil)
	if err != nil {
		return fmt.Errorf("creating shutdown request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing shutdown request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("daemon shutdown failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
