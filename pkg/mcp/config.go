package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type MCPConfigServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

type MCPConfigFile struct {
	MCPServers map[string]MCPConfigServerEntry `json:"mcpServers"`
}

// RegisterLocalMCPConfig writes or updates local .mcp.json in workspace directory to point to Zuri HTTP Streamable MCP server (§13.1).
func RegisterLocalMCPConfig(workspaceDir string, daemonHTTPPort int) (string, error) {
	configPath := filepath.Join(workspaceDir, ".mcp.json")
	return writeMCPConfig(configPath, daemonHTTPPort)
}

// RegisterClaudeDesktopMCPConfig registers Zuri MCP server in Claude Desktop configuration (§13.1, §3.4).
func RegisterClaudeDesktopMCPConfig(homeDir string, daemonHTTPPort int) (string, error) {
	var configPath string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		configPath = filepath.Join(appData, "Claude", "claude_desktop_config.json")
	case "darwin":
		configPath = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	default: // linux / unices
		configPath = filepath.Join(homeDir, ".config", "Claude", "claude_desktop_config.json")
	}
	return writeMCPConfig(configPath, daemonHTTPPort)
}

// RegisterGeminiCLIMCPConfig registers Zuri MCP server in Gemini CLI configuration (§13.1, §3.4).
func RegisterGeminiCLIMCPConfig(homeDir string, daemonHTTPPort int) (string, error) {
	configPath := filepath.Join(homeDir, ".gemini", "config.json")
	return writeMCPConfig(configPath, daemonHTTPPort)
}

// RegisterAllLocalAgentMCPConfigs registers Zuri across workspace .mcp.json, Claude Desktop, and Gemini CLI configs (§3.4).
func RegisterAllLocalAgentMCPConfigs(workspaceDir string, homeDir string, daemonHTTPPort int) ([]string, error) {
	var registeredPaths []string

	path1, err := RegisterLocalMCPConfig(workspaceDir, daemonHTTPPort)
	if err != nil {
		return nil, fmt.Errorf("failed registering .mcp.json: %w", err)
	}
	registeredPaths = append(registeredPaths, path1)

	path2, err := RegisterClaudeDesktopMCPConfig(homeDir, daemonHTTPPort)
	if err != nil {
		return nil, fmt.Errorf("failed registering Claude Desktop config: %w", err)
	}
	registeredPaths = append(registeredPaths, path2)

	path3, err := RegisterGeminiCLIMCPConfig(homeDir, daemonHTTPPort)
	if err != nil {
		return nil, fmt.Errorf("failed registering Gemini CLI config: %w", err)
	}
	registeredPaths = append(registeredPaths, path3)

	return registeredPaths, nil
}

func writeMCPConfig(configPath string, daemonHTTPPort int) (string, error) {
	serverURL := fmt.Sprintf("http://localhost:%d/mcp", daemonHTTPPort)

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return "", fmt.Errorf("failed creating directory for config %s: %w", configPath, err)
	}

	var cfg MCPConfigFile
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPConfigServerEntry)
	}

	cfg.MCPServers["zuri-brain"] = MCPConfigServerEntry{
		URL: serverURL,
	}

	updatedData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed marshaling MCP config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		return "", fmt.Errorf("failed writing MCP config %s: %w", configPath, err)
	}

	return configPath, nil
}
