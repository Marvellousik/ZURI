package sse

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServer_ServeHTTP(t *testing.T) {
	server := NewServer()
	ts := httptest.NewServer(server)
	defer ts.Close()

	// 1. Test invalid method
	req, _ := http.NewRequest(http.MethodPost, ts.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST, got %d", resp.StatusCode)
	}

	// 2. Test successful SSE connection and broadcast
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to SSE: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}

	// Wait briefly to ensure the client is registered in the map
	time.Sleep(100 * time.Millisecond)

	server.mu.Lock()
	clientCount := len(server.clients)
	server.mu.Unlock()
	if clientCount != 1 {
		t.Errorf("expected 1 connected client, got %d", clientCount)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	
	// Read from the stream concurrently
	go func() {
		defer wg.Done()
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		eventCount := 0
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event: retrieval") {
				eventCount++
			}
			if strings.HasPrefix(line, "data: {\"memory_id\":\"mem-123\"}") {
				eventCount++
			}
			if eventCount == 2 {
				return // we saw what we needed
			}
		}
	}()

	// Broadcast an event
	payload := map[string]string{"memory_id": "mem-123"}
	server.Broadcast(Event{
		Type:    EventRetrieval,
		Payload: payload,
	})

	// Wait for reader to confirm payload was received
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for SSE event")
	}

	// 3. Test client disconnect cleanup
	cancel() // Close the context to drop the connection
	time.Sleep(100 * time.Millisecond) // Give the handler a moment to clean up

	server.mu.Lock()
	clientCount = len(server.clients)
	server.mu.Unlock()
	if clientCount != 0 {
		t.Errorf("expected 0 connected clients after disconnect, got %d", clientCount)
	}
}

func TestServer_StalledClient(t *testing.T) {
	server := NewServer()
	
	// Create a client directly
	clientChan := make(chan Event, 1) // Buffer of 1
	server.clients[clientChan] = struct{}{}

	// Fill the buffer
	server.Broadcast(Event{Type: EventRetrieval})
	
	// The client map should still contain the client
	if len(server.clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(server.clients))
	}

	// Broadcast again. This should hit the default case, close the channel, and remove the client.
	server.Broadcast(Event{Type: EventConfirmation})

	if len(server.clients) != 0 {
		t.Fatalf("expected stalled client to be removed, got %d", len(server.clients))
	}
}

func TestServer_Close(t *testing.T) {
	server := NewServer()
	
	clientChan1 := make(chan Event, 1)
	clientChan2 := make(chan Event, 1)
	server.clients[clientChan1] = struct{}{}
	server.clients[clientChan2] = struct{}{}

	server.Close()

	if len(server.clients) != 0 {
		t.Fatalf("expected all clients to be removed, got %d", len(server.clients))
	}

	// Verifying channels are closed (will panic if we close an already closed channel)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic from closing an already closed channel, meaning it wasn't closed by Close()")
			}
		}()
		close(clientChan1)
	}()
}
