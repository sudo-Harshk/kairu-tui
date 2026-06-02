package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderHelpView(m model) string {
	footer := "[?] Close   [Esc] Close   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	lines := []string{
		"Timer continues while help is open.",
		"",
		"Global:",
		formatHelpLine("?", "Toggle help"),
		formatHelpLine("q", "Quit"),
		"",
		"Input mode:",
		formatHelpLine("Tab", "Switch field"),
		formatHelpLine("Enter", "Start session"),
		formatHelpLine("Ctrl+P", "Templates"),
		formatHelpLine("Ctrl+B", "Soundscapes"),
		"",
		"Timer/Break:",
		formatHelpLine("Space", "Pause/Resume"),
		formatHelpLine("E", "Edit time"),
		formatHelpLine("Enter", "End session"),
		formatHelpLine("Tab", "Stats"),
		formatHelpLine("S", "Settings"),
		formatHelpLine("Ctrl+B", "Soundscapes"),
		"",
		"Edit:",
		formatHelpLine("Enter", "Apply"),
		formatHelpLine("Esc", "Cancel"),
		"",
		"Stats Views:",
		formatHelpLine("Tab", "Cycle views"),
		formatHelpLine("Esc", "Back to timer"),
		formatHelpLine("R", "Daily report"),
		formatHelpLine("S", "Settings"),
		formatHelpLine("L", "Internal logs"),
		"",
	}
	body := lipgloss.NewStyle().Width(35).Render(strings.Join(lines, "\n"))
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  Help                               │
╰─────────────────────────────────────╯

%s

%s`, renderBanner(m.config), body, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderFatalView(m model) string {
	message := strings.TrimSpace(m.appError)
	if message == "" {
		message = "Failed to start due to an unexpected error."
	}
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  Startup Error                      │
╰─────────────────────────────────────╯

%s

[q] Quit`, renderBanner(m.config), message)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}
