package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/analytics"
	"kairu-tui/internal/streak"
	"kairu-tui/internal/ui"
)

func renderStatsView(m model) string {
	weeklyData := analytics.GetWeeklyData(m.entries)
	barChart := renderWeeklyBarChart(weeklyData)
	streakChart := renderStreakHistoryChart(m.entries)

	daily := streak.FormatDuration(analytics.GetDailyTotal(m.entries, "work"))
	streakState := streak.ComputeStreakState(m.entries)
	emptyMessage := ""
	if len(m.entries) == 0 {
		emptyMessage = "No sessions yet. Start a focus session to see stats."
	}
	tagSummary := renderTopTags(m.entries)

	workRatio := 0
	total := m.totalWorkTime + m.totalBreakTime
	if total > 0 {
		workRatio = m.totalWorkTime * 100 / total
	}
	errorLine := renderAppError(m)
	if emptyMessage != "" {
		emptyMessage = fmt.Sprintf("\n%s\n", emptyMessage)
	}

	tabs := renderStatsTabs("stats", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	theme := activeTheme(m.config)
	cardWidth := 18
	// Ensure uniform height by adding a newline to single-line cards
	card1 := ui.Panel("📅 TODAY", fmt.Sprintf("  %s\n", daily), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)
	card2 := ui.Panel("🔥 STREAKS", fmt.Sprintf("  Current: %d\n  Best: %d", streakState.Current, streakState.Best), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)
	card3 := ui.Panel("✦ RECOVERY", fmt.Sprintf("  %s\n", streak.RecoveryLabel(streakState)), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)
	card4 := ui.Panel("⚖️ RATIO", fmt.Sprintf("  Work: %d%%\n  Break: %d%%", workRatio, 100-workRatio), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)

	var cardsBlock string
	if m.width >= 80 {
		// Fit all in a single horizontal row
		cardsBlock = lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2, "  ", card3, "  ", card4)
	} else if m.width >= 40 {
		// 2x2 grid
		row1 := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, card3, "  ", card4)
		cardsBlock = row1 + "\n\n" + row2
	} else {
		// Stack vertically
		cardsBlock = strings.Join([]string{card1, card2, card3, card4}, "\n\n")
	}

	barWidth := 60
	if m.width >= 80 {
		barWidth = 80
	} else if m.width > 0 && m.width < 60 {
		barWidth = m.width - 6
	}

	shortcuts := []string{"[Tab] Cycle Views", "[S] Settings", "[?] Help", "[q] Quit"}
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, barWidth)

	block := fmt.Sprintf(`%s
%s

%s

Weekly Activity (7 days):

%s

Streak History (14 days):

%s

%s
%s
%s`, renderBanner(m.config), tabs, cardsBlock, barChart, streakChart, emptyMessage, tagSummary, statusBar)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderAnalyticsView(m model) string {
	taskTotals, tagTotals, summary := analytics.BuildAnalyticsSummary(m.entries)
	tabs := renderStatsTabs("analytics", m.config)
	errorLine := renderAppError(m)

	theme := activeTheme(m.config)

	// Responsive Card Width
	cardWidth := 60
	if m.width > 0 && m.width < 70 {
		cardWidth = m.width - 6
	}
	if cardWidth < 30 {
		cardWidth = 30
	}

	// Bar width inside card: cardWidth - padding(4) - name(14) - duration(16)
	barWidth := cardWidth - 34
	if barWidth < 6 {
		barWidth = 6
	}

	// 1. PRODUCTIVITY SUMMARY CARD
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(fmt.Sprintf("  Sessions analyzed: %d\n", summary.TotalSessions))
	summaryBuilder.WriteString(fmt.Sprintf("  Work time:         %s\n", streak.FormatDuration(summary.WorkSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Break time:        %s\n", streak.FormatDuration(summary.BreakSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Average session:   %s\n", streak.FormatDuration(summary.AverageSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Longest session:   %s\n", streak.FormatDuration(summary.LongestSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Busiest day:       %s", summary.BusiestDay))
	summaryCard := ui.Panel("PRODUCTIVITY SUMMARY", summaryBuilder.String(), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)

	// 2. TOP TASKS CARD
	taskLines := renderTopDurationBars(taskTotals, summary.WorkSeconds, 5, theme, true, barWidth)
	var tasksContent string
	if len(taskLines) == 0 {
		tasksContent = "  No task breakdown yet."
	} else {
		tasksContent = strings.Join(taskLines, "\n")
	}
	tasksCard := ui.Panel("Top tasks:", tasksContent, theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)

	// 3. TOP TAGS CARD
	totalTagSeconds := 0
	for _, secs := range tagTotals {
		totalTagSeconds += secs
	}
	tagLines := renderTopDurationBars(tagTotals, totalTagSeconds, 5, theme, false, barWidth)
	var tagsContent string
	if len(tagLines) == 0 {
		tagsContent = "  No tag breakdown yet."
	} else {
		tagsContent = strings.Join(tagLines, "\n")
	}
	tagsCard := ui.Panel("Top tags:", tagsContent, theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)

	shortcuts := []string{"[Tab] Cycle Views", "[S] Settings", "[?] Help", "[q] Quit"}
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, cardWidth)

	block := fmt.Sprintf(`%s
%s

%s

%s

%s

%s`, renderBanner(m.config), tabs, summaryCard, tasksCard, tagsCard, statusBar)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderHeatmapView(m model) string {
	heatmap := renderActivityHeatmap(m.entries, m.config, m.width)
	tabs := renderStatsTabs("heatmap", m.config)
	errorLine := renderAppError(m)

	theme := activeTheme(m.config)
	barWidth := 60
	if m.width > 0 && m.width < 70 {
		barWidth = m.width - 6
	}

	shortcuts := []string{"[Tab] Cycle Views", "[S] Settings", "[?] Help", "[q] Quit"}
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, barWidth)

	block := fmt.Sprintf(`%s
%s

%s

%s`, renderBanner(m.config), tabs, heatmap, statusBar)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}
