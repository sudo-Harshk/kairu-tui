package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Configuration (loaded from YAML/env with defaults)
type Config struct {
	WorkDuration         int    `yaml:"work_duration"`
	BreakDuration        int    `yaml:"break_duration"`
	Font                 string `yaml:"font"`
	Theme                string `yaml:"theme"`
	Notifications        bool   `yaml:"notifications"`
	DesktopNotifications bool   `yaml:"desktop_notifications"`
	NotifyWorkComplete   bool   `yaml:"notify_work_complete"`
	NotifyBreakComplete  bool   `yaml:"notify_break_complete"`
	NotifySessionStart   bool   `yaml:"notify_session_start"`
	NotifySessionEnd     bool   `yaml:"notify_session_end"`
	NotifyPauseResume    bool   `yaml:"notify_pause_resume"`
	NotifyEndingSoon     bool   `yaml:"notify_ending_soon"`
	QuietHoursStart      int    `yaml:"quiet_hours_start"`
	QuietHoursEnd        int    `yaml:"quiet_hours_end"`
	SoundCommand         string `yaml:"sound_command"`
	AutoBreak            bool   `yaml:"auto_break"`
	SessionsBeforeBreak  int    `yaml:"sessions_before_break"`
	TelegramBotToken     string `yaml:"-"`
	TelegramChatID       string `yaml:"-"`
}

var defaultConfig = Config{
	WorkDuration:         25,
	BreakDuration:        5,
	Font:                 "ansi",
	Theme:                "forest",
	Notifications:        false,
	DesktopNotifications: true,
	NotifyWorkComplete:   true,
	NotifyBreakComplete:  true,
	NotifySessionStart:   false,
	NotifySessionEnd:     false,
	NotifyPauseResume:    false,
	NotifyEndingSoon:     false,
	QuietHoursStart:      -1,
	QuietHoursEnd:        -1,
	SoundCommand:         "",
	AutoBreak:            false,
	SessionsBeforeBreak:  4,
	TelegramBotToken:     "",
	TelegramChatID:       "",
}

const (
	envTelegramBotToken = "KAIRU_TELEGRAM_BOT_TOKEN"
	envTelegramChatID   = "KAIRU_TELEGRAM_CHAT_ID"
)

func loadEnvFile(path string) error {
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			applyEnvOverrides(&cfg)
			return cfg, nil
		}
		applyEnvOverrides(&cfg)
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		applyEnvOverrides(&cfg)
		return cfg, err
	}
	applyEnvOverrides(&cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if val := strings.TrimSpace(os.Getenv(envTelegramBotToken)); val != "" {
		cfg.TelegramBotToken = val
	}

	if val := strings.TrimSpace(os.Getenv(envTelegramChatID)); val != "" {
		cfg.TelegramChatID = val
	}
}

type model struct {
	seconds            int
	sessionTarget      int
	sessionElapsed     int
	width              int
	running            bool
	mode               string
	editReturnMode     string
	editWasRunning     bool
	helpReturnMode     string
	helpWasRunning     bool
	templateReturnMode string
	templateWasRunning bool
	settingsReturnMode string
	statsReturnMode    string
	textInput          textinput.Model
	durationInput      textinput.Model
	noteInput          textinput.Model
	tagInput           textinput.Model
	templateIndex      int
	focusedField       int
	inputError         string
	appError           string
	notificationStatus string
	taskName           string
	recentTasks        []string
	recentTaskIndex    int
	settingsCursor     int
	entries            []Entry
	templates          []SessionTemplate
	dataFile           string
	templateFile       string
	configFile         string
	config             Config
	sessionStart       time.Time
	sessionCount       int
	totalWorkTime      int
	totalBreakTime     int
	notificationOutbox []notificationJob
	deliveredNotifyIDs map[string]time.Time
	outboxFile         string
}

type notificationJob struct {
	ID            string    `json:"id"`
	Event         string    `json:"event"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	CreatedAt     time.Time `json:"created_at"`
	Attempts      int       `json:"attempts"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	LastError     string    `json:"last_error,omitempty"`
}

type Entry struct {
	Task     string    `json:"task"`
	Note     string    `json:"note,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration int       `json:"duration_seconds"`
	Type     string    `json:"type"`
}

type SessionTemplate struct {
	Name     string   `json:"name"`
	Task     string   `json:"task"`
	Duration string   `json:"duration"`
	Note     string   `json:"note,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

func loadSessionTemplates(path string) ([]SessionTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []SessionTemplate{}, nil
		}
		return nil, err
	}
	var templates []SessionTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

func saveSessionTemplates(path string, templates []SessionTemplate) error {
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func loadEntries(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if err := validateEntries(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func validateEntries(entries []Entry) error {
	for i, entry := range entries {
		if strings.TrimSpace(entry.Task) == "" {
			return fmt.Errorf("entry %d: task is required", i)
		}
		if entry.Type != "work" && entry.Type != "break" {
			return fmt.Errorf("entry %d: type must be work or break", i)
		}
		if entry.Start.IsZero() || entry.End.IsZero() {
			return fmt.Errorf("entry %d: start and end must be set", i)
		}
		if entry.End.Before(entry.Start) {
			return fmt.Errorf("entry %d: end is before start", i)
		}
		if entry.Duration < 0 {
			return fmt.Errorf("entry %d: duration must be 0 or greater", i)
		}
	}
	return nil
}

func mergeEntries(existing, incoming []Entry) []Entry {
	seen := make(map[string]struct{}, len(existing))
	for _, entry := range existing {
		seen[entryKey(entry)] = struct{}{}
	}
	merged := append([]Entry{}, existing...)
	for _, entry := range incoming {
		key := entryKey(entry)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, entry)
	}
	return merged
}

func entryKey(entry Entry) string {
	return fmt.Sprintf("%s|%s|%s|%s",
		strings.TrimSpace(entry.Task),
		entry.Start.Format(time.RFC3339Nano),
		entry.End.Format(time.RFC3339Nano),
		entry.Type,
	)
}

func exportEntries(dataFile, exportPath string) error {
	entries, err := loadEntries(dataFile)
	if err != nil {
		return fmt.Errorf("failed to read entries: %w", err)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode entries: %w", err)
	}
	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	return nil
}

func importEntries(dataFile, importPath string) error {
	incomingData, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}
	var incoming []Entry
	if err := json.Unmarshal(incomingData, &incoming); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}
	if err := validateEntries(incoming); err != nil {
		return fmt.Errorf("import validation failed: %w", err)
	}
	existing, err := loadEntries(dataFile)
	if err != nil {
		return fmt.Errorf("failed to read existing entries: %w", err)
	}
	merged := mergeEntries(existing, incoming)
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode merged entries: %w", err)
	}
	if err := os.WriteFile(dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write entries: %w", err)
	}
	return nil
}

type tickTockMsg time.Time

type notifResultMsg struct {
	id     string
	status string
	err    error
}

type outboxFlushedMsg struct {
	remaining    []notificationJob
	deliveredIDs []string
	status       string
	err          error
}

const (
	focusTemplate = iota
	focusTask
	focusDuration
	focusNote
	focusTags
)

const (
	templateActionApply = iota
	templateActionSave
	templateActionRename
	templateActionDelete
	templateActionDuplicate
	templateActionCount
)

const (
	settingsNotifications = iota
	settingsDesktop
	settingsWorkComplete
	settingsBreakComplete
	settingsSessionStart
	settingsSessionEnd
	settingsPauseResume
	settingsEndingSoon
	settingsTheme
	settingsFont
	settingsQuietStart
	settingsQuietEnd
	settingsCount
)

type themeStyle struct {
	accent  string
	primary string
	notice  string
	warning string
}

var themeStyles = map[string]themeStyle{
	"forest": {accent: "10", primary: "2", notice: "3", warning: "1"},
	"ocean":  {accent: "14", primary: "6", notice: "12", warning: "9"},
	"ember":  {accent: "208", primary: "214", notice: "220", warning: "196"},
	"mono":   {accent: "15", primary: "7", notice: "8", warning: "9"},
}

var themeOrder = []string{"forest", "ocean", "ember", "mono"}

type timerFont struct {
	digits map[rune][]string
	label  string
}

var timerFonts = map[string]timerFont{
	"ansi": {
		label: "ANSI",
		digits: map[rune][]string{
			'0': {"███", "█ █", "█ █", "█ █", "███"},
			'1': {" █ ", "██ ", " █ ", " █ ", "███"},
			'2': {"███", "  █", "███", "█  ", "███"},
			'3': {"███", "  █", "███", "  █", "███"},
			'4': {"█ █", "█ █", "███", "  █", "  █"},
			'5': {"███", "█  ", "███", "  █", "███"},
			'6': {"███", "█  ", "███", "█ █", "███"},
			'7': {"███", "  █", "  █", "  █", "  █"},
			'8': {"███", "█ █", "███", "█ █", "███"},
			'9': {"███", "█ █", "███", "  █", "███"},
			':': {"   ", " █ ", "   ", " █ ", "   "},
		},
	},
	"block": {
		label: "Block",
		digits: map[rune][]string{
			'0': {"████", "█  █", "█  █", "█  █", "████"},
			'1': {" ██ ", "███ ", " ██ ", " ██ ", "████"},
			'2': {"████", "   █", "████", "█   ", "████"},
			'3': {"████", "   █", "████", "   █", "████"},
			'4': {"█  █", "█  █", "████", "   █", "   █"},
			'5': {"████", "█   ", "████", "   █", "████"},
			'6': {"████", "█   ", "████", "█  █", "████"},
			'7': {"████", "   █", "   █", "   █", "   █"},
			'8': {"████", "█  █", "████", "█  █", "████"},
			'9': {"████", "█  █", "████", "   █", "████"},
			':': {"    ", " ██ ", "    ", " ██ ", "    "},
		},
	},
	"thin": {
		label: "Thin",
		digits: map[rune][]string{
			'0': {"┌─┐", "│ │", "│ │", "│ │", "└─┘"},
			'1': {" ╷ ", " │ ", " │ ", " │ ", "─┴─"},
			'2': {"┌─┐", "  ┤", "┌─┘", "│  ", "└─┘"},
			'3': {"┌─┐", "  ┤", "┌─┘", "  ┤", "└─┘"},
			'4': {"│ │", "│ │", "└─┤", "  │", "  │"},
			'5': {"┌─┐", "│  ", "└─┐", "  ┤", "└─┘"},
			'6': {"┌─┐", "│  ", "├─┐", "│ │", "└─┘"},
			'7': {"┌─┐", "  │", "  │", "  │", "  │"},
			'8': {"┌─┐", "│ │", "├─┤", "│ │", "└─┘"},
			'9': {"┌─┐", "│ │", "└─┤", "  │", "└─┘"},
			':': {"   ", " ║ ", "   ", " ║ ", "   "},
		},
	},
}

var fontOrder = []string{"ansi", "block", "thin"}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickTockMsg(t) })
}

func (m model) Init() tea.Cmd {
	if m.mode == "fatal" {
		return nil
	}
	cmds := []tea.Cmd{textinput.Blink, m.flushOutboxCmd()}
	if (m.mode == "timer" || m.mode == "break") && m.running {
		cmds = append(cmds, tickCmd())
	}
	return tea.Batch(cmds...)
}

func (m model) activeSessionMode() string {
	switch m.mode {
	case "timer", "break":
		return m.mode
	case "edit":
		return m.editReturnMode
	case "settings":
		if m.settingsReturnMode == "stats" {
			return m.statsReturnMode
		}
		return m.settingsReturnMode
	case "stats":
		return m.statsReturnMode
	case "help":
		switch m.helpReturnMode {
		case "settings":
			if m.settingsReturnMode == "stats" {
				return m.statsReturnMode
			}
			return m.settingsReturnMode
		case "stats":
			return m.statsReturnMode
		default:
			return m.helpReturnMode
		}
	default:
		return ""
	}
}

func (m *model) saveOnQuit() {
	sessionMode := m.activeSessionMode()
	if (sessionMode != "timer" && sessionMode != "break") || m.seconds <= 0 {
		return
	}
	previous := m.mode
	m.mode = sessionMode
	m.saveSession()
	m.mode = previous
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickTockMsg:
		sessionMode := m.activeSessionMode()
		if m.running && (sessionMode == "timer" || sessionMode == "break") {
			if m.seconds > 0 {
				m.seconds--
				m.sessionElapsed++
				if sessionMode == "timer" {
					m.totalWorkTime++
				} else {
					m.totalBreakTime++
				}
			}
			if m.seconds == 0 {
				if m.mode != sessionMode {
					m.mode = sessionMode
				}
				var notifyC tea.Cmd
				if sessionMode == "timer" {
					notifyC = m.notifyCmd("work_complete")
				} else {
					notifyC = m.notifyCmd("break_complete")
				}
				model, cmd := m.completeSession()
				return model, tea.Batch(notifyC, cmd)
			}
			return m, tickCmd()
		}
		return m, nil

	case notifResultMsg:
		if msg.err != nil {
			m.setAppError(msg.err, "Notification failed")
		} else {
			if msg.status != "" {
				m.notificationStatus = msg.status
			}
			if msg.id != "" {
				if m.deliveredNotifyIDs == nil {
					m.deliveredNotifyIDs = make(map[string]time.Time)
				}
				m.deliveredNotifyIDs[msg.id] = time.Now()
			}
		}
		return m, nil

	case outboxFlushedMsg:
		m.notificationOutbox = msg.remaining
		if len(msg.deliveredIDs) > 0 {
			if m.deliveredNotifyIDs == nil {
				m.deliveredNotifyIDs = make(map[string]time.Time)
			}
			now := time.Now()
			for _, id := range msg.deliveredIDs {
				m.deliveredNotifyIDs[id] = now
			}
		}
		if msg.err != nil {
			m.setAppError(msg.err, "Failed to save notification queue")
		} else if msg.status != "" {
			m.notificationStatus = msg.status
		}
		if len(msg.remaining) > 0 {
			m.setAppError(fmt.Errorf("%s", msg.remaining[0].LastError), "Notification queued for retry")
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		key := msg.String()
		if m.mode == "fatal" {
			switch key {
			case "ctrl+c", "q", "enter", "esc":
				return m, tea.Quit
			}
			return m, nil
		}

		if key == "?" {
			if m.mode == "help" {
				return m.closeHelp(true)
			}
			m = m.openHelp()
			return m, nil
		}

		if m.mode == "help" {
			if key == "esc" {
				return m.closeHelp(true)
			}
			if key == "ctrl+c" || key == "q" {
				m.saveOnQuit()
				return m, tea.Quit
			}
			// Keep the help screen modal so timer shortcuts don't fire underneath it.
			return m, nil
		}

		if m.mode == "settings" {
			switch key {
			case "tab":
				m.settingsCursor = (m.settingsCursor + 1) % settingsCount
				return m, nil
			case "shift+tab":
				m.settingsCursor--
				if m.settingsCursor < 0 {
					m.settingsCursor = settingsCount - 1
				}
				return m, nil
			case "enter", " ", "space":
				m.toggleSetting()
				return m, nil
			case "left", "h":
				m.adjustSetting(-1)
				return m, nil
			case "right", "l":
				m.adjustSetting(1)
				return m, nil
			case "esc":
				target := m.settingsReturnMode
				if target == "" {
					target = m.activeSessionMode()
				}
				if target == "" {
					target = "timer"
				}
				m.mode = target
				return m, nil
			case "q", "ctrl+c":
				m.saveOnQuit()
				return m, tea.Quit
			}
			return m, nil
		}

		return m.handleKeyMsg(msg)
	}

	return m, nil
}

func (m model) setInputFocus(field int) model {
	m.focusedField = field
	m.textInput.Blur()
	m.durationInput.Blur()
	m.noteInput.Blur()
	m.tagInput.Blur()
	switch field {
	case focusTemplate:
		// Template selector is rendered manually; no input widget to focus.
	case focusTask:
		m.textInput.Focus()
	case focusDuration:
		m.durationInput.Focus()
	case focusNote:
		m.noteInput.Focus()
	case focusTags:
		m.tagInput.Focus()
	}
	return m
}

func (m model) openHelp() model {
	m.helpReturnMode = m.mode
	m.helpWasRunning = m.running
	m.mode = "help"
	return m
}

func (m model) closeHelp(resume bool) (model, tea.Cmd) {
	m.mode = m.helpReturnMode
	if resume && m.helpWasRunning {
		if !m.running {
			m.running = true
			if m.seconds > 0 {
				return m, tickCmd()
			}
		}
	} else {
		if !m.helpWasRunning {
			m.running = false
		}
	}
	return m, nil
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	key := msg.String()

	switch key {
	case "ctrl+c", "q":
		m.saveOnQuit()
		return m, tea.Quit

	case "tab":
		if m.mode == "input" {
			if m.focusedField == focusTemplate {
				m = m.setInputFocus(focusTask)
			} else if m.focusedField == focusTask {
				m = m.setInputFocus(focusDuration)
			} else if m.focusedField == focusDuration {
				m = m.setInputFocus(focusNote)
			} else if m.focusedField == focusNote {
				m = m.setInputFocus(focusTags)
			} else {
				m = m.setInputFocus(focusTemplate)
			}
			return m, nil
		}
		if m.mode == "templates" {
			if len(m.templates) == 0 {
				return m, nil
			}
			m = m.cycleTemplate(1)
			return m, nil
		}
		if m.mode == "timer" || m.mode == "break" {
			m.statsReturnMode = m.mode
			m.mode = "stats"
			return m, nil
		}
		if m.mode == "stats" {
			if m.statsReturnMode != "" {
				m.mode = m.statsReturnMode
			} else {
				m.mode = "timer"
			}
			return m, nil
		}

	case "up":
		if m.mode == "input" && m.focusedField == focusTask && len(m.recentTasks) > 0 {
			m = m.applyRecentTask(-1)
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(-1)
			return m, nil
		}

	case "down":
		if m.mode == "input" && m.focusedField == focusTask && len(m.recentTasks) > 0 {
			m = m.applyRecentTask(1)
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(1)
			return m, nil
		}

	case "left", "h":
		if m.mode == "input" && m.focusedField == focusTemplate && len(m.templates) > 0 {
			m.templateIndex = (m.templateIndex - 1 + len(m.templates)) % len(m.templates)
			m = m.applySelectedTemplate()
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(-1)
			return m, nil
		}

	case "right", "l":
		if m.mode == "input" && m.focusedField == focusTemplate && len(m.templates) > 0 {
			m.templateIndex = (m.templateIndex + 1) % len(m.templates)
			m = m.applySelectedTemplate()
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(1)
			return m, nil
		}

	case "ctrl+t":
		if m.mode == "input" {
			if err := m.saveCurrentTemplate(); err != nil {
				m.setAppError(err, "Failed to save template")
			}
			return m, nil
		}
		if m.mode == "templates" {
			if err := m.saveCurrentTemplate(); err != nil {
				m.setAppError(err, "Failed to save template")
			}
			return m, nil
		}

	case "ctrl+r":
		if m.mode == "input" && m.focusedField == focusTemplate {
			if err := m.renameSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to rename template")
			}
			return m, nil
		}
		if m.mode == "templates" {
			if err := m.renameSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to rename template")
			}
			return m, nil
		}

	case "ctrl+d":
		if m.mode == "input" && m.focusedField == focusTemplate {
			if err := m.deleteSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to delete template")
			}
			return m, nil
		}
		if m.mode == "templates" {
			if err := m.deleteSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to delete template")
			}
			return m, nil
		}

	case "ctrl+y":
		if m.mode == "templates" {
			if err := m.duplicateSelectedTemplate(); err != nil {
				m.setAppError(err, "Failed to duplicate template")
			}
			return m, nil
		}

	case "s":
		if m.mode == "timer" || m.mode == "break" || m.mode == "stats" {
			m.settingsReturnMode = m.mode
			m.settingsCursor = settingsNotifications
			m.mode = "settings"
			return m, nil
		}

	case "enter":
		if m.mode == "input" {
			if m.focusedField == focusTemplate {
				m = m.applySelectedTemplate()
				m = m.setInputFocus(focusTask)
				return m, nil
			}
			if m.focusedField == focusTask {
				m = m.setInputFocus(focusDuration)
				return m, nil
			}
			if m.focusedField == focusDuration {
				m = m.setInputFocus(focusNote)
				return m, nil
			}
			if m.focusedField == focusNote {
				m = m.setInputFocus(focusTags)
				return m, nil
			}
			if m.focusedField == focusTags {
				if strings.TrimSpace(m.textInput.Value()) == "" {
					m.inputError = "Task name is required."
					return m, nil
				}
				durationSeconds, err := parseDurationInput(m.durationInput.Value())
				if err != nil {
					m.inputError = err.Error()
					return m, nil
				}
				m.mode = "timer"
				m.taskName = strings.TrimSpace(m.textInput.Value())
				m.textInput.Blur()
				m.durationInput.Blur()
				m.noteInput.Blur()
				m.tagInput.Blur()
				m.sessionStart = time.Now()
				m.running = true
				m.sessionTarget = durationSeconds
				m.seconds = durationSeconds
				m.sessionElapsed = 0
				m.inputError = ""
				return m, tea.Batch(tickCmd(), m.notifyCmd("session_start"))
			}
			if strings.TrimSpace(m.textInput.Value()) == "" {
				m.inputError = "Task name is required."
				return m, nil
			}
			durationSeconds, err := parseDurationInput(m.durationInput.Value())
			if err != nil {
				m.inputError = err.Error()
				return m, nil
			}
			m.mode = "timer"
			m.taskName = strings.TrimSpace(m.textInput.Value())
			m.textInput.Blur()
			m.durationInput.Blur()
			m.noteInput.Blur()
			m.tagInput.Blur()
			m.sessionStart = time.Now()
			m.running = true
			m.sessionTarget = durationSeconds
			m.seconds = durationSeconds
			m.sessionElapsed = 0
			m.inputError = ""
			return m, tea.Batch(tickCmd(), m.notifyCmd("session_start"))
		}
		if m.mode == "templates" {
			m = m.applySelectedTemplate()
			m.mode = "input"
			m = m.setInputFocus(focusTask)
			return m, nil
		}

		if m.mode == "input" && key == "ctrl+t" {
			// handled above
		}

		if m.mode == "edit" {
			durationSeconds, err := parseDurationInput(m.durationInput.Value())
			if err != nil {
				m.inputError = err.Error()
				return m, nil
			}
			if durationSeconds <= m.sessionElapsed {
				m.inputError = "Duration must be greater than elapsed time."
				return m, nil
			}
			m.sessionTarget = durationSeconds
			m.seconds = durationSeconds - m.sessionElapsed
			m.mode = m.editReturnMode
			m.inputError = ""
			if m.editWasRunning && m.seconds > 0 {
				m.running = true
				return m, tickCmd()
			}
			return m, nil
		}

		if m.mode == "timer" || m.mode == "break" {
			notifyC := m.notifyCmd("session_end")
			model, cmd := m.completeSession()
			return model, tea.Batch(notifyC, cmd)
		}
	case " ", "space":
		if m.mode == "timer" || m.mode == "break" {
			m.running = !m.running
			notifyC := m.notifyCmd("pause_resume")
			if m.running && m.seconds > 0 {
				return m, tea.Batch(tickCmd(), notifyC)
			}
			return m, notifyC
		}
	case "e":
		if m.mode == "timer" || m.mode == "break" {
			m.editReturnMode = m.mode
			m.editWasRunning = m.running
			m.running = false
			m.mode = "edit"
			m.durationInput.SetValue(formatDurationInput(m.sessionTarget))
			m.durationInput.Focus()
			m.textInput.Blur()
			m.inputError = ""
			return m, nil
		}
	case "esc":
		if m.mode == "edit" {
			m.mode = m.editReturnMode
			m.inputError = ""
			if m.editWasRunning && m.seconds > 0 {
				m.running = true
				return m, tickCmd()
			}
			return m, nil
		}
	case "ctrl+p":
		if m.mode == "input" {
			m.templateReturnMode = m.mode
			m.templateWasRunning = m.running
			m.mode = "templates"
			return m, nil
		}
		if m.mode == "templates" {
			m.mode = "input"
			return m, nil
		}
	}

	if m.mode == "input" {
		if m.focusedField == focusTask {
			m.textInput, cmd = m.textInput.Update(msg)
			m.recentTaskIndex = -1
		} else if m.focusedField == focusDuration {
			m.durationInput, cmd = m.durationInput.Update(msg)
		} else if m.focusedField == focusNote {
			m.noteInput, cmd = m.noteInput.Update(msg)
		} else {
			m.tagInput, cmd = m.tagInput.Update(msg)
		}
		if m.inputError != "" {
			m.inputError = ""
		}
		return m, cmd
	}

	if m.mode == "edit" {
		m.durationInput, cmd = m.durationInput.Update(msg)
		if m.inputError != "" {
			m.inputError = ""
		}
		return m, cmd
	}

	return m, nil
}

func (m model) completeSession() (tea.Model, tea.Cmd) {
	flushCmd := m.saveSession()
	if m.mode == "timer" {
		m.sessionCount++
		if m.config.AutoBreak && m.sessionCount%m.config.SessionsBeforeBreak == 0 {
			m.mode = "break"
			m.sessionStart = time.Now()
			m.sessionTarget = m.config.BreakDuration * 60
			m.seconds = m.sessionTarget
			m.sessionElapsed = 0
			m.running = true
			return m, tea.Batch(tickCmd(), flushCmd)
		}
	}

	m.mode = "input"
	m.taskName = ""
	m.seconds = 0
	m.sessionTarget = 0
	m.sessionElapsed = 0
	m.running = false
	m.inputError = ""
	m.textInput.SetValue("")
	m.noteInput.SetValue("")
	m.tagInput.SetValue("")
	m = m.setInputFocus(focusTask)
	return m, flushCmd
}

func parseTags(input string) []string {
	parts := strings.Split(input, ",")
	tags := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		tag := strings.ToLower(strings.TrimSpace(part))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func (m model) currentSessionTags() []string {
	if m.tagInput.Value() == "" {
		return nil
	}
	return parseTags(m.tagInput.Value())
}

func renderTopTags(entries []Entry) string {
	counts := make(map[string]int)
	for _, entry := range entries {
		for _, tag := range entry.Tags {
			counts[strings.ToLower(strings.TrimSpace(tag))]++
		}
	}
	if len(counts) == 0 {
		return "Tags: none yet"
	}
	type tagCount struct {
		tag   string
		count int
	}
	items := make([]tagCount, 0, len(counts))
	for tag, count := range counts {
		items = append(items, tagCount{tag: tag, count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].tag < items[j].tag
		}
		return items[i].count > items[j].count
	})
	limit := 3
	if len(items) < limit {
		limit = len(items)
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s (%d)", items[i].tag, items[i].count))
	}
	return "Top tags: " + strings.Join(parts, ", ")
}

func (m *model) toggleSetting() {
	switch m.settingsCursor {
	case settingsNotifications:
		m.config.Notifications = !m.config.Notifications
	case settingsDesktop:
		m.config.DesktopNotifications = !m.config.DesktopNotifications
	case settingsWorkComplete:
		m.config.NotifyWorkComplete = !m.config.NotifyWorkComplete
	case settingsBreakComplete:
		m.config.NotifyBreakComplete = !m.config.NotifyBreakComplete
	case settingsSessionStart:
		m.config.NotifySessionStart = !m.config.NotifySessionStart
	case settingsSessionEnd:
		m.config.NotifySessionEnd = !m.config.NotifySessionEnd
	case settingsPauseResume:
		m.config.NotifyPauseResume = !m.config.NotifyPauseResume
	case settingsEndingSoon:
		m.config.NotifyEndingSoon = !m.config.NotifyEndingSoon
	case settingsTheme:
		m.config.Theme = nextValue(themeOrder, m.config.Theme, 1)
	case settingsFont:
		m.config.Font = nextValue(fontOrder, m.config.Font, 1)
	}
	if err := saveConfigFile(m.configFile, m.config); err != nil {
		m.setAppError(err, "Failed to save config")
	}
}

func (m *model) adjustSetting(delta int) {
	switch m.settingsCursor {
	case settingsTheme:
		m.config.Theme = nextValue(themeOrder, m.config.Theme, delta)
	case settingsFont:
		m.config.Font = nextValue(fontOrder, m.config.Font, delta)
	case settingsQuietStart:
		m.config.QuietHoursStart = wrapHour(m.config.QuietHoursStart + delta)
	case settingsQuietEnd:
		m.config.QuietHoursEnd = wrapHour(m.config.QuietHoursEnd + delta)
	}
	if err := saveConfigFile(m.configFile, m.config); err != nil {
		m.setAppError(err, "Failed to save config")
	}
}

func wrapHour(hour int) int {
	if hour < 0 {
		return 23
	}
	if hour > 23 {
		return 0
	}
	return hour
}

func parseDurationInput(input string) (int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return 0, fmt.Errorf("Duration is required.")
	}

	if strings.Contains(trimmed, ":") {
		parts := strings.Split(trimmed, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("Use mm or hh:mm for duration.")
		}
		hours, err := strconv.Atoi(parts[0])
		if err != nil || hours < 0 {
			return 0, fmt.Errorf("Hours must be a positive number.")
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil || minutes < 0 || minutes > 59 {
			return 0, fmt.Errorf("Minutes must be between 0 and 59.")
		}
		total := hours*3600 + minutes*60
		if total == 0 {
			return 0, fmt.Errorf("Duration must be greater than 0.")
		}
		return total, nil
	}

	minutes, err := strconv.Atoi(trimmed)
	if err != nil || minutes <= 0 {
		return 0, fmt.Errorf("Duration must be a positive number of minutes.")
	}
	return minutes * 60, nil
}

func formatDurationInput(seconds int) string {
	if seconds <= 0 {
		return "0"
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d", hours, minutes)
	}
	return fmt.Sprintf("%d", minutes)
}

func formatClock(seconds int) string {
	h, m, s := seconds/3600, (seconds%3600)/60, seconds%60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func centerBlock(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(content)
}

func (m *model) setAppError(err error, context string) {
	if err == nil {
		return
	}
	if context == "" {
		m.appError = err.Error()
		return
	}
	m.appError = fmt.Sprintf("%s: %v", context, err)
}

func (m *model) setNotificationStatus(status string) {
	m.notificationStatus = status
}

func defaultOutboxFile() string { return "notification_outbox.json" }

func loadNotificationOutbox(path string) ([]notificationJob, error) {
	var jobs []notificationJob
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return jobs, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func saveNotificationOutbox(path string, jobs []notificationJob) error {
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func saveConfigFile(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

var notificationBackoff = []time.Duration{
	2 * time.Second,
	5 * time.Second,
	15 * time.Second,
}

func newNotificationJob(id, event, title, body string) notificationJob {
	return notificationJob{
		ID:        id,
		Event:     event,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now(),
	}
}

func scheduleNextAttempt(job *notificationJob) {
	delay := notificationBackoff[len(notificationBackoff)-1]
	if job.Attempts > 0 && job.Attempts <= len(notificationBackoff) {
		delay = notificationBackoff[job.Attempts-1]
	}
	job.NextAttemptAt = time.Now().Add(delay)
}

func (m model) eventEnabled(event string) bool {
	switch event {
	case "work_complete":
		return m.config.NotifyWorkComplete
	case "break_complete":
		return m.config.NotifyBreakComplete
	case "session_start":
		return m.config.NotifySessionStart
	case "session_end":
		return m.config.NotifySessionEnd
	case "pause_resume":
		return m.config.NotifyPauseResume
	case "ending_soon":
		return m.config.NotifyEndingSoon
	default:
		return false
	}
}

func normalizeTheme(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range themeOrder {
		if candidate == name {
			return candidate
		}
	}
	return defaultConfig.Theme
}

func normalizeFont(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range fontOrder {
		if candidate == name {
			return candidate
		}
	}
	return defaultConfig.Font
}

func activeTheme(cfg Config) themeStyle {
	if theme, ok := themeStyles[normalizeTheme(cfg.Theme)]; ok {
		return theme
	}
	return themeStyles[defaultConfig.Theme]
}

func activeFont(cfg Config) timerFont {
	if font, ok := timerFonts[normalizeFont(cfg.Font)]; ok {
		return font
	}
	return timerFonts[defaultConfig.Font]
}

func themedStyle(cfg Config, color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func nextValue(order []string, current string, delta int) string {
	if len(order) == 0 {
		return current
	}
	current = strings.ToLower(strings.TrimSpace(current))
	index := 0
	for i, candidate := range order {
		if candidate == current {
			index = i
			break
		}
	}
	next := (index + delta) % len(order)
	if next < 0 {
		next += len(order)
	}
	return order[next]
}

func themeLabel(name string) string {
	return strings.Title(normalizeTheme(name))
}

func fontLabel(name string) string {
	font := activeFont(Config{Font: normalizeFont(name)})
	return font.label
}

func (m model) quietHoursActive(now time.Time) bool {
	start := m.config.QuietHoursStart
	end := m.config.QuietHoursEnd
	if start < 0 || start > 23 || end < 0 || end > 23 || start == end {
		return false
	}
	hour := now.Hour()
	if start < end {
		return hour >= start && hour < end
	}
	return hour >= start || hour < end
}

func (m model) notificationTitle(event string) string {
	switch event {
	case "work_complete":
		return "Work session complete"
	case "break_complete":
		return "Break complete"
	case "session_start":
		return "Session started"
	case "session_end":
		return "Session ended"
	case "pause_resume":
		if m.running {
			return "Session resumed"
		}
		return "Session paused"
	case "ending_soon":
		return "Session ending soon"
	default:
		return "Kairu"
	}
}

func (m model) notificationBody(event string) string {
	switch event {
	case "work_complete":
		return fmt.Sprintf("%s completed in %s", m.taskName, formatDuration(m.sessionElapsed))
	case "break_complete":
		return "Break is over. Ready to focus again?"
	case "session_start":
		return fmt.Sprintf("Focus session started: %s", m.taskName)
	case "session_end":
		return fmt.Sprintf("Session ended: %s", m.taskName)
	case "pause_resume":
		if m.running {
			return "Focus timer resumed."
		}
		return "Focus timer paused."
	case "ending_soon":
		return fmt.Sprintf("Only %s left in this session.", formatDuration(m.seconds))
	default:
		return ""
	}
}

func (m model) notificationID(event string) string {
	base := fmt.Sprintf("%s-%d", event, m.sessionStart.UnixNano())
	switch event {
	case "pause_resume":
		return fmt.Sprintf("%s-%t-%d", base, m.running, m.sessionElapsed)
	case "session_start":
		return fmt.Sprintf("%s-%s", base, m.taskName)
	case "session_end", "work_complete", "break_complete", "ending_soon":
		return fmt.Sprintf("%s-%d", base, m.sessionElapsed)
	default:
		return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
	}
}

func (m model) hasNotification(id string) bool {
	if id == "" {
		return false
	}
	if m.deliveredNotifyIDs != nil {
		if _, ok := m.deliveredNotifyIDs[id]; ok {
			return true
		}
	}
	for _, job := range m.notificationOutbox {
		if job.ID == id {
			return true
		}
	}
	return false
}

func (m model) notifyCmd(event string) tea.Cmd {
	if !m.config.Notifications || !m.eventEnabled(event) {
		return nil
	}
	if m.quietHoursActive(time.Now()) {
		return func() tea.Msg {
			return notifResultMsg{status: "Notification suppressed by quiet hours"}
		}
	}
	title := m.notificationTitle(event)
	body := m.notificationBody(event)
	if body == "" {
		return nil
	}
	id := m.notificationID(event)
	if m.hasNotification(id) {
		return func() tea.Msg {
			return notifResultMsg{id: id, status: "Duplicate notification suppressed"}
		}
	}
	job := newNotificationJob(id, event, title, body)
	cfg := m.config
	outboxFile := m.outboxFile
	existing := append([]notificationJob(nil), m.notificationOutbox...)
	return func() tea.Msg {
		status, err := sendNotification(cfg, job)
		if err == nil {
			return notifResultMsg{id: id, status: status}
		}
		job.Attempts = 1
		job.LastError = err.Error()
		scheduleNextAttempt(&job)
		updated := append(existing, job)
		if saveErr := saveNotificationOutbox(outboxFile, updated); saveErr != nil {
			return outboxFlushedMsg{remaining: updated, err: saveErr}
		}
		return outboxFlushedMsg{
			remaining: updated,
			status:    "Notification queued for retry",
		}
	}
}

func sendNotification(cfg Config, job notificationJob) (string, error) {
	if strings.TrimSpace(job.Body) == "" {
		return "", fmt.Errorf("notification body is empty")
	}
	return deliverNotification(cfg, job.Title, job.Body)
}

func deliverNotification(cfg Config, title, body string) (string, error) {
	if cfg.DesktopNotifications {
		if err := sendDesktopNotification(title, body); err == nil {
			return "Desktop notification delivered", nil
		}
	}
	if cfg.SoundCommand != "" {
		if err := exec.Command("sh", "-c", cfg.SoundCommand).Run(); err == nil {
			return "Sound fallback delivered", nil
		}
	}
	if token := strings.TrimSpace(cfg.TelegramBotToken); token != "" && strings.TrimSpace(cfg.TelegramChatID) != "" {
		if err := sendTelegramMessage(token, strings.TrimSpace(cfg.TelegramChatID), body); err == nil {
			return "Telegram fallback delivered", nil
		}
	}
	return "", fmt.Errorf("all notification channels failed")
}

func sendDesktopNotification(title, body string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		return exec.Command("osascript", "-e", script).Run()
	case "linux":
		if err := exec.Command("notify-send", title, body).Run(); err == nil {
			return nil
		}
		return exec.Command("sh", "-c", fmt.Sprintf("printf '\\a'; printf '%s: %s\\n'", shellEscape(title), shellEscape(body))).Run()
	case "windows":
		script := fmt.Sprintf(`
$ErrorActionPreference = 'SilentlyContinue'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$notify = New-Object System.Windows.Forms.NotifyIcon
$notify.Icon = [System.Drawing.SystemIcons]::Information
$notify.BalloonTipTitle = '%s'
$notify.BalloonTipText = '%s'
$notify.Visible = $true
$notify.ShowBalloonTip(4000)
Start-Sleep -Milliseconds 4500
$notify.Dispose()
`, psEscape(title), psEscape(body))
		return exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script).Run()
	default:
		return fmt.Errorf("desktop notifications are not supported on %s", runtime.GOOS)
	}
}

func shellEscape(s string) string {
	s = strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + s + "'"
}

func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func (m model) flushOutboxCmd() tea.Cmd {
	if !m.config.Notifications || len(m.notificationOutbox) == 0 {
		return nil
	}
	jobs := make([]notificationJob, len(m.notificationOutbox))
	copy(jobs, m.notificationOutbox)
	cfg := m.config
	outboxFile := m.outboxFile
	return func() tea.Msg {
		remaining := make([]notificationJob, 0)
		var delivered []string
		var lastStatus string
		now := time.Now()
		for _, job := range jobs {
			if !job.NextAttemptAt.IsZero() && job.NextAttemptAt.After(now) {
				remaining = append(remaining, job)
				continue
			}
			status, err := sendNotification(cfg, job)
			if err != nil {
				job.Attempts++
				job.LastError = err.Error()
				scheduleNextAttempt(&job)
				remaining = append(remaining, job)
				continue
			}
			lastStatus = status
			delivered = append(delivered, job.ID)
		}
		var saveErr error
		if err := saveNotificationOutbox(outboxFile, remaining); err != nil {
			saveErr = err
		}
		return outboxFlushedMsg{remaining: remaining, deliveredIDs: delivered, status: lastStatus, err: saveErr}
	}
}

func (m *model) saveSession() tea.Cmd {
	duration := m.sessionElapsed
	sessionType := "work"
	if m.mode == "break" {
		sessionType = "break"
	}

	entry := Entry{
		Task:     m.taskName,
		Note:     strings.TrimSpace(m.noteInput.Value()),
		Tags:     parseTags(m.tagInput.Value()),
		Start:    m.sessionStart,
		End:      time.Now(),
		Duration: duration,
		Type:     sessionType,
	}

	var entries []Entry
	if data, err := os.ReadFile(m.dataFile); err == nil {
		if err := json.Unmarshal(data, &entries); err != nil {
			m.setAppError(err, "Failed to parse entries")
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		m.setAppError(err, "Failed to read entries")
		return nil
	}
	entries = append(entries, entry)
	fileData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		m.setAppError(err, "Failed to encode entries")
		return nil
	}
	if err := os.WriteFile(m.dataFile, fileData, 0644); err != nil {
		m.setAppError(err, "Failed to write entries")
		return nil
	}
	m.entries = entries
	m.recentTasks = buildRecentTasks(entries)
	m.recentTaskIndex = -1
	return nil
}

func buildRecentTasks(entries []Entry) []string {
	seen := make(map[string]struct{})
	recent := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		task := strings.TrimSpace(entries[i].Task)
		if task == "" {
			continue
		}
		key := strings.ToLower(task)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		recent = append(recent, task)
	}
	return recent
}

func (m model) applyRecentTask(delta int) model {
	if len(m.recentTasks) == 0 {
		return m
	}
	if m.recentTaskIndex < 0 || m.recentTaskIndex >= len(m.recentTasks) {
		m.recentTaskIndex = 0
	} else {
		m.recentTaskIndex = (m.recentTaskIndex + delta + len(m.recentTasks)) % len(m.recentTasks)
	}
	m.textInput.SetValue(m.recentTasks[m.recentTaskIndex])
	m.textInput.CursorEnd()
	return m
}

func (m model) currentTemplate() (SessionTemplate, bool) {
	if len(m.templates) == 0 || m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		return SessionTemplate{}, false
	}
	return m.templates[m.templateIndex], true
}

func (m model) applySelectedTemplate() model {
	template, ok := m.currentTemplate()
	if !ok {
		return m
	}
	if strings.TrimSpace(template.Task) != "" {
		m.textInput.SetValue(template.Task)
		m.textInput.CursorEnd()
	}
	if strings.TrimSpace(template.Duration) != "" {
		m.durationInput.SetValue(template.Duration)
		m.durationInput.CursorEnd()
	}
	m.noteInput.SetValue(strings.TrimSpace(template.Note))
	m.tagInput.SetValue(strings.Join(template.Tags, ", "))
	m.inputError = ""
	m.appError = ""
	return m
}

func (m model) cycleTemplate(delta int) model {
	if len(m.templates) == 0 {
		return m
	}
	if m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		m.templateIndex = 0
	} else {
		m.templateIndex = (m.templateIndex + delta + len(m.templates)) % len(m.templates)
	}
	return m.applySelectedTemplate()
}

func (m *model) saveCurrentTemplate() error {
	task := strings.TrimSpace(m.textInput.Value())
	if task == "" {
		return fmt.Errorf("task name is required before saving a template")
	}
	template := SessionTemplate{
		Name:     task,
		Task:     task,
		Duration: strings.TrimSpace(m.durationInput.Value()),
		Note:     strings.TrimSpace(m.noteInput.Value()),
		Tags:     parseTags(m.tagInput.Value()),
	}
	if strings.TrimSpace(template.Duration) == "" {
		return fmt.Errorf("duration is required before saving a template")
	}
	replaced := false
	for i, existing := range m.templates {
		if strings.EqualFold(strings.TrimSpace(existing.Name), task) {
			m.templates[i] = template
			m.templateIndex = i
			replaced = true
			break
		}
	}
	if !replaced {
		m.templates = append([]SessionTemplate{template}, m.templates...)
		m.templateIndex = 0
	}
	if err := saveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.notificationStatus = fmt.Sprintf("Saved template: %s", task)
	return nil
}

func (m *model) renameSelectedTemplate() error {
	template, ok := m.currentTemplate()
	if !ok {
		return fmt.Errorf("no template selected")
	}
	newName := strings.TrimSpace(m.textInput.Value())
	if newName == "" {
		return fmt.Errorf("task name is required before renaming a template")
	}
	template.Name = newName
	template.Task = newName
	template.Duration = strings.TrimSpace(m.durationInput.Value())
	template.Note = strings.TrimSpace(m.noteInput.Value())
	template.Tags = parseTags(m.tagInput.Value())
	m.templates[m.templateIndex] = template
	if err := saveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.notificationStatus = fmt.Sprintf("Renamed template to: %s", newName)
	return nil
}

func (m *model) deleteSelectedTemplate() error {
	if len(m.templates) == 0 || m.templateIndex < 0 || m.templateIndex >= len(m.templates) {
		return fmt.Errorf("no template selected")
	}
	removed := m.templates[m.templateIndex]
	m.templates = append(m.templates[:m.templateIndex], m.templates[m.templateIndex+1:]...)
	if len(m.templates) == 0 {
		m.templateIndex = 0
	} else if m.templateIndex >= len(m.templates) {
		m.templateIndex = len(m.templates) - 1
	}
	if err := saveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	if len(m.templates) > 0 {
		updated := m.applySelectedTemplate()
		*m = updated
	}
	m.notificationStatus = fmt.Sprintf("Deleted template: %s", removed.Name)
	return nil
}

func (m *model) duplicateSelectedTemplate() error {
	template, ok := m.currentTemplate()
	if !ok {
		return fmt.Errorf("no template selected")
	}
	copy := template
	copy.Name = template.Name + " Copy"
	copy.Task = template.Task + " Copy"
	m.templates = append([]SessionTemplate{copy}, m.templates...)
	m.templateIndex = 0
	if err := saveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.notificationStatus = fmt.Sprintf("Duplicated template: %s", template.Name)
	return nil
}

func (m model) currentTemplateDetails() string {
	template, ok := m.currentTemplate()
	if !ok {
		return "No templates saved yet."
	}
	tags := "none"
	if len(template.Tags) > 0 {
		tags = strings.Join(template.Tags, ", ")
	}
	note := template.Note
	if strings.TrimSpace(note) == "" {
		note = "none"
	}
	return fmt.Sprintf("Task: %s\nDuration: %s\nNote: %s\nTags: %s", template.Task, template.Duration, note, tags)
}

func sendTelegramMessage(token, chatID, text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if err != nil {
			return fmt.Errorf("telegram send failed: %w", err)
		}
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("telegram send failed: %s", message)
	}
	return nil
}

func (m model) View() string {
	switch m.mode {
	case "input":
		return renderInputView(m)
	case "timer", "break":
		return renderTimerView(m)
	case "edit":
		return renderEditView(m)
	case "stats":
		return renderStatsView(m)
	case "settings":
		return renderSettingsView(m)
	case "templates":
		return renderTemplateManagerView(m)
	case "help":
		return renderHelpView(m)
	case "fatal":
		return renderFatalView(m)
	default:
		return renderInputView(m)
	}
}

func joinNonEmptyLines(lines ...string) string {
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			trimmed = append(trimmed, line)
		}
	}
	return strings.Join(trimmed, "\n")
}

func renderAppError(m model) string {
	if strings.TrimSpace(m.appError) == "" {
		return ""
	}
	return themedStyle(m.config, activeTheme(m.config).warning).Render(m.appError)
}

func renderNotificationStatus(m model) string {
	if strings.TrimSpace(m.notificationStatus) == "" {
		return ""
	}
	return themedStyle(m.config, activeTheme(m.config).primary).Render(m.notificationStatus)
}

func renderInputView(m model) string {
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	templateLine := "Template: none"
	if len(m.templates) > 0 {
		if template, ok := m.currentTemplate(); ok {
			templateLine = fmt.Sprintf("Template: %s (%d/%d)", template.Name, m.templateIndex+1, len(m.templates))
		}
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))
	return fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📝  What are you working on?      │
╰─────────────────────────────────────╯

%s

%s

%s

%s

%s

%s

[Tab] Switch Field   [Enter] Start/Apply   [Ctrl+T] Save Template   [?] Help   [q] Quit
Recent tasks: Up/Down to cycle from history
Templates: Left/Right while Template is focused

`, templateLine, m.textInput.View(), m.durationInput.View(), m.noteInput.View(), m.tagInput.View(), errorBlock)
}

func renderEditView(m model) string {
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))
	elapsed := formatClock(m.sessionElapsed)
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  ✏️  Adjust Session Time           │
╰─────────────────────────────────────╯

Task: %s
Elapsed: %s

%s

%s

	[Enter] Apply   [Esc] Cancel   [?] Help   [q] Quit`, renderBanner(m.config), m.taskName, elapsed, m.durationInput.View(), errorBlock)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderTimerView(m model) string {
	timeStr := formatClock(m.seconds)
	theme := activeTheme(m.config)

	modeStr := "WORK"
	if m.mode == "break" {
		modeStr = "BREAK"
	}

	// Progress bar
	targetSeconds := m.sessionTarget
	if targetSeconds <= 0 {
		targetSeconds = 1
	}
	remainingPct := float64(m.seconds) / float64(targetSeconds) * 100
	if remainingPct > 100 {
		remainingPct = 100
	}
	if remainingPct < 0 {
		remainingPct = 0
	}
	barWidth := 40
	filled := int(remainingPct / 100 * float64(barWidth))
	empty := barWidth - filled
	progress := fmt.Sprintf("[%s%s] %.0f%%", strings.Repeat("█", filled), strings.Repeat("░", empty), remainingPct)

	hint := "[Space] Pause  [E] Edit  [Enter] End  [Tab] Stats  [S] Settings  [?] Help  [q] Quit"
	if !m.running {
		hint = "[Space] Resume  [E] Edit  [Enter] End  [Tab] Stats  [S] Settings  [?] Help  [q] Quit"
	}

	header := fmt.Sprintf("%s • %s", modeStr, m.taskName)
	if tags := strings.Join(m.currentSessionTags(), ", "); tags != "" {
		header += fmt.Sprintf(" [%s]", tags)
	}
	ascii := renderASCIITimer(timeStr, m.config)
	innerWidth := lipgloss.Width(progress)
	if asciiWidth := lipgloss.Width(ascii); asciiWidth > innerWidth {
		innerWidth = asciiWidth
	}
	ascii = themedStyle(m.config, theme.accent).Width(innerWidth).Align(lipgloss.Center).Render(ascii)
	timerFrame := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(0, 1).
		Render(fmt.Sprintf("%s\n\n%s", ascii, progress))

	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	details := hint
	if errorLine != "" {
		details = fmt.Sprintf("%s\n%s", errorLine, hint)
	}
	if statusLine != "" {
		details = fmt.Sprintf("%s\n%s", details, statusLine)
	}
	block := fmt.Sprintf(`%s

%s

%s`, header, timerFrame, details)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderASCIITimer(timeStr string, cfg Config) string {
	chars := activeFont(cfg).digits
	lines := make([]string, 5)
	for _, ch := range timeStr {
		if art, ok := chars[ch]; ok {
			for i := 0; i < 5; i++ {
				lines[i] += art[i] + " "
			}
		}
	}

	return strings.Join(lines, "\n")
}

func renderStatsView(m model) string {
	weeklyData := getWeeklyData(m.entries)
	barChart := renderWeeklyBarChart(weeklyData)

	daily := formatDuration(getDailyTotal(m.entries, "work"))
	current, longest := calculateStreaks(m.entries)
	emptyMessage := ""
	if len(m.entries) == 0 {
		emptyMessage = "No sessions yet. Start a focus session to see stats."
	}
	tagSummary := renderTopTags(m.entries)

	workRatio := 0
	total := m.totalWorkTime + m.totalBreakTime
	if total > 0 {
		workRatio = m.totalWorkTime * 100 / total
	}
	footer := "[Tab] Back   [S] Settings   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	if emptyMessage != "" {
		emptyMessage = fmt.Sprintf("\n%s\n", emptyMessage)
	}

	return fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📊  Activity Dashboard            │
╰─────────────────────────────────────╯

┌─────────────────┐
│  📅  Today      │
│  %-13s  │
└─────────────────┘

┌─────────────────┐
│  🔥  Streaks    │
│  Current: %-3d  │
│  Longest: %-3d  │
└─────────────────┘

┌─────────────────┐
│  ⚖️  Ratio      │
│  Work: %d%%     │
│  Break: %d%%    │
└─────────────────┘

Weekly Activity (7 days):

%s

%s

%s

%s
`, daily, current, longest, workRatio, 100-workRatio, emptyMessage, tagSummary, barChart, footer)
}

func renderSettingsView(m model) string {
	footer := "[Tab] Switch   [Space] Toggle   [Left/Right] Adjust   [Esc] Back   [q] Quit"
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	if statusLine != "" {
		footer = fmt.Sprintf("%s\n%s", statusLine, footer)
	}

	leftColumn := strings.Join([]string{
		renderSettingsSection(m.config, "Theme", []string{
			renderSettingLine(m.settingsCursor == settingsTheme, "Theme", themeLabel(m.config.Theme)),
		}),
		renderSettingsSection(m.config, "Typography", []string{
			renderSettingLine(m.settingsCursor == settingsFont, "Timer font", fontLabel(m.config.Font)),
		}),
		renderSettingsSection(m.config, "Notifications", []string{
			renderSettingLine(m.settingsCursor == settingsNotifications, "Notifications", boolLabel(m.config.Notifications)),
			renderSettingLine(m.settingsCursor == settingsDesktop, "Desktop notifications", boolLabel(m.config.DesktopNotifications)),
			renderSettingLine(m.settingsCursor == settingsWorkComplete, "Work complete", boolLabel(m.config.NotifyWorkComplete)),
			renderSettingLine(m.settingsCursor == settingsBreakComplete, "Break complete", boolLabel(m.config.NotifyBreakComplete)),
			renderSettingLine(m.settingsCursor == settingsSessionStart, "Session start", boolLabel(m.config.NotifySessionStart)),
			renderSettingLine(m.settingsCursor == settingsSessionEnd, "Session end", boolLabel(m.config.NotifySessionEnd)),
			renderSettingLine(m.settingsCursor == settingsPauseResume, "Pause/resume", boolLabel(m.config.NotifyPauseResume)),
			renderSettingLine(m.settingsCursor == settingsEndingSoon, "Ending soon", boolLabel(m.config.NotifyEndingSoon)),
		}),
		renderSettingsSection(m.config, "Quiet Hours", []string{
			renderSettingLine(m.settingsCursor == settingsQuietStart, "Quiet start", hourLabel(m.config.QuietHoursStart)),
			renderSettingLine(m.settingsCursor == settingsQuietEnd, "Quiet end", hourLabel(m.config.QuietHoursEnd)),
		}),
	}, "\n\n")

	rightColumn := renderSettingsPreview(m)

	var body string
	if m.width >= 110 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, "    ", rightColumn)
	} else {
		body = leftColumn + "\n\n" + rightColumn
	}

	hints := renderSettingsHintRow(m)

	block := renderBanner(m.config) + "\n\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  Notification Settings              │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		body + "\n\n" +
		hints + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderTemplateManagerView(m model) string {
	footer := "[Tab/Arrows] Browse   [Enter] Apply   [Ctrl+T] Save current form   [Ctrl+R] Rename   [Ctrl+D] Delete   [Ctrl+Y] Duplicate   [Ctrl+P/Esc] Back   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	if statusLine != "" {
		footer = fmt.Sprintf("%s\n%s", statusLine, footer)
	}

	listLines := []string{"Templates:"}
	if len(m.templates) == 0 {
		listLines = append(listLines, "  No templates saved yet.")
	} else {
		for i, template := range m.templates {
			prefix := "  "
			if i == m.templateIndex {
				prefix = "> "
			}
			listLines = append(listLines, fmt.Sprintf("%s%s (%s)", prefix, template.Name, template.Duration))
		}
	}

	preview := m.currentTemplateDetails()
	if len(m.templates) > 0 {
		preview = "Selected Template:\n" + preview
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(32).Render(strings.Join(listLines, "\n")),
		"    ",
		lipgloss.NewStyle().Width(40).Render(preview),
	)
	block := renderBanner(m.config) + "\n\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  Session Templates                 │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		body + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderSettingsSection(cfg Config, title string, lines []string) string {
	theme := activeTheme(cfg)
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.accent)).Render(title)
	return fmt.Sprintf(`╭─────────────────────────────────────╮
│  %s
╰─────────────────────────────────────╯

%s`, header, strings.Join(lines, "\n"))
}

func renderSettingsPreview(m model) string {
	theme := activeTheme(m.config)
	font := activeFont(m.config)
	timer := renderASCIITimer("08:45", m.config)
	lines := strings.Split(timer, "\n")
	if len(lines) > 5 {
		lines = lines[:5]
	}

	themeLine := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.accent)).Render("Theme: " + themeLabel(m.config.Theme))
	fontLine := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.primary)).Render("Font: " + font.label)
	timerBlock := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.accent)).Render(strings.Join(lines, "\n"))

	return fmt.Sprintf(`╭─────────────────────────────────────╮
│  Live Preview                       │
╰─────────────────────────────────────╯

%s
%s
%s`, themeLine, fontLine, timerBlock)
}

func renderSettingsHintRow(m model) string {
	switch m.settingsCursor {
	case settingsTheme:
		return "Theme: Left/Right cycles presets. Space also cycles."
	case settingsFont:
		return "Typography: Left/Right cycles timer fonts. Space also cycles."
	case settingsQuietStart, settingsQuietEnd:
		return "Quiet hours: Left/Right adjusts the hour."
	case settingsNotifications, settingsDesktop, settingsWorkComplete, settingsBreakComplete, settingsSessionStart, settingsSessionEnd, settingsPauseResume, settingsEndingSoon:
		return "Toggles: Space, Enter, or Left/Right flips the setting."
	default:
		return "Use Tab to move between sections."
	}
}

func renderSettingLine(selected bool, label, value string) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	return fmt.Sprintf("%s%-22s %s", prefix, label, value)
}

func boolLabel(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}

func hourLabel(hour int) string {
	if hour < 0 {
		return "off"
	}
	return fmt.Sprintf("%02d:00", hour)
}

func renderHelpView(m model) string {
	footer := "[?] Close   [Esc] Close   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	lines := []string{
		"Timer continues while help is open.",
		"",
		"Global:",
		formatHelpLine("?", "Toggle help"),
		formatHelpLine("q", "Quit"),
		"",
		"Input mode:",
		formatHelpLine("Tab", "Switch field"),
		formatHelpLine("Enter", "Start session"),
		"",
		"Timer/Break:",
		formatHelpLine("Space", "Pause/Resume"),
		formatHelpLine("E", "Edit time"),
		formatHelpLine("Enter", "End session"),
		formatHelpLine("Tab", "Stats"),
		formatHelpLine("S", "Settings"),
		"",
		"Edit:",
		formatHelpLine("Enter", "Apply"),
		formatHelpLine("Esc", "Cancel"),
		"",
		"Stats:",
		formatHelpLine("Tab", "Back"),
		formatHelpLine("S", "Settings"),
		"",
	}
	body := lipgloss.NewStyle().Width(35).Render(strings.Join(lines, "\n"))
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  Help                               │
╰─────────────────────────────────────╯

%s

%s`, renderBanner(m.config), body, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func formatHelpLine(key, description string) string {
	return fmt.Sprintf("  %-8s %s", key, description)
}

func renderFatalView(m model) string {
	message := strings.TrimSpace(m.appError)
	if message == "" {
		message = "Failed to start due to an unexpected error."
	}
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  Startup Error                      │
╰─────────────────────────────────────╯

%s

[q] Quit`, renderBanner(m.config), message)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func getWeeklyData(entries []Entry) map[string]int {
	weekly := make(map[string]int)
	today := time.Now()
	for i := 0; i < 7; i++ {
		date := dateKey(today.AddDate(0, 0, -i))
		weekly[date] = 0
	}
	for _, e := range entries {
		date := dateKey(e.Start)
		if _, ok := weekly[date]; ok && e.Type == "work" {
			weekly[date] += e.Duration
		}
	}
	return weekly
}

func renderWeeklyBarChart(weeklyData map[string]int) string {
	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	today := time.Now()

	maxMinutes := 0
	for _, secs := range weeklyData {
		mins := secs / 60
		if mins > maxMinutes {
			maxMinutes = mins
		}
	}
	if maxMinutes == 0 {
		return "No activity yet."
	}

	var b strings.Builder
	for i := 6; i >= 0; i-- {
		dateValue := today.AddDate(0, 0, -i)
		date := dateKey(dateValue)
		dayName := days[dateValue.Weekday()]
		minutes := weeklyData[date] / 60

		barLen := minutes * 40 / maxMinutes
		bar := strings.Repeat("█", barLen) + strings.Repeat("░", 40-barLen)

		b.WriteString(fmt.Sprintf("%s │%s│ %dm\n", dayName, bar, minutes))
	}

	return b.String()
}

func getDailyTotal(entries []Entry, sessionType string) int {
	today := dateKey(time.Now())
	total := 0
	for _, e := range entries {
		if dateKey(e.Start) == today && e.Type == sessionType {
			total += e.Duration
		}
	}
	return total
}

func calculateStreaks(entries []Entry) (int, int) {
	days := make(map[string]bool)
	for _, e := range entries {
		if e.Type == "work" {
			days[dateKey(e.Start)] = true
		}
	}

	var list []string
	for d := range days {
		list = append(list, d)
	}
	sort.Strings(list)

	longest, temp := 0, 0
	var last time.Time
	for _, d := range list {
		date, err := time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Kairu: invalid entry date:", d)
			continue
		}
		if last.IsZero() {
			temp = 1
		} else if last.AddDate(0, 0, 1).Equal(date) {
			temp++
		} else {
			if temp > longest {
				longest = temp
			}
			temp = 1
		}
		last = date
	}
	if temp > longest {
		longest = temp
	}

	today := time.Now()
	if !days[dateKey(today)] {
		return 0, longest
	}
	current := 0
	for i := 0; i < 365; i++ {
		if days[dateKey(today.AddDate(0, 0, -i))] {
			current++
		} else if i > 0 {
			break
		}
	}

	return current, longest
}

func dateKey(value time.Time) string {
	return value.In(time.Local).Format("2006-01-02")
}

func formatDuration(seconds int) string {
	h, m := seconds/3600, (seconds%3600)/60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return "0m"
}

func renderBanner(cfg Config) string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme(cfg).accent)).Padding(0, 2).Render("KAIRU  •  Grow Your Focus")
}

func main() {
	dataFile := "entries.json"
	exportPath := flag.String("export", "", "Export entries.json to the provided file path")
	importPath := flag.String("import", "", "Import entries from the provided file path into entries.json")
	flag.Parse()

	if *exportPath != "" && *importPath != "" {
		fmt.Println("Error: --export and --import cannot be used together.")
		os.Exit(1)
	}
	if *exportPath != "" {
		if err := exportEntries(dataFile, *exportPath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Export complete:", *exportPath)
		return
	}
	if *importPath != "" {
		if err := importEntries(dataFile, *importPath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Import complete:", *importPath)
		return
	}
	startupErrors := []string{}
	if err := loadEnvFile(".env"); err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to load .env: %v", err))
	}
	cfg, err := loadConfig("kairu.yaml")
	fatalConfig := false
	if err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to load config: %v", err))
		fatalConfig = true
	}
	templates, err := loadSessionTemplates("templates.json")
	if err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to load templates: %v", err))
	}
	ti := textinput.New()
	ti.Placeholder = "Task name"
	ti.CharLimit = 50
	ti.Width = 40
	ti.Prompt = "Task: "

	di := textinput.New()
	di.Placeholder = "25"
	di.CharLimit = 8
	di.Width = 16
	di.Prompt = "Duration (mm or hh:mm): "
	di.SetValue(fmt.Sprintf("%d", cfg.WorkDuration))
	di.Blur()

	ni := textinput.New()
	ni.Placeholder = "Optional note"
	ni.CharLimit = 120
	ni.Width = 40
	ni.Prompt = "Note: "
	ni.Blur()

	gi := textinput.New()
	gi.Placeholder = "Optional tags, comma separated"
	gi.CharLimit = 120
	gi.Width = 40
	gi.Prompt = "Tags: "
	gi.Blur()

	var entryList []Entry
	if data, err := os.ReadFile(dataFile); err == nil {
		if err := json.Unmarshal(data, &entryList); err != nil {
			startupErrors = append(startupErrors, fmt.Sprintf("Failed to parse entries: %v", err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to read entries: %v", err))
	}
	mode := "input"
	if fatalConfig {
		mode = "fatal"
	}
	initialFocus := focusTask
	if len(templates) > 0 {
		initialFocus = focusTemplate
	}

	m := model{
		mode:               mode,
		textInput:          ti,
		durationInput:      di,
		noteInput:          ni,
		tagInput:           gi,
		focusedField:       initialFocus,
		entries:            entryList,
		templates:          templates,
		recentTasks:        buildRecentTasks(entryList),
		recentTaskIndex:    -1,
		dataFile:           dataFile,
		templateFile:       "templates.json",
		configFile:         "kairu.yaml",
		config:             cfg,
		appError:           strings.Join(startupErrors, " | "),
		outboxFile:         defaultOutboxFile(),
		deliveredNotifyIDs: make(map[string]time.Time),
	}
	m = m.setInputFocus(initialFocus)

	if jobs, err := loadNotificationOutbox(m.outboxFile); err == nil {
		m.notificationOutbox = jobs
	} else {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to read notification queue: %v", err))
		m.appError = strings.Join(startupErrors, " | ")
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
