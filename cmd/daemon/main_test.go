package main_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestEndToEndMCPRouting(t *testing.T) {
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

	// Run the binary
	cmd := exec.Command("./test-daemon.exe")
	cmd.Env = append(os.Environ(),
		"ZURI_DB_PORT=5455",
		"ZURI_PORT=7345",
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
		// Force kill on teardown
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Wait for server to start (polling health endpoint)
	serverUp := false
	for i := 0; i < 40; i++ { // 20 seconds max for DB init
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:7345/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				serverUp = true
				break
			}
		}
	}

	if !serverUp {
		t.Fatalf("Server did not start in time")
	}

	// POST through HTTP
	reqBody := []byte(`{"jsonrpc": "2.0", "method": "ping", "id": 1}`)
	resp, err := http.Post("http://127.0.0.1:7345/mcp/", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to make POST request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	// Verify it reached the MCP handler
	if resp.StatusCode == http.StatusNotImplemented {
		t.Errorf("Expected request to hit MCP handler, but got 501 Not Implemented. Body: %s", string(body))
	} else if resp.StatusCode == http.StatusNotFound {
		t.Errorf("Expected request to hit MCP handler, but got 404 Not Found")
	} else {
		t.Logf("Success: hit MCP handler, status = %d, body = %s", resp.StatusCode, string(body))
	}
}
