package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/pet"
)

func renderInputView(m model) string {
	if m.showPetLevelUpOverlay {
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, renderPetLevelUpCard(m.petState)))
	}

	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	templateLine := "Template: none"
	if len(m.templates) > 0 {
		if template, ok := m.currentTemplate(); ok {
			templateLine = fmt.Sprintf("Template: %s (%d/%d)", template.Name, m.templateIndex+1, len(m.templates))
		}
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))

	recoveryMsg := ""
	if m.streakState.RecoveryAvailable {
		recoveryMsg = "✦ Recovery mode — complete a session today to save your streak!"
	} else if m.streakState.RecoveryNeeded {
		recoveryMsg = "◌ Streak lost — start fresh today and rebuild your momentum!"
	}

	recentOverlay := ""
	if m.showRecentOverlay && len(m.taskSuggestions) > 0 {
		limit := 5
		if len(m.taskSuggestions) < limit {
			limit = len(m.taskSuggestions)
		}
		overlayLines := []string{"Suggested tasks:"}
		for i := 0; i < limit; i++ {
			cursor := "  "
			if i == m.suggestionIndex {
				cursor = "> "
			}
			overlayLines = append(overlayLines, cursor+m.taskSuggestions[i])
		}
		recentOverlay = strings.Join(overlayLines, "\n")
	}

	hintLine := "[Tab] Switch Field   [Enter] Start/Apply   [Ctrl+T] Save Template   [Ctrl+B] Soundscapes   [?] Help   [q] Quit"
	if m.petEnabled {
		hintLine += "   [Ctrl+G] Toggle Pet"
	}

	inputForm := fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📝  What are you working on?      │
╰─────────────────────────────────────╯

%s
%s
%s

%s

%s

%s

%s

%s

%s
Templates: Left/Right while Template is focused   Up/Down to browse suggested tasks`,
		templateLine, recoveryMsg, recentOverlay, m.textInput.View(), m.durationInput.View(), m.noteInput.View(), m.tagInput.View(), errorBlock, hintLine)

	if m.petEnabled && m.showPetSidebar && m.width >= 90 {
		m.petState.UpdateMood(m.running, m.mode, m.sessionStart)
		petBox := pet.RenderPetBox(m.petState, m.width)

		formFrame := lipgloss.NewStyle().Padding(0, 1).Render(inputForm)

		theme := activeTheme(m.config)
		petFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(0, 1).
			Render(petBox)

		block := lipgloss.JoinHorizontal(lipgloss.Center, formFrame, petFrame)
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
	}

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, inputForm))
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
