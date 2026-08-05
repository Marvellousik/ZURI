package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskState tracks the active state of an engineering task session.
type TaskState string

const (
	StateIdle         TaskState = "idle"
	StatePlanning     TaskState = "planning"
	StateExecuting    TaskState = "executing"
	StateVerifying    TaskState = "verifying"
	StateAwaitingUser TaskState = "awaiting_user"
)

// QueryTurn encapsulates a user prompt and system response turn.
type QueryTurn struct {
	Prompt    string    `json:"prompt"`
	Response  string    `json:"response"`
	Timestamp time.Time `json:"timestamp"`
}

// ToolExecutionRecord records a tool call made during session execution.
type ToolExecutionRecord struct {
	ToolName   string    `json:"tool_name"`
	Input      string    `json:"input"`
	Output     string    `json:"output"`
	Success    bool      `json:"success"`
	ExecutedAt time.Time `json:"executed_at"`
}

// Session encapsulates state, query history, working files, and reasoning history for a task.
type Session struct {
	ID                 string                `json:"id"`
	WorkspaceID        string                `json:"workspace_id"`
	ActiveRepositories []string              `json:"active_repositories"`
	CurrentObjective   string                `json:"current_objective"`
	State              TaskState             `json:"state"`
	WorkingFiles       []string              `json:"working_files"`
	QueryHistory       []QueryTurn           `json:"query_history"`
	ToolExecutions     []ToolExecutionRecord `json:"tool_executions"`
	ReasoningHistory   []string              `json:"reasoning_history"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

// Manager manages in-memory and disk-persistent sessions.
type Manager struct {
	mu       sync.RWMutex
	baseDir  string
	sessions map[string]*Session
}

// NewManager creates a new Session Manager instance.
func NewManager(baseDir string) (*Manager, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			baseDir = ".zuri/sessions"
		} else {
			baseDir = filepath.Join(home, ".zuri", "sessions")
		}
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("creating session directory: %w", err)
	}

	return &Manager{
		baseDir:  baseDir,
		sessions: make(map[string]*Session),
	}, nil
}

// CreateSession initializes a new persistent session.
func (m *Manager) CreateSession(ctx context.Context, id, workspaceID, objective string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "" {
		id = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}

	sess := &Session{
		ID:               id,
		WorkspaceID:      workspaceID,
		CurrentObjective: objective,
		State:            StateIdle,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	m.sessions[id] = sess
	_ = m.persistSession(sess)

	return sess, nil
}

// GetSession retrieves a session by ID.
func (m *Manager) GetSession(ctx context.Context, id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, exists := m.sessions[id]
	if !exists {
		// Try loading from disk
		loaded, err := m.loadSessionFromDisk(id)
		if err != nil {
			return nil, fmt.Errorf("session %s not found: %w", id, err)
		}
		m.sessions[id] = loaded
		return loaded, nil
	}

	return sess, nil
}

// RecordTurn appends a query turn to the session history.
func (m *Manager) RecordTurn(ctx context.Context, sessionID, prompt, response string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	sess.QueryHistory = append(sess.QueryHistory, QueryTurn{
		Prompt:    prompt,
		Response:  response,
		Timestamp: time.Now(),
	})
	sess.UpdatedAt = time.Now()

	return m.persistSession(sess)
}

// UpdateState modifies session state and objective.
func (m *Manager) UpdateState(ctx context.Context, sessionID string, state TaskState, objective string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	sess.State = state
	if objective != "" {
		sess.CurrentObjective = objective
	}
	sess.UpdatedAt = time.Now()

	return m.persistSession(sess)
}

func (m *Manager) persistSession(sess *Session) error {
	filePath := filepath.Join(m.baseDir, fmt.Sprintf("%s.json", sess.ID))
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}

func (m *Manager) loadSessionFromDisk(id string) (*Session, error) {
	filePath := filepath.Join(m.baseDir, fmt.Sprintf("%s.json", id))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	return &sess, nil
}
