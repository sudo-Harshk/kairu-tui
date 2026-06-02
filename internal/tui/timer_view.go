package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
	"kairu-tui/internal/pet"
	"kairu-tui/internal/timer"
)

func renderEditView(m model) string {
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))
	elapsed := timer.FormatClock(m.sessionElapsed)
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  ✏️  Adjust Session Time           │
╰─────────────────────────────────────╯

Task: %s
Elapsed: %s

%s

%s

	[Enter] Apply   [Esc] Cancel   [?] Help   [q] Quit`, renderBanner(m.config), m.taskName, elapsed, m.durationInput.View(), errorBlock)
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

	// Progress bar
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
	barWidth := 40
	filled := int(remainingPct / 100 * float64(barWidth))
	empty := barWidth - filled
	progress := fmt.Sprintf("[%s%s] %.0f%%", strings.Repeat("█", filled), strings.Repeat("░", empty), remainingPct)

	petHint := ""
	if m.petEnabled {
		petHint = "  [Ctrl+G] Pet"
	}
	hint := "[Space] Pause  [E] Edit  [Enter] End  [Tab] Stats  [S] Settings  [Ctrl+B] Soundscapes  [?] Help" + petHint
	if m.guardianLocked {
		hint += "  [Esc] Abort"
	} else {
		hint += "  [q] Quit"
	}
	if !m.running {
		hint = "[Space] Resume  [E] Edit  [Enter] End  [Tab] Stats  [S] Settings  [Ctrl+B] Soundscapes  [?] Help" + petHint
		if m.guardianLocked {
			hint += "  [Esc] Abort"
		} else {
			hint += "  [q] Quit"
		}
	}

	header := fmt.Sprintf("%s • %s", modeStr, m.taskName)
	if m.guardianLocked {
		header = "🔒 Guardian Active • " + header
	}
	if tags := strings.Join(m.currentSessionTags(), ", "); tags != "" {
		header += fmt.Sprintf(" [%s]", tags)
	}
	if m.streakState.Current > 0 {
		header += fmt.Sprintf("  🔥%d", m.streakState.Current)
	} else if m.streakState.RecoveryAvailable {
		header += "  ✦ recoverable"
	} else if m.streakState.RecoveryNeeded {
		header += "  ◌ rebuild"
	}
	if m.activeSoundscapeCmd != nil && m.soundscapeIndex >= 0 && m.soundscapeIndex < len(m.soundscapes) {
		track := strings.TrimSuffix(m.soundscapes[m.soundscapeIndex], filepath.Ext(m.soundscapes[m.soundscapeIndex]))
		header += fmt.Sprintf("  🎵 %s", track)
	}

	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	details := hint
	if errorLine != "" {
		details = fmt.Sprintf("%s\n%s", errorLine, hint)
	}
	if statusLine != "" {
		details = fmt.Sprintf("%s\n%s", details, statusLine)
	}

	var block string
	switch layout {
	case "minimal":
		timerLine := themedStyle(m.config, theme.Accent).Bold(true).Render(timeStr)
		block = fmt.Sprintf("%s\n\n%s  %s\n\n%s", header, timerLine, progress, details)
	case "compact":
		timerFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			Render(fmt.Sprintf("%s  %s", themedStyle(m.config, theme.Accent).Bold(true).Render(timeStr), progress))
		block = fmt.Sprintf("%s\n\n%s\n\n%s", header, timerFrame, details)
	default: // classic
		ascii := renderASCIITimer(timeStr, m.config)
		innerWidth := lipgloss.Width(progress)
		if asciiWidth := lipgloss.Width(ascii); asciiWidth > innerWidth {
			innerWidth = asciiWidth
		}
		ascii = themedStyle(m.config, theme.Accent).Width(innerWidth).Align(lipgloss.Center).Render(ascii)
		timerFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			Render(fmt.Sprintf("%s\n\n%s", ascii, progress))
		block = fmt.Sprintf("%s\n\n%s\n\n%s", header, timerFrame, details)
	}

	if m.petEnabled && m.showPetSidebar && m.width >= 90 {
		m.petState.UpdateMood(m.running, m.mode, m.sessionStart)
		petBox := pet.RenderPetBox(m.petState, m.width)

		timerFrame := lipgloss.NewStyle().Padding(0, 1).Render(block)

		petFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(0, 1).
			Render(petBox)

		joinedBlock := lipgloss.JoinHorizontal(lipgloss.Center, timerFrame, petFrame)
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, joinedBlock))
	}

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
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
