package github

import (
	"fmt"
	"strings"
	"zuri-daemon/pkg/extraction"
)

// FormatProposalComment generates the markdown body for Zuri's PR comment proposal.
func FormatProposalComment(result *extraction.ExtractionResult) string {
	if result == nil || len(result.Decisions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("🤖 **Zuri Memory Proposal**\n\n")
	sb.WriteString("I extracted the following architectural decision(s) from this PR:\n\n")

	for _, d := range result.Decisions {
		sb.WriteString(fmt.Sprintf("- **Decision:** %s\n", d.Decision))
		sb.WriteString(fmt.Sprintf("  **Reasoning:** %s\n\n", d.Reasoning))
	}

	sb.WriteString("---\n")
	sb.WriteString("*Reply to this comment with `confirm`, `reject`, or `edit <modified decision>` to update Zuri's brain.*")

	return sb.String()
}
