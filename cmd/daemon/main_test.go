package main_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestEndToEndDaemonIntegration(t *testing.T) {
	// Build the binary
	buildCmd := exec.Command("go", "build", "-o", "test-daemon.exe", ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build daemon: %v", err)
	}
	defer os.Remove("test-daemon.exe")

	tmpDir, err := os.MkdirTemp("", "zuri_main_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	port := "7346"
	dbPort := "5456"
	baseURL := "http://127.0.0.1:" + port

	// Run the binary
	cmd := exec.Command("./test-daemon.exe")
	cmd.Env = append(os.Environ(),
		"ZURI_DB_PORT="+dbPort,
		"ZURI_PORT="+port,
		"ZURI_HOST=127.0.0.1",
		"ZURI_DB_PATH="+tmpDir,
		"ZURI_DISABLE_PGVECTOR_VALIDATION_FOR_TESTS=1",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start daemon: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// 1. Wait for server to start (polling health endpoint)
	serverUp := false
	for i := 0; i < 120; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverUp = true
				break
			}
		}
	}

	if !serverUp {
		t.Fatalf("Daemon HTTP server did not start in time")
	}

	t.Log("✓ Task 1: Daemon booted cleanly and responded to /api/health")

	// 2. Test MCP JSON-RPC ping
	pingReqBody := []byte(`{"jsonrpc": "2.0", "method": "ping", "id": 1}`)
	req, err := http.NewRequest("POST", baseURL+"/mcp/", bytes.NewBuffer(pingReqBody))
	if err != nil {
		t.Fatalf("Failed to create MCP ping request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute MCP ping request: %v", err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK from MCP ping, got %d. Body: %s", resp.StatusCode, string(body))
	} else {
		t.Logf("✓ Task 1 (MCP Endpoint): MCP ping successful, status = %d", resp.StatusCode)
	}

	// 3. Test SSE Event Stream connection
	sseReq, err := http.NewRequest("GET", baseURL+"/events", nil)
	if err == nil {
		sseClient := &http.Client{Timeout: 2 * time.Second}
		sseResp, err := sseClient.Do(sseReq)
		if err == nil {
			if sseResp.Header.Get("Content-Type") == "text/event-stream" {
				t.Log("✓ Task 1 (SSE Endpoint): Connected to /events with text/event-stream content type")
			}
			sseResp.Body.Close()
		}
	}

	// 4. Test MCP Tool Execution: get_relevant_memory
	toolReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "get_relevant_memory",
			"arguments": map[string]interface{}{
				"prompt_text": "database conventions",
				"repo_id":     "test-repo",
			},
		},
	}
	toolReqBytes, _ := json.Marshal(toolReq)
	req2, err := http.NewRequest("POST", baseURL+"/mcp/", bytes.NewBuffer(toolReqBytes))
	if err != nil {
		t.Fatalf("Failed to create tool call request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Accept", "application/json, text/event-stream")

	resp, err = client.Do(req2)
	if err != nil {
		t.Fatalf("Failed to call get_relevant_memory: %v", err)
	}
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK from get_relevant_memory, got %d. Body: %s", resp.StatusCode, string(body))
	} else {
		t.Logf("✓ Task 1 (MCP Tool Execution): get_relevant_memory call succeeded, body = %s", string(body))
	}
}
