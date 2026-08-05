package cli_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zuri-daemon/pkg/cli"
)

func TestClient_GetHealth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cli.HealthResponse{
			Status:    "running",
			Uptime:    "5m",
			Database:  "connected",
			Version:   "1.0.0",
			Timestamp: time.Now(),
		})
	}))
	defer ts.Close()

	client := cli.NewClient(ts.URL)
	health, err := client.GetHealth(context.Background())
	if err != nil {
		t.Fatalf("unexpected error fetching health: %v", err)
	}

	if health.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", health.Status)
	}
	if health.Database != "connected" {
		t.Errorf("expected database 'connected', got '%s'", health.Database)
	}
}

func TestClient_QueryMemory(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/memory/query" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []cli.MemoryRecordDTO{
				{
					ID:                   "mem-101",
					DecisionKey:          "boundary:auth/concern:security/decision_type:jwt",
					Concern:              "security",
					DecisionType:         "jwt",
					Boundary:             "auth",
					Summary:              "Use RS256 for token signing",
					ExtractionConfidence: 0.95,
					EvidenceStrength:     0.90,
					CreatedAt:            time.Now(),
				},
			},
			"total": 1,
		})
	}))
	defer ts.Close()

	client := cli.NewClient(ts.URL)
	records, err := client.QueryMemory(context.Background(), cli.QueryMemoryRequest{
		Query: "jwt signing",
	})
	if err != nil {
		t.Fatalf("unexpected error querying memory: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].Boundary != "auth" {
		t.Errorf("expected boundary 'auth', got '%s'", records[0].Boundary)
	}
}

func TestClient_ListGaps(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/gaps" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]cli.KnowledgeGapDTO{
			{
				ID:          "gap-001",
				DecisionKey: "boundary:api/concern:reliability/decision_type:retry",
				Scope:       "global",
				GapType:     "unowned_decision",
				Status:      "open",
				DetectedAt:  time.Now(),
			},
		})
	}))
	defer ts.Close()

	client := cli.NewClient(ts.URL)
	gaps, err := client.ListGaps(context.Background(), "open")
	if err != nil {
		t.Fatalf("unexpected error listing gaps: %v", err)
	}

	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(gaps))
	}
	if gaps[0].ID != "gap-001" {
		t.Errorf("expected gap ID 'gap-001', got '%s'", gaps[0].ID)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := cli.NewClient(ts.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GetHealth(ctx)
	if err == nil {
		t.Fatal("expected error on context timeout, got nil")
	}
}
