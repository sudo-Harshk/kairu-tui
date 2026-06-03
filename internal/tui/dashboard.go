package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/ui"
)

func renderDashboard(m model, leftContent string) string {
	theme := activeTheme(m.config)

	// 1. Outer margin — leave 1-cell border around entire app
	availWidth := m.width - 2
	availHeight := m.height - 2

	if availWidth < 0 {
		availWidth = 0
	}
	if availHeight < 0 {
		availHeight = 0
	}

	var dashboard string

	// Phase 7: Responsive Breakpoints
	if m.width < 60 {
		// When width < 60: sidebar hidden, show only left pane
		leftPanel := lipgloss.NewStyle().
			Width(availWidth).
			Padding(1, 3).
			Render(leftContent)
		dashboard = leftPanel
	} else if m.width < 90 {
		// When width < 90: collapse to single-column (stack panes vertically, sidebar below clock)
		leftPanel := lipgloss.NewStyle().
			Width(availWidth).
			Padding(1, 3).
			Render(leftContent)

		divWidth := availWidth - 6
		if divWidth < 0 {
			divWidth = 0
		}
		divider := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Border)).Render(strings.Repeat("─", divWidth))

		// Call renderRightContextPane with the full available inner width of the column
		rightContent := renderRightContextPane(m, availWidth - 6)
		rightPanel := lipgloss.NewStyle().
			Width(availWidth).
			Padding(1, 3).
			Render(rightContent)

		dashboard = lipgloss.JoinVertical(lipgloss.Left, leftPanel, divider, rightPanel)
	} else {
		// Normal: 70/30 horizontal split (gives left pane mega clock room and breathing space)
		leftWidth := int(float64(availWidth) * 0.70)
		rightWidth := availWidth - leftWidth

		// Left panel has a vertical divider border on its right side, with breathing padding
		leftPanel := lipgloss.NewStyle().
			Width(leftWidth).
			Padding(1, 3).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color(theme.Border)).
			Render(leftContent)

		// Call renderRightContextPane with the correct inner width for the 30% pane
		rightContent := renderRightContextPane(m, rightWidth - 6)
		rightPanel := lipgloss.NewStyle().
			Width(rightWidth).
			Padding(1, 3).
			Render(rightContent)

		dashboard = lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	}

	// Hotkeys for StatusBar
	var shortcuts []string
	if m.mode == "input" {
		shortcuts = []string{
			"[Tab] Switch Field",
			"[Enter] Start/Apply",
			"[Ctrl+T] Save Template",
			"[Ctrl+B] Soundscapes",
			"[?] Help",
			"[q] Quit",
		}
		if m.petEnabled {
			shortcuts = append(shortcuts, "[Ctrl+G] Toggle Pet")
		}
	} else {
		petHint := ""
		if m.petEnabled {
			petHint = "[Ctrl+G] Pet"
		}
		shortcuts = []string{"[Space] Pause", "[E] Edit", "[Enter] End", "[Tab] Stats", "[S] Settings", "[Ctrl+B] Soundscapes", "[?] Help"}
		if petHint != "" {
			shortcuts = append(shortcuts, petHint)
		}
		if m.guardianLocked {
			shortcuts = append(shortcuts, "[Esc] Abort")
		} else {
			shortcuts = append(shortcuts, "[q] Quit")
		}
		if !m.running {
			shortcuts[0] = "[Space] Resume"
		}
	}

	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m, availWidth)

	// Combine status messages and global status bar
	dashboardWidth := lipgloss.Width(dashboard)
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, dashboardWidth)

	var fullLayout string
	if statusLine != "" {
		fullLayout = dashboard + "\n\n" + statusLine + "\n" + statusBar
	} else {
		fullLayout = dashboard + "\n\n\n" + statusBar // 3 newlines before statusbar for nice breathing space!
	}

	// 5. Center in terminal vertically & horizontally
	framed := centerVertical(m.height, outerMargin(m.width, fullLayout))

	return framed
}
