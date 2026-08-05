package tui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"zuri-daemon/pkg/cli"
)

// Tab represents the active view tab in the TUI dashboard.
type Tab int

const (
	TabGaps Tab = iota
	TabMemory
	TabAudit
	TabRepos
	TabHealth
)

// App orchestrates the Terminal User Interface application loop and state rendering.
type App struct {
	client    cli.DaemonClient
	activeTab Tab
	mu        sync.Mutex
	in        io.Reader
	out       io.Writer

	// View State
	gaps           []cli.KnowledgeGapDTO
	selectedGapIdx int
	memoryRecords  []cli.MemoryRecordDTO
	repositories   []cli.ConnectedRepositoryDTO
	selectedRepoIdx int
	searchQuery    string
	auditLogs      []cli.AuditLogDTO
	health         *cli.HealthResponse
	statusMessage  string
}

// NewApp creates a new TUI App instance given a DaemonClient.
func NewApp(client cli.DaemonClient) *App {
	if client == nil {
		client = cli.NewClient("")
	}
	return &App{
		client:        client,
		activeTab:     TabGaps,
		in:            os.Stdin,
		out:           os.Stdout,
		statusMessage: "System operational. Press [1-5] to switch tabs, [j/k] navigate, [r] refresh, [q] quit.",
	}
}

// Run executes the main interactive TUI loop until context cancellation or user quit.
func (a *App) Run(ctx context.Context) error {
	// Initial data fetch
	a.refreshData(ctx)

	// Background ticker for periodic metrics refresh
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	keyChan := make(chan string, 16)
	go a.readInput(ctx, keyChan)

	a.render()

	for {
		select {
		case <-ctx.Done():
			a.clearScreen()
			fmt.Fprintln(a.out, Style("ZURI TUI exited cleanly.", FgBrightGreen, Bold))
			return nil

		case key := <-keyChan:
			switch strings.ToLower(key) {
			case "q", "ctrl+c":
				a.clearScreen()
				fmt.Fprintln(a.out, Style("ZURI TUI exited cleanly.", FgBrightGreen, Bold))
				return nil
			case "1":
				a.activeTab = TabGaps
				a.statusMessage = "Switched to Knowledge Gaps Inbox."
			case "2":
				a.activeTab = TabMemory
				a.statusMessage = "Switched to Memory Search & Inspector."
			case "3":
				a.activeTab = TabAudit
				a.statusMessage = "Switched to Live System Audit Stream."
			case "4":
				a.activeTab = TabRepos
				a.statusMessage = "Switched to Project Repositories Manager."
			case "5":
				a.activeTab = TabHealth
				a.statusMessage = "Switched to System Health & Diagnostics."
			case "r":
				a.refreshData(ctx)
				a.statusMessage = "Dashboard metrics refreshed successfully."
			case "j", "down":
				a.moveSelection(1)
			case "k", "up":
				a.moveSelection(-1)
			}
			a.render()

		case <-ticker.C:
			a.refreshData(ctx)
			a.render()
		}
	}
}

func (a *App) readInput(ctx context.Context, keyChan chan<- string) {
	reader := bufio.NewReader(a.in)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				keyChan <- trimmed
			}
		}
	}
}

func (a *App) refreshData(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	health, err := a.client.GetHealth(ctx)
	if err == nil {
		a.health = health
	} else {
		a.health = &cli.HealthResponse{Status: "offline", Database: "disconnected"}
	}

	gaps, err := a.client.ListGaps(ctx, "")
	if err == nil {
		a.gaps = gaps
	}

	repos, err := a.client.ListRepositories(ctx)
	if err == nil {
		a.repositories = repos
	}

	logs, err := a.client.GetAuditLogs(ctx, 20)
	if err == nil {
		a.auditLogs = logs
	}
}

func (a *App) moveSelection(delta int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.activeTab == TabGaps {
		if len(a.gaps) == 0 {
			return
		}
		a.selectedGapIdx += delta
		if a.selectedGapIdx < 0 {
			a.selectedGapIdx = 0
		}
		if a.selectedGapIdx >= len(a.gaps) {
			a.selectedGapIdx = len(a.gaps) - 1
		}
	} else if a.activeTab == TabRepos {
		if len(a.repositories) == 0 {
			return
		}
		a.selectedRepoIdx += delta
		if a.selectedRepoIdx < 0 {
			a.selectedRepoIdx = 0
		}
		if a.selectedRepoIdx >= len(a.repositories) {
			a.selectedRepoIdx = len(a.repositories) - 1
		}
	}
}

func (a *App) clearScreen() {
	fmt.Fprint(a.out, "\033[H\033[2J")
}

func (a *App) render() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.clearScreen()
	boxWidth := 96

	// 1. Render Header Banner
	dbBadge := Badge("CONNECTED", FgBlack, BgGreen)
	if a.health == nil || strings.ToLower(a.health.Database) != "connected" {
		dbBadge = Badge("DISCONNECTED", FgWhite, BgRed)
	}

	uptimeStr := "0s"
	if a.health != nil && a.health.Uptime != "" {
		uptimeStr = a.health.Uptime
	}

	headerTitle := Style("⚡ ZURI INTELLIGENCE ENGINE DASHBOARD", FgBrightCyan, Bold)
	timeStr := Style(time.Now().Format("15:04:05"), FgBrightWhite, Bold)

	fmt.Fprintln(a.out, DrawBoxBorderTop(boxWidth, ""))
	fmt.Fprintf(a.out, "│  %s   DB: %s  │  Uptime: %s  │  %s  │\n",
		headerTitle, dbBadge, Style(uptimeStr, FgBrightYellow, Bold), timeStr)
	fmt.Fprintln(a.out, DrawBoxDivider(boxWidth))

	// 2. Render Navigation Bar
	tabTitles := []string{"1: GAPS INBOX", "2: MEMORY ENGINE", "3: AUDIT STREAM", "4: REPOSITORIES", "5: SYSTEM HEALTH"}
	var navParts []string
	for i, t := range tabTitles {
		if Tab(i) == a.activeTab {
			navParts = append(navParts, Badge(fmt.Sprintf(" ❯ %s ❮ ", t), FgBlack, BgBrightCyan))
		} else {
			navParts = append(navParts, Style(fmt.Sprintf("   %s   ", t), FgGray, Dim))
		}
	}
	fmt.Fprintf(a.out, "│ %s │\n", strings.Join(navParts, " "))
	fmt.Fprintln(a.out, DrawBoxDivider(boxWidth))

	// 3. Render Tab Content
	switch a.activeTab {
	case TabGaps:
		a.renderGapsTab(boxWidth)
	case TabMemory:
		a.renderMemoryTab(boxWidth)
	case TabAudit:
		a.renderAuditTab(boxWidth)
	case TabRepos:
		a.renderReposTab(boxWidth)
	case TabHealth:
		a.renderHealthTab(boxWidth)
	}

	// 4. Render Footer & Status Bar
	fmt.Fprintln(a.out, DrawBoxDivider(boxWidth))
	statusFormatted := Style(fmt.Sprintf("ℹ️  STATUS: %s", a.statusMessage), FgBrightYellow)
	fmt.Fprintf(a.out, "│ %-110s │\n", statusFormatted)
	
	keyHint := Style("⌨️  KEYBINDINGS: [1-5] Switch Tabs  │  [j/k] Navigate  │  [r] Refresh  │  [q] Quit", FgGray, Dim)
	fmt.Fprintf(a.out, "│ %-117s │\n", keyHint)
	fmt.Fprintln(a.out, DrawBoxBorderBottom(boxWidth))
}

func (a *App) renderGapsTab(width int) {
	headerStr := Style("📥 KNOWLEDGE GAP INBOX", FgBrightWhite, Bold)
	countStr := Style(fmt.Sprintf("[%d Gaps Detected]", len(a.gaps)), FgBrightCyan)
	fmt.Fprintf(a.out, "│ %s %-80s │\n", headerStr, countStr)
	fmt.Fprintln(a.out, DrawBoxDivider(width))

	if len(a.gaps) == 0 {
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  🟢 No unowned knowledge gaps detected in database.", FgBrightGreen))
		fmt.Fprintln(a.out, "│                                                                                              │")
		return
	}

	// Table Header
	tableHeader := Style(fmt.Sprintf("  %-3s  %-12s  %-14s  %-42s  %-14s", "SEL", "STATUS", "GAP ID", "DECISION KEY", "ROUTED TO"), FgGray, Bold, Underline)
	fmt.Fprintf(a.out, "│ %s │\n", tableHeader)

	for i, g := range a.gaps {
		cursor := " "
		if i == a.selectedGapIdx {
			cursor = Style("❯", FgBrightCyan, Bold)
		}

		statusBadge := Badge("OPEN", FgBlack, BgYellow)
		if strings.ToLower(g.Status) == "resolved" {
			statusBadge = Badge("RESOLVED", FgBlack, BgGreen)
		} else if strings.ToLower(g.Status) == "triaged" {
			statusBadge = Badge("TRIAGED", FgWhite, BgBlue)
		}

		idFormatted := Style(Truncate(g.ID, 14), FgBrightWhite)
		keyFormatted := Style(Truncate(g.DecisionKey, 42), FgBrightCyan, Bold)
		routedFormatted := Style(Truncate(g.RoutedTo, 14), FgMagenta)

		fmt.Fprintf(a.out, "│ %s  %s  %-14s  %-42s  %-14s │\n",
			cursor, statusBadge, idFormatted, keyFormatted, routedFormatted)
	}

	// Render Selected Gap Card Details if available
	if a.selectedGapIdx >= 0 && a.selectedGapIdx < len(a.gaps) {
		selected := a.gaps[a.selectedGapIdx]
		fmt.Fprintln(a.out, DrawBoxDivider(width))
		detailHeader := Style(fmt.Sprintf("🔍 INSPECTING GAP [%s]", selected.ID), FgBrightYellow, Bold)
		fmt.Fprintf(a.out, "│ %-107s │\n", detailHeader)
		fmt.Fprintf(a.out, "│   Decision Key : %s\n", Style(selected.DecisionKey, FgBrightCyan, Bold))
		fmt.Fprintf(a.out, "│   Gap Type     : %s  │  Scope: %s  │  Status: %s\n",
			Style(selected.GapType, FgBrightWhite), Style(selected.Scope, FgGray), Style(selected.Status, FgBrightGreen))
		fmt.Fprintf(a.out, "│   Routed Owner : %s  │  Detected At: %s\n",
			Style(selected.RoutedTo, FgMagenta, Bold), Style(selected.DetectedAt.Format("2006-01-02 15:04:05"), FgGray))
	}
}

func (a *App) renderMemoryTab(width int) {
	headerStr := Style("🧠 DUAL-CONFIDENCE MEMORY INSPECTOR", FgBrightWhite, Bold)
	fmt.Fprintf(a.out, "│ %-107s │\n", headerStr)
	fmt.Fprintln(a.out, DrawBoxDivider(width))

	if len(a.memoryRecords) == 0 {
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  ℹ️  No memory records found in database.", FgGray, Italic))
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  Use 'zuri repo add <path>' and 'zuri query \"<terms>\"' via CLI.", FgBrightCyan))
		return
	}

	for _, rec := range a.memoryRecords {
		extMeter := ProgressBar(rec.ExtractionConfidence, 20)
		evidMeter := ProgressBar(rec.EvidenceStrength, 20)
		
		fmt.Fprintf(a.out, "│  RECORD %s  │  Key: %s\n", Style(rec.ID, FgBrightWhite, Bold), Style(rec.DecisionKey, FgBrightCyan))
		fmt.Fprintf(a.out, "│    Extraction Confidence : %s\n", extMeter)
		fmt.Fprintf(a.out, "│    Materialized Evidence : %s\n", evidMeter)
		fmt.Fprintf(a.out, "│    Summary               : %s\n\n", Style(rec.Summary, FgWhite))
	}
}

func (a *App) renderAuditTab(width int) {
	headerStr := Style("📜 LIVE SYSTEM AUDIT EVENT STREAM", FgBrightWhite, Bold)
	countStr := Style(fmt.Sprintf("[%d Events Recorded]", len(a.auditLogs)), FgBrightCyan)
	fmt.Fprintf(a.out, "│ %s %-80s │\n", headerStr, countStr)
	fmt.Fprintln(a.out, DrawBoxDivider(width))

	if len(a.auditLogs) == 0 {
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  No recent audit events recorded in database.", FgGray, Italic))
		return
	}

	for _, l := range a.auditLogs {
		timeFormatted := Style(l.Timestamp.Format("15:04:05"), FgGray)
		
		eventBadge := Badge(strings.ToUpper(l.EventType), FgBlack, BgCyan)
		if strings.Contains(strings.ToLower(l.EventType), "error") || strings.Contains(strings.ToLower(l.EventType), "fail") {
			eventBadge = Badge(strings.ToUpper(l.EventType), FgWhite, BgRed)
		} else if strings.Contains(strings.ToLower(l.EventType), "gap") {
			eventBadge = Badge(strings.ToUpper(l.EventType), FgBlack, BgYellow)
		}

		actorFormatted := Style(l.Actor, FgBrightGreen, Bold)
		detailsStr := fmt.Sprintf("%v", l.Details)
		if detailsStr == "map[]" {
			detailsStr = "{}"
		}
		detailsFormatted := Style(Truncate(detailsStr, 40), FgGray)

		fmt.Fprintf(a.out, "│  [%s] %s  ACTOR: %-15s  DETAILS: %-40s │\n",
			timeFormatted, eventBadge, actorFormatted, detailsFormatted)
	}
}

func (a *App) renderReposTab(width int) {
	headerStr := Style("📁 CONNECTED PROJECT REPOSITORIES", FgBrightWhite, Bold)
	countStr := Style(fmt.Sprintf("[%d Repositories Connected]", len(a.repositories)), FgBrightCyan)
	fmt.Fprintf(a.out, "│ %s %-80s │\n", headerStr, countStr)
	fmt.Fprintln(a.out, DrawBoxDivider(width))

	if len(a.repositories) == 0 {
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  ℹ️  No repositories onboarded to database.", FgGray, Italic))
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  Run 'zuri repo add <path-or-url>' to onboard your code codebase.", FgBrightCyan, Bold))
		return
	}

	// Table Header
	tableHeader := Style(fmt.Sprintf("  %-3s  %-20s  %-35s  %-12s  %-12s", "SEL", "PROJECT NAME", "LOCAL PATH", "BRANCH", "INDEXING"), FgGray, Bold, Underline)
	fmt.Fprintf(a.out, "│ %s │\n", tableHeader)

	for i, repo := range a.repositories {
		cursor := " "
		if i == a.selectedRepoIdx {
			cursor = Style("❯", FgBrightCyan, Bold)
		}

		idxBadge := Badge("INDEXED", FgBlack, BgGreen)
		if strings.ToLower(repo.IndexingStatus) == "indexing" {
			idxBadge = Badge("INDEXING", FgBlack, BgYellow)
		}

		nameFormatted := Style(Truncate(repo.Name, 20), FgBrightWhite, Bold)
		pathFormatted := Style(Truncate(repo.LocalPath, 35), FgGray)
		branchFormatted := Style(Truncate(repo.DefaultBranch, 12), FgBrightCyan)

		fmt.Fprintf(a.out, "│ %s  %-20s  %-35s  %-12s  %s │\n",
			cursor, nameFormatted, pathFormatted, branchFormatted, idxBadge)
	}

	if a.selectedRepoIdx >= 0 && a.selectedRepoIdx < len(a.repositories) {
		selected := a.repositories[a.selectedRepoIdx]
		fmt.Fprintln(a.out, DrawBoxDivider(width))
		detailHeader := Style(fmt.Sprintf("🔍 INSPECTING REPOSITORY [%s]", selected.ID), FgBrightYellow, Bold)
		fmt.Fprintf(a.out, "│ %-107s │\n", detailHeader)
		fmt.Fprintf(a.out, "│   Name       : %s\n", Style(selected.Name, FgBrightWhite, Bold))
		fmt.Fprintf(a.out, "│   Local Path : %s\n", Style(selected.LocalPath, FgBrightCyan))
		fmt.Fprintf(a.out, "│   Status     : Health=%s  │  Indexing=%s  │  Last Synced=%s\n",
			Style(selected.Health, FgBrightGreen), Style(selected.IndexingStatus, FgBrightYellow), Style(selected.LastSyncedAt.Format("2006-01-02 15:04:05"), FgGray))
	}
}

func (a *App) renderHealthTab(width int) {
	headerStr := Style("🏥 SYSTEM HEALTH & DIAGNOSTICS", FgBrightWhite, Bold)
	fmt.Fprintf(a.out, "│ %-107s │\n", headerStr)
	fmt.Fprintln(a.out, DrawBoxDivider(width))

	if a.health == nil {
		fmt.Fprintf(a.out, "│ %-94s │\n", Style("  🔴 Daemon health status service unavailable.", FgBrightRed, Bold))
		return
	}

	daemonStatusBadge := Badge("RUNNING", FgBlack, BgGreen)
	if strings.ToLower(a.health.Status) != "running" && strings.ToLower(a.health.Status) != "ok" {
		daemonStatusBadge = Badge(strings.ToUpper(a.health.Status), FgWhite, BgRed)
	}

	dbStatusBadge := Badge("CONNECTED", FgBlack, BgGreen)
	if strings.ToLower(a.health.Database) != "connected" {
		dbStatusBadge = Badge(strings.ToUpper(a.health.Database), FgWhite, BgRed)
	}

	fmt.Fprintf(a.out, "│   %-24s : %s\n", Style("Daemon Engine Status", FgBrightWhite, Bold), daemonStatusBadge)
	fmt.Fprintf(a.out, "│   %-24s : %s\n", Style("PostgreSQL Database", FgBrightWhite, Bold), dbStatusBadge)
	fmt.Fprintf(a.out, "│   %-24s : %s\n", Style("Engine Build Version", FgBrightWhite, Bold), Style(a.health.Version, FgBrightCyan, Bold))
	fmt.Fprintf(a.out, "│   %-24s : %s\n", Style("Active Uptime", FgBrightYellow, Bold), Style(a.health.Uptime, FgBrightYellow, Bold))
	fmt.Fprintf(a.out, "│   %-24s : %s\n", Style("Last Diagnostics Timestamp", FgBrightWhite, Bold), Style(a.health.Timestamp.Format("2006-01-02 15:04:05 MST"), FgGray))
}
