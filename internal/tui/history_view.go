package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/analytics"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
	"kairu-tui/internal/ui"
)

func renderHistoryView(m model) string {
	theme := activeTheme(m.config)

	// Determine date bounds based on historyFilter.dateRange
	var dateFrom, dateTo time.Time
	now := time.Now()
	switch m.historyFilter.dateRange {
	case "today":
		dateFrom = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dateTo = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	case "week":
		offset := int(now.Weekday()) - 1
		if offset < 0 {
			offset = 6
		}
		monday := now.AddDate(0, 0, -offset)
		dateFrom = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())
		sunday := monday.AddDate(0, 0, 6)
		dateTo = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 999999999, now.Location())
	case "month":
		dateFrom = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		nextMonth := dateFrom.AddDate(0, 1, 0)
		dateTo = nextMonth.Add(-time.Nanosecond)
	}

	// Filter entries
	opt := entries.FilterOption{
		Query:    m.historyFilter.searchInput.Value(),
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Type:     m.historyFilter.typeFilter,
	}
	filtered := entries.FilterEntries(m.entries, opt)

	// Sort filtered entries by Start time descending
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Start.Equal(filtered[j].Start) {
			return filtered[i].End.After(filtered[j].End)
		}
		return filtered[i].Start.After(filtered[j].Start)
	})

	// Construct card/view widths
	cardWidth := 76
	if m.width > 0 && m.width < 82 {
		cardWidth = m.width - 6
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	// Render the Search Input view inside a FormField
	searchView := ui.FormField("[/] Search", m.historyFilter.searchInput.View(), m.historyFilter.searchFocused, theme)

	// Construct Badges for Type and Range
	typeLabel := "All"
	if m.historyFilter.typeFilter == "work" {
		typeLabel = "Work"
	} else if m.historyFilter.typeFilter == "break" {
		typeLabel = "Break"
	}
	typeBadge := ui.Badge(typeLabel, theme, m.historyFilter.typeFilter != "all")

	rangeLabel := "All Time"
	if m.historyFilter.dateRange == "today" {
		rangeLabel = "Today"
	} else if m.historyFilter.dateRange == "week" {
		rangeLabel = "This Week"
	} else if m.historyFilter.dateRange == "month" {
		rangeLabel = "This Month"
	}
	rangeBadge := ui.Badge(rangeLabel, theme, m.historyFilter.dateRange != "all")

	tHint := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.WorkAccent)).Bold(true).Render("[t]")
	dHint := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.WorkAccent)).Bold(true).Render("[d]")
	filterLine := fmt.Sprintf("  %s Type: %s    %s Range: %s", tHint, typeBadge, dHint, rangeBadge)

	// Counts and duration
	filteredDuration := 0
	for _, entry := range filtered {
		filteredDuration += entry.Duration
	}
	summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice)).Italic(true)
	summaryText := fmt.Sprintf("  Showing %d of %d sessions  (%s filtered)",
		len(filtered), len(m.entries), streak.FormatDuration(filteredDuration))

	// Group sessions by day and print
	var sessionLines []string
	if len(filtered) == 0 {
		sessionLines = append(sessionLines, "  No matching sessions found.")
	} else {
		grouped := make(map[string][]entries.Entry)
		order := make([]string, 0)
		for _, entry := range filtered {
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
			sessionLines = append(sessionLines, fmt.Sprintf("  %s  (%d sessions, %s)",
				dayLabel, len(groupEntries), streak.FormatDuration(dayTotal)))
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
				sessionLines = append(sessionLines, fmt.Sprintf("    - %-18s %s", task, strings.Join(meta, " | ")))
			}
			sessionLines = append(sessionLines, "")
		}
	}

	var panelContentParts []string
	panelContentParts = append(panelContentParts, "  "+searchView)
	panelContentParts = append(panelContentParts, "")
	panelContentParts = append(panelContentParts, filterLine)
	panelContentParts = append(panelContentParts, "")
	panelContentParts = append(panelContentParts, summaryStyle.Render(summaryText))
	panelContentParts = append(panelContentParts, "")
	panelContentParts = append(panelContentParts, ui.Divider(cardWidth-4, theme))
	panelContentParts = append(panelContentParts, "")
	panelContentParts = append(panelContentParts, strings.Join(sessionLines, "\n"))

	tabs := renderStatsTabs("history", m.config)
	historyCard := ui.Panel("📜 Focus History", strings.Join(panelContentParts, "\n"), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)

	shortcuts := []string{"[/] Search", "[t] Type", "[d] Range", "[Esc] Clear/Back", "[Tab] Cycle Views", "[S] Settings", "[?] Help", "[q] Quit"}
	errorLine := renderAppError(m)
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, cardWidth)

	block := renderBanner(m.config) + "\n" +
		tabs + "\n" +
		historyCard + "\n\n" +
		statusBar
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderDailyReportView(m model) string {
	lines := analytics.BuildDailyReport(m.entries, time.Now())
	tabs := renderStatsTabs("report", m.config)

	theme := activeTheme(m.config)
	cardWidth := 76
	if m.width > 0 && m.width < 82 {
		cardWidth = m.width - 6
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	reportCard := ui.Panel("📊 Daily Productivity Report", strings.Join(lines, "\n"), theme, cardWidth, lipgloss.RoundedBorder(), theme.Primary)

	shortcuts := []string{"[Tab] Cycle Views", "[E] Export markdown", "[S] Settings", "[?] Help", "[q] Quit"}
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m, cardWidth)
	statusBar := ui.StatusBar(shortcuts, errorLine, theme, cardWidth)

	var block string
	if statusLine != "" {
		block = renderBanner(m.config) + "\n" +
			tabs + "\n" +
			reportCard + "\n\n" +
			statusLine + "\n\n" +
			statusBar
	} else {
		block = renderBanner(m.config) + "\n" +
			tabs + "\n" +
			reportCard + "\n\n" +
			statusBar
	}
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
