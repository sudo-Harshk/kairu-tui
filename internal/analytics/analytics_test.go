package analytics

import (
	"strings"
	"testing"
	"time"

	"kairu-tui/internal/entries"
	"kairu-tui/internal/streak"
)

func TestGetDailyTotal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	entryList := []entries.Entry{
		{Task: "a", Start: today, Duration: 600, Type: "work"},
		{Task: "b", Start: today.Add(2 * time.Hour), Duration: 120, Type: "work"},
		{Task: "c", Start: today.Add(3 * time.Hour), Duration: 300, Type: "break"},
		{Task: "d", Start: yesterday, Duration: 999, Type: "work"},
	}

	if got := GetDailyTotal(entryList, "work"); got != 720 {
		t.Fatalf("work total got %d, want %d", got, 720)
	}
	if got := GetDailyTotal(entryList, "break"); got != 300 {
		t.Fatalf("break total got %d, want %d", got, 300)
	}
}

func TestGetWeeklyData(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	twoDaysAgo := today.AddDate(0, 0, -2)
	eightDaysAgo := today.AddDate(0, 0, -8)

	entryList := []entries.Entry{
		{Task: "today", Start: today, Duration: 600, Type: "work"},
		{Task: "today-break", Start: today.Add(1 * time.Hour), Duration: 300, Type: "break"},
		{Task: "two-days", Start: twoDaysAgo, Duration: 120, Type: "work"},
		{Task: "old", Start: eightDaysAgo, Duration: 999, Type: "work"},
	}

	weekly := GetWeeklyData(entryList)
	if len(weekly) != 7 {
		t.Fatalf("weekly data size got %d, want 7", len(weekly))
	}

	if got := weekly[streak.DateKey(today)]; got != 600 {
		t.Fatalf("today total got %d, want %d", got, 600)
	}
	if got := weekly[streak.DateKey(twoDaysAgo)]; got != 120 {
		t.Fatalf("two-days-ago total got %d, want %d", got, 120)
	}

	if _, ok := weekly[streak.DateKey(eightDaysAgo)]; ok {
		t.Fatalf("expected date %s to be out of range", streak.DateKey(eightDaysAgo))
	}
}

func TestBuildDailyReport(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	entriesList := []entries.Entry{
		{Task: "Deep work", Start: today, End: today.Add(45 * time.Minute), Duration: 2700, Type: "work", Tags: []string{"writing"}},
		{Task: "Reset", Start: today.Add(1 * time.Hour), End: today.Add(75 * time.Minute), Duration: 900, Type: "break"},
	}

	lines := BuildDailyReport(entriesList, today)
	report := strings.Join(lines, "\n")
	if !strings.Contains(report, "Kairu Daily Report") {
		t.Fatalf("expected report title, got %q", report)
	}
	if !strings.Contains(report, "Work: 45m") || !strings.Contains(report, "Break: 15m") {
		t.Fatalf("expected totals in report, got %q", report)
	}
	if !strings.Contains(report, "Deep work") || !strings.Contains(report, "Reset") {
		t.Fatalf("expected entries in report, got %q", report)
	}
}

