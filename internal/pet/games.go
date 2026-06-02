package pet

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
)

// Pomo-Type Quotes
var HackerQuotes = []string{
	"git commit -m \"feed Neko\"",
	"sudo rm -rf /distractions",
	"func focus() { work() }",
	"go run main.go --deep-focus",
	"grep -r \"productivity\" .",
	"chmod +x play_with_kitty",
	"cat happiness.log",
	"ping -c 4 robo_kitty",
}

// TypingGameState holds states for the coding typing challenge
type TypingGameState struct {
	TargetText string
	TypedText  string
	StartTime  time.Time
	Finished   bool
	Accuracy   float64
	WPM        int
	CoinsWon   int
}

// BinaryGameState holds states for the higher/lower binary game
type BinaryGameState struct {
	TargetNum int
	Attempts  int
	LastHint  string
	InputStr  string
	Finished  bool
	Won       bool
}

// InitTypingGame starts a typing challenge
func InitTypingGame() TypingGameState {
	quote := HackerQuotes[rand.Intn(len(HackerQuotes))]
	return TypingGameState{
		TargetText: quote,
		TypedText:  "",
		StartTime:  time.Now(),
		Finished:   false,
	}
}

// InitBinaryGame starts a higher/lower binary game
func InitBinaryGame() BinaryGameState {
	target := rand.Intn(14) + 2 // Number between 2 and 15
	return BinaryGameState{
		TargetNum: target,
		Attempts:  0,
		LastHint:  "Enter a decimal number between 2 and 15!",
		InputStr:  "",
		Finished:  false,
		Won:       false,
	}
}

// RenderTypingGame renders the typing game LCD panel
func RenderTypingGame(game TypingGameState, width int, theme config.ThemeStyle) string {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent))
	primary := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary))
	subtle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice))
	wrongStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Warning))

	var rows []string
	rows = append(rows, primary.Bold(true).Render("   ⌨️  P O M O - T Y P E   C H A L L E N G E"))
	rows = append(rows, subtle.Render("   "+strings.Repeat("─", width-6)))
	rows = append(rows, "   Type the following text exactly:")
	rows = append(rows, "")
	
	// Highlight target text vs typed text character by character
	var builder strings.Builder
	builder.WriteString("   ")
	for i, char := range game.TargetText {
		if i < len(game.TypedText) {
			if byte(char) == game.TypedText[i] {
				builder.WriteString(primary.Render(string(char)))
			} else {
				builder.WriteString(wrongStyle.Underline(true).Render(string(char)))
			}
		} else {
			builder.WriteString(subtle.Render(string(char)))
		}
	}
	rows = append(rows, builder.String())
	rows = append(rows, "")

	if game.Finished {
		rows = append(rows, accent.Bold(true).Render(fmt.Sprintf("   🎉 CHALLENGE COMPLETE! WPM: %d | Acc: %.1f%%", game.WPM, game.Accuracy)))
		rows = append(rows, accent.Render(fmt.Sprintf("   Awarded +25 Happiness and +%d Pomo-Coins! 🪙", game.CoinsWon)))
		rows = append(rows, subtle.Render("   [Enter] Go back"))
	} else {
		rows = append(rows, subtle.Render("   Type now! [Esc] Abort and return"))
	}

	// Pad game rows to uniform 8 rows height
	for len(rows) < 8 {
		rows = append(rows, "")
	}

	return strings.Join(rows, "\n")
}

// RenderBinaryGame renders the binary/guessing game LCD panel
func RenderBinaryGame(game BinaryGameState, width int, theme config.ThemeStyle) string {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent))
	primary := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary))
	subtle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice))
	gold := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent))

	var rows []string
	rows = append(rows, primary.Bold(true).Render("   🤖  B I N A R Y   G U E S S I N G   G A M E"))
	rows = append(rows, subtle.Render("   "+strings.Repeat("─", width-6)))

	// Show target binary representation!
	binaryRep := fmt.Sprintf("%04b", game.TargetNum)
	rows = append(rows, fmt.Sprintf("   Robo-Pet's secret binary byte: %s", gold.Bold(true).Render(binaryRep)))
	rows = append(rows, fmt.Sprintf("   Attempts: %d / 4", game.Attempts))
	rows = append(rows, "")
	rows = append(rows, "   Hint: "+accent.Italic(true).Render(game.LastHint))

	if game.Finished {
		if game.Won {
			rows = append(rows, accent.Bold(true).Render(fmt.Sprintf("   🎉 CORRECT! The secret number was %d!", game.TargetNum)))
			rows = append(rows, accent.Render("   Awarded +15 Happiness and +5 Pomo-Coins! 🪙"))
		} else {
			rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Warning)).Render(fmt.Sprintf("   GAME OVER! The secret number was %d.", game.TargetNum)))
		}
		rows = append(rows, subtle.Render("   [Enter] Go back"))
	} else {
		rows = append(rows, fmt.Sprintf("   Your guess (dec): %s_", game.InputStr))
		rows = append(rows, subtle.Render("   [0-9] Guess  [Enter] Submit  [Esc] Quit"))
	}

	// Pad game rows to uniform 8 rows height
	for len(rows) < 8 {
		rows = append(rows, "")
	}

	return strings.Join(rows, "\n")
}
