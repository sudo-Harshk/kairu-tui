package analytics

import (
	"strings"
	"testing"
	"time"

	"kairu-tui/internal/entries"
)

func TestBuildWeeklyReport(t *testing.T) {
	t.Parallel()

	// Monday Jun 1, 2026
	loc := time.Local
	refTime := time.Date(2026, 6, 3, 15, 0, 0, 0, loc) // Wednesday Jun 3, 2026

	entryList := []entries.Entry{
		// Monday: 2 sessions of Task A (1h each)
		{Task: "Task A", Start: time.Date(2026, 6, 1, 9, 0, 0, 0, loc), End: time.Date(2026, 6, 1, 10, 0, 0, 0, loc), Duration: 3600, Type: "work"},
		{Task: "Task A", Start: time.Date(2026, 6, 1, 14, 0, 0, 0, loc), End: time.Date(2026, 6, 1, 15, 0, 0, 0, loc), Duration: 3600, Type: "work"},

		// Tuesday: 1 session of Task B (2h)
		{Task: "Task B", Start: time.Date(2026, 6, 2, 10, 0, 0, 0, loc), End: time.Date(2026, 6, 2, 12, 0, 0, 0, loc), Duration: 7200, Type: "work"},

		// Wednesday: 3 sessions of Task A (30m each)
		{Task: "Task A", Start: time.Date(2026, 6, 3, 10, 0, 0, 0, loc), End: time.Date(2026, 6, 3, 10, 30, 0, 0, loc), Duration: 1800, Type: "work"},
		{Task: "Task A", Start: time.Date(2026, 6, 3, 10, 35, 0, 0, loc), End: time.Date(2026, 6, 3, 11, 05, 0, 0, loc), Duration: 1800, Type: "work"},
		{Task: "Task A", Start: time.Date(2026, 6, 3, 16, 0, 0, 0, loc), End: time.Date(2026, 6, 3, 16, 30, 0, 0, loc), Duration: 1800, Type: "work"},

		// Thursday: 4 sessions of Task C (1h each)
		{Task: "Task C", Start: time.Date(2026, 6, 4, 10, 0, 0, 0, loc), End: time.Date(2026, 6, 4, 11, 0, 0, 0, loc), Duration: 3600, Type: "work"},
		{Task: "Task C", Start: time.Date(2026, 6, 4, 11, 10, 0, 0, loc), End: time.Date(2026, 6, 4, 12, 10, 0, 0, loc), Duration: 3600, Type: "work"},
		{Task: "Task C", Start: time.Date(2026, 6, 4, 13, 0, 0, 0, loc), End: time.Date(2026, 6, 4, 14, 0, 0, 0, loc), Duration: 3600, Type: "work"},
		{Task: "Task C", Start: time.Date(2026, 6, 4, 15, 0, 0, 0, loc), End: time.Date(2026, 6, 4, 16, 0, 0, 0, loc), Duration: 3600, Type: "work"},

		// Saturday: 1 session of Task B (1h)
		{Task: "Task B", Start: time.Date(2026, 6, 6, 9, 0, 0, 0, loc), End: time.Date(2026, 6, 6, 10, 0, 0, 0, loc), Duration: 3600, Type: "work"},

		// Out of week entry (prior week)
		{Task: "Old Task", Start: time.Date(2026, 5, 25, 10, 0, 0, 0, loc), End: time.Date(2026, 5, 25, 11, 0, 0, 0, loc), Duration: 3600, Type: "work"},
	}

	report := BuildWeeklyReportAt(entryList, refTime)

	// Validate week boundaries
	expectedStart := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	if !report.WeekStart.Equal(expectedStart) {
		t.Errorf("WeekStart got %v, want %v", report.WeekStart, expectedStart)
	}

	// Validate Total sessions and seconds
	// Total seconds should be: 7200 (Mon) + 7200 (Tue) + 5400 (Wed) + 14400 (Thu) + 3600 (Sat) = 37800
	if report.TotalSessions != 11 {
		t.Errorf("TotalSessions got %d, want %d", report.TotalSessions, 11)
	}
	if report.TotalSeconds != 37800 {
		t.Errorf("TotalSeconds got %d, want %d", report.TotalSeconds, 37800)
	}

	// Validate DailyBreakdown
	// Mon: index 0 (7200)
	// Tue: index 1 (7200)
	// Wed: index 2 (5400)
	// Thu: index 3 (14400)
	// Fri: index 4 (0)
	// Sat: index 5 (3600)
	// Sun: index 6 (0)
	expectedBreakdown := []struct {
		day      string
		seconds  int
		sessions int
	}{
		{"Mon", 7200, 2},
		{"Tue", 7200, 1},
		{"Wed", 5400, 3},
		{"Thu", 14400, 4},
		{"Fri", 0, 0},
		{"Sat", 3600, 1},
		{"Sun", 0, 0},
	}

	for i, expected := range expectedBreakdown {
		got := report.DailyBreakdown[i]
		if got.Day != expected.day {
			t.Errorf("Index %d day got %s, want %s", i, got.Day, expected.day)
		}
		if got.Seconds != expected.seconds {
			t.Errorf("Index %d seconds got %d, want %d", i, got.Seconds, expected.seconds)
		}
		if got.Sessions != expected.sessions {
			t.Errorf("Index %d sessions got %d, want %d", i, got.Sessions, expected.sessions)
		}
	}

	// Best Day should be Thursday (14400 seconds)
	if report.BestDay != "Thursday" {
		t.Errorf("BestDay got %s, want Thursday", report.BestDay)
	}

	// Best Hour should be 10 (10 AM) as we had:
	// - 10:00 (Tue)
	// - 10:00, 10:35 (Wed)
	// - 10:00 (Thu)
	// (4 sessions total started in the 10:00-11:00 hour)
	if report.BestHour != 10 {
		t.Errorf("BestHour got %d, want 10", report.BestHour)
	}

	// Validate TopTasks sorting
	// Task C: 14400
	// Task A: 3600+3600+1800+1800+1800 = 12600
	// Task B: 7200+3600 = 10800
	if len(report.TopTasks) != 3 {
		t.Fatalf("TopTasks len got %d, want 3", len(report.TopTasks))
	}
	if report.TopTasks[0].Name != "Task C" || report.TopTasks[0].Seconds != 14400 {
		t.Errorf("First top task got %s (%d), want Task C (14400)", report.TopTasks[0].Name, report.TopTasks[0].Seconds)
	}
	if report.TopTasks[1].Name != "Task A" || report.TopTasks[1].Seconds != 12600 {
		t.Errorf("Second top task got %s (%d), want Task A (12600)", report.TopTasks[1].Name, report.TopTasks[1].Seconds)
	}
	if report.TopTasks[2].Name != "Task B" || report.TopTasks[2].Seconds != 10800 {
		t.Errorf("Third top task got %s (%d), want Task B (10800)", report.TopTasks[2].Name, report.TopTasks[2].Seconds)
	}

	// Render report and ensure some fields are visible
	rendered := RenderReport(report)
	if !strings.Contains(rendered, "Kairu Focus Report") {
		t.Errorf("Rendered report does not contain header")
	}
	if !strings.Contains(rendered, "Week of Jun 1 – Jun 7, 2026") {
		t.Errorf("Rendered report does not contain week label, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Total Focus: 10h 30m") { // 37800s = 10.5 hours = 10h 30m
		t.Errorf("Rendered report does not contain total focus duration, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Best Day: Thursday (4h 0m)") { // 14400s = 4h
		t.Errorf("Rendered report does not contain Best Day, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Best Hour: 10 AM - 11 AM") {
		t.Errorf("Rendered report does not contain Best Hour, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Task C") || !strings.Contains(rendered, "Task A") || !strings.Contains(rendered, "Task B") {
		t.Errorf("Rendered report missing top tasks, got: %s", rendered)
	}
}
