package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
)

func renderBanner(cfg config.Config) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme(cfg).Accent)).Padding(0, 2).Render("KAIRU  •  Grow Your Focus")
}

func renderStatsTabs(currentMode string, cfg config.Config) string {
	tabs := []struct {
		mode  string
		label string
	}{
		{"stats", "Dashboard"},
		{"analytics", "Analytics"},
		{"heatmap", "Heatmap"},
		{"history", "Timeline"},
		{"report", "Report"},
	}

	theme := activeTheme(cfg)
	var renderedTabs []string

	for _, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if t.mode == currentMode {
			style = style.
				Foreground(lipgloss.Color(theme.Accent)).
				Bold(true).
				Underline(true)
		} else {
			style = style.Foreground(lipgloss.Color(theme.Primary))
		}
		renderedTabs = append(renderedTabs, style.Render(t.label))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	return "\n" + row + "\n" + themedStyle(cfg, theme.Primary).Render(strings.Repeat("─", lipgloss.Width(row))) + "\n"
}

func renderHorizontalProgressBar(percent float64, totalWidth int, theme config.ThemeStyle, useAccent bool) string {
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

	filledChar := "█"
	emptyChar := "░"

	color := theme.Primary
	if useAccent {
		color = theme.Accent
	}

	filledStr := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat(filledChar, filledLength))
	emptyStr := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice)).Render(strings.Repeat(emptyChar, emptyLength))

	return filledStr + emptyStr
}

func renderDashboardCard(title string, content string, theme config.ThemeStyle, width int) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Primary)).
		Padding(0, 1).
		Width(width)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Accent)).
		Bold(true)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Primary)).
		Render(strings.Repeat("─", width-2))

	cardContent := titleStyle.Render(title) + "\n" + divider + "\n" + content
	return borderStyle.Render(cardContent)
}

func renderTopDurationBars(totals map[string]int, totalSeconds int, limit int, theme config.ThemeStyle, useAccent bool, barWidth int) []string {
	if len(totals) == 0 {
		return nil
	}
	type item struct {
		name    string
		seconds int
	}
	items := make([]item, 0, len(totals))
	for name, seconds := range totals {
		items = append(items, item{name: name, seconds: seconds})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].seconds == items[j].seconds {
			return items[i].name < items[j].name
		}
		return items[i].seconds > items[j].seconds
	})
	if len(items) < limit {
		limit = len(items)
	}

	lines := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		pct := 0.0
		if totalSeconds > 0 {
			pct = float64(items[i].seconds) * 100.0 / float64(totalSeconds)
		}

		progressBar := renderHorizontalProgressBar(pct, barWidth, theme, useAccent)
		durStr := streak.FormatDuration(items[i].seconds)

		name := items[i].name
		if len(name) > 12 {
			name = name[:9] + "..."
		}

		lines = append(lines, fmt.Sprintf("  %-12s %s %7s (%.1f%%)", name, progressBar, durStr, pct))
	}
	return lines
}

func renderTopTags(entriesList []entries.Entry) string {
	counts := make(map[string]int)
	for _, entry := range entriesList {
		for _, tag := range entry.Tags {
			counts[strings.ToLower(strings.TrimSpace(tag))]++
		}
	}
	if len(counts) == 0 {
		return "Tags: none yet"
	}
	type tagCount struct {
		tag   string
		count int
	}
	items := make([]tagCount, 0, len(counts))
	for tag, count := range counts {
		items = append(items, tagCount{tag: tag, count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].tag < items[j].tag
		}
		return items[i].count > items[j].count
	})
	limit := 3
	if len(items) < limit {
		limit = len(items)
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s (%d)", items[i].tag, items[i].count))
	}
	return "Top tags: " + strings.Join(parts, ", ")
}

func renderWeeklyBarChart(weeklyData map[string]int) string {
	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	today := time.Now()

	maxMinutes := 0
	for _, secs := range weeklyData {
		mins := secs / 60
		if mins > maxMinutes {
			maxMinutes = mins
		}
	}
	if maxMinutes == 0 {
		return "No activity yet."
	}

	var b strings.Builder
	for i := 6; i >= 0; i-- {
		dateValue := today.AddDate(0, 0, -i)
		date := streak.DateKey(dateValue)
		dayName := days[dateValue.Weekday()]
		minutes := weeklyData[date] / 60

		barLen := minutes * 40 / maxMinutes
		bar := strings.Repeat("█", barLen) + strings.Repeat("░", 40-barLen)

		b.WriteString(fmt.Sprintf("%s │%s│ %dm\n", dayName, bar, minutes))
	}

	return b.String()
}

func renderActivityHeatmap(entriesList []entries.Entry, cfg config.Config, width int) string {
	dayTotals := make(map[string]int)
	for _, entry := range entriesList {
		if entry.Type == "work" {
			dayTotals[streak.DateKey(entry.Start)] += entry.Duration
		}
	}

	maxWeeks := (width - 8) / 2
	if maxWeeks > 52 {
		maxWeeks = 52
	}
	if maxWeeks < 4 {
		return "Terminal too narrow for heatmap."
	}

	today := time.Now()
	currentSunday := today.AddDate(0, 0, -int(today.Weekday()))
	startDate := currentSunday.AddDate(0, 0, -7*(maxWeeks-1))

	var b strings.Builder

	b.WriteString("    ")
	lastMonth := -1
	for w := 0; w < maxWeeks; w++ {
		date := startDate.AddDate(0, 0, w*7)
		if int(date.Month()) != lastMonth {
			label := date.Format("Jan")
			b.WriteString(label)
			if len(label) < 2 {
				b.WriteString(" ")
			}
			lastMonth = int(date.Month())
			skip := (len(label) + 1) / 2
			for i := 1; i < skip && w+i < maxWeeks; i++ {
				w++
			}
		} else {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")

	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	theme := activeTheme(cfg)

	for d := 0; d < 7; d++ {
		b.WriteString(fmt.Sprintf("%3s ", days[d]))
		for w := 0; w < maxWeeks; w++ {
			date := startDate.AddDate(0, 0, w*7+d)
			if date.After(today) {
				b.WriteString("  ")
				continue
			}

			seconds := dayTotals[streak.DateKey(date)]
			b.WriteString(renderHeatmapBlock(seconds, cfg, theme) + " ")
		}
		b.WriteString("\n")
	}

	b.WriteString("\n    Less ")
	b.WriteString(renderHeatmapBlock(0, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(15*60, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(45*60, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(90*60, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(150*60, cfg, theme) + " ")
	b.WriteString(" More\n")

	return b.String()
}

func renderHeatmapBlock(seconds int, cfg config.Config, theme config.ThemeStyle) string {
	minutes := seconds / 60
	style := lipgloss.NewStyle()

	if minutes <= 0 {
		return style.Foreground(lipgloss.Color("8")).Render("·")
	}

	if minutes < 30 {
		return style.Foreground(lipgloss.Color(theme.Primary)).Faint(true).Render("█")
	} else if minutes < 60 {
		return style.Foreground(lipgloss.Color(theme.Primary)).Render("█")
	} else if minutes < 120 {
		return style.Foreground(lipgloss.Color(theme.Primary)).Bold(true).Render("█")
	} else {
		return style.Foreground(lipgloss.Color(theme.Accent)).Render("█")
	}
}

func renderStreakHistoryChart(entriesList []entries.Entry) string {
	streakDays := make(map[string]bool)
	for _, e := range entriesList {
		if e.Type == "work" {
			streakDays[streak.DateKey(e.Start)] = true
		}
	}
	if len(streakDays) == 0 {
		return "No streak history yet."
	}

	var b strings.Builder
	for i := 13; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i)
		key := streak.DateKey(day)
		marker := "·"
		if streakDays[key] {
			marker = "█"
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", day.Format("Jan 02"), marker, statusForStreakDay(streakDays, day)))
	}
	return b.String()
}

func statusForStreakDay(days map[string]bool, day time.Time) string {
	key := streak.DateKey(day)
	if days[key] {
		return "work logged"
	}
	return "no work"
}
