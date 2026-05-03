package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestParseDurationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		wantSeconds int
		wantErr     bool
	}{
		{name: "minutes", input: "25", wantSeconds: 25 * 60},
		{name: "hoursMinutes", input: "1:00", wantSeconds: 60 * 60},
		{name: "zeroHoursMinutes", input: "0:30", wantSeconds: 30 * 60},
		{name: "trimmed", input: "  5  ", wantSeconds: 5 * 60},
		{name: "empty", input: "", wantErr: true},
		{name: "zeroMinutes", input: "0", wantErr: true},
		{name: "negativeMinutes", input: "-5", wantErr: true},
		{name: "invalidMinutes", input: "1:60", wantErr: true},
		{name: "invalidFormat", input: "1:2:3", wantErr: true},
		{name: "notNumber", input: "abc", wantErr: true},
		{name: "negativePart", input: "1:-1", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDurationInput(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (input=%q)", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error (input=%q): %v", tc.input, err)
			}
			if got != tc.wantSeconds {
				t.Fatalf("got %d seconds, want %d (input=%q)", got, tc.wantSeconds, tc.input)
			}
		})
	}
}

func TestFormatDurationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seconds int
		want    string
	}{
		{seconds: 0, want: "0"},
		{seconds: -10, want: "0"},
		{seconds: 60, want: "1"},
		{seconds: 600, want: "10"},
		{seconds: 3600, want: "1:00"},
		{seconds: 3660, want: "1:01"},
		{seconds: 7320, want: "2:02"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			got := formatDurationInput(tc.seconds)
			if got != tc.want {
				t.Fatalf("got %q, want %q (seconds=%d)", got, tc.want, tc.seconds)
			}
		})
	}
}

func TestGetDailyTotal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	entries := []Entry{
		{Task: "a", Start: today, Duration: 600, Type: "work"},
		{Task: "b", Start: today.Add(2 * time.Hour), Duration: 120, Type: "work"},
		{Task: "c", Start: today.Add(3 * time.Hour), Duration: 300, Type: "break"},
		{Task: "d", Start: yesterday, Duration: 999, Type: "work"},
	}

	if got := getDailyTotal(entries, "work"); got != 720 {
		t.Fatalf("work total got %d, want %d", got, 720)
	}
	if got := getDailyTotal(entries, "break"); got != 300 {
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

	entries := []Entry{
		{Task: "today", Start: today, Duration: 600, Type: "work"},
		{Task: "today-break", Start: today.Add(1 * time.Hour), Duration: 300, Type: "break"},
		{Task: "two-days", Start: twoDaysAgo, Duration: 120, Type: "work"},
		{Task: "old", Start: eightDaysAgo, Duration: 999, Type: "work"},
	}

	weekly := getWeeklyData(entries)
	if len(weekly) != 7 {
		t.Fatalf("weekly data size got %d, want 7", len(weekly))
	}

	if got := weekly[dateKey(today)]; got != 600 {
		t.Fatalf("today total got %d, want %d", got, 600)
	}
	if got := weekly[dateKey(twoDaysAgo)]; got != 120 {
		t.Fatalf("two-days-ago total got %d, want %d", got, 120)
	}

	if _, ok := weekly[dateKey(eightDaysAgo)]; ok {
		t.Fatalf("expected date %s to be out of range", dateKey(eightDaysAgo))
	}
}

func TestCalculateStreaks(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)

	makeEntry := func(date time.Time, sessionType string) Entry {
		return Entry{Task: "t", Start: date, End: date.Add(30 * time.Minute), Duration: 1800, Type: sessionType}
	}

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		current, longest := calculateStreaks(nil)
		if current != 0 || longest != 0 {
			t.Fatalf("got current=%d longest=%d, want 0,0", current, longest)
		}
	})

	t.Run("consecutiveIncludingToday", func(t *testing.T) {
		t.Parallel()

		entries := []Entry{
			makeEntry(today.AddDate(0, 0, -2), "work"),
			makeEntry(today.AddDate(0, 0, -1), "work"),
			makeEntry(today, "work"),
			makeEntry(today, "break"),
		}
		current, longest := calculateStreaks(entries)
		if current != 3 || longest != 3 {
			t.Fatalf("got current=%d longest=%d, want 3,3", current, longest)
		}
	})

	t.Run("gapBeforeToday", func(t *testing.T) {
		t.Parallel()

		entries := []Entry{
			makeEntry(today.AddDate(0, 0, -2), "work"),
			makeEntry(today, "work"),
		}
		current, longest := calculateStreaks(entries)
		if current != 1 || longest != 1 {
			t.Fatalf("got current=%d longest=%d, want 1,1", current, longest)
		}
	})

	t.Run("longestWithoutToday", func(t *testing.T) {
		t.Parallel()

		entries := []Entry{
			makeEntry(today.AddDate(0, 0, -5), "work"),
			makeEntry(today.AddDate(0, 0, -4), "work"),
			makeEntry(today.AddDate(0, 0, -3), "work"),
		}
		current, longest := calculateStreaks(entries)
		if current != 0 || longest != 3 {
			t.Fatalf("got current=%d longest=%d, want 0,3", current, longest)
		}
	})
}

func TestDateKeyUsesLocal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	got := dateKey(now)
	want := now.In(time.Local).Format("2006-01-02")
	if !strings.EqualFold(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateEntries(t *testing.T) {
	t.Parallel()

	now := time.Now()
	valid := []Entry{
		{Task: "focus", Start: now, End: now, Duration: 0, Type: "work"},
		{Task: "break", Start: now, End: now.Add(2 * time.Minute), Duration: 120, Type: "break"},
	}
	if err := validateEntries(valid); err != nil {
		t.Fatalf("expected valid entries, got error: %v", err)
	}

	cases := []struct {
		name   string
		entry  Entry
		hasErr bool
	}{
		{name: "missingTask", entry: Entry{Task: " ", Start: now, End: now, Duration: 10, Type: "work"}, hasErr: true},
		{name: "invalidType", entry: Entry{Task: "x", Start: now, End: now, Duration: 10, Type: "other"}, hasErr: true},
		{name: "missingStart", entry: Entry{Task: "x", End: now, Duration: 10, Type: "work"}, hasErr: true},
		{name: "missingEnd", entry: Entry{Task: "x", Start: now, Duration: 10, Type: "work"}, hasErr: true},
		{name: "endBeforeStart", entry: Entry{Task: "x", Start: now, End: now.Add(-time.Minute), Duration: 10, Type: "work"}, hasErr: true},
		{name: "negativeDuration", entry: Entry{Task: "x", Start: now, End: now, Duration: -1, Type: "work"}, hasErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := validateEntries([]Entry{tc.entry}); (err != nil) != tc.hasErr {
				t.Fatalf("expected error=%t, got %v", tc.hasErr, err)
			}
		})
	}
}

func TestMergeEntriesDedup(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 6, 1, 9, 0, 0, 0, time.Local)
	entryA := Entry{Task: "a", Start: start, End: start.Add(25 * time.Minute), Duration: 1500, Type: "work"}
	entryB := Entry{Task: "b", Start: start.Add(2 * time.Hour), End: start.Add(2*time.Hour + 5*time.Minute), Duration: 300, Type: "break"}
	entryC := Entry{Task: "c", Start: start.Add(3 * time.Hour), End: start.Add(3*time.Hour + 10*time.Minute), Duration: 600, Type: "work"}

	existing := []Entry{entryA, entryB}
	incoming := []Entry{entryA, entryC}
	merged := mergeEntries(existing, incoming)

	if len(merged) != 3 {
		t.Fatalf("expected 3 merged entries, got %d", len(merged))
	}

	seen := map[string]bool{}
	for _, entry := range merged {
		seen[entryKey(entry)] = true
	}
	if !seen[entryKey(entryA)] || !seen[entryKey(entryB)] || !seen[entryKey(entryC)] {
		t.Fatalf("merged entries missing expected items")
	}
}

func TestNotificationIDDedup(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 6, 1, 9, 0, 0, 0, time.Local)
	m := model{
		sessionStart:   start,
		sessionElapsed: 120,
		taskName:       "Focus",
		running:        true,
	}

	first := m.notificationID("session_start")
	second := m.notificationID("session_start")
	if first != second {
		t.Fatalf("expected stable notification ID, got %q and %q", first, second)
	}

	pause := m.notificationID("pause_resume")
	m.running = false
	resume := m.notificationID("pause_resume")
	if pause == resume {
		t.Fatalf("expected pause/resume IDs to differ")
	}
}

func TestHasNotification(t *testing.T) {
	t.Parallel()

	m := model{
		deliveredNotifyIDs: map[string]time.Time{"delivered": time.Now()},
		notificationOutbox: []notificationJob{{ID: "queued"}},
	}

	if !m.hasNotification("delivered") {
		t.Fatalf("expected delivered notification to be found")
	}
	if !m.hasNotification("queued") {
		t.Fatalf("expected queued notification to be found")
	}
	if m.hasNotification("missing") {
		t.Fatalf("did not expect missing notification to be found")
	}
}

func TestActiveSessionMode(t *testing.T) {
	t.Parallel()

	m := model{mode: "settings", settingsReturnMode: "break"}
	if got := m.activeSessionMode(); got != "break" {
		t.Fatalf("expected break from settings, got %q", got)
	}

	m = model{
		mode:               "help",
		helpReturnMode:     "settings",
		settingsReturnMode: "break",
	}
	if got := m.activeSessionMode(); got != "break" {
		t.Fatalf("expected break via help->settings, got %q", got)
	}
}

func TestLoadAndSaveSessionTemplates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")

	templates := []SessionTemplate{
		{Name: "Deep Work", Task: "Deep Work", Duration: "25", Note: "Focus block", Tags: []string{"deep work", "writing"}},
	}
	if err := saveSessionTemplates(path, templates); err != nil {
		t.Fatalf("saveSessionTemplates failed: %v", err)
	}

	got, err := loadSessionTemplates(path)
	if err != nil {
		t.Fatalf("loadSessionTemplates failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 template, got %d", len(got))
	}
	if got[0].Name != "Deep Work" || got[0].Duration != "25" {
		t.Fatalf("unexpected template contents: %+v", got[0])
	}
}

func TestApplySelectedTemplate(t *testing.T) {
	t.Parallel()

	m := model{
		templates: []SessionTemplate{
			{Name: "Writing", Task: "Write outline", Duration: "45", Note: "Blog draft", Tags: []string{"writing", "deep work"}},
		},
		templateIndex: 0,
	}

	m = m.applySelectedTemplate()
	if got := m.textInput.Value(); got != "Write outline" {
		t.Fatalf("task got %q, want %q", got, "Write outline")
	}
	if got := m.durationInput.Value(); got != "45" {
		t.Fatalf("duration got %q, want %q", got, "45")
	}
	if got := m.noteInput.Value(); got != "Blog draft" {
		t.Fatalf("note got %q, want %q", got, "Blog draft")
	}
	if got := m.tagInput.Value(); got != "writing, deep work" {
		t.Fatalf("tags got %q, want %q", got, "writing, deep work")
	}
}

func TestSaveCurrentTemplateUpserts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	m := model{
		textInput:     textInputWithValue("Focus"),
		durationInput: textInputWithValue("25"),
		noteInput:     textInputWithValue("First pass"),
		tagInput:      textInputWithValue("deep work, writing"),
		templateFile:  path,
		templates: []SessionTemplate{
			{Name: "Focus", Task: "Old", Duration: "10"},
		},
	}

	if err := m.saveCurrentTemplate(); err != nil {
		t.Fatalf("saveCurrentTemplate failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read template file: %v", err)
	}
	if !strings.Contains(string(data), "First pass") {
		t.Fatalf("expected saved template data to contain note, got %s", string(data))
	}
	if len(m.templates) != 1 || m.templates[0].Duration != "25" {
		t.Fatalf("unexpected in-memory templates: %+v", m.templates)
	}
}

func textInputWithValue(value string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	return ti
}
