package tui

import (
	"fmt"
	"io"
	"strings"
)

// PaletteItem defines an entry in the ZURI Command Palette overlay.
type PaletteItem struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Preview     string `json:"preview"`
}

// CommandPalette manages fuzzy command searching and interactive preview rendering.
type CommandPalette struct {
	items         []PaletteItem
	selectedIndex int
}

// NewCommandPalette initializes the default command palette items.
func NewCommandPalette() *CommandPalette {
	return &CommandPalette{
		items: []PaletteItem{
			{
				Command:     "repo add <path-or-url>",
				Description: "Onboard a local repository path or remote Git URL",
				Preview:     "Triggers Tree-sitter AST parsing and vector embedding indexer",
			},
			{
				Command:     "repo sync [repo-id]",
				Description: "Trigger incremental re-indexing on a repository",
				Preview:     "Computes SHA-256 Git commit deltas and updates modified AST nodes",
			},
			{
				Command:     "model switch <profile>",
				Description: "Switch runtime profile (e.g. local-gpu -> cloud-anthropic)",
				Preview:     "Updates active LLM, embedding, and reranker model allocations",
			},
			{
				Command:     "gaps triage",
				Description: "Interactive review inbox for unresolved architectural gaps",
				Preview:     "Displays open decision key gaps and surfaces CODEOWNERS routes",
			},
			{
				Command:     "context view",
				Description: "Inspect current Context Engine payload and token packing",
				Preview:     "Displays 9-stage context synthesis results and proximity multipliers",
			},
		},
	}
}

// Filter returns palette items matching query string.
func (cp *CommandPalette) Filter(query string) []PaletteItem {
	if query == "" {
		return cp.items
	}

	q := strings.ToLower(query)
	var matches []PaletteItem
	for _, item := range cp.items {
		if strings.Contains(strings.ToLower(item.Command), q) || strings.Contains(strings.ToLower(item.Description), q) {
			matches = append(matches, item)
		}
	}
	return matches
}

// RenderOverlay renders the styled Command Palette overlay on the terminal writer.
func (cp *CommandPalette) RenderOverlay(w io.Writer, query string) {
	width := 96
	fmt.Fprintln(w, DrawBoxBorderTop(width, "ZURI COMMAND PALETTE (Fuzzy Search)"))
	
	searchHeader := fmt.Sprintf("  Search: %s", Style(query, FgBrightYellow, Bold, Underline))
	fmt.Fprintf(w, "│ %-107s │\n", searchHeader)
	fmt.Fprintln(w, DrawBoxDivider(width))

	matches := cp.Filter(query)
	if len(matches) == 0 {
		fmt.Fprintf(w, "│ %-94s │\n", Style("  No matching commands found.", FgGray, Italic))
	} else {
		for i, item := range matches {
			cmdStr := Truncate(item.Command, 28)
			descStr := Truncate(item.Description, 54)
			
			if i == cp.selectedIndex {
				cursor := Style("❯", FgBrightCyan, Bold)
				cmdFormatted := Style(fmt.Sprintf("%-28s", cmdStr), FgBrightCyan, Bold)
				descFormatted := Style(fmt.Sprintf("%-54s", descStr), FgBrightWhite)
				fmt.Fprintf(w, "│ %s %s  %s │\n", cursor, cmdFormatted, descFormatted)
			} else {
				cursor := " "
				cmdFormatted := Style(fmt.Sprintf("%-28s", cmdStr), FgWhite)
				descFormatted := Style(fmt.Sprintf("%-54s", descStr), FgGray)
				fmt.Fprintf(w, "│ %s %s  %s │\n", cursor, cmdFormatted, descFormatted)
			}
		}
	}

	fmt.Fprintln(w, DrawBoxDivider(width))
	hint := Style("  [UP/DOWN] Select  │  [ENTER] Execute  │  [ESC] Dismiss", FgGray, Dim)
	fmt.Fprintf(w, "│ %-107s │\n", hint)
	fmt.Fprintln(w, DrawBoxBorderBottom(width))
}
