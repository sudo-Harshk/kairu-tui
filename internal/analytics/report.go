package analytics

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
)

type FocusReport struct {
	WeekStart      time.Time
	WeekEnd        time.Time
	TotalSessions  int
	TotalSeconds   int
	Streak         int
	BestStreak     int
	DailyBreakdown []DayBreakdown
	TopTasks       []TaskBreakdown
	BestDay        string
	BestHour       int
}

type DayBreakdown struct {
	Day      string // "Mon", "Tue", etc.
	Seconds  int
	Sessions int
}

type TaskBreakdown struct {
	Name    string
	Seconds int
}

// BuildDailyReport constructs a list of lines representing a daily productivity markdown report.
func BuildDailyReport(entryList []entries.Entry, day time.Time) []string {
	dayKey := streak.DateKey(day)
	workSeconds := 0
	breakSeconds := 0
	var dayEntries []entries.Entry
	for _, entry := range entryList {
		if streak.DateKey(entry.Start) != dayKey {
			continue
		}
		dayEntries = append(dayEntries, entry)
		if entry.Type == "break" {
			breakSeconds += entry.Duration
		} else {
			workSeconds += entry.Duration
		}
	}

	sort.Slice(dayEntries, func(i, j int) bool {
		if dayEntries[i].Start.Equal(dayEntries[j].Start) {
			return dayEntries[i].End.After(dayEntries[j].End)
		}
		return dayEntries[i].Start.Before(dayEntries[j].Start)
	})

	lines := []string{
		fmt.Sprintf("# Kairu Daily Report - %s", day.Format("2006-01-02")),
		"",
		fmt.Sprintf("- Work: %s", streak.FormatDuration(workSeconds)),
		fmt.Sprintf("- Break: %s", streak.FormatDuration(breakSeconds)),
		fmt.Sprintf("- Sessions: %d", len(dayEntries)),
	}
	if len(dayEntries) == 0 {
		lines = append(lines, "- No sessions recorded today.")
		return lines
	}

	lines = append(lines, "", "## Sessions")
	for _, entry := range dayEntries {
		task := strings.TrimSpace(entry.Task)
		if task == "" {
			task = "(untitled)"
		}
		kind := strings.ToUpper(strings.TrimSpace(entry.Type))
		if kind == "" {
			kind = "WORK"
		}
		meta := []string{
			entry.Start.Local().Format("15:04"),
			kind,
			streak.FormatDuration(entry.Duration),
		}
		if note := strings.TrimSpace(entry.Note); note != "" {
			meta = append(meta, "note: "+note)
		}
		if len(entry.Tags) > 0 {
			meta = append(meta, "tags: "+strings.Join(entry.Tags, ", "))
		}
		lines = append(lines, fmt.Sprintf("- %s | %s", task, strings.Join(meta, " | ")))
	}
	return lines
}

// BuildWeeklyReport builds the weekly focus report for the current week.
func BuildWeeklyReport(entryList []entries.Entry) FocusReport {
	return BuildWeeklyReportAt(entryList, time.Now())
}

// BuildWeeklyReportAt builds the weekly focus report at a specific reference time (for testing).
func BuildWeeklyReportAt(entryList []entries.Entry, referenceTime time.Time) FocusReport {
	refLocal := referenceTime.Local()
	daysToSubtract := int(refLocal.Weekday()) - 1
	if daysToSubtract < 0 {
		daysToSubtract = 6 // Sunday is 0
	}

	weekStart := time.Date(refLocal.Year(), refLocal.Month(), refLocal.Day(), 0, 0, 0, 0, refLocal.Location()).AddDate(0, 0, -daysToSubtract)
	weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Nanosecond)

	var weeklyWorkEntries []entries.Entry
	for _, entry := range entryList {
		if entry.Type == "work" && !entry.Start.Before(weekStart) && !entry.Start.After(weekEnd) {
			weeklyWorkEntries = append(weeklyWorkEntries, entry)
		}
	}

	totalSeconds := 0
	for _, entry := range weeklyWorkEntries {
		totalSeconds += entry.Duration
	}

	daily := make([]DayBreakdown, 7)
	daysOfWeek := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for i, day := range daysOfWeek {
		daily[i] = DayBreakdown{Day: day}
	}

	for _, entry := range weeklyWorkEntries {
		wd := int(entry.Start.Local().Weekday())
		idx := wd - 1
		if idx < 0 {
			idx = 6 // Sunday
		}
		daily[idx].Seconds += entry.Duration
		daily[idx].Sessions++
	}

	taskMap := make(map[string]int)
	for _, entry := range weeklyWorkEntries {
		name := strings.TrimSpace(entry.Task)
		if name == "" {
			name = "(untitled)"
		}
		taskMap[name] += entry.Duration
	}

	var tasks []TaskBreakdown
	for name, sec := range taskMap {
		tasks = append(tasks, TaskBreakdown{Name: name, Seconds: sec})
	}
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Seconds == tasks[j].Seconds {
			return tasks[i].Name < tasks[j].Name
		}
		return tasks[i].Seconds > tasks[j].Seconds
	})

	bestDayName := "None"
	maxSeconds := 0
	weekdayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	for i, db := range daily {
		if db.Seconds > maxSeconds {
			maxSeconds = db.Seconds
			bestDayName = weekdayNames[i]
		}
	}

	bestHour := -1
	if len(weeklyWorkEntries) > 0 {
		hourCounts := make(map[int]int)
		hourSeconds := make(map[int]int)
		for _, entry := range weeklyWorkEntries {
			hour := entry.Start.Local().Hour()
			hourCounts[hour]++
			hourSeconds[hour] += entry.Duration
		}
		maxHourCount := 0
		maxHourSeconds := 0
		for h := 0; h < 24; h++ {
			count := hourCounts[h]
			sec := hourSeconds[h]
			if count > maxHourCount {
				maxHourCount = count
				maxHourSeconds = sec
				bestHour = h
			} else if count == maxHourCount && count > 0 {
				if sec > maxHourSeconds {
					maxHourSeconds = sec
					bestHour = h
				}
			}
		}
	}

	sState := streak.ComputeStreakState(entryList)

	return FocusReport{
		WeekStart:      weekStart,
		WeekEnd:        weekEnd,
		TotalSessions:  len(weeklyWorkEntries),
		TotalSeconds:   totalSeconds,
		Streak:         sState.Current,
		BestStreak:     sState.Best,
		DailyBreakdown: daily,
		TopTasks:       tasks,
		BestDay:        bestDayName,
		BestHour:       bestHour,
	}
}

var blockFractions = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

func renderFractionalBar(val, maxVal float64, maxWidth int) string {
	if maxVal <= 0 || val <= 0 {
		return ""
	}
	totalEighths := int((val / maxVal) * float64(maxWidth) * 8.0 + 0.5)
	if totalEighths > maxWidth*8 {
		totalEighths = maxWidth * 8
	}
	fullBlocks := totalEighths / 8
	fraction := totalEighths % 8

	bar := strings.Repeat("█", fullBlocks)
	if fraction > 0 {
		bar += blockFractions[fraction]
	}
	return bar
}

func wrapInBorders(content string, width int) string {
	visualWidth := lipgloss.Width(content)
	padding := width - visualWidth
	if padding < 0 {
		padding = 0
	}
	return "│" + content + strings.Repeat(" ", padding) + "│"
}

func formatWeekRange(start, end time.Time) string {
	if start.Year() != end.Year() {
		return fmt.Sprintf("%s %d, %d – %s %d, %d",
			start.Format("Jan"), start.Day(), start.Year(),
			end.Format("Jan"), end.Day(), end.Year())
	}
	return fmt.Sprintf("%s %d – %s %d, %d",
		start.Format("Jan"), start.Day(),
		end.Format("Jan"), end.Day(), start.Year())
}

func formatHourRange(hour int) string {
	if hour < 0 || hour > 23 {
		return "n/a"
	}
	formatHour := func(h int) string {
		h = h % 24
		if h == 0 {
			return "12 AM"
		}
		if h == 12 {
			return "12 PM"
		}
		if h < 12 {
			return fmt.Sprintf("%d AM", h)
		}
		return fmt.Sprintf("%d PM", h-12)
	}
	return fmt.Sprintf("%s - %s", formatHour(hour), formatHour(hour+1))
}

// RenderReport generates the formatted ASCII weekly focus report string.
func RenderReport(report FocusReport) string {
	var sb strings.Builder

	// Top border
	sb.WriteString("╭" + strings.Repeat("─", 21) + " Kairu Focus Report " + strings.Repeat("─", 21) + "╮\n")

	// Empty line
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Week range line
	weekStr := fmt.Sprintf("  📊  Week of %s", formatWeekRange(report.WeekStart, report.WeekEnd))
	sb.WriteString(wrapInBorders(weekStr, 62) + "\n")
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Total Focus summary
	focusStr := fmt.Sprintf("  Total Focus: %s  ·  %d sessions  ·  ⚡ %d-day streak",
		streak.FormatDuration(report.TotalSeconds),
		report.TotalSessions,
		report.Streak)
	sb.WriteString(wrapInBorders(focusStr, 62) + "\n")
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Daily breakdown inner box
	sb.WriteString(wrapInBorders("  ┌─────────────────────────────────────────────────────┐", 62) + "\n")

	// Find max daily seconds for scaling
	maxDailySecs := 0
	for _, db := range report.DailyBreakdown {
		if db.Seconds > maxDailySecs {
			maxDailySecs = db.Seconds
		}
	}

	for _, db := range report.DailyBreakdown {
		barStr := ""
		if maxDailySecs > 0 && db.Seconds > 0 {
			barStr = renderFractionalBar(float64(db.Seconds), float64(maxDailySecs), 20)
		}

		durStr := streak.FormatDuration(db.Seconds)

		maxSessionsToShow := 15
		numBlocks := db.Sessions
		if numBlocks > maxSessionsToShow {
			numBlocks = maxSessionsToShow
		}
		sessionBlocks := strings.Repeat("■", numBlocks)

		dayPrefix := fmt.Sprintf("   %s ", db.Day)
		barAndPadding := barStr
		barVisualLen := lipgloss.Width(barStr)
		if barVisualLen < 20 {
			barAndPadding += strings.Repeat(" ", 20-barVisualLen)
		}

		durPadded := fmt.Sprintf("%6s", durStr)
		innerLineContent := fmt.Sprintf("%s%s  %s  %s", dayPrefix, barAndPadding, durPadded, sessionBlocks)
		innerLineVisualLen := lipgloss.Width(innerLineContent)
		if innerLineVisualLen < 53 {
			innerLineContent += strings.Repeat(" ", 53-innerLineVisualLen)
		} else if innerLineVisualLen > 53 {
			innerLineContent = innerLineContent[:53]
		}

		sb.WriteString(wrapInBorders("  │" + innerLineContent + "│", 62) + "\n")
	}

	sb.WriteString(wrapInBorders("  └─────────────────────────────────────────────────────┘", 62) + "\n")
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Best Day
	bestDayStr := ""
	if report.BestDay != "None" {
		bestDaySeconds := 0
		weekdayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		for i, db := range report.DailyBreakdown {
			if weekdayNames[i] == report.BestDay {
				bestDaySeconds = db.Seconds
				break
			}
		}
		bestDayStr = fmt.Sprintf("  🔥  Best Day: %s (%s)", report.BestDay, streak.FormatDuration(bestDaySeconds))
	} else {
		bestDayStr = "  🔥  Best Day: None"
	}
	sb.WriteString(wrapInBorders(bestDayStr, 62) + "\n")

	// Best Hour
	bestHourStr := fmt.Sprintf("  🎯  Best Hour: %s (most sessions started)", formatHourRange(report.BestHour))
	sb.WriteString(wrapInBorders(bestHourStr, 62) + "\n")
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Top Tasks title
	sb.WriteString(wrapInBorders("  Top Tasks:", 62) + "\n")

	numTasksToShow := 5
	if len(report.TopTasks) < numTasksToShow {
		numTasksToShow = len(report.TopTasks)
	}

	if numTasksToShow == 0 {
		sb.WriteString(wrapInBorders("    No tasks recorded this week.", 62) + "\n")
	} else {
		topTaskSeconds := report.TopTasks[0].Seconds
		for i := 0; i < numTasksToShow; i++ {
			task := report.TopTasks[i]

			name := task.Name
			maxNameLen := 28
			nameVisualLen := lipgloss.Width(name)
			if nameVisualLen > maxNameLen {
				runes := []rune(name)
				if len(runes) > maxNameLen-3 {
					name = string(runes[:maxNameLen-3]) + "..."
				} else {
					name = name[:maxNameLen-3] + "..."
				}
				nameVisualLen = lipgloss.Width(name)
			}
			namePadded := name
			if nameVisualLen < maxNameLen {
				namePadded += strings.Repeat(" ", maxNameLen-nameVisualLen)
			}

			taskBarLen := 0
			if topTaskSeconds > 0 {
				taskBarLen = int((float64(task.Seconds) / float64(topTaskSeconds)) * 14.0 + 0.5)
			}
			taskBar := strings.Repeat("█", taskBarLen)

			durStr := streak.FormatDuration(task.Seconds)

			durPadded := fmt.Sprintf("%6s", durStr)
			taskLine := fmt.Sprintf("    %s  %s  %s", namePadded, durPadded, taskBar)
			sb.WriteString(wrapInBorders(taskLine, 62) + "\n")
		}
	}

	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Streak info
	streakStr := fmt.Sprintf("  🏆  Streak: %d days active · Best ever: %d days", report.Streak, report.BestStreak)
	sb.WriteString(wrapInBorders(streakStr, 62) + "\n")
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Premium tier
	sb.WriteString(wrapInBorders("  ⚡  Premium: Get per-repo breakdown & team reports", 62) + "\n")
	sb.WriteString(wrapInBorders("      → kairu subscribe", 62) + "\n")
	sb.WriteString(wrapInBorders("", 62) + "\n")

	// Bottom border
	sb.WriteString("╰" + strings.Repeat("─", 62) + "╯\n")

	return sb.String()
}
