package tui

import (
	"fmt"
	"strings"
)

// ANSI terminal escape codes for colors and text formatting.
const (
	Reset         = "\033[0m"
	Bold          = "\033[1m"
	Dim           = "\033[2m"
	Italic        = "\033[3m"
	Underline     = "\033[4m"
	Inverse       = "\033[7m"

	// Foreground Colors
	FgBlack       = "\033[30m"
	FgRed         = "\033[31m"
	FgGreen       = "\033[32m"
	FgYellow      = "\033[33m"
	FgBlue        = "\033[34m"
	FgMagenta     = "\033[35m"
	FgCyan        = "\033[36m"
	FgWhite       = "\033[37m"
	FgBrightRed   = "\033[91m"
	FgBrightGreen = "\033[92m"
	FgBrightYellow= "\033[93m"
	FgBrightBlue  = "\033[94m"
	FgBrightCyan  = "\033[96m"
	FgBrightWhite = "\033[97m"
	FgGray        = "\033[90m"

	// Background Colors
	BgBlack       = "\033[40m"
	BgRed         = "\033[41m"
	BgGreen       = "\033[42m"
	BgYellow      = "\033[43m"
	BgBlue        = "\033[44m"
	BgMagenta     = "\033[45m"
	BgCyan        = "\033[46m"
	BgWhite       = "\033[47m"
	BgBrightCyan  = "\033[106m"
	BgDarkGray    = "\033[100m"
)

// Style wraps text with ANSI escape codes and resets formatting afterwards.
func Style(text string, ansiCodes ...string) string {
	if len(ansiCodes) == 0 {
		return text
	}
	var sb strings.Builder
	for _, code := range ansiCodes {
		sb.WriteString(code)
	}
	sb.WriteString(text)
	sb.WriteString(Reset)
	return sb.String()
}

// Badge returns a padded badge with specified foreground and background ANSI colors.
func Badge(text string, fg string, bg string) string {
	return fmt.Sprintf("%s%s%s %s %s", bg, fg, Bold, text, Reset)
}

// ProgressBar renders a visual meter (e.g. [██████████░░░░░░░░░░] 50%).
func ProgressBar(value float64, width int) string {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	filledLen := int(value * float64(width))
	if filledLen > width {
		filledLen = width
	}
	emptyLen := width - filledLen

	var filledStr string
	if value >= 0.8 {
		filledStr = Style(strings.Repeat("█", filledLen), FgBrightGreen, Bold)
	} else if value >= 0.5 {
		filledStr = Style(strings.Repeat("█", filledLen), FgBrightYellow, Bold)
	} else {
		filledStr = Style(strings.Repeat("█", filledLen), FgBrightCyan, Bold)
	}

	emptyStr := Style(strings.Repeat("░", emptyLen), FgGray)
	pctStr := Style(fmt.Sprintf("%3d%%", int(value*100)), FgBrightWhite, Bold)

	return fmt.Sprintf("[%s%s] %s", filledStr, emptyStr, pctStr)
}

// Truncate ensures a string does not exceed maxLen, appending ellipsis if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// DrawBoxBorderTop renders a rounded box top border line.
func DrawBoxBorderTop(width int, title string) string {
	if title != "" {
		titlePadded := fmt.Sprintf(" %s ", title)
		titleLen := len(titlePadded)
		if titleLen < width-4 {
			rightLen := width - 2 - titleLen
			return Style("╭", FgGray) + Style(titlePadded, FgBrightCyan, Bold) + Style(strings.Repeat("─", rightLen)+"╮", FgGray)
		}
	}
	return Style("╭"+strings.Repeat("─", width-2)+"╮", FgGray)
}

// DrawBoxBorderBottom renders a rounded box bottom border line.
func DrawBoxBorderBottom(width int) string {
	return Style("╰"+strings.Repeat("─", width-2)+"╯", FgGray)
}

// DrawBoxDivider renders a box section divider.
func DrawBoxDivider(width int) string {
	return Style("├"+strings.Repeat("─", width-2)+"┤", FgGray)
}
