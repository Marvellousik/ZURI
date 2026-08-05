package mcp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"zuri-daemon/pkg/mcp"
)

func TestRegisterAllLocalAgentMCPConfigs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zuri_mcp_config_test_*")
	if err != nil {
		t.Fatalf("Failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	workspaceDir := filepath.Join(tmpDir, "workspace")
	homeDir := filepath.Join(tmpDir, "home")

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("Failed creating workspace dir: %v", err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatalf("Failed creating home dir: %v", err)
	}

	port := 7331
	paths, err := mcp.RegisterAllLocalAgentMCPConfigs(workspaceDir, homeDir, port)
	if err != nil {
		t.Fatalf("RegisterAllLocalAgentMCPConfigs failed: %v", err)
	}

	if len(paths) != 3 {
		t.Errorf("Expected 3 registered config paths, got %d", len(paths))
	}

	// Verify workspace .mcp.json
	mcpPath := filepath.Join(workspaceDir, ".mcp.json")
	verifyMCPConfigFile(t, mcpPath, port)

	// Verify Gemini CLI config
	geminiPath := filepath.Join(homeDir, ".gemini", "config.json")
	verifyMCPConfigFile(t, geminiPath, port)

	// Verify Claude Desktop config path exists and is populated
	claudePath := paths[1]
	verifyMCPConfigFile(t, claudePath, port)
}

func verifyMCPConfigFile(t *testing.T, path string, port int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed reading config file %s: %v", path, err)
	}

	var cfg mcp.MCPConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed unmarshaling config file %s: %v", path, err)
	}

	entry, ok := cfg.MCPServers["zuri-brain"]
	if !ok {
		t.Fatalf("Expected key 'zuri-brain' in %s", path)
	}

	expectedURL := "http://localhost:7331/mcp"
	if entry.URL != expectedURL {
		t.Errorf("Expected URL %s in %s, got %s", expectedURL, path, entry.URL)
	}
}
