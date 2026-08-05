package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"zuri-daemon/pkg/cli"
	"zuri-daemon/pkg/tui"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Single-word launch ("zuri" with zero arguments) or "zuri tui"
	if len(os.Args) == 1 || (len(os.Args) > 1 && os.Args[1] == "tui") {
		client := cli.NewClient("")

		// Check if daemon is active
		health, err := client.GetHealth(ctx)
		if err != nil || health == nil || health.Status == "offline" {
			fmt.Fprintln(os.Stdout, "Starting ZURI daemon in background (Postgres init may take 15-30s)...")
			if err := ensureDaemonStarted(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting ZURI daemon: %v\n", err)
				os.Exit(1)
			}

			// Poll for daemon health up to 30 seconds for embedded postgres recovery / extraction
			if err := waitForDaemon(ctx, client, 30*time.Second); err != nil {
				fmt.Fprintf(os.Stderr, "Daemon health check timed out: %v\n", err)
				os.Exit(1)
			}
		}

		// Launch Interactive Terminal IDE Dashboard
		app := tui.NewApp(client)
		if err := app.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 2. Subcommand or help execution
	runner := cli.NewRunner(nil, os.Stdout, os.Stderr)
	exitCode := runner.Execute(ctx, os.Args[1:])
	os.Exit(exitCode)
}

func ensureDaemonStarted() error {
	exPath, err := os.Executable()
	if err != nil {
		exPath = "."
	}

	daemonPath := filepath.Join(filepath.Dir(exPath), "zuri-daemon.exe")
	if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
		daemonPath = filepath.Join(".", "zuri-daemon.exe")
	}

	cmd := exec.Command(daemonPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func waitForDaemon(ctx context.Context, client cli.DaemonClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			health, err := client.GetHealth(ctx)
			if err == nil && health != nil && (health.Status == "ok" || health.Status == "running") {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("daemon failed to reach healthy state within %s", timeout)
			}
		}
	}
}
