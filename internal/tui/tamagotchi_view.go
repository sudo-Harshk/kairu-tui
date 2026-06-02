package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
	"kairu-tui/internal/pet"
	"kairu-tui/internal/ui"
)

func renderTamagotchiView(m model) string {
	theme := activeTheme(m.config)

	var screenContent string
	if m.tamagotchiActiveMenu == "typing" {
		screenContent = pet.RenderTypingGame(m.typingGame, m.width, theme)
	} else if m.tamagotchiActiveMenu == "guessing" {
		screenContent = pet.RenderBinaryGame(m.binaryGame, m.width, theme)
	} else if m.tamagotchiActiveMenu == "rebirth" {
		content := "Your companion has reached the end of their digital cycle.\n" +
			"Please enter a new name for your companion:\n\n" +
			"  > " + m.textInput.View() + "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice)).Render("Press [Enter] to rebirth or [Esc] to cancel")

		// Rebirth screen styled as a Card with warning border and accent title
		rebirthCard := ui.Panel("🧬 REBIRTH SANCTUARY", content, theme, 52, lipgloss.DoubleBorder(), theme.Warning)
		screenContent = pet.RenderTamagotchiScreen(m.petState, m.width, m.tamagotchiActiveMenu, m.tamagotchiMenuSelect, rebirthCard, theme)
	} else {
		screenContent = pet.RenderTamagotchiScreen(m.petState, m.width, m.tamagotchiActiveMenu, m.tamagotchiMenuSelect, m.tamagotchiFeedback, theme)
	}
	return fmt.Sprintf("\n%s\n", screenContent)
}

func renderPetLevelUpCard(petState pet.PetState, theme config.ThemeStyle) string {
	stage := petState.EvolutionStage()
	stageName := "Baby"
	if stage == 2 {
		stageName = "Teenager"
	} else if stage == 3 {
		stageName = "Cyber-Ascended God!"
	}

	content := fmt.Sprintf("       ★  %s HAS REACHED LEVEL %d! ★\n\n                 Evolution Stage: %s\n\n            \"Quack! Gaining power! Thank you!\"\n\n[ Press any key to continue... ]", strings.ToUpper(petState.Name), petState.Level, stageName)

	// Level-up card styled as a Card with success (accent) border and accent title
	return ui.Panel("🎉 LEVEL UP! LEVEL UP! 🎉", content, theme, 52, lipgloss.RoundedBorder(), theme.Accent)
}
