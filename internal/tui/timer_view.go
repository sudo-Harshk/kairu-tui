package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
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
	theme := activeTheme(m.config)
	layout := normalizeLayout(m.config.Layout)

	modeStr := "WORK"
	if m.mode == "break" {
		modeStr = "BREAK"
	}

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

	// Calculate leftWidth
	availWidth := m.width - 2
	leftWidth := int(float64(availWidth) * 0.70)
	if m.width < 60 {
		leftWidth = availWidth
	} else if m.width < 90 {
		leftWidth = availWidth
	}

	contentWidth := leftWidth - 8
	if contentWidth < 20 {
		contentWidth = 20
	}

	progressBarWidth := contentWidth - 12
	if progressBarWidth < 10 {
		progressBarWidth = 10
	}

	stateColor := config.StateAccent(theme, m.mode)
	progress := fmt.Sprintf("%s %.0f%%", ui.ProgressBar(remainingPct, progressBarWidth, stateColor, theme.Border), remainingPct)

	// Header badges
	var headerParts []string
	modeBadge := ui.Badge(modeStr, theme, m.mode == "work")
	headerParts = append(headerParts, modeBadge)

	taskNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Bold(true)
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
	}

	if m.activeSoundscapeCmd != nil && m.soundscapeIndex >= 0 && m.soundscapeIndex < len(m.soundscapes) {
		track := strings.TrimSuffix(m.soundscapes[m.soundscapeIndex], filepath.Ext(m.soundscapes[m.soundscapeIndex]))
		headerParts = append(headerParts, ui.Badge("🎵 "+track, theme, true))
	}

	headerRow := lipgloss.JoinHorizontal(lipgloss.Center, headerParts...)

	// Clock text formatting
	var clockStr string
	timeStr := timer.FormatClock(m.seconds)
	if strings.HasPrefix(timeStr, "00:") {
		timeStr = timeStr[3:]
	}

	stateStyle := themedStyle(m.config, stateColor).Bold(true)

	if m.height < 20 || layout == "minimal" {
		clockStr = stateStyle.Render("⏰ " + timeStr)
	} else {
		font := activeFont(m.config)
		ascii := renderClock(m.seconds, font)
		asciiWidth := lipgloss.Width(ascii)

		if asciiWidth > contentWidth && m.config.Font == "mega" {
			font = config.TimerFonts["ansi"]
			ascii = renderClock(m.seconds, font)
			asciiWidth = lipgloss.Width(ascii)
		}

		if asciiWidth > contentWidth || layout == "compact" {
			clockStr = stateStyle.Render("⏰ " + timeStr)
		} else {
			clockStr = stateStyle.Render(ascii)
		}
	}

	if layout == "minimal" {
		return headerRow + "\n\n" + clockStr + "  " + progress
	} else if layout == "compact" {
		timerFrame := ui.Panel("", fmt.Sprintf("%s  %s", clockStr, progress), theme, contentWidth, lipgloss.RoundedBorder(), theme.Primary)
		return headerRow + "\n\n" + timerFrame
	} else {
		timerFrame := ui.Panel("", fmt.Sprintf("%s\n\n%s", clockStr, progress), theme, contentWidth, lipgloss.RoundedBorder(), theme.Primary)
		return headerRow + "\n\n" + timerFrame
	}
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
