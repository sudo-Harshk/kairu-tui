package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/pet"
	"kairu-tui/internal/ui"
)

func renderInputView(m model) string {
	theme := activeTheme(m.config)

	if m.showPetLevelUpOverlay {
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, renderPetLevelUpCard(m.petState, theme)))
	}

	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Warning)).Render(m.inputError)
	}
	templateLine := ""
	if len(m.templates) > 0 {
		if template, ok := m.currentTemplate(); ok {
			templateLine = fmt.Sprintf("%s (%d/%d)", template.Name, m.templateIndex+1, len(m.templates))
		}
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))

	// Streak Recovery Panel (Success or Warning border)
	var recoveryPanel string
	if m.streakState.RecoveryAvailable {
		recoveryPanel = ui.Panel("", "✦ Recovery mode — complete a session today to save your streak!", theme, 42, lipgloss.NormalBorder(), theme.Accent)
	} else if m.streakState.RecoveryNeeded {
		recoveryPanel = ui.Panel("", "◌ Streak lost — start fresh today and rebuild your momentum!", theme, 42, lipgloss.NormalBorder(), theme.Warning)
	}

	recentOverlay := ""
	if m.showRecentOverlay && len(m.taskSuggestions) > 0 {
		limit := 5
		if len(m.taskSuggestions) < limit {
			limit = len(m.taskSuggestions)
		}
		var items []string
		for i := 0; i < limit; i++ {
			items = append(items, m.taskSuggestions[i])
		}
		recentOverlay = "Suggested tasks:\n" + ui.Menu(items, m.suggestionIndex, theme)
	}

	// Hotkeys for StatusBar
	shortcuts := []string{
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

	formWidth := 46
	var formContentBuilder strings.Builder

	if templateLine != "" {
		formContentBuilder.WriteString(ui.FormField("Active Template", templateLine, m.focusedField == focusTemplate, theme) + "\n\n")
	}
	if recoveryPanel != "" {
		formContentBuilder.WriteString(recoveryPanel + "\n\n")
	}

	formContentBuilder.WriteString(ui.FormField("Task Name", m.textInput.View(), m.focusedField == focusTask, theme) + "\n\n")
	formContentBuilder.WriteString(ui.FormField("Duration (minutes)", m.durationInput.View(), m.focusedField == focusDuration, theme) + "\n\n")
	formContentBuilder.WriteString(ui.FormField("Session Notes", m.noteInput.View(), m.focusedField == focusNote, theme) + "\n\n")
	formContentBuilder.WriteString(ui.FormField("Tags (comma separated)", m.tagInput.View(), m.focusedField == focusTags, theme))

	if recentOverlay != "" {
		formContentBuilder.WriteString("\n\n" + recentOverlay)
	}

	formContentBuilder.WriteString("\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice)).Render("Templates: Left/Right while Template is focused\nUp/Down to browse suggested tasks"))

	// Render the form inside a ui.Panel with themed borders
	inputForm := ui.Panel("📝 What are you working on?", formContentBuilder.String(), theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)

	// Combine with status bar
	statusBar := ui.StatusBar(shortcuts, errorBlock, theme, formWidth)

	fullForm := inputForm + "\n\n" + statusBar

	if m.petEnabled && m.showPetSidebar && m.width >= 90 {
		m.petState.UpdateMood(m.running, m.mode, m.sessionStart)
		petBox := pet.RenderPetBox(m.petState, m.width, theme)

		formFrame := lipgloss.NewStyle().Padding(0, 1).Render(fullForm)

		// Styled pet sidebar using ui.Panel
		petFrame := ui.Panel("", petBox, theme, 36, lipgloss.RoundedBorder(), theme.Primary)

		block := lipgloss.JoinHorizontal(lipgloss.Center, formFrame, petFrame)
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
	}

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, fullForm))
}

func (m model) applyTaskSuggestion(delta int) model {
	if len(m.taskSuggestions) == 0 {
		return m
	}
	if m.suggestionIndex < 0 || m.suggestionIndex >= len(m.taskSuggestions) {
		m.suggestionIndex = 0
	} else {
		m.suggestionIndex = (m.suggestionIndex + delta + len(m.taskSuggestions)) % len(m.taskSuggestions)
	}
	m.textInput.SetValue(m.taskSuggestions[m.suggestionIndex])
	m.textInput.CursorEnd()
	return m
}
