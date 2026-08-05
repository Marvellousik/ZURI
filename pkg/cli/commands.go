package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"zuri-daemon/pkg/mcp"
)

// Runner encapsulates CLI command execution with standard input/output stream dependency injection.
type Runner struct {
	client DaemonClient
	out    io.Writer
	errOut io.Writer
	in     io.Reader
}

// NewRunner creates a new CLI Runner instance.
func NewRunner(client DaemonClient, out, errOut io.Writer) *Runner {
	if client == nil {
		client = NewClient("")
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	return &Runner{
		client: client,
		out:    out,
		errOut: errOut,
		in:     os.Stdin,
	}
}

// Execute parses command line arguments and routes execution to the corresponding subcommand.
func (r *Runner) Execute(ctx context.Context, args []string) int {
	if len(args) == 0 {
		r.printUsage()
		return 0
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "status", "health":
		return r.runStatus(ctx)
	case "query":
		return r.runQuery(ctx, subArgs)
	case "gaps":
		return r.runGaps(ctx, subArgs)
	case "repo", "repository", "repositories":
		return r.runRepo(ctx, subArgs)
	case "onboard", "onboarding":
		return r.runOnboard(ctx, subArgs)
	case "audit":
		return r.runAudit(ctx, subArgs)
	case "daemon":
		return r.runDaemon(ctx, subArgs)
	case "mcp":
		return r.runMCP(ctx, subArgs)
	case "help", "-h", "--help":
		r.printUsage()
		return 0
	default:
		fmt.Fprintf(r.errOut, "Unknown command: %s\nRun 'zuri help' for usage instructions.\n", subcommand)
		return 1
	}
}

func (r *Runner) runStatus(ctx context.Context) int {
	health, err := r.client.GetHealth(ctx)
	if err != nil {
		fmt.Fprintf(r.errOut, "Error connecting to ZURI daemon: %v\n", err)
		fmt.Fprintln(r.errOut, "Ensure zuri-daemon is running (execute 'zuri daemon start').")
		return 1
	}

	w := tabwriter.NewWriter(r.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC\tVALUE")
	fmt.Fprintln(w, "------\t-----")
	fmt.Fprintf(w, "Status\t%s\n", health.Status)
	fmt.Fprintf(w, "Database\t%s\n", health.Database)
	fmt.Fprintf(w, "Version\t%s\n", health.Version)
	fmt.Fprintf(w, "Uptime\t%s\n", health.Uptime)
	fmt.Fprintf(w, "Server Time\t%s\n", health.Timestamp.Format(time.RFC3339))
	_ = w.Flush()

	return 0
}

func (r *Runner) runRepo(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: zuri repo [list|add|remove]")
		return 1
	}

	action := args[0]
	subArgs := args[1:]

	switch action {
	case "list":
		repos, err := r.client.ListRepositories(ctx)
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed listing connected repositories: %v\n", err)
			return 1
		}

		if len(repos) == 0 {
			fmt.Fprintln(r.out, "No repositories connected to Zuri memory yet.")
			fmt.Fprintln(r.out, "Use 'zuri repo add <path-or-github-url>' to onboard a project.")
			return 0
		}

		w := tabwriter.NewWriter(r.out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "REPO ID\tNAME\tLOCAL PATH\tBRANCH\tINDEXING STATUS\tHEALTH")
		fmt.Fprintln(w, "-------\t----\t----------\t------\t---------------\t------")
		for _, repo := range repos {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				repo.ID, repo.Name, repo.LocalPath, repo.DefaultBranch, repo.IndexingStatus, repo.Health)
		}
		_ = w.Flush()
		return 0

	case "add":
		fs := flag.NewFlagSet("repo add", flag.ContinueOnError)
		fs.SetOutput(r.errOut)
		nameFlag := fs.String("name", "", "Display name for project repository")
		if err := fs.Parse(subArgs); err != nil {
			return 1
		}

		targetPath := fs.Arg(0)
		if targetPath == "" {
			var err error
			targetPath, err = os.Getwd()
			if err != nil {
				fmt.Fprintln(r.errOut, "Usage: zuri repo add <path-or-github-url> [--name \"Project Name\"]")
				return 1
			}
		}

		absPath, err := filepath.Abs(targetPath)
		if err == nil {
			targetPath = absPath
		}

		projName := *nameFlag
		if projName == "" {
			projName = filepath.Base(targetPath)
		}

		repo, err := r.client.AddRepository(ctx, projName, targetPath, "")
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed connecting repository: %v\n", err)
			return 1
		}

		fmt.Fprintf(r.out, "Successfully connected project '%s' (ID: %s).\n", repo.Name, repo.ID)
		fmt.Fprintln(r.out, "Background AST parsing and vector indexing started.")
		return 0

	case "remove", "rm":
		if len(subArgs) == 0 {
			fmt.Fprintln(r.errOut, "Usage: zuri repo remove <repo-id>")
			return 1
		}
		repoID := subArgs[0]
		err := r.client.RemoveRepository(ctx, repoID)
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed removing repository %s: %v\n", repoID, err)
			return 1
		}
		fmt.Fprintf(r.out, "Repository %s successfully removed from Zuri memory.\n", repoID)
		return 0

	default:
		fmt.Fprintf(r.errOut, "Unknown repo command '%s'. Use 'list', 'add', or 'remove'.\n", action)
		return 1
	}
}

func (r *Runner) runOnboard(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("onboard", flag.ContinueOnError)
	fs.SetOutput(r.errOut)
	repoID := fs.String("repo", "", "Target repository ID")
	founder := fs.String("founder", "", "Founder or Lead Architect name")
	decision := fs.String("decision", "", "Architectural decision summary")
	reasoning := fs.String("reasoning", "", "Tradeoff rationale & reasoning")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targetRepo := *repoID
	if targetRepo == "" {
		repos, err := r.client.ListRepositories(ctx)
		if err == nil && len(repos) > 0 {
			targetRepo = repos[0].ID
		} else {
			targetRepo = "repo-main"
		}
	}

	targetFounder := *founder
	if targetFounder == "" {
		targetFounder = "lead-architect"
	}

	var memories []OnboardMemoryDTO

	if strings.TrimSpace(*decision) != "" {
		memories = append(memories, OnboardMemoryDTO{
			Decision:  *decision,
			Reasoning: *reasoning,
		})
	} else {
		// Interactive CLI onboarding survey
		fmt.Fprintln(r.out, "==========================================================")
		fmt.Fprintln(r.out, "  ZURI PROJECT ONBOARDING SURVEY (Canonical Memory)")
		fmt.Fprintln(r.out, "==========================================================")
		fmt.Fprintln(r.out, "Capture founder/lead architectural decisions into PostgreSQL memory.")
		fmt.Fprintln(r.out, "")

		scanner := bufio.NewScanner(r.in)
		
		fmt.Fprint(r.out, "1. Key Architectural Decision (e.g. Auth model, Database choice): ")
		if scanner.Scan() {
			decText := strings.TrimSpace(scanner.Text())
			if decText != "" {
				fmt.Fprint(r.out, "2. Rationale / Tradeoff Reasoning for this decision: ")
				reasonText := ""
				if scanner.Scan() {
					reasonText = strings.TrimSpace(scanner.Text())
				}
				memories = append(memories, OnboardMemoryDTO{
					Decision:  decText,
					Reasoning: reasonText,
				})
			}
		}
	}

	if len(memories) == 0 {
		fmt.Fprintln(r.errOut, "No onboarding memories entered. Exiting.")
		return 1
	}

	err := r.client.OnboardProject(ctx, OnboardRequestDTO{
		RepoID:   targetRepo,
		Founder:  targetFounder,
		Memories: memories,
	})
	if err != nil {
		fmt.Fprintf(r.errOut, "Failed persisting onboarding survey memories: %v\n", err)
		return 1
	}

	fmt.Fprintf(r.out, "Successfully persisted %d canonical architectural memory record(s) to database.\n", len(memories))
	return 0
}

func (r *Runner) runQuery(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(r.errOut)

	boundary := fs.String("boundary", "", "Filter by system boundary (e.g. auth, api)")
	concern := fs.String("concern", "", "Filter by concern enum (e.g. security, reliability)")
	minConfidence := fs.Float64("min-confidence", 0.0, "Filter by minimum confidence score (0.0 to 1.0)")
	limit := fs.Int("limit", 10, "Maximum records to return")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	queryStr := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(queryStr) == "" {
		fmt.Fprintln(r.errOut, "Usage: zuri query [flags] \"<search query>\"")
		return 1
	}

	records, err := r.client.QueryMemory(ctx, QueryMemoryRequest{
		Query:         queryStr,
		Boundary:      *boundary,
		Concern:       *concern,
		MinConfidence: *minConfidence,
		Limit:         *limit,
	})
	if err != nil {
		fmt.Fprintf(r.errOut, "Memory query failed: %v\n", err)
		return 1
	}

	if len(records) == 0 {
		fmt.Fprintln(r.out, "No matching memory records found in database.")
		return 0
	}

	w := tabwriter.NewWriter(r.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDECISION KEY\tCONF (EXT/EVID)\tSUMMARY")
	fmt.Fprintln(w, "--\t------------\t---------------\t-------")
	for _, rec := range records {
		dKey := rec.DecisionKey
		if dKey == "" {
			dKey = fmt.Sprintf("boundary:%s/concern:%s/decision_type:%s", rec.Boundary, rec.Concern, rec.DecisionType)
		}
		fmt.Fprintf(w, "%s\t%s\t%.2f / %.2f\t%s\n", rec.ID, dKey, rec.ExtractionConfidence, rec.EvidenceStrength, rec.Summary)
	}
	_ = w.Flush()

	return 0
}

func (r *Runner) runGaps(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: zuri gaps [list|resolve]")
		return 1
	}

	action := args[0]
	subArgs := args[1:]

	switch action {
	case "list":
		fs := flag.NewFlagSet("gaps list", flag.ContinueOnError)
		fs.SetOutput(r.errOut)
		statusFilter := fs.String("status", "open", "Filter gaps by status (open, surfaced, answered, acknowledged_unknown)")
		if err := fs.Parse(subArgs); err != nil {
			return 1
		}

		gaps, err := r.client.ListGaps(ctx, *statusFilter)
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed listing knowledge gaps: %v\n", err)
			return 1
		}

		if len(gaps) == 0 {
			fmt.Fprintf(r.out, "No knowledge gaps found with status '%s'.\n", *statusFilter)
			return 0
		}

		w := tabwriter.NewWriter(r.out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GAP ID\tGAP TYPE\tSTATUS\tDECISION KEY\tROUTED TO")
		fmt.Fprintln(w, "------\t--------\t------\t------------\t---------")
		for _, g := range gaps {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", g.ID, g.GapType, g.Status, g.DecisionKey, g.RoutedTo)
		}
		_ = w.Flush()
		return 0

	case "resolve":
		fs := flag.NewFlagSet("gaps resolve", flag.ContinueOnError)
		fs.SetOutput(r.errOut)
		answer := fs.String("answer", "", "Resolution answer text for the knowledge gap")
		ackUnknown := fs.Bool("acknowledge-unknown", false, "Flag gap as acknowledged unknown architectural decision")
		if err := fs.Parse(subArgs); err != nil {
			return 1
		}

		gapID := fs.Arg(0)
		if gapID == "" {
			fmt.Fprintln(r.errOut, "Usage: zuri gaps resolve <gap_id> [--answer \"<text>\" | --acknowledge-unknown]")
			return 1
		}

		reqAction := "answer"
		if *ackUnknown {
			reqAction = "acknowledge_unknown"
		} else if strings.TrimSpace(*answer) == "" {
			fmt.Fprintln(r.errOut, "Error: Must provide either --answer \"<text>\" or --acknowledge-unknown flag.")
			return 1
		}

		err := r.client.ResolveGap(ctx, gapID, ResolveGapRequest{
			Action: reqAction,
			Answer: *answer,
		})
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed resolving gap %s: %v\n", gapID, err)
			return 1
		}

		fmt.Fprintf(r.out, "Knowledge gap %s successfully resolved (%s).\n", gapID, reqAction)
		return 0

	default:
		fmt.Fprintf(r.errOut, "Unknown gaps command '%s'. Use 'list' or 'resolve'.\n", action)
		return 1
	}
}

func (r *Runner) runAudit(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(r.errOut)
	limit := fs.Int("limit", 20, "Maximum log entries to return")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	logs, err := r.client.GetAuditLogs(ctx, *limit)
	if err != nil {
		fmt.Fprintf(r.errOut, "Failed fetching audit logs: %v\n", err)
		return 1
	}

	if len(logs) == 0 {
		fmt.Fprintln(r.out, "No audit log entries recorded in database.")
		return 0
	}

	w := tabwriter.NewWriter(r.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIMESTAMP\tEVENT TYPE\tACTOR\tDETAILS")
	fmt.Fprintln(w, "---------\t----------\t-----\t-------")
	for _, l := range logs {
		detailsStr, _ := json.Marshal(l.Details)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", l.Timestamp.Format(time.RFC3339), l.EventType, l.Actor, string(detailsStr))
	}
	_ = w.Flush()

	return 0
}

func (r *Runner) runDaemon(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(r.errOut, "Usage: zuri daemon [start|stop|status]")
		return 1
	}

	action := args[0]
	switch action {
	case "status":
		return r.runStatus(ctx)

	case "stop":
		err := r.client.ShutdownDaemon(ctx)
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed stopping zuri-daemon: %v\n", err)
			return 1
		}
		fmt.Fprintln(r.out, "Zuri daemon shutdown signal transmitted successfully.")
		return 0

	case "start":
		exPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(r.errOut, "Failed resolving executable path: %v\n", err)
			return 1
		}

		daemonPath := filepath.Join(filepath.Dir(exPath), "zuri-daemon.exe")
		if _, err := os.Stat(daemonPath); os.IsNotExist(err) {
			daemonPath = filepath.Join(".", "zuri-daemon.exe")
		}

		cmd := exec.Command(daemonPath)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(r.errOut, "Failed starting daemon binary at %s: %v\n", daemonPath, err)
			return 1
		}

		fmt.Fprintf(r.out, "Zuri daemon process launched in background (PID: %d).\n", cmd.Process.Pid)
		return 0

	default:
		fmt.Fprintf(r.errOut, "Unknown daemon command '%s'. Use start, stop, or status.\n", action)
		return 1
	}
}

func (r *Runner) runMCP(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] != "config" {
		fmt.Fprintln(r.errOut, "Usage: zuri mcp config")
		return 1
	}

	execName := "zuri-daemon.exe"
	host := "127.0.0.1"
	port := 7331

	mcp.RegisterAllLocalAgentMCPConfigs(execName, host, port)
	fmt.Fprintln(r.out, "Successfully registered ZURI MCP server into local configuration files:")
	fmt.Fprintln(r.out, "  - Workspace .mcp.json")
	fmt.Fprintln(r.out, "  - Claude Desktop config")
	fmt.Fprintln(r.out, "  - Gemini CLI config")

	return 0
}

func (r *Runner) printUsage() {
	fmt.Fprintln(r.out, "ZURI — Evidence-First Engineering Memory System (CLI)")
	fmt.Fprintln(r.out, "")
	fmt.Fprintln(r.out, "USAGE:")
	fmt.Fprintln(r.out, "  zuri <command> [flags] [arguments]")
	fmt.Fprintln(r.out, "")
	fmt.Fprintln(r.out, "COMMANDS:")
	fmt.Fprintln(r.out, "  status                Check daemon connectivity and database health")
	fmt.Fprintln(r.out, "  repo <list|add|rm>    Onboard and manage connected project repositories")
	fmt.Fprintln(r.out, "  onboard               Capture founder/lead architectural survey memories into PostgreSQL")
	fmt.Fprintln(r.out, "  query <query>         Query memory records (flags: --boundary, --concern, --min-confidence, --limit)")
	fmt.Fprintln(r.out, "  gaps list             List detected knowledge gaps (flags: --status)")
	fmt.Fprintln(r.out, "  gaps resolve <id>     Resolve a gap (flags: --answer, --acknowledge-unknown)")
	fmt.Fprintln(r.out, "  audit                 View operational audit logs (flags: --limit)")
	fmt.Fprintln(r.out, "  daemon <start|stop>   Control background zuri-daemon process")
	fmt.Fprintln(r.out, "  mcp config            Auto-register ZURI MCP server in local agent configs")
	fmt.Fprintln(r.out, "  tui                   Launch interactive terminal user interface dashboard")
	fmt.Fprintln(r.out, "  help                  Display usage information")
}
