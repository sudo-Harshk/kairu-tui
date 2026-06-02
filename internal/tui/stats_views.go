package tui

import (
	"fmt"
	"strings"

	"kairu-tui/internal/analytics"
	"kairu-tui/internal/streak"
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

	block := fmt.Sprintf(`%s
%s

┌─────────────────┐
│  📅  Today      │
│  %-13s  │
└─────────────────┘

┌─────────────────┐
│  🔥  Streaks    │
│  Current: %-3d  │
│  Best: %-7d│
└─────────────────┘

┌─────────────────┐
│  Recovery       │
│  %-13s  │
└─────────────────┘

┌─────────────────┐
│  ⚖️  Ratio      │
│  Work: %d%%     │
│  Break: %d%%    │
└─────────────────┘

Weekly Activity (7 days):

%s

Streak History (14 days):

%s

%s

%s

%s
`, renderBanner(m.config), tabs, daily, streakState.Current, streakState.Best, streak.RecoveryLabel(streakState), workRatio, 100-workRatio, barChart, streakChart, emptyMessage, tagSummary, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderAnalyticsView(m model) string {
	taskTotals, tagTotals, summary := analytics.BuildAnalyticsSummary(m.entries)
	tabs := renderStatsTabs("analytics", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

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
	summaryCard := renderDashboardCard("PRODUCTIVITY SUMMARY", summaryBuilder.String(), theme, cardWidth)

	// 2. TOP TASKS CARD
	taskLines := renderTopDurationBars(taskTotals, summary.WorkSeconds, 5, theme, true, barWidth)
	var tasksContent string
	if len(taskLines) == 0 {
		tasksContent = "  No task breakdown yet."
	} else {
		tasksContent = strings.Join(taskLines, "\n")
	}
	tasksCard := renderDashboardCard("Top tasks:", tasksContent, theme, cardWidth)

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
	tagsCard := renderDashboardCard("Top tags:", tagsContent, theme, cardWidth)

	block := fmt.Sprintf(`%s
%s

%s

%s

%s

%s
`, renderBanner(m.config), tabs, summaryCard, tasksCard, tagsCard, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderHeatmapView(m model) string {
	heatmap := renderActivityHeatmap(m.entries, m.config, m.width)
	tabs := renderStatsTabs("heatmap", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	block := fmt.Sprintf(`%s
%s

%s

%s
`, renderBanner(m.config), tabs, heatmap, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}
