package tui_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"zuri-daemon/pkg/cli"
	"zuri-daemon/pkg/tui"
)

type mockDaemonClient struct {
	cli.DaemonClient
}

func (m *mockDaemonClient) GetHealth(ctx context.Context) (*cli.HealthResponse, error) {
	return &cli.HealthResponse{
		Status:    "running",
		Uptime:    "10m",
		Database:  "connected",
		Version:   "1.0.0",
		Timestamp: time.Now(),
	}, nil
}

func (m *mockDaemonClient) ListGaps(ctx context.Context, status string) ([]cli.KnowledgeGapDTO, error) {
	return []cli.KnowledgeGapDTO{
		{
			ID:          "gap-100",
			DecisionKey: "boundary:db/concern:data/decision_type:migration",
			Scope:       "core",
			GapType:     "unowned_decision",
			Status:      "open",
			RoutedTo:    "backend-team",
			DetectedAt:  time.Now(),
		},
	}, nil
}

func (m *mockDaemonClient) ListRepositories(ctx context.Context) ([]cli.ConnectedRepositoryDTO, error) {
	return []cli.ConnectedRepositoryDTO{
		{
			ID:             "repo-100",
			Name:           "ZURI Core",
			LocalPath:      "C:/Users/Agada Bartholomew/Documents/ZURI",
			DefaultBranch:  "main",
			IndexingStatus: "indexed",
			Health:         "healthy",
			LastSyncedAt:   time.Now(),
			CreatedAt:      time.Now(),
		},
	}, nil
}

func (m *mockDaemonClient) GetAuditLogs(ctx context.Context, limit int) ([]cli.AuditLogDTO, error) {
	return []cli.AuditLogDTO{
		{
			ID:        "log-1",
			EventType: "gap_detected",
			Actor:     "gap-detector",
			Details:   map[string]interface{}{"note": "Detected unowned decision key"},
			Timestamp: time.Now(),
		},
	}, nil
}

func TestTUI_AppLifecycle(t *testing.T) {
	mockClient := &mockDaemonClient{}
	app := tui.NewApp(mockClient)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := app.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected TUI run error: %v", err)
	}
}

func TestCLI_RunnerExecuteHelp(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	mockClient := &mockDaemonClient{}
	runner := cli.NewRunner(mockClient, &outBuf, &errBuf)

	exitCode := runner.Execute(context.Background(), []string{"help"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 on help, got %d", exitCode)
	}

	output := outBuf.String()
	if !strings.Contains(output, "USAGE:") {
		t.Errorf("expected usage header in help output, got: %s", output)
	}
}
