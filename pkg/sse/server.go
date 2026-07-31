package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// EventType defines the allowed SSE broadcast types.
type EventType string

const (
	EventRetrieval    EventType = "retrieval"
	EventConfirmation EventType = "confirmation"
	EventRejection    EventType = "rejection"
	EventLapse        EventType = "lapse"
	EventRevival      EventType = "revival"
)

// Event represents a broadcastable SSE event.
type Event struct {
	Type    EventType
	Payload interface{}
}

// Server handles SSE client connections and event broadcasting.
type Server struct {
	mu      sync.Mutex
	clients map[chan Event]struct{}
}

// NewServer creates a new SSE server.
func NewServer() *Server {
	return &Server{
		clients: make(map[chan Event]struct{}),
	}
}

// Broadcast sends an event to all connected clients.
func (s *Server) Broadcast(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		select {
		case client <- event:
		default:
			// Client channel is full (stalled client). 
			// Drop the client entirely rather than silently dropping events, 
			// forcing them to reconnect and re-sync if necessary.
			close(client)
			delete(s.clients, client)
		}
	}
}

// Close gracefully disconnects all active clients.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		close(client)
	}
	s.clients = make(map[chan Event]struct{})
}

// ServeHTTP implements the http.Handler interface for the SSE endpoint.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Removed wildcard CORS policy. Electron typically communicates over localhost. 
	// If cross-origin requests are made from the renderer without proper headers, 
	// it should be addressed via strict origin validation, not a wildcard.

	// Buffer events so temporary spikes don't disconnect clients immediately.
	clientChan := make(chan Event, 100)

	s.mu.Lock()
	s.clients[clientChan] = struct{}{}
	s.mu.Unlock()

	// Clean up client connection on normal disconnect
	defer func() {
		s.mu.Lock()
		// Only close if it hasn't been removed by the broadcaster already
		if _, exists := s.clients[clientChan]; exists {
			delete(s.clients, clientChan)
			close(clientChan)
		}
		s.mu.Unlock()
	}()

	// Send headers to client immediately
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return // Client disconnected
		case ev, ok := <-clientChan:
			if !ok {
				// Channel was closed by the server (e.g. stalled client or shutdown)
				return
			}
			data, err := json.Marshal(ev.Payload)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
			flusher.Flush()
		}
	}
}
