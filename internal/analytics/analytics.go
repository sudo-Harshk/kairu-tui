package analytics

import (
	"fmt"
	"strings"
	"time"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
)

type AnalyticsSummary struct {
	TotalSessions  int
	WorkSeconds    int
	BreakSeconds   int
	AverageSeconds int
	LongestSeconds int
	BusiestDay     string
}

func BuildAnalyticsSummary(entryList []entries.Entry) (map[string]int, map[string]int, AnalyticsSummary) {
	taskTotals := make(map[string]int)
	tagTotals := make(map[string]int)
	dayTotals := make(map[string]int)
	summary := AnalyticsSummary{BusiestDay: "n/a"}
	for _, entry := range entryList {
		summary.TotalSessions++
		if entry.Type == "break" {
			summary.BreakSeconds += entry.Duration
		} else {
			summary.WorkSeconds += entry.Duration
		}
		if entry.Duration > summary.LongestSeconds {
			summary.LongestSeconds = entry.Duration
		}
		task := strings.TrimSpace(entry.Task)
		if task == "" {
			task = "(untitled)"
		}
		taskTotals[task] += entry.Duration
		for _, tag := range entry.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagTotals[tag] += entry.Duration
			}
		}
		dayTotals[streak.DateKey(entry.Start)] += entry.Duration
	}

	if summary.TotalSessions > 0 {
		summary.AverageSeconds = (summary.WorkSeconds + summary.BreakSeconds) / summary.TotalSessions
	}
	if len(dayTotals) > 0 {
		var maxDay string
		var maxTotal int
		for day, total := range dayTotals {
			if total > maxTotal || maxDay == "" {
				maxDay = day
				maxTotal = total
			}
		}
		if parsed, err := time.ParseInLocation("2006-01-02", maxDay, time.Local); err == nil {
			summary.BusiestDay = fmt.Sprintf("%s (%s)", parsed.Format("Mon, Jan 02"), streak.FormatDuration(maxTotal))
		}
	}

	return taskTotals, tagTotals, summary
}

func GetDailyTotal(entryList []entries.Entry, sessionType string) int {
	today := streak.DateKey(time.Now())
	total := 0
	for _, e := range entryList {
		if streak.DateKey(e.Start) == today && e.Type == sessionType {
			total += e.Duration
		}
	}
	return total
}

func GetWeeklyData(entryList []entries.Entry) map[string]int {
	weekly := make(map[string]int)
	today := time.Now()
	for i := 0; i < 7; i++ {
		date := streak.DateKey(today.AddDate(0, 0, -i))
		weekly[date] = 0
	}
	for _, e := range entryList {
		date := streak.DateKey(e.Start)
		if _, ok := weekly[date]; ok && e.Type == "work" {
			weekly[date] += e.Duration
		}
	}
	return weekly
}
