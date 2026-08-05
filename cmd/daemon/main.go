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

	"zuri-daemon/pkg/api"
	"zuri-daemon/pkg/db"
	"zuri-daemon/pkg/mcp"
	"zuri-daemon/pkg/server"
	"zuri-daemon/pkg/sse"
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
	defer dbMgr.Close()

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
	apiHandler := api.NewAPIHandler(dbMgr.GetDB())
	sseServer := sse.NewServer()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthSvr.HandleHealthCheck)
	mux.HandleFunc("/api/health", healthSvr.HandleHealthCheck)
	mux.HandleFunc("/api/repositories", apiHandler.HandleRepositories)
	mux.HandleFunc("/api/onboarding", apiHandler.HandleOnboarding)
	mux.HandleFunc("/api/memory/query", apiHandler.HandleQueryMemory)
	mux.HandleFunc("/api/audit-log", apiHandler.HandleAuditLog)
	mux.HandleFunc("/api/agents", apiHandler.HandleAgentRegistrations)
	mux.HandleFunc("/api/gaps", apiHandler.HandleGaps)
	mux.HandleFunc("/api/gaps/resolve", apiHandler.HandleGapResolution)
	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})
		go func() {
			time.Sleep(100 * time.Millisecond)
			shutdownChan <- syscall.SIGTERM
		}()
	})
	mux.Handle("/mcp/", mcpHandler)
	mux.Handle("/mcp", mcpHandler)
	mux.Handle("/events/", sseServer)
	mux.Handle("/events", sseServer)

	githubWebhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	webhookHandler := webhooks.NewGitHubWebhookHandler(githubWebhookSecret)
	mux.Handle("/webhooks/github", webhookHandler)

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

	addr := fmt.Sprintf("%s:%s", host, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

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
