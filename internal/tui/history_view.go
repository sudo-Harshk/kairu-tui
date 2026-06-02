package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"kairu-tui/internal/analytics"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
)

func renderHistoryView(m model) string {
	entryList := make([]entries.Entry, len(m.entries))
	copy(entryList, m.entries)
	sort.Slice(entryList, func(i, j int) bool {
		if entryList[i].Start.Equal(entryList[j].Start) {
			return entryList[i].End.After(entryList[j].End)
		}
		return entryList[i].Start.After(entryList[j].Start)
	})

	lines := []string{"Recent Sessions by Day:"}
	if len(entryList) == 0 {
		lines = append(lines, "  No sessions recorded yet.")
	} else {
		limit := 15
		if len(entryList) < limit {
			limit = len(entryList)
		}
		grouped := make(map[string][]entries.Entry)
		order := make([]string, 0, len(entryList))
		for i := 0; i < limit; i++ {
			entry := entryList[i]
			key := streak.DateKey(entry.Start)
			if _, ok := grouped[key]; !ok {
				order = append(order, key)
			}
			grouped[key] = append(grouped[key], entry)
		}

		for _, key := range order {
			groupEntries := grouped[key]
			dayLabel := groupEntries[0].Start.Local().Format("Mon, Jan 02, 2006")
			dayTotal := 0
			for _, entry := range groupEntries {
				dayTotal += entry.Duration
			}
			lines = append(lines, fmt.Sprintf("  %s  (%d sessions, %s)", dayLabel, len(groupEntries), streak.FormatDuration(dayTotal)))
			for _, entry := range groupEntries {
				kind := strings.ToUpper(strings.TrimSpace(entry.Type))
				if kind == "" {
					kind = "WORK"
				}
				task := strings.TrimSpace(entry.Task)
				if task == "" {
					task = "(untitled)"
				}
				when := entry.Start.Local().Format("15:04")
				meta := []string{when, kind, streak.FormatDuration(entry.Duration)}
				if note := strings.TrimSpace(entry.Note); note != "" {
					meta = append(meta, "note: "+note)
				}
				if len(entry.Tags) > 0 {
					meta = append(meta, "tags: "+strings.Join(entry.Tags, ", "))
				}
				lines = append(lines, fmt.Sprintf("    - %-18s %s", task, strings.Join(meta, " | ")))
			}
			lines = append(lines, "")
		}
		if len(entryList) > limit {
			lines = append(lines, fmt.Sprintf("  ... and %d more sessions", len(entryList)-limit))
		}
	}

	tabs := renderStatsTabs("history", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	block := renderBanner(m.config) + "\n" +
		tabs + "\n" +
		strings.Join(lines, "\n") + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderDailyReportView(m model) string {
	lines := analytics.BuildDailyReport(m.entries, time.Now())
	tabs := renderStatsTabs("report", m.config)
	footer := "[Tab] Cycle Views   [E] Export markdown   [S] Settings   [?] Help   [q] Quit"
	if strings.TrimSpace(m.notificationStatus) != "" {
		footer = fmt.Sprintf("%s\n%s", renderNotificationStatus(m), footer)
	}
	if errorLine := renderAppError(m); errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	block := renderBanner(m.config) + "\n" +
		tabs + "\n" +
		strings.Join(lines, "\n") + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func (m *model) exportDailyReport() (string, error) {
	path := fmt.Sprintf("kairu-report-%s.md", streak.DateKey(time.Now()))
	lines := analytics.BuildDailyReport(m.entries, time.Now())
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return "", err
	}
	return path, nil
}
