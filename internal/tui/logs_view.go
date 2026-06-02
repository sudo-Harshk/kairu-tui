package tui

import (
	"fmt"
	"strings"
)

func renderLogView(m model) string {
	lines := make([]string, 0, len(m.internalLogs))
	for _, l := range m.internalLogs {
		lines = append(lines, "  "+l)
	}
	if len(lines) == 0 {
		lines = append(lines, "  (No logs recorded yet)")
	}

	footer := "[Esc] Back   [q] Quit"
	block := renderBanner(m.config) + "\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  Internal Event Logs                │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		strings.Join(lines, "\n") + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}
