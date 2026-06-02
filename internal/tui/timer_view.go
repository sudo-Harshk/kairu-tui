package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
	"kairu-tui/internal/pet"
	"kairu-tui/internal/timer"
	"kairu-tui/internal/ui"
)

func renderEditView(m model) string {
	theme := activeTheme(m.config)
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Warning)).Render(m.inputError)
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))
	elapsed := timer.FormatClock(m.sessionElapsed)

	formWidth := 46
	body := fmt.Sprintf("Task: %s\nElapsed: %s\n\n%s", m.taskName, elapsed, m.durationInput.View())

	editCard := ui.Panel("✏️ Adjust Session Time", body, theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)
	statusBar := ui.StatusBar([]string{"[Enter] Apply", "[Esc] Cancel", "[?] Help", "[q] Quit"}, errorBlock, theme, formWidth)

	block := renderBanner(m.config) + "\n\n" + editCard + "\n\n" + statusBar
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderTimerView(m model) string {
	timeStr := timer.FormatClock(m.seconds)
	theme := activeTheme(m.config)
	layout := normalizeLayout(m.config.Layout)

	modeStr := "WORK"
	if m.mode == "break" {
		modeStr = "BREAK"
	}

	// Progress bar - unified ui.ProgressBar
	targetSeconds := m.sessionTarget
	if targetSeconds <= 0 {
		targetSeconds = 1
	}
	remainingPct := float64(m.seconds) / float64(targetSeconds) * 100
	if remainingPct > 100 {
		remainingPct = 100
	}
	if remainingPct < 0 {
		remainingPct = 0
	}
	progress := fmt.Sprintf("%s %.0f%%", ui.ProgressBar(remainingPct, 28, theme.Primary, theme.Notice), remainingPct)

	petHint := ""
	if m.petEnabled {
		petHint = "[Ctrl+G] Pet"
	}
	shortcuts := []string{"[Space] Pause", "[E] Edit", "[Enter] End", "[Tab] Stats", "[S] Settings", "[Ctrl+B] Soundscapes", "[?] Help"}
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

	// Header text - Structured badges row
	var headerParts []string
	modeBadge := ui.Badge(modeStr, theme, m.mode == "work")
	headerParts = append(headerParts, modeBadge)

	taskNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true)
	taskNameStr := taskNameStyle.Render(" " + m.taskName + " ")
	if m.guardianLocked {
		taskNameStr = "🔒 " + taskNameStr
	}
	headerParts = append(headerParts, taskNameStr)

	if tags := strings.Join(m.currentSessionTags(), ", "); tags != "" {
		headerParts = append(headerParts, ui.Badge("🏷️ "+tags, theme, false))
	}

	if m.streakState.Current > 0 {
		headerParts = append(headerParts, ui.Badge(fmt.Sprintf("🔥 %d", m.streakState.Current), theme, true))
	} else if m.streakState.RecoveryAvailable {
		headerParts = append(headerParts, ui.Badge("✦ recoverable", theme, false))
	} else if m.streakState.RecoveryNeeded {
		headerParts = append(headerParts, ui.Badge("◌ rebuild", theme, false))
	}

	if m.activeSoundscapeCmd != nil && m.soundscapeIndex >= 0 && m.soundscapeIndex < len(m.soundscapes) {
		track := strings.TrimSuffix(m.soundscapes[m.soundscapeIndex], filepath.Ext(m.soundscapes[m.soundscapeIndex]))
		headerParts = append(headerParts, ui.Badge("🎵 "+track, theme, true))
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Center, headerParts...)

	formWidth := 46
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m, formWidth)

	var mainFrame string

	switch layout {
	case "minimal":
		timerLine := themedStyle(m.config, theme.Accent).Bold(true).Render(timeStr)
		mainFrame = fmt.Sprintf("%s\n\n%s  %s", headerRow, timerLine, progress)
	case "compact":
		timerFrame := ui.Panel("", fmt.Sprintf("%s  %s", themedStyle(m.config, theme.Accent).Bold(true).Render(timeStr), progress), theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)
		mainFrame = fmt.Sprintf("%s\n\n%s", headerRow, timerFrame)
	default: // classic
		ascii := renderASCIITimer(timeStr, m.config)
		// Constrain ASCII timer to width-4. Overflow triggers a compact fallback.
		if lipgloss.Width(ascii) > 42 || m.width < 48 {
			ascii = themedStyle(m.config, theme.Accent).Bold(true).Render("⏰ " + timeStr)
		} else {
			ascii = themedStyle(m.config, theme.Accent).Width(42).Align(lipgloss.Center).Render(ascii)
		}
		timerFrame := ui.Panel("", fmt.Sprintf("%s\n\n%s", ascii, progress), theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)
		mainFrame = fmt.Sprintf("%s\n\n%s", headerRow, timerFrame)
	}

	statusBar := ui.StatusBar(shortcuts, errorLine, theme, formWidth)
	var fullBlock string
	if statusLine != "" {
		fullBlock = mainFrame + "\n\n" + statusLine + "\n\n" + statusBar
	} else {
		fullBlock = mainFrame + "\n\n" + statusBar
	}

	if m.petEnabled && m.showPetSidebar && m.width >= 90 {
		m.petState.UpdateMood(m.running, m.mode, m.sessionStart)
		petBox := pet.RenderPetBox(m.petState, m.width, theme)

		timerFrame := lipgloss.NewStyle().Padding(0, 1).Render(fullBlock)

		// Styled pet sidebar matching the main panel container style
		petFrame := ui.Panel("", petBox, theme, 36, lipgloss.RoundedBorder(), theme.Primary)

		joinedBlock := lipgloss.JoinHorizontal(lipgloss.Center, timerFrame, petFrame)
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, joinedBlock))
	}

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, fullBlock))
}

func renderASCIITimer(timeStr string, cfg config.Config) string {
	chars := activeFont(cfg).Digits
	lines := make([]string, 5)
	for _, ch := range timeStr {
		if art, ok := chars[ch]; ok {
			for i := 0; i < 5; i++ {
				lines[i] += art[i] + " "
			}
		}
	}

	return strings.Join(lines, "\n")
}
