package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"kairu-tui/internal/config"
	"kairu-tui/internal/ui"
)

func themedStyle(cfg config.Config, color string) lipgloss.Style {
	return config.ThemedStyle(cfg, color)
}

func activeTheme(cfg config.Config) config.ThemeStyle {
	return config.ActiveTheme(cfg)
}

func activeFont(cfg config.Config) config.TimerFont {
	return config.ActiveFont(cfg)
}

func themeLabel(name string) string {
	return config.ThemeLabel(name)
}

func fontLabel(name string) string {
	return config.FontLabel(name)
}

func layoutLabel(name string) string {
	return config.LayoutLabel(name)
}

func normalizeLayout(name string) string {
	return config.NormalizeLayout(name)
}

func nextValue(order []string, current string, delta int) string {
	return config.NextValue(order, current, delta)
}

func renderAppError(m model) string {
	if strings.TrimSpace(m.appError) == "" {
		return ""
	}
	return themedStyle(m.config, activeTheme(m.config).Warning).Render(m.appError)
}

func renderNotificationStatus(m model, width int) string {
	if strings.TrimSpace(m.notificationStatus) == "" {
		return ""
	}
	return ui.Toast(m.notificationStatus, activeTheme(m.config), width)
}

func centerBlock(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(content)
}

func joinNonEmptyLines(lines ...string) string {
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			trimmed = append(trimmed, line)
		}
	}
	return strings.Join(trimmed, "\n")
}

func formatHelpLine(key, description string) string {
	return fmt.Sprintf("%-10s %s", key, description)
}

func shellEscape(s string) string {
	s = strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + s + "'"
}

func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func wrapHour(hour int) int {
	if hour < 0 {
		return 23
	}
	if hour > 23 {
		return 0
	}
	return hour
}

func (m model) openHelp() model {
	m.helpReturnMode = m.mode
	m.helpWasRunning = m.running
	m.mode = "help"
	return m
}

func (m model) closeHelp(resume bool) (model, tea.Cmd) {
	m.mode = m.helpReturnMode
	if resume && m.helpWasRunning {
		if !m.running {
			m.running = true
			if m.seconds > 0 {
				return m, tickCmd()
			}
		}
	} else {
		if !m.helpWasRunning {
			m.running = false
		}
	}
	return m, nil
}

func parseTags(input string) []string {
	parts := strings.Split(input, ",")
	tags := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		tag := strings.ToLower(strings.TrimSpace(part))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func (m model) currentSessionTags() []string {
	if m.tagInput.Value() == "" {
		return nil
	}
	return parseTags(m.tagInput.Value())
}
