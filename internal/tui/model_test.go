package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/notification"
	"kairu-tui/internal/templates"
	"kairu-tui/internal/ui"
)

func TestRenderStreakHistoryChart(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
	entriesList := []entries.Entry{
		{Task: "a", Start: today, End: today.Add(30 * time.Minute), Duration: 1800, Type: "work"},
	}

	got := renderStreakHistoryChart(entriesList)
	if !strings.Contains(got, "work logged") {
		t.Fatalf("expected chart to show work logged, got %q", got)
	}
}

func TestRenderHistoryView(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)
	m := model{
		entries: []entries.Entry{
			{Task: "Review PR", Start: today.Add(2 * time.Hour), End: today.Add(2*time.Hour + 10*time.Minute), Duration: 600, Type: "break"},
			{Task: "Write docs", Start: today, End: today.Add(25 * time.Minute), Duration: 1500, Type: "work", Note: "draft", Tags: []string{"docs"}},
			{Task: "Plan sprint", Start: yesterday, End: yesterday.Add(40 * time.Minute), Duration: 2400, Type: "work"},
		},
	}

	got := renderHistoryView(m)
	if !strings.Contains(got, "📜 Focus History") {
		t.Fatalf("expected timeline header, got %q", got)
	}
	first := strings.Index(got, "Review PR")
	second := strings.Index(got, "Write docs")
	third := strings.Index(got, "Plan sprint")
	if first == -1 || second == -1 || third == -1 {
		t.Fatalf("expected all entries in view, got %q", got)
	}
	if first > second {
		t.Fatalf("expected newest entry first, got %q", got)
	}
	if !strings.Contains(got, today.Format("Mon, Jan 02, 2006")) || !strings.Contains(got, yesterday.Format("Mon, Jan 02, 2006")) {
		t.Fatalf("expected day headers, got %q", got)
	}
	if !strings.Contains(got, "note: draft") || !strings.Contains(got, "tags: docs") {
		t.Fatalf("expected note and tag details, got %q", got)
	}
}

func TestRenderHistoryViewWithFilters(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	si := textinput.New()
	si.SetValue("Write")

	m := model{
		entries: []entries.Entry{
			{Task: "Review PR", Start: today.Add(2 * time.Hour), End: today.Add(2*time.Hour + 10*time.Minute), Duration: 600, Type: "break"},
			{Task: "Write docs", Start: today, End: today.Add(25 * time.Minute), Duration: 1500, Type: "work", Note: "draft", Tags: []string{"docs"}},
			{Task: "Plan sprint", Start: yesterday, End: yesterday.Add(40 * time.Minute), Duration: 2400, Type: "work"},
		},
		historyFilter: historyFilterState{
			searchInput:   si,
			typeFilter:    "work",
			dateRange:     "today",
			searchFocused: false,
		},
	}

	got := renderHistoryView(m)
	if !strings.Contains(got, "📜 Focus History") {
		t.Fatalf("expected timeline header, got %q", got)
	}

	// Should contain "Write docs" but NOT "Plan sprint" (yesterday) and NOT "Review PR" (type break)
	if !strings.Contains(got, "Write docs") {
		t.Error("expected 'Write docs' in filtered history view")
	}
	if strings.Contains(got, "Plan sprint") {
		t.Error("did not expect 'Plan sprint' in filtered history view")
	}
	if strings.Contains(got, "Review PR") {
		t.Error("did not expect 'Review PR' in filtered history view")
	}
}


func TestRenderAnalyticsView(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, loc)
	entriesList := []entries.Entry{
		{Task: "Deep work", Start: today, End: today.Add(45 * time.Minute), Duration: 2700, Type: "work", Tags: []string{"writing", "focus"}},
		{Task: "Reset", Start: today.Add(1 * time.Hour), End: today.Add(75 * time.Minute), Duration: 900, Type: "break"},
		{Task: "Deep work", Start: today.Add(2 * time.Hour), End: today.Add(2*time.Hour + 30*time.Minute), Duration: 1800, Type: "work", Tags: []string{"writing"}},
	}

	got := renderAnalyticsView(model{entries: entriesList})
	for _, want := range []string{"Analytics", "Sessions analyzed: 3", "Top tasks:", "Top tags:", "Deep work", "writing"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected analytics view to contain %q, got %q", want, got)
		}
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
		notificationOutbox: []notification.NotificationJob{{ID: "queued"}},
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

func TestReturnModeForModal(t *testing.T) {
	t.Parallel()

	m := model{mode: "help", helpReturnMode: "settings", settingsReturnMode: "break"}
	if got := m.returnModeForModal(); got != "break" {
		t.Fatalf("expected break return mode, got %q", got)
	}
}

func TestSettingsEscRestoresBreakMode(t *testing.T) {
	t.Parallel()

	m := model{
		mode:               "settings",
		settingsReturnMode: "break",
	}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := got.(model)
	if updated.mode != "break" {
		t.Fatalf("expected settings esc to restore break mode, got %q", updated.mode)
	}
}

func TestSettingsTabCanOpenStats(t *testing.T) {
	t.Parallel()

	m := model{
		mode:               "settings",
		settingsCursor:     settingsCount - 1,
		settingsReturnMode: "timer",
	}

	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	updated := got.(model)
	if updated.mode != "stats" {
		t.Fatalf("expected tab from last settings item to open stats, got %q", updated.mode)
	}
	if updated.statsReturnMode != "timer" {
		t.Fatalf("expected stats return mode timer, got %q", updated.statsReturnMode)
	}
}

func TestSaveOnQuitFromHelpSavesActiveBreakSession(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dataFile := filepath.Join(dir, "entries.json")
	start := time.Now().Add(-10 * time.Minute)

	m := model{
		mode:           "help",
		helpReturnMode: "break",
		seconds:        120,
		taskName:       "Focus task",
		sessionStart:   start,
		sessionElapsed: 600,
		dataFile:       dataFile,
		noteInput:      textinput.New(),
		tagInput:       textinput.New(),
	}

	m.saveOnQuit()

	entriesList, err := entries.LoadEntries(dataFile)
	if err != nil {
		t.Fatalf("loadEntries failed: %v", err)
	}
	if len(entriesList) != 1 {
		t.Fatalf("expected 1 saved entry, got %d", len(entriesList))
	}
	if entriesList[0].Type != "break" {
		t.Fatalf("expected break entry, got %q", entriesList[0].Type)
	}
	if entriesList[0].Task != "Focus task" {
		t.Fatalf("unexpected task saved: %q", entriesList[0].Task)
	}
}

func TestTickContinuesWhileSettingsOpen(t *testing.T) {
	t.Parallel()

	m := model{
		mode:               "settings",
		settingsReturnMode: "timer",
		running:            true,
		seconds:            10,
		sessionTarget:      10,
		sessionElapsed:     0,
		taskName:           "Focus task",
	}

	got, cmd := m.Update(tickTockMsg(time.Now()))
	updated := got.(model)
	if updated.seconds != 9 {
		t.Fatalf("expected seconds to continue ticking in settings, got %d", updated.seconds)
	}
	if cmd == nil {
		t.Fatalf("expected background tick to be scheduled while settings are open")
	}
}

func TestApplySelectedTemplate(t *testing.T) {
	t.Parallel()

	m := model{
		templates: []templates.SessionTemplate{
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
		templates: []templates.SessionTemplate{
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

func TestRenameSelectedTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	m := model{
		textInput:     textInputWithValue("Renamed"),
		durationInput: textInputWithValue("30"),
		noteInput:     textInputWithValue("Updated note"),
		tagInput:      textInputWithValue("focus, writing"),
		templateFile:  path,
		templates: []templates.SessionTemplate{
			{Name: "Old", Task: "Old", Duration: "15"},
		},
		templateIndex: 0,
	}

	if err := m.renameSelectedTemplate(); err != nil {
		t.Fatalf("renameSelectedTemplate failed: %v", err)
	}
	if got := m.templates[0].Name; got != "Renamed" {
		t.Fatalf("template name got %q, want %q", got, "Renamed")
	}
	if got := m.templates[0].Duration; got != "30" {
		t.Fatalf("template duration got %q, want %q", got, "30")
	}
}

func TestDeleteSelectedTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	m := model{
		templateFile: path,
		templates: []templates.SessionTemplate{
			{Name: "One", Task: "One", Duration: "15"},
			{Name: "Two", Task: "Two", Duration: "30"},
		},
		templateIndex: 0,
	}

	if err := m.deleteSelectedTemplate(); err != nil {
		t.Fatalf("deleteSelectedTemplate failed: %v", err)
	}
	if len(m.templates) != 1 {
		t.Fatalf("expected 1 template after delete, got %d", len(m.templates))
	}
	if got := m.templates[0].Name; got != "Two" {
		t.Fatalf("remaining template got %q, want %q", got, "Two")
	}
}

func TestUndoLastTemplateDelete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	m := model{
		templateFile: path,
		templates: []templates.SessionTemplate{
			{Name: "One", Task: "One", Duration: "25"},
			{Name: "Two", Task: "Two", Duration: "30"},
			{Name: "Three", Task: "Three", Duration: "15"},
		},
		templateIndex: 1,
	}

	if err := m.deleteSelectedTemplate(); err != nil {
		t.Fatalf("deleteSelectedTemplate failed: %v", err)
	}
	if err := m.undoLastTemplateDelete(); err != nil {
		t.Fatalf("undoLastTemplateDelete failed: %v", err)
	}

	if len(m.templates) != 3 {
		t.Fatalf("expected 3 templates after undo, got %d", len(m.templates))
	}
	if got := m.templates[1].Name; got != "Two" {
		t.Fatalf("restored template got %q, want %q", got, "Two")
	}
	if m.templateIndex != 1 {
		t.Fatalf("expected restored template index 1, got %d", m.templateIndex)
	}
	if m.lastDeletedTemplate != nil {
		t.Fatalf("expected undo buffer to be cleared")
	}
}

func TestDuplicateSelectedTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "templates.json")
	m := model{
		templateFile: path,
		templates: []templates.SessionTemplate{
			{Name: "One", Task: "One", Duration: "15"},
		},
		templateIndex: 0,
	}

	if err := m.duplicateSelectedTemplate(); err != nil {
		t.Fatalf("duplicateSelectedTemplate failed: %v", err)
	}
	if len(m.templates) != 2 {
		t.Fatalf("expected 2 templates after duplicate, got %d", len(m.templates))
	}
	if got := m.templates[0].Name; got != "One Copy" {
		t.Fatalf("duplicated template got %q, want %q", got, "One Copy")
	}
}

func TestRenderActivityHeatmap(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entriesList := []entries.Entry{
		{Task: "Focus", Start: now, Duration: 3600, Type: "work"},
		{Task: "Old", Start: now.AddDate(0, 0, -10), Duration: 7200, Type: "work"},
	}

	got := renderActivityHeatmap(entriesList, config.DefaultConfig, 80)
	if !strings.Contains(got, "Sun") || !strings.Contains(got, "Sat") {
		t.Fatalf("expected day labels in heatmap, got %q", got)
	}
}

func textInputWithValue(value string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	return ti
}

func TestVisualAnalyticsRendering(t *testing.T) {
	t.Parallel()

	theme := config.ThemeStyle{
		Primary: "2",
		Accent:  "10",
		Notice:  "3",
		Warning: "1",
	}

	t.Run("ProgressBar", func(t *testing.T) {
		got := renderHorizontalProgressBar(50.0, 10, theme, false)
		if !strings.Contains(got, "█████") || !strings.Contains(got, "░░░░░") {
			t.Fatalf("unexpected progress bar: %q", got)
		}
	})

	t.Run("DashboardCard", func(t *testing.T) {
		got := ui.Panel("Card Title", "Card Content", theme, 30, lipgloss.RoundedBorder(), theme.Primary)
		if !strings.Contains(got, "Card Title") || !strings.Contains(got, "Card Content") {
			t.Fatalf("expected card title and content: %q", got)
		}
	})

	t.Run("TopDurationBars", func(t *testing.T) {
		totals := map[string]int{
			"coding":  3600,
			"writing": 1800,
		}
		got := renderTopDurationBars(totals, 5400, 2, theme, true, 10)
		if len(got) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(got))
		}
		if !strings.Contains(got[0], "coding") || !strings.Contains(got[0], "(66.7%)") {
			t.Fatalf("unexpected top bar: %q", got[0])
		}
		if !strings.Contains(got[1], "writing") || !strings.Contains(got[1], "(33.3%)") {
			t.Fatalf("unexpected second bar: %q", got[1])
		}
	})
}
