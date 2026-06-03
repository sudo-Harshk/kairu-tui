package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
	"kairu-tui/internal/tasks"
	"kairu-tui/internal/ui"
)

// computeRecentEntries gets the last 'limit' entries in reverse chronological order
func computeRecentEntries(allEntries []entries.Entry, limit int) []entries.Entry {
	if len(allEntries) == 0 {
		return nil
	}
	n := len(allEntries)
	start := n - limit
	if start < 0 {
		start = 0
	}
	recent := make([]entries.Entry, 0, n-start)
	for i := n - 1; i >= start; i-- {
		recent = append(recent, allEntries[i])
	}
	return recent
}

// computeSidebarMetrics calculates streak, daily progress, weekly progress, and tasks
func computeSidebarMetrics(allEntries []entries.Entry, streakState streak.StreakState, tasksFile string) sidebarMetrics {
	var metrics sidebarMetrics
	metrics.Streak = streakState.Current

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Start of the week (Monday)
	daysSinceMonday := int(now.Weekday()) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6
	}
	weekStart := todayStart.AddDate(0, 0, -daysSinceMonday)

	var todayWorkTime time.Duration
	var todayBreakTime time.Duration
	var todayWorkCount int
	var weekWorkCount int

	for _, e := range allEntries {
		if e.Start.After(todayStart) {
			if e.Type == "work" {
				todayWorkTime += time.Duration(e.Duration) * time.Second
				todayWorkCount++
			} else if e.Type == "break" {
				todayBreakTime += time.Duration(e.Duration) * time.Second
			}
		}
		if e.Start.After(weekStart) && e.Type == "work" {
			weekWorkCount++
		}
	}

	metrics.TodayTotal = todayWorkTime
	metrics.SessionCount = todayWorkCount

	if todayBreakTime > 0 {
		metrics.WorkBreakRatio = float64(todayWorkTime) / float64(todayBreakTime)
	} else if todayWorkTime > 0 {
		metrics.WorkBreakRatio = 99.9
	} else {
		metrics.WorkBreakRatio = 0
	}

	dailyTarget := 4
	weeklyTarget := 20
	metrics.DailyProgress = (float64(todayWorkCount) / float64(dailyTarget)) * 100
	if metrics.DailyProgress > 100 {
		metrics.DailyProgress = 100
	}
	metrics.WeeklyProgress = (float64(weekWorkCount) / float64(weeklyTarget)) * 100
	if metrics.WeeklyProgress > 100 {
		metrics.WeeklyProgress = 100
	}

	fileTasks := tasks.LoadTasksFromFile(tasksFile)
	var microTasks []string
	var microTasksDone []bool

	for _, t := range fileTasks {
		if len(microTasks) >= 4 {
			break
		}
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		microTasks = append(microTasks, t)

		completedToday := false
		for _, e := range allEntries {
			if e.Start.After(todayStart) && e.Type == "work" && strings.EqualFold(strings.TrimSpace(e.Task), t) {
				completedToday = true
				break
			}
		}
		microTasksDone = append(microTasksDone, completedToday)
	}

	metrics.MicroTasks = microTasks
	metrics.MicroTasksDone = microTasksDone

	return metrics
}

// renderRightContextPane builds the right sidebar stacked with stats, history, queue, and metrics.
func renderRightContextPane(m model, width int) string {
	theme := activeTheme(m.config)

	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary))
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.WorkAccent))

	divWidth := width
	if divWidth < 0 {
		divWidth = 0
	}
	divider := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Border)).Render(strings.Repeat("─", divWidth))

	// 1. Session Stats
	statsTitle := accentStyle.Render("◆ ") + primaryStyle.Bold(true).Render("Session Stats")
	statsLines := []string{
		accentStyle.Render("⚡ ") + mutedStyle.Render("Focus:  ") + primaryStyle.Render(streak.FormatDuration(int(m.sidebarMetrics.TodayTotal.Seconds()))),
		accentStyle.Render("🏆 ") + mutedStyle.Render("Streak: ") + primaryStyle.Render(fmt.Sprintf("%d days", m.sidebarMetrics.Streak)),
		accentStyle.Render("⏱️ ") + mutedStyle.Render("Count:  ") + primaryStyle.Render(fmt.Sprintf("%d sessions", m.sidebarMetrics.SessionCount)),
	}
	ratioStr := "0.0"
	if m.sidebarMetrics.WorkBreakRatio > 90 {
		ratioStr = "∞"
	} else if m.sidebarMetrics.WorkBreakRatio > 0 {
		ratioStr = fmt.Sprintf("%.1f", m.sidebarMetrics.WorkBreakRatio)
	}
	statsLines = append(statsLines, accentStyle.Render("⚖️ ") + mutedStyle.Render("Ratio:  ") + primaryStyle.Render(ratioStr))
	statsSection := statsTitle + "\n" + strings.Join(statsLines, "\n")

	// 2. Recent History
	historyTitle := accentStyle.Render("◆ ") + primaryStyle.Bold(true).Render("Recent History")
	var historyLines []string
	for _, e := range m.sidebarEntries {
		indicator := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.WorkAccent)).Render("●")
		if e.Type == "break" {
			indicator = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.BreakAccent)).Render("○")
		}
		taskName := e.Task
		
		// Dynamically truncate task name based on width (indicator + space + tags = ~8-9 cells)
		maxTaskNameLen := width - 9
		if maxTaskNameLen < 10 {
			maxTaskNameLen = 10
		}
		if len(taskName) > maxTaskNameLen {
			taskName = taskName[:maxTaskNameLen-3] + "..."
		}
		
		durStr := streak.FormatDuration(e.Duration)
		historyLines = append(historyLines, fmt.Sprintf("%s %s %s", indicator, primaryStyle.Render(taskName), mutedStyle.Render("("+durStr+")")))
	}
	if len(historyLines) == 0 {
		historyLines = append(historyLines, mutedStyle.Render("No history yet"))
	}
	historySection := historyTitle + "\n" + strings.Join(historyLines, "\n")

	// 3. Micro-task Queue
	queueTitle := accentStyle.Render("◆ ") + primaryStyle.Bold(true).Render("Task Queue")
	var queueLines []string
	if m.running && m.taskName != "" {
		activeName := m.taskName
		maxActiveLen := width - 15
		if maxActiveLen < 10 {
			maxActiveLen = 10
		}
		if len(activeName) > maxActiveLen {
			activeName = activeName[:maxActiveLen-3] + "..."
		}
		queueLines = append(queueLines, accentStyle.Render("▸ ") + primaryStyle.Bold(true).Render(activeName) + " " + accentStyle.Render("(Active)"))
	}
	for i, t := range m.sidebarMetrics.MicroTasks {
		if m.running && strings.EqualFold(t, m.taskName) {
			continue
		}
		chk := " [ ] "
		var statusStyle lipgloss.Style
		if m.sidebarMetrics.MicroTasksDone[i] {
			chk = " [✔] "
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.BreakAccent))
		} else {
			statusStyle = mutedStyle
		}
		
		displayName := t
		maxDispLen := width - 8
		if maxDispLen < 10 {
			maxDispLen = 10
		}
		if len(displayName) > maxDispLen {
			displayName = displayName[:maxDispLen-3] + "..."
		}
		
		queueLines = append(queueLines, statusStyle.Render(chk) + mutedStyle.Render(displayName))
	}
	if len(queueLines) == 0 {
		queueLines = append(queueLines, mutedStyle.Render("No tasks in queue"))
	}
	if len(queueLines) > 4 {
		queueLines = queueLines[:4]
	}
	queueSection := queueTitle + "\n" + strings.Join(queueLines, "\n")

	// 4. Progress Metrics
	progressTitle := accentStyle.Render("◆ ") + primaryStyle.Bold(true).Render("Goal Progress")
	barWidth := divWidth - 14
	if barWidth < 6 {
		barWidth = 6
	}

	dailyPct := m.sidebarMetrics.DailyProgress
	dailyBar := ui.ProgressBar(dailyPct, barWidth, theme.WorkAccent, theme.Border)
	dailyLine := fmt.Sprintf("%s %s %s", mutedStyle.Render("Daily:  "), dailyBar, primaryStyle.Render(fmt.Sprintf("%.0f%%", dailyPct)))

	weeklyPct := m.sidebarMetrics.WeeklyProgress
	weeklyBar := ui.ProgressBar(weeklyPct, barWidth, theme.BreakAccent, theme.Border)
	weeklyLine := fmt.Sprintf("%s %s %s", mutedStyle.Render("Weekly: "), weeklyBar, primaryStyle.Render(fmt.Sprintf("%.0f%%", weeklyPct)))

	progressSection := progressTitle + "\n" + dailyLine + "\n" + weeklyLine

	// Assemble all sections with dividers
	sections := []string{statsSection, historySection, queueSection, progressSection}
	return strings.Join(sections, "\n\n"+divider+"\n\n")
}
