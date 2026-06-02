package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/ui"
)

func renderLogView(m model) string {
	theme := activeTheme(m.config)
	lines := make([]string, 0, len(m.internalLogs))
	for _, l := range m.internalLogs {
		lines = append(lines, "  "+l)
	}
	if len(lines) == 0 {
		lines = append(lines, "  (No logs recorded yet)")
	}

	formWidth := 60
	if m.width > 0 && m.width < 70 {
		formWidth = m.width - 6
	}

	logsCard := ui.Panel("📋 Internal Event Logs", strings.Join(lines, "\n"), theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)
	statusBar := ui.StatusBar([]string{"[Esc] Back", "[q] Quit"}, "", theme, formWidth)

	block := renderBanner(m.config) + "\n\n" + logsCard + "\n\n" + statusBar
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}
