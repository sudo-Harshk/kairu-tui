package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/ui"
)

func renderHelpView(m model) string {
	theme := activeTheme(m.config)
	errorLine := renderAppError(m)
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
	}
	body := strings.Join(lines, "\n")

	formWidth := 46
	helpCard := ui.Panel("📖 Help Guidelines", body, theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)

	shortcuts := []string{"[?] Close", "[Esc] Close", "[q] Quit"}
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, formWidth)

	block := helpCard + "\n\n" + statusBar
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderFatalView(m model) string {
	theme := activeTheme(m.config)
	message := strings.TrimSpace(m.appError)
	if message == "" {
		message = "Failed to start due to an unexpected error."
	}

	formWidth := 46
	errorCard := ui.Panel("❌ Startup Error", message, theme, formWidth, lipgloss.DoubleBorder(), theme.Warning)
	statusBar := ui.StatusBar([]string{"[q] Quit"}, "", theme, formWidth)

	block := errorCard + "\n\n" + statusBar
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}
