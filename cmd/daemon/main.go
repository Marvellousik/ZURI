package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zuri-daemon/pkg/db"
	"zuri-daemon/pkg/mcp"
	"zuri-daemon/pkg/server"
	"zuri-daemon/pkg/webhooks"
)

func main() {
	startTime := time.Now()
	log.Println("Starting Zuri daemon...")

	port := os.Getenv("ZURI_PORT")
	if port == "" {
		port = "7331"
	}

	host := os.Getenv("ZURI_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	dbMgr := db.NewDBManager()
	if err := dbMgr.Init(); err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer func() {
		if err := dbMgr.Close(); err != nil {
			log.Printf("Error closing database manager: %v", err)
		}
	}()

	log.Println("Embedded Postgres database initialized successfully.")

	appliedCount, err := db.RunMigrations(dbMgr.GetDB())
	if err != nil {
		log.Fatalf("Fatal: Database migration failed: %v", err)
	}

	if appliedCount > 0 {
		log.Printf("Applied %d new database migration(s).", appliedCount)
	} else {
		log.Println("Database schema is up to date.")
	}

	healthSvr := server.NewHealthServer(dbMgr.GetDB(), startTime)
	mcpHandler := mcp.NewServerHandler(dbMgr.GetDB())

	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthSvr.HandleHealthCheck)
	mux.HandleFunc("/api/health", healthSvr.HandleHealthCheck)
	mux.Handle("/mcp/", mcpHandler)
	mux.Handle("/mcp", mcpHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Not Found",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"service": "zuri-daemon",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// Placeholder route groups for future build phases
	// MCP server, GitHub webhook receiver, and SSE activity stream attach here in later sessions.
	unimplementedHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Endpoint not implemented in session S1",
		})
	}

	githubWebhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	webhookHandler := webhooks.NewGitHubWebhookHandler(githubWebhookSecret)
	mux.Handle("/webhooks/github", webhookHandler)

	mux.HandleFunc("/events/", unimplementedHandler)

	addr := fmt.Sprintf("%s:%s", host, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Zuri daemon listening on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sig := <-shutdownChan
	log.Printf("Received signal %v. Shutting down Zuri daemon...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Zuri daemon stopped gracefully.")
}
