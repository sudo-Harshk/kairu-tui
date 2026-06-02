package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
)

// ProgressBar renders a themed filled/empty progress bar.
func ProgressBar(percent float64, totalWidth int, fillColor string, emptyColor string) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filledLength := int(math.Round(percent * float64(totalWidth) / 100.0))
	if filledLength < 0 {
		filledLength = 0
	}
	if filledLength > totalWidth {
		filledLength = totalWidth
	}
	emptyLength := totalWidth - filledLength

	filledStr := lipgloss.NewStyle().Foreground(lipgloss.Color(fillColor)).Render(strings.Repeat("█", filledLength))
	emptyStr := lipgloss.NewStyle().Foreground(lipgloss.Color(emptyColor)).Render(strings.Repeat("░", emptyLength))

	return filledStr + emptyStr
}

// Badge renders a styled text tag with a solid background and inverted text.
func Badge(text string, theme config.ThemeStyle, useAccent bool) string {
	color := theme.Primary
	if useAccent {
		color = theme.Accent
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color(color)).
		Foreground(lipgloss.Color(theme.Surface)).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

// Button renders a retro classic console-styled button.
func Button(text string, selected bool, theme config.ThemeStyle) string {
	if selected {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Surface)).
			Background(lipgloss.Color(theme.Accent)). // Accent background
			Bold(true).
			Padding(0, 1).
			Render(text)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Primary)).
		Padding(0, 1).
		Render("[ " + text + " ]")
}


// Panel renders a card box with optional titles and warning/success border colors.
func Panel(title string, content string, theme config.ThemeStyle, width int, border lipgloss.Border, borderColor string) string {
	if borderColor == "" {
		borderColor = theme.Primary
	}

	borderS := lipgloss.NewStyle().
		Border(border).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(width)

	if title == "" {
		return borderS.Render(content)
	}

	titleS := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Accent)).
		Bold(true)

	// Lipgloss border takes 2 chars, padding takes 2 chars (left + right).
	divWidth := width - 4
	if divWidth < 0 {
		divWidth = 0
	}
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(borderColor)).
		Render(strings.Repeat("─", divWidth))

	cardContent := titleS.Render(title) + "\n" + divider + "\n" + content
	return borderS.Render(cardContent)
}

// StatusBar renders a consistent bottom status bar containing active hotkeys and status messages.
func StatusBar(shortcuts []string, errMessage string, theme config.ThemeStyle, width int) string {
	shortcutsStr := ""
	for i, s := range shortcuts {
		if i > 0 {
			shortcutsStr += "  "
		}
		if strings.HasPrefix(s, "[") && strings.Contains(s, "]") {
			parts := strings.SplitN(s, "]", 2)
			keyPart := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true).Render(parts[0] + "]")
			shortcutsStr += keyPart + parts[1]
		} else {
			shortcutsStr += s
		}
	}

	barStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false). // Top border only
		BorderForeground(lipgloss.Color(theme.Primary)).
		Padding(0, 1).
		Width(width)

	if errMessage != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Warning)).Bold(true).Render("⚠️  " + errMessage)
		return barStyle.Render(shortcutsStr + "\n" + errStyle)
	}

	return barStyle.Render(shortcutsStr)
}

// Toast renders a styled floating notification toast bar.
func Toast(message string, theme config.ThemeStyle, width int) string {
	if message == "" {
		return ""
	}
	toastContent := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Accent)).
		Bold(true).
		Render("🔔 " + message)
	return Panel("", toastContent, theme, width, lipgloss.RoundedBorder(), theme.Accent)
}

// KeyHints renders a themed helper/hints line at the bottom of a panel.
func KeyHints(message string, theme config.ThemeStyle) string {
	if message == "" {
		return ""
	}
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice))
	
	// If the message has format "Title: Description"
	parts := strings.SplitN(message, ":", 2)
	if len(parts) == 2 {
		return keyStyle.Render("💡 "+parts[0]+":") + " " + descStyle.Render(parts[1])
	}
	return descStyle.Render("💡 " + message)
}

// FormField renders a styled input field with a label and highlighted focus state.
func FormField(label string, inputView string, focused bool, theme config.ThemeStyle) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Primary)).
		Bold(true)
	if focused {
		labelStyle = labelStyle.Foreground(lipgloss.Color(theme.Accent))
	}
	
	inputStyle := lipgloss.NewStyle().Padding(0, 1)
	if focused {
		inputStyle = inputStyle.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(theme.Accent))
	} else {
		inputStyle = inputStyle.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(theme.Dim))
	}
	
	return labelStyle.Render(label) + "\n" + inputStyle.Render(inputView)
}

// Menu renders a list of items with a highlighted cursor/selection.
func Menu(items []string, selectedIndex int, theme config.ThemeStyle) string {
	var rendered []string
	for i, item := range items {
		if i == selectedIndex {
			chevron := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true).Render("> ")
			text := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true).Render(item)
			rendered = append(rendered, chevron+text)
		} else {
			chevron := "  "
			text := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Dim)).Render(item)
			rendered = append(rendered, chevron+text)
		}
	}
	return strings.Join(rendered, "\n")
}

// Divider renders a horizontal dividing line of specified width.
func Divider(width int, theme config.ThemeStyle) string {
	if width <= 0 {
		return ""
	}
	borderColor := theme.Border
	if borderColor == "" {
		borderColor = theme.Primary
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(borderColor)).
		Render(strings.Repeat("─", width))
}

type KeyHintPair struct {
	Key  string
	Desc string
}

// KeyHint renders structured key+desc pairs consistently using theme tokens.
func KeyHint(pairs []KeyHintPair, theme config.ThemeStyle) string {
	var parts []string
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Dim))
	
	for _, p := range pairs {
		parts = append(parts, fmt.Sprintf("%s %s", keyStyle.Render(p.Key), descStyle.Render(p.Desc)))
	}
	return "💡 " + strings.Join(parts, "  •  ")
}



