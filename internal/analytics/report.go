package analytics

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
)

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
