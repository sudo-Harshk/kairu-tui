package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/pet"
)

func renderTamagotchiView(m model) string {
	theme := activeTheme(m.config)

	var screenContent string
	if m.tamagotchiActiveMenu == "typing" {
		screenContent = pet.RenderTypingGame(m.typingGame, m.width, theme.Accent, theme.Primary)
	} else if m.tamagotchiActiveMenu == "guessing" {
		screenContent = pet.RenderBinaryGame(m.binaryGame, m.width, theme.Accent, theme.Primary)
	} else if m.tamagotchiActiveMenu == "rebirth" {
		subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

		var rows []string
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Bold(true).Render("   🧬  R E B I R T H   S A N C T U A R Y"))
		rows = append(rows, subtle.Render("   "+strings.Repeat("─", 52)))
		rows = append(rows, "   Your companion has reached the end of their digital cycle.")
		rows = append(rows, "   Please enter a new name for your companion:")
		rows = append(rows, "")
		rows = append(rows, "   > "+m.textInput.View())
		rows = append(rows, "")
		rows = append(rows, subtle.Render("   Press [Enter] to rebirth or [Esc] to cancel"))

		for len(rows) < 8 {
			rows = append(rows, "")
		}
		screenContent = pet.RenderTamagotchiScreen(m.petState, m.width, m.tamagotchiActiveMenu, m.tamagotchiMenuSelect, strings.Join(rows, "\n"), theme.Accent, theme.Primary)
	} else {
		screenContent = pet.RenderTamagotchiScreen(m.petState, m.width, m.tamagotchiActiveMenu, m.tamagotchiMenuSelect, m.tamagotchiFeedback, theme.Accent, theme.Primary)
	}
	return fmt.Sprintf("\n%s\n", screenContent)
}

func renderPetLevelUpCard(petState pet.PetState) string {
	stage := petState.EvolutionStage()
	stageName := "Baby"
	if stage == 2 {
		stageName = "Teenager"
	} else if stage == 3 {
		stageName = "Cyber-Ascended God!"
	}

	return fmt.Sprintf(`
╭───────────────────────────────────────────────────╮
│             🎉   LEVEL UP!  LEVEL UP!   🎉        │
╰───────────────────────────────────────────────────╯

        ★  %s HAS REACHED LEVEL %d! ★

                 Evolution Stage: %s

            "Quack! Gaining power! Thank you!"

[ Press any key to continue... ]`,
		strings.ToUpper(petState.Name), petState.Level, stageName)
}
