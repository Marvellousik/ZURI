package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status        string `json:"status"`
	Database      string `json:"database"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Timestamp     string `json:"timestamp"`
	Error         string `json:"error,omitempty"`
}

type HealthServer struct {
	db        *sql.DB
	startTime time.Time
}

func NewHealthServer(db *sql.DB, startTime time.Time) *HealthServer {
	return &HealthServer{
		db:        db,
		startTime: startTime,
	}
}

func (h *HealthServer) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	uptime := int64(time.Since(h.startTime).Seconds())
	now := time.Now().UTC().Format(time.RFC3339)

	var alive int
	err := h.db.QueryRowContext(r.Context(), "SELECT 1;").Scan(&alive)

	if err != nil || alive != 1 {
		w.WriteHeader(http.StatusServiceUnavailable)
		errMsg := "database connection failed"
		if err != nil {
			errMsg = err.Error()
		}
		json.NewEncoder(w).Encode(HealthResponse{
			Status:        "error",
			Database:      "disconnected",
			UptimeSeconds: uptime,
			Timestamp:     now,
			Error:         errMsg,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:        "ok",
		Database:      "connected",
		UptimeSeconds: uptime,
		Timestamp:     now,
	})
}
