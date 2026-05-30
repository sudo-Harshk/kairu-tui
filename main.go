package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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
	SoundscapesDir       string   `yaml:"soundscapes_dir"`
	SoundscapePlayer     string   `yaml:"soundscape_player"`
	PinnedTasks          []string `yaml:"pinned_tasks"`
	TasksFile            string   `yaml:"tasks_file"`
	GuardianMode         bool     `yaml:"guardian_mode"`
	LockdownCommand      string   `yaml:"lockdown_command"`
	UnlockCommand        string   `yaml:"unlock_command"`
	Layout               string   `yaml:"layout"`
	SynthVolume          float64  `yaml:"synth_volume"`
	BinauralPreset       string   `yaml:"binaural_preset"`
	BinauralCarrier      float64  `yaml:"binaural_carrier"`
	BinauralBeat         float64  `yaml:"binaural_beat"`
	FadeInDuration       int      `yaml:"fade_in_duration"`
	FadeOutDuration      int      `yaml:"fade_out_duration"`
	TelegramBotToken     string   `yaml:"-"`
	TelegramChatID       string   `yaml:"-"`
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
	SoundscapesDir:       "soundscapes",
	SoundscapePlayer:     "mpv --loop --no-video",
	PinnedTasks:          []string{},
	TasksFile:            "tasks.txt",
	GuardianMode:         false,
	LockdownCommand:      "",
	UnlockCommand:        "",
	Layout:               "classic",
	SynthVolume:          0.5,
	BinauralPreset:       "alpha",
	BinauralCarrier:      120.0,
	BinauralBeat:         10.0,
	FadeInDuration:       500,
	FadeOutDuration:      200,
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
	seconds             int
	sessionTarget       int
	sessionElapsed      int
	width               int
	running             bool
	mode                string
	editReturnMode      string
	editWasRunning      bool
	helpReturnMode      string
	helpWasRunning      bool
	templateReturnMode  string
	templateWasRunning  bool
	settingsReturnMode  string
	statsReturnMode     string
	textInput           textinput.Model
	durationInput       textinput.Model
	noteInput           textinput.Model
	tagInput            textinput.Model
	templateIndex       int
	focusedField        int
	inputError          string
	appError            string
	notificationStatus  string
	lastDeletedTemplate *deletedTemplateState
	taskName            string
	taskSuggestions     []string
	suggestionIndex     int
	showRecentOverlay   bool
	settingsCursor      int
	entries             []Entry
	templates           []SessionTemplate
	dataFile            string
	templateFile        string
	configFile          string
	config              Config
	sessionStart        time.Time
	sessionCount        int
	totalWorkTime       int
	totalBreakTime      int
	streakState         StreakState
	notificationOutbox  []notificationJob
	deliveredNotifyIDs  map[string]time.Time
	outboxFile          string
	soundscapeReturnMode string
	soundscapes         []string
	soundscapeIndex     int
	activeSoundscapeCmd *exec.Cmd
	internalLogs        []string
	guardianLocked      bool
	abortConfirmation   bool
	petState            PetState
	petEnabled          bool
	showPetSidebar      bool
	petOnboardingStage  int
	petOnboardingIndex  int
	petNameInput        textinput.Model
	showPetLevelUpOverlay bool

	// Cyber-Tamagotchi TUI States
	tamagotchiReturnMode   string
	tamagotchiActiveMenu   string // "", "feed", "shop", "play"
	tamagotchiMenuSelect   int
	tamagotchiFeedback     string
	tamagotchiFeedbackTime time.Time
	typingGame             TypingGameState
	binaryGame             BinaryGameState
}

func loadSoundscapes(dir string) ([]string, error) {
	var soundscapes []string
	soundscapes = append(soundscapes,
		"[Synth] White Noise",
		"[Synth] Pink Noise",
		"[Synth] Brown Noise",
		"[Synth] Focus Binaural Beats",
	)
	if dir == "" {
		return soundscapes, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return soundscapes, nil
		}
		return nil, err
	}
	var fileTracks []string
	extensions := map[string]bool{
		".mp3":  true,
		".wav":  true,
		".ogg":  true,
		".flac": true,
		".m4a":  true,
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if extensions[ext] {
			fileTracks = append(fileTracks, entry.Name())
		}
	}
	sort.Strings(fileTracks)
	soundscapes = append(soundscapes, fileTracks...)
	return soundscapes, nil
}

type deletedTemplateState struct {
	template  SessionTemplate
	index     int
	expiresAt time.Time
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

type ProjectBackup struct {
	Version            int               `json:"version"`
	CreatedAt          time.Time         `json:"created_at"`
	Entries            []Entry           `json:"entries"`
	Templates          []SessionTemplate `json:"templates"`
	ConfigYAML         string            `json:"config_yaml"`
	NotificationOutbox []notificationJob `json:"notification_outbox,omitempty"`
}

type StreakState struct {
	Current           int
	Best              int
	LastWorkDay       string
	RecoveryAvailable bool
	RecoveryNeeded    bool
	RecoveryPrompt    string
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

func loadFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func backupProject(dataFile, templateFile, configFile, outboxFile, backupPath string) error {
	entries, err := loadEntries(dataFile)
	if err != nil {
		return fmt.Errorf("failed to read entries: %w", err)
	}
	templates, err := loadSessionTemplates(templateFile)
	if err != nil {
		return fmt.Errorf("failed to read templates: %w", err)
	}
	configYAML, err := loadFileString(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	outbox, err := loadNotificationOutbox(outboxFile)
	if err != nil {
		return fmt.Errorf("failed to read notification queue: %w", err)
	}
	backup := ProjectBackup{
		Version:            1,
		CreatedAt:          time.Now().UTC(),
		Entries:            entries,
		Templates:          templates,
		ConfigYAML:         configYAML,
		NotificationOutbox: outbox,
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode backup: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	return nil
}

func restoreProject(dataFile, templateFile, configFile, outboxFile, backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	var backup ProjectBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}
	if backup.Version != 1 {
		return fmt.Errorf("unsupported backup version: %d", backup.Version)
	}
	if err := validateEntries(backup.Entries); err != nil {
		return fmt.Errorf("backup entries validation failed: %w", err)
	}
	entriesData, err := json.MarshalIndent(backup.Entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode entries: %w", err)
	}
	if err := os.WriteFile(dataFile, entriesData, 0644); err != nil {
		return fmt.Errorf("failed to restore entries: %w", err)
	}
	templatesData, err := json.MarshalIndent(backup.Templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode templates: %w", err)
	}
	if err := os.WriteFile(templateFile, templatesData, 0644); err != nil {
		return fmt.Errorf("failed to restore templates: %w", err)
	}
	if err := os.WriteFile(configFile, []byte(backup.ConfigYAML), 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}
	outboxData, err := json.MarshalIndent(backup.NotificationOutbox, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode notification queue: %w", err)
	}
	if err := os.WriteFile(outboxFile, outboxData, 0644); err != nil {
		return fmt.Errorf("failed to restore notification queue: %w", err)
	}
	return nil
}

type tickTockMsg time.Time

type petAnimTickMsg time.Time

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
	settingsLayout
	settingsQuietStart
	settingsQuietEnd
	settingsSynthVolume
	settingsBinauralPreset
	settingsBinauralCarrier
	settingsBinauralBeat
	settingsFadeIn
	settingsFadeOut
	settingsBackup
	settingsRestore
	settingsClearOutbox
	settingsCount
)

type themeStyle struct {
	accent  string
	primary string
	notice  string
	warning string
}

var themeStyles = map[string]themeStyle{
	"forest":     {accent: "10", primary: "2", notice: "3", warning: "1"},
	"ocean":      {accent: "14", primary: "6", notice: "12", warning: "9"},
	"ember":      {accent: "208", primary: "214", notice: "220", warning: "196"},
	"mono":       {accent: "15", primary: "7", notice: "8", warning: "9"},
	"matrix":     {accent: "46", primary: "22", notice: "28", warning: "1"},
	"cyberpunk":  {accent: "201", primary: "51", notice: "226", warning: "196"},
	"minimalist": {accent: "244", primary: "248", notice: "240", warning: "1"},
}

var themeOrder = []string{"forest", "ocean", "ember", "mono", "matrix", "cyberpunk", "minimalist"}

var layoutOrder = []string{"classic", "compact", "minimal"}

var binauralPresetsOrder = []string{"alpha", "beta", "theta", "delta", "custom"}

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

func petAnimTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return petAnimTickMsg(t) })
}

func (m model) Init() tea.Cmd {
	if m.mode == "fatal" {
		return nil
	}
	cmds := []tea.Cmd{textinput.Blink, m.flushOutboxCmd()}
	if (m.activeSessionMode() == "timer" || m.activeSessionMode() == "break") && m.running {
		cmds = append(cmds, tickCmd())
	}
	if m.petEnabled {
		cmds = append(cmds, petAnimTick())
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
	case "analytics":
		return m.statsReturnMode
	case "heatmap":
		return m.statsReturnMode
	case "report":
		return m.statsReturnMode
	case "templates":
		return m.templateReturnMode
	case "soundscapes":
		return m.soundscapeReturnMode
	case "logs":
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
		case "heatmap":
			return m.statsReturnMode
		default:
			return m.helpReturnMode
		}
	default:
		return ""
	}
}

func (m model) returnModeForModal() string {
	if mode := m.activeSessionMode(); mode != "" {
		return mode
	}
	if m.mode == "stats" {
		return m.statsReturnMode
	}
	if m.mode == "analytics" {
		return m.statsReturnMode
	}
	if m.mode == "heatmap" {
		return m.statsReturnMode
	}
	if m.mode == "report" {
		return m.statsReturnMode
	}
	return m.mode
}

func (m *model) saveOnQuit() {
	m.stopSoundscape()
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
			if m.mode == "timer" || m.mode == "break" || m.mode == "settings" || m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" || m.mode == "report" || m.mode == "help" || m.mode == "logs" || m.mode == "templates" || m.mode == "soundscapes" || m.mode == "tamagotchi" {
				return m, tickCmd()
			}
			return m, nil
		}
		if m.mode == "tamagotchi" {
			m.petState.TickStateDecay(time.Now())
			return m, tickCmd()
		}
		return m, nil

	case petAnimTickMsg:
		if m.petEnabled && (m.mode == "tamagotchi" || (m.showPetSidebar && m.width >= 90)) {
			m.petState.TickStateDecay(time.Now())
			return m, petAnimTick()
		}
		return m, nil

	case notifResultMsg:
		if msg.err != nil {
			m.setAppError(msg.err, "Notification failed")
		} else {
			if msg.status != "" {
				m.setNotificationStatus(msg.status)
			}
			if msg.id != "" {
				if m.deliveredNotifyIDs == nil {
					m.deliveredNotifyIDs = make(map[string]time.Time)
				}
				m.deliveredNotifyIDs[msg.id] = time.Now()
				m.logInternal("NOTIF: Delivered %s", msg.id)
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
				m.logInternal("NOTIF: Flushed %s", id)
			}
		}
		if msg.err != nil {
			m.setAppError(msg.err, "Failed to save notification queue")
		} else if msg.status != "" {
			m.setNotificationStatus(msg.status)
		}
		if len(msg.remaining) > 0 {
			m.setAppError(fmt.Errorf("%s", msg.remaining[0].LastError), "Notification queued for retry")
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.petEnabled && m.showPetSidebar && m.width >= 90 {
			return m, petAnimTick()
		}
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		if m.showPetLevelUpOverlay {
			m.showPetLevelUpOverlay = false
			return m, nil
		}

		// Global toggle for pet sidebar
		if key == "ctrl+g" && m.mode != "fatal" {
			if m.petEnabled {
				m.showPetSidebar = !m.showPetSidebar
				if m.showPetSidebar && m.width >= 90 {
					return m, petAnimTick()
				}
				return m, nil
			}
		}

		// Global toggle for Tamagotchi Screen
		if key == "ctrl+t" && m.mode != "fatal" {
			if m.petEnabled {
				if m.mode == "tamagotchi" {
					m.mode = m.tamagotchiReturnMode
					if m.mode == "" {
						m.mode = "input"
					}
				} else {
					m.tamagotchiReturnMode = m.mode
					m.mode = "tamagotchi"
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					// Tick decay immediately on opening to keep state fresh!
					m.petState.TickStateDecay(time.Now())
					_ = SavePetState("pet.json", m.petState)
				}
				return m, tea.Batch(tickCmd(), petAnimTick())
			}
		}

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

		if m.mode == "tamagotchi" {
			// Clear temporary feedback message if enough time has passed
			if !m.tamagotchiFeedbackTime.IsZero() && time.Since(m.tamagotchiFeedbackTime) > 3*time.Second {
				m.tamagotchiFeedback = ""
			}

			// If active menu is rebirth (renaming pet)
			if m.tamagotchiActiveMenu == "rebirth" {
				switch key {
				case "enter":
					name := strings.TrimSpace(m.textInput.Value())
					if name == "" {
						name = "Neko"
					}
					m.tamagotchiFeedback = m.petState.RebirthPet(name)
					m.tamagotchiFeedbackTime = time.Now()
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					_ = SavePetState("pet.json", m.petState)
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					return m, nil
				default:
					var cmd tea.Cmd
					m.textInput, cmd = m.textInput.Update(msg)
					return m, cmd
				}
			}

			// If active game is typing challenge
			if m.tamagotchiActiveMenu == "typing" {
				if m.typingGame.Finished {
					if key == "enter" {
						m.tamagotchiActiveMenu = ""
						m.tamagotchiMenuSelect = 0
						m.tamagotchiFeedback = ""
					}
					return m, nil
				}
				
				switch key {
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				case "backspace":
					if len(m.typingGame.TypedText) > 0 {
						m.typingGame.TypedText = m.typingGame.TypedText[:len(m.typingGame.TypedText)-1]
					}
					return m, nil
				case "enter", "ctrl+m", "space", "tab":
					// Allow entering space or characters
					char := " "
					if key == "enter" || key == "ctrl+m" {
						char = "\n"
					} else if key == "tab" {
						char = "\t"
					}
					if len(m.typingGame.TypedText) < len(m.typingGame.TargetText) {
						m.typingGame.TypedText += char
					}
				default:
					if len(key) == 1 && len(m.typingGame.TypedText) < len(m.typingGame.TargetText) {
						m.typingGame.TypedText += key
					}
				}

				// Check typing game completion
				if len(m.typingGame.TypedText) >= len(m.typingGame.TargetText) {
					m.typingGame.Finished = true
					// Calculate WPM and Accuracy
					correctCount := 0
					for i := 0; i < len(m.typingGame.TargetText); i++ {
						if i < len(m.typingGame.TypedText) && m.typingGame.TargetText[i] == m.typingGame.TypedText[i] {
							correctCount++
						}
					}
					m.typingGame.Accuracy = (float64(correctCount) / float64(len(m.typingGame.TargetText))) * 100.0
					duration := time.Since(m.typingGame.StartTime).Minutes()
					if duration <= 0 {
						duration = 0.01
					}
					words := float64(len(m.typingGame.TargetText)) / 5.0
					m.typingGame.WPM = int(words / duration)
					if m.typingGame.WPM > 200 {
						m.typingGame.WPM = 200 // Cap typing WPM
					}

					// Rewards
					coinsReward := 5
					if m.typingGame.Accuracy >= 90.0 {
						coinsReward = 10
						if m.typingGame.WPM > 60 {
							coinsReward = 15 // Bonus hacker reward
						}
					}
					m.typingGame.CoinsWon = coinsReward
					m.petState.Coins += coinsReward
					m.petState.Happiness += 25
					if m.petState.Happiness > 100 {
						m.petState.Happiness = 100
					}
					_ = SavePetState("pet.json", m.petState)
				}
				return m, nil
			}

			// If active game is binary guessing game
			if m.tamagotchiActiveMenu == "guessing" {
				if m.binaryGame.Finished {
					if key == "enter" {
						m.tamagotchiActiveMenu = ""
						m.tamagotchiMenuSelect = 0
						m.tamagotchiFeedback = ""
					}
					return m, nil
				}

				switch key {
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				case "backspace":
					if len(m.binaryGame.InputStr) > 0 {
						m.binaryGame.InputStr = m.binaryGame.InputStr[:len(m.binaryGame.InputStr)-1]
					}
					return m, nil
				case "enter":
					// Submit guess
					var guess int
					_, err := fmt.Sscanf(m.binaryGame.InputStr, "%d", &guess)
					if err != nil {
						m.binaryGame.LastHint = "Invalid decimal number! Enter digits only."
						m.binaryGame.InputStr = ""
						return m, nil
					}

					m.binaryGame.Attempts++
					if guess == m.binaryGame.TargetNum {
						m.binaryGame.Won = true
						m.binaryGame.Finished = true
						m.petState.Coins += 5
						m.petState.Happiness += 15
						if m.petState.Happiness > 100 {
							m.petState.Happiness = 100
						}
						_ = SavePetState("pet.json", m.petState)
					} else if m.binaryGame.Attempts >= 4 {
						m.binaryGame.Finished = true
					} else if guess < m.binaryGame.TargetNum {
						m.binaryGame.LastHint = "Higher!"
						m.binaryGame.InputStr = ""
					} else {
						m.binaryGame.LastHint = "Lower!"
						m.binaryGame.InputStr = ""
					}
					return m, nil
				default:
					if len(key) == 1 && key >= "0" && key <= "9" && len(m.binaryGame.InputStr) < 3 {
						m.binaryGame.InputStr += key
					}
				}
				return m, nil
			}

			// Main sub-menus keyboard handling
			switch m.tamagotchiActiveMenu {
			case "feed":
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect - 1 + 3) % 3
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 1) % 3
					return m, nil
				case "enter":
					items := []string{"fish", "treat", "drink"}
					item := items[m.tamagotchiMenuSelect]
					var leveledUp bool
					m.tamagotchiFeedback, leveledUp = m.petState.FeedItem(item)
					m.tamagotchiFeedbackTime = time.Now()
					if leveledUp {
						m.showPetLevelUpOverlay = true
					}
					_ = SavePetState("pet.json", m.petState)
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				}

			case "shop":
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect - 1 + 4) % 4
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 1) % 4
					return m, nil
				case "enter":
					items := []string{"fish", "treat", "drink", "medicine"}
					costs := []int{5, 10, 8, 15}
					item := items[m.tamagotchiMenuSelect]
					cost := costs[m.tamagotchiMenuSelect]
					m.tamagotchiFeedback = m.petState.BuyItem(item, cost)
					m.tamagotchiFeedbackTime = time.Now()
					_ = SavePetState("pet.json", m.petState)
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				}

			case "play":
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect - 1 + 2) % 2
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 1) % 2
					return m, nil
				case "enter":
					if m.tamagotchiMenuSelect == 0 {
						m.tamagotchiActiveMenu = "typing"
						m.typingGame = InitTypingGame()
					} else {
						m.tamagotchiActiveMenu = "guessing"
						m.binaryGame = InitBinaryGame()
					}
					return m, nil
				case "esc":
					m.tamagotchiActiveMenu = ""
					m.tamagotchiMenuSelect = 0
					m.tamagotchiFeedback = ""
					return m, nil
				}

			default:
				// Grid navigation for main actions (8 choices in 4x2 layout)
				// options: [0] FEED, [1] PLAY, [2] CLEAN, [3] HEAL, [4] SLEEP, [5] SHOP, [6] REBIRTH, [7] EXIT
				switch key {
				case "up", "k":
					m.tamagotchiMenuSelect -= 2
					if m.tamagotchiMenuSelect < 0 {
						m.tamagotchiMenuSelect += 8
					}
					return m, nil
				case "down", "j":
					m.tamagotchiMenuSelect = (m.tamagotchiMenuSelect + 2) % 8
					return m, nil
				case "left", "h":
					if m.tamagotchiMenuSelect%2 == 1 {
						m.tamagotchiMenuSelect--
					} else {
						m.tamagotchiMenuSelect++
					}
					return m, nil
				case "right", "l":
					if m.tamagotchiMenuSelect%2 == 0 {
						m.tamagotchiMenuSelect++
					} else {
						m.tamagotchiMenuSelect--
					}
					return m, nil
				case "esc", "q":
					m.mode = m.tamagotchiReturnMode
					if m.mode == "" {
						m.mode = "input"
					}
					return m, nil
				case "enter":
					switch m.tamagotchiMenuSelect {
					case 0: // FEED
						m.tamagotchiActiveMenu = "feed"
						m.tamagotchiMenuSelect = 0
					case 1: // PLAY
						m.tamagotchiActiveMenu = "play"
						m.tamagotchiMenuSelect = 0
					case 2: // CLEAN
						m.tamagotchiFeedback = m.petState.CleanPoop()
						m.tamagotchiFeedbackTime = time.Now()
						_ = SavePetState("pet.json", m.petState)
					case 3: // HEAL
						m.tamagotchiFeedback = m.petState.HealSick()
						m.tamagotchiFeedbackTime = time.Now()
						_ = SavePetState("pet.json", m.petState)
					case 4: // SLEEP
						m.tamagotchiFeedback = m.petState.ToggleSleep()
						m.tamagotchiFeedbackTime = time.Now()
						_ = SavePetState("pet.json", m.petState)
					case 5: // SHOP
						m.tamagotchiActiveMenu = "shop"
						m.tamagotchiMenuSelect = 0
					case 6: // REBIRTH
						m.tamagotchiActiveMenu = "rebirth"
						m.textInput.SetValue("")
						m.textInput.Focus()
					case 7: // EXIT
						m.mode = m.tamagotchiReturnMode
						if m.mode == "" {
							m.mode = "input"
						}
					}
					return m, nil
				}
			}
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
				if m.settingsCursor == settingsCount-1 && m.settingsReturnMode != "" {
					m.statsReturnMode = m.settingsReturnMode
					m.mode = "stats"
					return m, nil
				}
				m.settingsCursor = (m.settingsCursor + 1) % settingsCount
				return m, nil
			case "down", "j":
				m.settingsCursor = (m.settingsCursor + 1) % settingsCount
				return m, nil
			case "shift+tab", "up", "k":
				m.settingsCursor--
				if m.settingsCursor < 0 {
					m.settingsCursor = settingsCount - 1
				}
				return m, nil
			case "enter", " ", "space":
				if m.settingsCursor == settingsBackup {
					if err := backupProject(m.dataFile, m.templateFile, m.configFile, m.outboxFile, defaultBackupFile()); err != nil {
						m.setAppError(err, "Failed to create backup")
					} else {
						m.setNotificationStatus("Backup saved to backup.json")
					}
					return m, nil
				}
				if m.settingsCursor == settingsRestore {
					if err := restoreProject(m.dataFile, m.templateFile, m.configFile, m.outboxFile, defaultBackupFile()); err != nil {
						m.setAppError(err, "Failed to restore backup")
					} else {
						if err := reloadProjectState(&m); err != nil {
							m.setAppError(err, "Backup restored but reload failed")
						} else {
							m.setNotificationStatus("Backup restored from backup.json")
						}
					}
					return m, nil
				}
				if m.settingsCursor == settingsClearOutbox {
					m.notificationOutbox = nil
					if err := saveNotificationOutbox(m.outboxFile, m.notificationOutbox); err != nil {
						m.setAppError(err, "Failed to clear outbox")
					} else {
						m.setNotificationStatus("Notification queue cleared")
					}
					return m, nil
				}
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
	m.showRecentOverlay = false
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
		if m.guardianLocked {
			m.abortConfirmation = false
			m.appError = "Guardian Mode Active: Press Esc to Abort"
			m.logInternal("GUARDIAN: Blocked exit attempt")
			return m, nil
		}
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
			m.mode = "analytics"
			return m, nil
		}
		if m.mode == "analytics" {
			m.mode = "heatmap"
			return m, nil
		}
		if m.mode == "heatmap" {
			m.mode = "history"
			return m, nil
		}
		if m.mode == "history" {
			m.mode = "report"
			return m, nil
		}
		if m.mode == "report" {
			m.mode = "stats"
			return m, nil
		}

	case "esc":
		if m.guardianLocked {
			if !m.abortConfirmation {
				m.abortConfirmation = true
				m.appError = "Confirm Abort? Press Esc again to force exit (Streak Penalty)"
				return m, nil
			}
			// Penalty Abort
			m.logInternal("GUARDIAN: Session Forcefully Aborted")
			m.stopSoundscape()
			m.runGuardianCommand(m.config.UnlockCommand)
			m.guardianLocked = false
			m.abortConfirmation = false
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
			return m, nil
		}
		m.abortConfirmation = false
		if m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" || m.mode == "report" || m.mode == "logs" {
			if m.statsReturnMode != "" {
				m.mode = m.statsReturnMode
			} else {
				m.mode = "timer"
			}
			return m, nil
		}
		if m.mode == "edit" {
			m.mode = m.editReturnMode
			m.inputError = ""
			if m.editWasRunning && m.seconds > 0 {
				m.running = true
				return m, tickCmd()
			}
			return m, nil
		}
		return m, nil

	case "up":
		if m.mode == "input" && m.focusedField == focusTask && len(m.taskSuggestions) > 0 {
			m.showRecentOverlay = true
			m = m.applyTaskSuggestion(-1)
			return m, nil
		}
		if m.mode == "templates" {
			m = m.cycleTemplate(-1)
			return m, nil
		}

	case "down":
		if m.mode == "input" && m.focusedField == focusTask && len(m.taskSuggestions) > 0 {
			m.showRecentOverlay = true
			m = m.applyTaskSuggestion(1)
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

	case "ctrl+z":
		if m.mode == "templates" {
			if err := m.undoLastTemplateDelete(); err != nil {
				m.setAppError(err, "Failed to undo template delete")
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
		if m.mode == "timer" || m.mode == "break" || m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" || m.mode == "report" {
			m.settingsReturnMode = m.mode
			m.settingsCursor = settingsNotifications
			m.mode = "settings"
			return m, nil
		}

	case "L":
		if m.mode != "logs" {
			m.statsReturnMode = m.mode
			m.mode = "logs"
		} else {
			m.mode = m.statsReturnMode
			if m.mode == "" {
				m.mode = "stats"
			}
		}
		return m, nil

	case "r":
		if m.mode == "stats" || m.mode == "analytics" || m.mode == "heatmap" || m.mode == "history" {
			m.mode = "report"
			return m, nil
		}

	case "e":
		if m.mode == "report" {
			if path, err := m.exportDailyReport(); err != nil {
				m.setAppError(err, "Failed to export daily report")
			} else {
				m.setNotificationStatus(fmt.Sprintf("Daily report exported to %s", path))
			}
			return m, nil
		}
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
				m.startSoundscape()
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
			m.startSoundscape()
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
			if m.config.GuardianMode {
				m.guardianLocked = true
				m.runGuardianCommand(m.config.LockdownCommand)
			}
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

			// Duck volume or pause/resume native synth if running
			if m.soundscapeIndex >= 0 && m.soundscapeIndex < len(m.soundscapes) {
				track := m.soundscapes[m.soundscapeIndex]
				if IsSyntheticTrack(track) {
					pauseNativeSynth(!m.running)
					if m.running {
						fadeNativeVolume(m.config.SynthVolume, time.Duration(m.config.FadeInDuration)*time.Millisecond)
					} else {
						fadeNativeVolume(0.15 * m.config.SynthVolume, time.Duration(m.config.FadeOutDuration)*time.Millisecond)
					}
				}
			}

			notifyC := m.notifyCmd("pause_resume")
			if m.running && m.seconds > 0 {
				return m, tea.Batch(tickCmd(), notifyC)
			}
			return m, notifyC
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
	case "ctrl+m":
		if m.mode == "soundscapes" {
			m.mode = m.soundscapeReturnMode
			return m, nil
		}
		m.soundscapeReturnMode = m.mode
		m.mode = "soundscapes"
		return m, nil
	}

	if m.mode == "soundscapes" {
		switch key {
		case "up", "k":
			if len(m.soundscapes) > 0 {
				m.soundscapeIndex--
				if m.soundscapeIndex < -1 {
					m.soundscapeIndex = len(m.soundscapes) - 1
				}
			}
			return m, nil
		case "down", "j":
			if len(m.soundscapes) > 0 {
				m.soundscapeIndex++
				if m.soundscapeIndex >= len(m.soundscapes) {
					m.soundscapeIndex = -1
				}
			}
			return m, nil
		case "enter":
			if m.activeSessionMode() == "timer" {
				if m.soundscapeIndex == -1 {
					m.stopSoundscape()
				} else {
					m.startSoundscape()
				}
			}
			m.mode = m.soundscapeReturnMode
			return m, nil
		case "esc":
			m.mode = m.soundscapeReturnMode
			return m, nil
		}
		return m, nil
	}

	if m.mode == "input" {
		if m.focusedField == focusTask {
			m.textInput, cmd = m.textInput.Update(msg)
			m.suggestionIndex = -1
			m.showRecentOverlay = false
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

func (m *model) startSoundscape() {
	if m.soundscapeIndex < 0 || m.soundscapeIndex >= len(m.soundscapes) {
		return
	}
	m.stopSoundscape()

	track := m.soundscapes[m.soundscapeIndex]
	m.logInternal("AUDIO: Starting %s", track)

	if IsSyntheticTrack(track) {
		if err := startNativeSynth(track, m.config); err != nil {
			m.setAppError(err, "Failed to start native synthesizer")
		}
		// Trigger a gentle volume fade-in
		fadeNativeVolume(m.config.SynthVolume, time.Duration(m.config.FadeInDuration)*time.Millisecond)
		return
	}

	path := filepath.Join(m.config.SoundscapesDir, track)

	parts := strings.Fields(m.config.SoundscapePlayer)
	if len(parts) == 0 {
		return
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	if err := cmd.Start(); err != nil {
		m.setAppError(err, fmt.Sprintf("Audio player not found. Ensure '%s' is installed or update 'soundscape_player' in kairu.yaml", parts[0]))
		return
	}
	m.activeSoundscapeCmd = cmd
}

func (m *model) stopSoundscape() {
	// Stop native synthesizer
	stopNativeSynth()

	// Stop external player
	if m.activeSoundscapeCmd != nil && m.activeSoundscapeCmd.Process != nil {
		m.logInternal("AUDIO: Stopping playback")
		_ = m.activeSoundscapeCmd.Process.Kill()
		_ = m.activeSoundscapeCmd.Wait()
		m.activeSoundscapeCmd = nil
	}
}

func (m model) completeSession() (tea.Model, tea.Cmd) {
	m.stopSoundscape()
	if m.guardianLocked {
		m.runGuardianCommand(m.config.UnlockCommand)
		m.guardianLocked = false
	}
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
	case settingsLayout:
		m.config.Layout = nextValue(layoutOrder, m.config.Layout, 1)
	case settingsSynthVolume:
		m.config.SynthVolume += 0.1
		if m.config.SynthVolume > 1.05 {
			m.config.SynthVolume = 0.0
		}
		// Apply volume immediately if playing
		setNativeVolume(m.config.SynthVolume)
	case settingsBinauralPreset:
		m.config.BinauralPreset = nextValue(binauralPresetsOrder, m.config.BinauralPreset, 1)
		updateActiveBinauralFrequencies(m.config)
	case settingsBinauralCarrier:
		carriers := []float64{70.0, 100.0, 120.0, 150.0, 200.0}
		idx := -1
		for i, c := range carriers {
			if math.Abs(m.config.BinauralCarrier-c) < 0.1 {
				idx = i
				break
			}
		}
		m.config.BinauralCarrier = carriers[(idx+1)%len(carriers)]
		updateActiveBinauralFrequencies(m.config)
	case settingsBinauralBeat:
		beats := []float64{3.0, 6.0, 10.0, 15.0, 20.0}
		idx := -1
		for i, b := range beats {
			if math.Abs(m.config.BinauralBeat-b) < 0.1 {
				idx = i
				break
			}
		}
		m.config.BinauralBeat = beats[(idx+1)%len(beats)]
		updateActiveBinauralFrequencies(m.config)
	case settingsFadeIn:
		fades := []int{100, 200, 500, 1000, 2000}
		idx := -1
		for i, f := range fades {
			if m.config.FadeInDuration == f {
				idx = i
				break
			}
		}
		m.config.FadeInDuration = fades[(idx+1)%len(fades)]
	case settingsFadeOut:
		fades := []int{100, 200, 500, 1000, 2000}
		idx := -1
		for i, f := range fades {
			if m.config.FadeOutDuration == f {
				idx = i
				break
			}
		}
		m.config.FadeOutDuration = fades[(idx+1)%len(fades)]
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
	case settingsLayout:
		m.config.Layout = nextValue(layoutOrder, m.config.Layout, delta)
	case settingsQuietStart:
		m.config.QuietHoursStart = wrapHour(m.config.QuietHoursStart + delta)
	case settingsQuietEnd:
		m.config.QuietHoursEnd = wrapHour(m.config.QuietHoursEnd + delta)
	case settingsSynthVolume:
		m.config.SynthVolume += float64(delta) * 0.05
		if m.config.SynthVolume < 0.0 {
			m.config.SynthVolume = 0.0
		} else if m.config.SynthVolume > 1.0 {
			m.config.SynthVolume = 1.0
		}
		setNativeVolume(m.config.SynthVolume)
	case settingsBinauralPreset:
		m.config.BinauralPreset = nextValue(binauralPresetsOrder, m.config.BinauralPreset, delta)
		updateActiveBinauralFrequencies(m.config)
	case settingsBinauralCarrier:
		m.config.BinauralCarrier += float64(delta) * 5.0
		if m.config.BinauralCarrier < 20.0 {
			m.config.BinauralCarrier = 20.0
		} else if m.config.BinauralCarrier > 20000.0 {
			m.config.BinauralCarrier = 20000.0
		}
		updateActiveBinauralFrequencies(m.config)
	case settingsBinauralBeat:
		m.config.BinauralBeat += float64(delta) * 0.5
		if m.config.BinauralBeat < 0.1 {
			m.config.BinauralBeat = 0.1
		} else if m.config.BinauralBeat > 100.0 {
			m.config.BinauralBeat = 100.0
		}
		updateActiveBinauralFrequencies(m.config)
	case settingsFadeIn:
		m.config.FadeInDuration += delta * 100
		if m.config.FadeInDuration < 0 {
			m.config.FadeInDuration = 0
		} else if m.config.FadeInDuration > 10000 {
			m.config.FadeInDuration = 10000
		}
	case settingsFadeOut:
		m.config.FadeOutDuration += delta * 100
		if m.config.FadeOutDuration < 0 {
			m.config.FadeOutDuration = 0
		} else if m.config.FadeOutDuration > 10000 {
			m.config.FadeOutDuration = 10000
		}
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

func (m *model) runGuardianCommand(cmdStr string) {
	if strings.TrimSpace(cmdStr) == "" {
		return
	}
	m.logInternal("GUARDIAN: Running command: %s", cmdStr)
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	if err := cmd.Run(); err != nil {
		m.logInternal("GUARDIAN: Command failed: %v", err)
	} else {
		m.logInternal("GUARDIAN: Command completed successfully")
	}
}

func (m *model) setAppError(err error, context string) {
	if err == nil {
		return
	}
	msg := err.Error()
	if context != "" {
		msg = fmt.Sprintf("%s: %v", context, err)
	}
	m.appError = msg
	m.logInternal("ERROR: %s", msg)
}

func (m *model) setNotificationStatus(status string) {
	m.notificationStatus = status
	if status != "" {
		m.logInternal("STATUS: %s", status)
	}
}

func (m *model) logInternal(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	m.internalLogs = append(m.internalLogs, entry)
	if len(m.internalLogs) > 100 {
		m.internalLogs = m.internalLogs[len(m.internalLogs)-100:]
	}
}

func defaultOutboxFile() string { return "notification_outbox.json" }
func defaultBackupFile() string { return "backup.json" }

func reloadProjectState(m *model) error {
	cfg, err := loadConfig(m.configFile)
	if err != nil {
		return err
	}
	templates, err := loadSessionTemplates(m.templateFile)
	if err != nil {
		return err
	}
	entries, err := loadEntries(m.dataFile)
	if err != nil {
		return err
	}
	outbox, err := loadNotificationOutbox(m.outboxFile)
	if err != nil {
		return err
	}
	soundscapes, _ := loadSoundscapes(cfg.SoundscapesDir)
	fileTasks := loadTasksFromFile(cfg.TasksFile)
	m.config = cfg
	m.templates = templates
	m.entries = entries
	m.taskSuggestions = buildTaskSuggestions(entries, cfg.PinnedTasks, fileTasks)
	m.suggestionIndex = -1
	m.streakState = computeStreakState(entries)
	m.notificationOutbox = outbox
	m.soundscapes = soundscapes
	if len(m.templates) > 0 && (m.templateIndex < 0 || m.templateIndex >= len(m.templates)) {
		m.templateIndex = 0
	}
	return nil
}

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

func normalizeLayout(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range layoutOrder {
		if candidate == name {
			return candidate
		}
	}
	return defaultConfig.Layout
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

func layoutLabel(name string) string {
	return strings.Title(normalizeLayout(name))
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
	var successes []string
	var failures []string

	if cfg.DesktopNotifications {
		if err := sendDesktopNotification(title, body); err == nil {
			successes = append(successes, "desktop")
		} else {
			failures = append(failures, fmt.Sprintf("desktop: %v", err))
		}
	}

	if cfg.SoundCommand != "" {
		var soundErr error
		if runtime.GOOS == "windows" {
			soundErr = exec.Command("cmd", "/c", cfg.SoundCommand).Run()
		} else {
			soundErr = exec.Command("sh", "-c", cfg.SoundCommand).Run()
		}
		if soundErr == nil {
			successes = append(successes, "sound")
		} else {
			failures = append(failures, fmt.Sprintf("sound: %v", soundErr))
		}
	}

	if token := strings.TrimSpace(cfg.TelegramBotToken); token != "" && strings.TrimSpace(cfg.TelegramChatID) != "" {
		if err := sendTelegramMessage(token, strings.TrimSpace(cfg.TelegramChatID), body); err == nil {
			successes = append(successes, "telegram")
		} else {
			failures = append(failures, fmt.Sprintf("telegram: %v", err))
		}
	}

	if len(successes) > 0 {
		status := "Delivered via " + strings.Join(successes, ", ")
		if len(failures) > 0 {
			status += " (failed: " + strings.Join(failures, "; ") + ")"
		}
		return status, nil
	}

	if len(failures) > 0 {
		return "", fmt.Errorf("all channels failed: %s", strings.Join(failures, "; "))
	}
	return "No active notification channels", nil
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
Add-Type -AssemblyName System.Windows.Forms
$notify = New-Object System.Windows.Forms.NotifyIcon
$notify.Icon = [System.Drawing.SystemIcons]::Information
$notify.BalloonTipTitle = '%s'
$notify.BalloonTipText = '%s'
$notify.Visible = $true
$notify.ShowBalloonTip(3000)
$notify.Dispose()
`, psEscape(title), psEscape(body))
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script).Run()
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

	if m.petEnabled {
		leveledUp := false
		coinsGained := 0
		if sessionType == "work" {
			// Feed pet based on work duration (minutes)
			leveledUp = m.petState.Feed(duration / 60)
			coinsGained = duration / 60
			if coinsGained < 5 {
				coinsGained = 5 // Minimum reward
			}
			m.petState.Coins += coinsGained
			m.logInternal("PET: Work block complete. Gained XP and earned %d Pomo-Coins! (Total: %d)", coinsGained, m.petState.Coins)
		} else {
			// Award XP for break completion
			leveledUp = m.petState.AddXP(20)
			coinsGained = 5
			m.petState.Coins += coinsGained
			m.logInternal("PET: Break complete. Gained 20 XP and earned 5 Pomo-Coins! (Total: %d)", m.petState.Coins)
		}

		// Random chance (15%) to discover a cosmetic item if a work session completes
		if sessionType == "work" && rand.Float64() < 0.15 && m.petState.ActiveItem == "" {
			items := []string{"Wizard Hat", "Cyber Visor", "Golden Crown", "Laser Goggles", "Mini Cape"}
			rand.Seed(time.Now().UnixNano())
			discovered := items[rand.Intn(len(items))]
			m.petState.ActiveItem = discovered
			m.petState.AddXP(50) // Bonus XP for finding a rare item
			m.logInternal("PET: %s found a rare item: %s!", m.petState.Name, discovered)
		}

		if err := SavePetState("pet.json", m.petState); err != nil {
			m.setAppError(err, "Failed to save pet state")
		}

		if leveledUp {
			m.logInternal("PET: Level Up! %s reached Level %d", m.petState.Name, m.petState.Level)
			m.showPetLevelUpOverlay = true
		}
	}

	m.logInternal("SESSION: Saved %s (%s)", m.taskName, formatDuration(duration))
	m.entries = entries
	fileTasks := loadTasksFromFile(m.config.TasksFile)
	m.taskSuggestions = buildTaskSuggestions(entries, m.config.PinnedTasks, fileTasks)
	m.suggestionIndex = -1
	return nil
}

func loadTasksFromFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var tasks []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

func buildTaskSuggestions(entries []Entry, pinned []string, fileTasks []string) []string {
	seen := make(map[string]struct{})
	var suggestions []string

	// 1. Pinned tasks (highest priority)
	for _, t := range pinned {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			suggestions = append(suggestions, t)
		}
	}

	// 2. File tasks
	for _, t := range fileTasks {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			suggestions = append(suggestions, t)
		}
	}

	// 3. Recent tasks from history
	for i := len(entries) - 1; i >= 0; i-- {
		t := strings.TrimSpace(entries[i].Task)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			suggestions = append(suggestions, t)
		}
	}
	return suggestions
}

func (m model) applyTaskSuggestion(delta int) model {
	if len(m.taskSuggestions) == 0 {
		return m
	}
	if m.suggestionIndex < 0 || m.suggestionIndex >= len(m.taskSuggestions) {
		m.suggestionIndex = 0
	} else {
		m.suggestionIndex = (m.suggestionIndex + delta + len(m.taskSuggestions)) % len(m.taskSuggestions)
	}
	m.textInput.SetValue(m.taskSuggestions[m.suggestionIndex])
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
	m.lastDeletedTemplate = &deletedTemplateState{
		template:  removed,
		index:     m.templateIndex,
		expiresAt: time.Now().Add(10 * time.Second),
	}
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
	m.notificationStatus = fmt.Sprintf("Deleted template: %s (Ctrl+Z to undo)", removed.Name)
	return nil
}

func (m *model) undoLastTemplateDelete() error {
	if m.lastDeletedTemplate == nil {
		return fmt.Errorf("no deleted template to undo")
	}
	if time.Now().After(m.lastDeletedTemplate.expiresAt) {
		m.lastDeletedTemplate = nil
		return fmt.Errorf("deleted template can no longer be restored")
	}
	restore := *m.lastDeletedTemplate
	if restore.index < 0 {
		restore.index = 0
	}
	if restore.index > len(m.templates) {
		restore.index = len(m.templates)
	}
	m.templates = append(m.templates, SessionTemplate{})
	copy(m.templates[restore.index+1:], m.templates[restore.index:])
	m.templates[restore.index] = restore.template
	m.templateIndex = restore.index
	if err := saveSessionTemplates(m.templateFile, m.templates); err != nil {
		return err
	}
	m.lastDeletedTemplate = nil
	updated := m.applySelectedTemplate()
	*m = updated
	m.notificationStatus = fmt.Sprintf("Restored template: %s", restore.template.Name)
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
	case "analytics":
		return renderAnalyticsView(m)
	case "heatmap":
		return renderHeatmapView(m)
	case "history":
		return renderHistoryView(m)
	case "report":
		return renderDailyReportView(m)
	case "logs":
		return renderLogView(m)
	case "settings":
		return renderSettingsView(m)
	case "templates":
		return renderTemplateManagerView(m)
	case "soundscapes":
		return renderSoundscapeMenuView(m)
	case "help":
		return renderHelpView(m)
	case "fatal":
		return renderFatalView(m)
	case "tamagotchi":
		return renderTamagotchiView(m)
	default:
		return renderInputView(m)
	}
}

func renderTamagotchiView(m model) string {
	theme := activeTheme(m.config)

	var screenContent string
	if m.tamagotchiActiveMenu == "typing" {
		screenContent = RenderTypingGame(m.typingGame, m.width, theme.accent, theme.primary)
	} else if m.tamagotchiActiveMenu == "guessing" {
		screenContent = RenderBinaryGame(m.binaryGame, m.width, theme.accent, theme.primary)
	} else if m.tamagotchiActiveMenu == "rebirth" {
		subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		
		var rows []string
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color(theme.primary)).Bold(true).Render("   🧬  R E B I R T H   S A N C T U A R Y"))
		rows = append(rows, subtle.Render("   "+strings.Repeat("─", 52)))
		rows = append(rows, "   Your companion has reached the end of their digital cycle.")
		rows = append(rows, "   Please enter a new name for your companion:")
		rows = append(rows, "")
		rows = append(rows, "   > "+m.textInput.View())
		rows = append(rows, "")
		rows = append(rows, subtle.Render("   Press [Enter] to rebirth or [Esc] to cancel"))
		
		for len(rows) < 8 {
			rows = append(rows, "")
		}
		// Pass the prompt as feedbackMsg so that RenderTamagotchiScreen will output it directly in the LCD screen
		screenContent = RenderTamagotchiScreen(m.petState, m.width, m.tamagotchiActiveMenu, m.tamagotchiMenuSelect, strings.Join(rows, "\n"), theme.accent, theme.primary)
	} else {
		screenContent = RenderTamagotchiScreen(m.petState, m.width, m.tamagotchiActiveMenu, m.tamagotchiMenuSelect, m.tamagotchiFeedback, theme.accent, theme.primary)
	}
	return fmt.Sprintf("\n%s\n", screenContent)
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

func renderPetLevelUpCard(pet PetState) string {
	stage := pet.EvolutionStage()
	stageName := "Baby"
	if stage == 2 {
		stageName = "Teenager"
	} else if stage == 3 {
		stageName = "Cyber-Ascended God!"
	}

	return fmt.Sprintf(`
╭───────────────────────────────────────────────────╮
│             🎉   LEVEL UP!  LEVEL UP!   🎉        │
╰───────────────────────────────────────────────────╯

        ★  %s HAS REACHED LEVEL %d! ★

                 Evolution Stage: %s

           "Quack! Gaining power! Thank you!"

[ Press any key to continue... ]`,
		strings.ToUpper(pet.Name), pet.Level, stageName)
}

func renderInputView(m model) string {
	if m.showPetLevelUpOverlay {
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, renderPetLevelUpCard(m.petState)))
	}

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

	recoveryMsg := ""
	if m.streakState.RecoveryAvailable {
		recoveryMsg = "✦ Recovery mode — complete a session today to save your streak!"
	} else if m.streakState.RecoveryNeeded {
		recoveryMsg = "◌ Streak lost — start fresh today and rebuild your momentum!"
	}

	recentOverlay := ""
	if m.showRecentOverlay && len(m.taskSuggestions) > 0 {
		limit := 5
		if len(m.taskSuggestions) < limit {
			limit = len(m.taskSuggestions)
		}
		overlayLines := []string{"Suggested tasks:"}
		for i := 0; i < limit; i++ {
			cursor := "  "
			if i == m.suggestionIndex {
				cursor = "> "
			}
			overlayLines = append(overlayLines, cursor+m.taskSuggestions[i])
		}
		recentOverlay = strings.Join(overlayLines, "\n")
	}

	hintLine := "[Tab] Switch Field   [Enter] Start/Apply   [Ctrl+T] Save Template   [Ctrl+M] Soundscapes   [?] Help   [q] Quit"
	if m.petEnabled {
		hintLine += "   [Ctrl+G] Toggle Pet"
	}

	inputForm := fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📝  What are you working on?      │
╰─────────────────────────────────────╯

%s
%s
%s

%s

%s

%s

%s

%s

%s
Templates: Left/Right while Template is focused   Up/Down to browse suggested tasks`,
		templateLine, recoveryMsg, recentOverlay, m.textInput.View(), m.durationInput.View(), m.noteInput.View(), m.tagInput.View(), errorBlock, hintLine)

	if m.petEnabled && m.showPetSidebar && m.width >= 90 {
		m.petState.UpdateMood(m.running, m.mode, m.sessionStart)
		petBox := RenderPetBox(m.petState, m.width)

		formFrame := lipgloss.NewStyle().Padding(0, 1).Render(inputForm)

		theme := activeTheme(m.config)
		petFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.primary)).
			Padding(0, 1).
			Render(petBox)

		block := lipgloss.JoinHorizontal(lipgloss.Center, formFrame, petFrame)
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
	}

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, inputForm))
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
	layout := normalizeLayout(m.config.Layout)

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

	petHint := ""
	if m.petEnabled {
		petHint = "  [Ctrl+G] Pet"
	}
	hint := "[Space] Pause  [E] Edit  [Enter] End  [Tab] Stats  [S] Settings  [Ctrl+M] Soundscapes  [?] Help" + petHint
	if m.guardianLocked {
		hint += "  [Esc] Abort"
	} else {
		hint += "  [q] Quit"
	}
	if !m.running {
		hint = "[Space] Resume  [E] Edit  [Enter] End  [Tab] Stats  [S] Settings  [Ctrl+M] Soundscapes  [?] Help" + petHint
		if m.guardianLocked {
			hint += "  [Esc] Abort"
		} else {
			hint += "  [q] Quit"
		}
	}

	header := fmt.Sprintf("%s • %s", modeStr, m.taskName)
	if m.guardianLocked {
		header = "🔒 Guardian Active • " + header
	}
	if tags := strings.Join(m.currentSessionTags(), ", "); tags != "" {
		header += fmt.Sprintf(" [%s]", tags)
	}
	if m.streakState.Current > 0 {
		header += fmt.Sprintf("  🔥%d", m.streakState.Current)
	} else if m.streakState.RecoveryAvailable {
		header += "  ✦ recoverable"
	} else if m.streakState.RecoveryNeeded {
		header += "  ◌ rebuild"
	}
	if m.activeSoundscapeCmd != nil && m.soundscapeIndex >= 0 && m.soundscapeIndex < len(m.soundscapes) {
		track := strings.TrimSuffix(m.soundscapes[m.soundscapeIndex], filepath.Ext(m.soundscapes[m.soundscapeIndex]))
		header += fmt.Sprintf("  🎵 %s", track)
	}

	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	details := hint
	if errorLine != "" {
		details = fmt.Sprintf("%s\n%s", errorLine, hint)
	}
	if statusLine != "" {
		details = fmt.Sprintf("%s\n%s", details, statusLine)
	}

	var block string
	switch layout {
	case "minimal":
		timerLine := themedStyle(m.config, theme.accent).Bold(true).Render(timeStr)
		block = fmt.Sprintf("%s\n\n%s  %s\n\n%s", header, timerLine, progress, details)
	case "compact":
		timerFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 1).
			Render(fmt.Sprintf("%s  %s", themedStyle(m.config, theme.accent).Bold(true).Render(timeStr), progress))
		block = fmt.Sprintf("%s\n\n%s\n\n%s", header, timerFrame, details)
	default: // classic
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
		block = fmt.Sprintf("%s\n\n%s\n\n%s", header, timerFrame, details)
	}

	if m.petEnabled && m.showPetSidebar && m.width >= 90 {
		m.petState.UpdateMood(m.running, m.mode, m.sessionStart)
		petBox := RenderPetBox(m.petState, m.width)

		timerFrame := lipgloss.NewStyle().Padding(0, 1).Render(block)

		theme := activeTheme(m.config)
		petFrame := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.primary)).
			Padding(0, 1).
			Render(petBox)

		joinedBlock := lipgloss.JoinHorizontal(lipgloss.Center, timerFrame, petFrame)
		return fmt.Sprintf("\n%s\n", centerBlock(m.width, joinedBlock))
	}

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

func renderStatsTabs(currentMode string, cfg Config) string {
	tabs := []struct {
		mode  string
		label string
	}{
		{"stats", "Dashboard"},
		{"analytics", "Analytics"},
		{"heatmap", "Heatmap"},
		{"history", "Timeline"},
		{"report", "Report"},
	}

	theme := activeTheme(cfg)
	var renderedTabs []string

	for _, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if t.mode == currentMode {
			style = style.
				Foreground(lipgloss.Color(theme.accent)).
				Bold(true).
				Underline(true)
		} else {
			style = style.Foreground(lipgloss.Color(theme.primary))
		}
		renderedTabs = append(renderedTabs, style.Render(t.label))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	return "\n" + row + "\n" + themedStyle(cfg, theme.primary).Render(strings.Repeat("─", lipgloss.Width(row))) + "\n"
}

func renderStatsView(m model) string {
	weeklyData := getWeeklyData(m.entries)
	barChart := renderWeeklyBarChart(weeklyData)
	streakChart := renderStreakHistoryChart(m.entries)

	daily := formatDuration(getDailyTotal(m.entries, "work"))
	streak := computeStreakState(m.entries)
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
	errorLine := renderAppError(m)
	if emptyMessage != "" {
		emptyMessage = fmt.Sprintf("\n%s\n", emptyMessage)
	}

	tabs := renderStatsTabs("stats", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	block := fmt.Sprintf(`%s
%s

┌─────────────────┐
│  📅  Today      │
│  %-13s  │
└─────────────────┘

┌─────────────────┐
│  🔥  Streaks    │
│  Current: %-3d  │
│  Best: %-7d│
└─────────────────┘

┌─────────────────┐
│  Recovery       │
│  %-13s  │
└─────────────────┘

┌─────────────────┐
│  ⚖️  Ratio      │
│  Work: %d%%     │
│  Break: %d%%    │
└─────────────────┘

Weekly Activity (7 days):

%s

Streak History (14 days):

%s

%s

%s

%s
`, renderBanner(m.config), tabs, daily, streak.Current, streak.Best, recoveryLabel(streak), workRatio, 100-workRatio, emptyMessage, tagSummary, barChart, streakChart, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderHorizontalProgressBar(percent float64, totalWidth int, theme themeStyle, useAccent bool) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filledLength := int(math.Round(percent * float64(totalWidth) / 100.0))
	if filledLength < 0 {
		filledLength = 0
	}
	if filledLength > totalWidth {
		filledLength = totalWidth
	}
	emptyLength := totalWidth - filledLength

	filledChar := "█"
	emptyChar := "░"

	color := theme.primary
	if useAccent {
		color = theme.accent
	}

	filledStr := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat(filledChar, filledLength))
	emptyStr := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.notice)).Render(strings.Repeat(emptyChar, emptyLength))

	return filledStr + emptyStr
}

func renderDashboardCard(title string, content string, theme themeStyle, width int) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.primary)).
		Padding(0, 1).
		Width(width)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.accent)).
		Bold(true)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.primary)).
		Render(strings.Repeat("─", width-2))

	cardContent := titleStyle.Render(title) + "\n" + divider + "\n" + content
	return borderStyle.Render(cardContent)
}

func renderTopDurationBars(totals map[string]int, totalSeconds int, limit int, theme themeStyle, useAccent bool, barWidth int) []string {
	if len(totals) == 0 {
		return nil
	}
	type item struct {
		name    string
		seconds int
	}
	items := make([]item, 0, len(totals))
	for name, seconds := range totals {
		items = append(items, item{name: name, seconds: seconds})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].seconds == items[j].seconds {
			return items[i].name < items[j].name
		}
		return items[i].seconds > items[j].seconds
	})
	if len(items) < limit {
		limit = len(items)
	}

	lines := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		pct := 0.0
		if totalSeconds > 0 {
			pct = float64(items[i].seconds) * 100.0 / float64(totalSeconds)
		}

		progressBar := renderHorizontalProgressBar(pct, barWidth, theme, useAccent)
		durStr := formatDuration(items[i].seconds)

		name := items[i].name
		if len(name) > 12 {
			name = name[:9] + "..."
		}

		lines = append(lines, fmt.Sprintf("  %-12s %s %7s (%.1f%%)", name, progressBar, durStr, pct))
	}
	return lines
}

func renderAnalyticsView(m model) string {
	taskTotals, tagTotals, summary := buildAnalyticsSummary(m.entries)
	tabs := renderStatsTabs("analytics", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	theme := activeTheme(m.config)

	// Responsive Card Width
	cardWidth := 60
	if m.width > 0 && m.width < 70 {
		cardWidth = m.width - 6
	}
	if cardWidth < 30 {
		cardWidth = 30
	}

	// Bar width inside card: cardWidth - padding(4) - name(14) - duration(16)
	barWidth := cardWidth - 34
	if barWidth < 6 {
		barWidth = 6
	}

	// 1. PRODUCTIVITY SUMMARY CARD
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString(fmt.Sprintf("  Sessions analyzed: %d\n", summary.totalSessions))
	summaryBuilder.WriteString(fmt.Sprintf("  Work time:         %s\n", formatDuration(summary.workSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Break time:        %s\n", formatDuration(summary.breakSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Average session:   %s\n", formatDuration(summary.averageSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Longest session:   %s\n", formatDuration(summary.longestSeconds)))
	summaryBuilder.WriteString(fmt.Sprintf("  Busiest day:       %s", summary.busiestDay))
	summaryCard := renderDashboardCard("PRODUCTIVITY SUMMARY", summaryBuilder.String(), theme, cardWidth)

	// 2. TOP TASKS CARD
	taskLines := renderTopDurationBars(taskTotals, summary.workSeconds, 5, theme, true, barWidth)
	var tasksContent string
	if len(taskLines) == 0 {
		tasksContent = "  No task breakdown yet."
	} else {
		tasksContent = strings.Join(taskLines, "\n")
	}
	tasksCard := renderDashboardCard("Top tasks:", tasksContent, theme, cardWidth)

	// 3. TOP TAGS CARD
	totalTagSeconds := 0
	for _, secs := range tagTotals {
		totalTagSeconds += secs
	}
	tagLines := renderTopDurationBars(tagTotals, totalTagSeconds, 5, theme, false, barWidth)
	var tagsContent string
	if len(tagLines) == 0 {
		tagsContent = "  No tag breakdown yet."
	} else {
		tagsContent = strings.Join(tagLines, "\n")
	}
	tagsCard := renderDashboardCard("Top tags:", tagsContent, theme, cardWidth)

	block := fmt.Sprintf(`%s
%s

%s

%s

%s

%s
`, renderBanner(m.config), tabs, summaryCard, tasksCard, tagsCard, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderHeatmapView(m model) string {
	heatmap := renderActivityHeatmap(m.entries, m.config, m.width)
	tabs := renderStatsTabs("heatmap", m.config)
	footer := "[Tab] Cycle Views   [S] Settings   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}

	block := fmt.Sprintf(`%s
%s

%s

%s
`, renderBanner(m.config), tabs, heatmap, footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderActivityHeatmap(entries []Entry, cfg Config, width int) string {
	dayTotals := make(map[string]int)
	for _, entry := range entries {
		if entry.Type == "work" {
			dayTotals[dateKey(entry.Start)] += entry.Duration
		}
	}

	// Each column is 2 chars wide (block + space)
	// Left labels: "Sun " (4 chars)
	// We want to fit within the width
	maxWeeks := (width - 8) / 2
	if maxWeeks > 52 {
		maxWeeks = 52
	}
	if maxWeeks < 4 {
		return "Terminal too narrow for heatmap."
	}

	today := time.Now()
	// Current week's Sunday
	currentSunday := today.AddDate(0, 0, -int(today.Weekday()))
	startDate := currentSunday.AddDate(0, 0, -7*(maxWeeks-1))

	var b strings.Builder

	// Month labels
	b.WriteString("    ")
	lastMonth := -1
	for w := 0; w < maxWeeks; w++ {
		date := startDate.AddDate(0, 0, w*7)
		if int(date.Month()) != lastMonth {
			label := date.Format("Jan")
			b.WriteString(label)
			if len(label) < 2 {
				b.WriteString(" ")
			}
			lastMonth = int(date.Month())
			// Skip columns covered by the label
			skip := (len(label) + 1) / 2
			for i := 1; i < skip && w+i < maxWeeks; i++ {
				w++
			}
		} else {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n")

	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	theme := activeTheme(cfg)

	for d := 0; d < 7; d++ {
		b.WriteString(fmt.Sprintf("%3s ", days[d]))
		for w := 0; w < maxWeeks; w++ {
			date := startDate.AddDate(0, 0, w*7+d)
			if date.After(today) {
				b.WriteString("  ")
				continue
			}

			seconds := dayTotals[dateKey(date)]
			b.WriteString(renderHeatmapBlock(seconds, cfg, theme) + " ")
		}
		b.WriteString("\n")
	}

	// Legend
	b.WriteString("\n    Less ")
	b.WriteString(renderHeatmapBlock(0, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(15*60, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(45*60, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(90*60, cfg, theme) + " ")
	b.WriteString(renderHeatmapBlock(150*60, cfg, theme) + " ")
	b.WriteString(" More\n")

	return b.String()
}

func renderHeatmapBlock(seconds int, cfg Config, theme themeStyle) string {
	minutes := seconds / 60
	style := lipgloss.NewStyle()

	if minutes <= 0 {
		return style.Foreground(lipgloss.Color("8")).Render("·")
	}

	if minutes < 30 {
		return style.Foreground(lipgloss.Color(theme.primary)).Faint(true).Render("█")
	} else if minutes < 60 {
		return style.Foreground(lipgloss.Color(theme.primary)).Render("█")
	} else if minutes < 120 {
		return style.Foreground(lipgloss.Color(theme.primary)).Bold(true).Render("█")
	} else {
		return style.Foreground(lipgloss.Color(theme.accent)).Render("█")
	}
}

type analyticsSummary struct {
	totalSessions  int
	workSeconds    int
	breakSeconds   int
	averageSeconds int
	longestSeconds int
	busiestDay     string
}

func buildAnalyticsSummary(entries []Entry) (map[string]int, map[string]int, analyticsSummary) {
	taskTotals := make(map[string]int)
	tagTotals := make(map[string]int)
	dayTotals := make(map[string]int)
	summary := analyticsSummary{busiestDay: "n/a"}
	for _, entry := range entries {
		summary.totalSessions++
		if entry.Type == "break" {
			summary.breakSeconds += entry.Duration
		} else {
			summary.workSeconds += entry.Duration
		}
		if entry.Duration > summary.longestSeconds {
			summary.longestSeconds = entry.Duration
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
		dayTotals[dateKey(entry.Start)] += entry.Duration
	}

	if summary.totalSessions > 0 {
		summary.averageSeconds = (summary.workSeconds + summary.breakSeconds) / summary.totalSessions
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
			summary.busiestDay = fmt.Sprintf("%s (%s)", parsed.Format("Mon, Jan 02"), formatDuration(maxTotal))
		}
	}

	return taskTotals, tagTotals, summary
}

func renderHistoryView(m model) string {
	entries := make([]Entry, len(m.entries))
	copy(entries, m.entries)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Start.Equal(entries[j].Start) {
			return entries[i].End.After(entries[j].End)
		}
		return entries[i].Start.After(entries[j].Start)
	})

	lines := []string{"Recent Sessions by Day:"}
	if len(entries) == 0 {
		lines = append(lines, "  No sessions recorded yet.")
	} else {
		limit := 15
		if len(entries) < limit {
			limit = len(entries)
		}
		grouped := make(map[string][]Entry)
		order := make([]string, 0, len(entries))
		for i := 0; i < limit; i++ {
			entry := entries[i]
			key := dateKey(entry.Start)
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
			lines = append(lines, fmt.Sprintf("  %s  (%d sessions, %s)", dayLabel, len(groupEntries), formatDuration(dayTotal)))
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
				meta := []string{when, kind, formatDuration(entry.Duration)}
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
		if len(entries) > limit {
			lines = append(lines, fmt.Sprintf("  ... and %d more sessions", len(entries)-limit))
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

func buildDailyReport(entries []Entry, day time.Time) []string {
	dayKey := dateKey(day)
	workSeconds := 0
	breakSeconds := 0
	var dayEntries []Entry
	for _, entry := range entries {
		if dateKey(entry.Start) != dayKey {
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
		fmt.Sprintf("- Work: %s", formatDuration(workSeconds)),
		fmt.Sprintf("- Break: %s", formatDuration(breakSeconds)),
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
			formatDuration(entry.Duration),
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

func renderDailyReportView(m model) string {
	lines := buildDailyReport(m.entries, time.Now())
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
	path := fmt.Sprintf("kairu-report-%s.md", dateKey(time.Now()))
	lines := buildDailyReport(m.entries, time.Now())
	data := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func renderLogView(m model) string {
	lines := make([]string, 0, len(m.internalLogs))
	// Show newest at bottom, or maybe newest at top?
	// Heatmap/History show oldest at top usually.
	// Logs are usually read top to bottom.
	for _, l := range m.internalLogs {
		lines = append(lines, "  "+l)
	}
	if len(lines) == 0 {
		lines = append(lines, "  (No logs recorded yet)")
	}

	footer := "[Esc] Back   [q] Quit"
	block := renderBanner(m.config) + "\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  Internal Event Logs                │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		strings.Join(lines, "\n") + "\n\n" +
		footer
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderSettingsView(m model) string {
	footer := "[Tab] Switch   [Space] Toggle   [Left/Right] Adjust   [Enter] Run action   [Esc] Back   [q] Quit"
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	if statusLine != "" {
		footer = fmt.Sprintf("%s\n%s", statusLine, footer)
	}

	leftColumn := strings.Join([]string{
		renderSettingsSection(m.config, "Appearance", []string{
			renderSettingLine(m.settingsCursor == settingsTheme, "Theme", themeLabel(m.config.Theme)),
			renderSettingLine(m.settingsCursor == settingsFont, "Timer font", fontLabel(m.config.Font)),
			renderSettingLine(m.settingsCursor == settingsLayout, "Timer layout", layoutLabel(m.config.Layout)),
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
		renderSettingsSection(m.config, "Synthesizer & Audio", []string{
			renderSettingLine(m.settingsCursor == settingsSynthVolume, "Synth volume", renderVolumeBar(m.config.SynthVolume)),
			renderSettingLine(m.settingsCursor == settingsBinauralPreset, "Binaural preset", binauralPresetLabel(m.config.BinauralPreset)),
			renderSettingLine(m.settingsCursor == settingsBinauralCarrier, "Binaural carrier", customOrPresetCarrier(m)),
			renderSettingLine(m.settingsCursor == settingsBinauralBeat, "Binaural detune", customOrPresetBeat(m)),
			renderSettingLine(m.settingsCursor == settingsFadeIn, "Audio fade-in speed", fmt.Sprintf("%d ms", m.config.FadeInDuration)),
			renderSettingLine(m.settingsCursor == settingsFadeOut, "Audio fade-out speed", fmt.Sprintf("%d ms", m.config.FadeOutDuration)),
		}),
		renderSettingsSection(m.config, "Backup/Tools", []string{
			renderSettingLine(m.settingsCursor == settingsBackup, "Create backup", "Write snapshot to backup.json"),
			renderSettingLine(m.settingsCursor == settingsRestore, "Restore backup", "Load snapshot from backup.json"),
			renderSettingLine(m.settingsCursor == settingsClearOutbox, "Clear queue", fmt.Sprintf("Delete %d pending notifications", len(m.notificationOutbox))),
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
	footer := "[Tab/Arrows] Browse   [Enter] Apply   [Ctrl+T] Save current form   [Ctrl+R] Rename   [Ctrl+D] Delete   [Ctrl+Z] Undo delete   [Ctrl+Y] Duplicate   [Ctrl+P/Esc] Back   [?] Help   [q] Quit"
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
			tagStr := ""
			if len(template.Tags) > 0 {
				tagStr = fmt.Sprintf(" [%s]", strings.Join(template.Tags, ", "))
			}
			listLines = append(listLines, fmt.Sprintf("%s%s (%s)%s", prefix, template.Name, template.Duration, tagStr))
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

func renderSoundscapeMenuView(m model) string {
	lines := []string{"Select a Soundscape (Work Only):"}
	noneLabel := "  [ ] None"
	if m.soundscapeIndex == -1 {
		noneLabel = "  [*] None"
	}
	lines = append(lines, noneLabel)

	for i, track := range m.soundscapes {
		prefix := "  [ ] "
		if i == m.soundscapeIndex {
			prefix = "  [*] "
		}
		lines = append(lines, prefix+track)
	}

	if len(m.soundscapes) == 0 {
		lines = append(lines, "", "  (No audio files found in "+m.config.SoundscapesDir+")")
	}

	footer := "[Enter] Select   [Esc/Ctrl+M] Cancel"
	block := renderBanner(m.config) + "\n\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  🎵  Soundscapes                   │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		strings.Join(lines, "\n") + "\n\n" +
		footer

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
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
	layoutLine := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.notice)).Render("Layout: " + layoutLabel(m.config.Layout))
	timerBlock := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.accent)).Render(strings.Join(lines, "\n"))

	return fmt.Sprintf(`╭─────────────────────────────────────╮
│  Live Preview                       │
╰─────────────────────────────────────╯

%s
%s
%s
%s`, themeLine, fontLine, layoutLine, timerBlock)
}

func renderSettingsHintRow(m model) string {
	switch m.settingsCursor {
	case settingsTheme:
		return "Theme: Left/Right cycles presets. Space also cycles."
	case settingsFont:
		return "Typography: Left/Right cycles timer fonts. Space also cycles."
	case settingsLayout:
		return "Layout: Left/Right cycles timer layouts. Space also cycles."
	case settingsQuietStart, settingsQuietEnd:
		return "Quiet hours: Left/Right adjusts the hour."
	case settingsNotifications, settingsDesktop, settingsWorkComplete, settingsBreakComplete, settingsSessionStart, settingsSessionEnd, settingsPauseResume, settingsEndingSoon:
		return "Toggles: Space, Enter, or Left/Right flips the setting."
	case settingsSynthVolume:
		return "Synth Volume: Left/Right adjusts. Space cycles in 10% steps."
	case settingsBinauralPreset:
		return "Binaural Preset: Left/Right or Space cycles brainwave presets."
	case settingsBinauralCarrier:
		if strings.ToLower(m.config.BinauralPreset) != "custom" {
			return "Binaural Carrier: Locked to preset. Choose 'Custom' preset to edit."
		}
		return "Binaural Carrier: Left/Right adjusts carrier frequency by 5 Hz."
	case settingsBinauralBeat:
		if strings.ToLower(m.config.BinauralPreset) != "custom" {
			return "Binaural Beat: Locked to preset. Choose 'Custom' preset to edit."
		}
		return "Binaural Beat: Left/Right adjusts detuning gap by 0.5 Hz."
	case settingsFadeIn:
		return "Fade-in duration: Left/Right or Space adjusts the speed (ms)."
	case settingsFadeOut:
		return "Fade-out duration: Left/Right or Space adjusts the speed (ms)."
	case settingsBackup:
		return "Backup: Enter writes a project snapshot to backup.json."
	case settingsRestore:
		return "Restore: Enter loads backup.json and overwrites local project files."
	case settingsClearOutbox:
		return "Outbox: Enter clears the notification retry queue."
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

func renderVolumeBar(vol float64) string {
	pct := int(vol * 100)
	filled := int(vol * 10)
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("[%s] %d%%", bar, pct)
}

func binauralPresetLabel(preset string) string {
	switch strings.ToLower(preset) {
	case "alpha":
		return "Alpha (Relaxed Focus, 10Hz)"
	case "beta":
		return "Beta (Logical Work, 20Hz)"
	case "theta":
		return "Theta (Deep Insight, 6Hz)"
	case "delta":
		return "Delta (Deep Restore, 3Hz)"
	case "custom":
		return "Custom (Tune below)"
	default:
		return "Alpha (Relaxed Focus, 10Hz)"
	}
}

func customOrPresetCarrier(m model) string {
	if strings.ToLower(m.config.BinauralPreset) != "custom" {
		c, _ := getPresetFrequencies(m.config.BinauralPreset)
		return fmt.Sprintf("%.1f Hz (Locked)", c)
	}
	return fmt.Sprintf("%.1f Hz", m.config.BinauralCarrier)
}

func customOrPresetBeat(m model) string {
	if strings.ToLower(m.config.BinauralPreset) != "custom" {
		_, b := getPresetFrequencies(m.config.BinauralPreset)
		return fmt.Sprintf("%.1f Hz (Locked)", b)
	}
	return fmt.Sprintf("%.1f Hz", m.config.BinauralBeat)
}

func getPresetFrequencies(preset string) (float64, float64) {
	switch strings.ToLower(preset) {
	case "alpha":
		return 120.0, 10.0
	case "beta":
		return 150.0, 20.0
	case "theta":
		return 100.0, 6.0
	case "delta":
		return 70.0, 3.0
	default:
		return 120.0, 10.0
	}
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
		formatHelpLine("Ctrl+P", "Templates"),
		formatHelpLine("Ctrl+M", "Soundscapes"),
		"",
		"Timer/Break:",
		formatHelpLine("Space", "Pause/Resume"),
		formatHelpLine("E", "Edit time"),
		formatHelpLine("Enter", "End session"),
		formatHelpLine("Tab", "Stats"),
		formatHelpLine("S", "Settings"),
		formatHelpLine("Ctrl+M", "Soundscapes"),
		"",
		"Edit:",
		formatHelpLine("Enter", "Apply"),
		formatHelpLine("Esc", "Cancel"),
		"",
		"Stats Views:",
		formatHelpLine("Tab", "Cycle views"),
		formatHelpLine("Esc", "Back to timer"),
		formatHelpLine("R", "Daily report"),
		formatHelpLine("S", "Settings"),
		formatHelpLine("L", "Internal logs"),
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

func renderStreakHistoryChart(entries []Entry) string {
	streakDays := make(map[string]bool)
	for _, e := range entries {
		if e.Type == "work" {
			streakDays[dateKey(e.Start)] = true
		}
	}
	if len(streakDays) == 0 {
		return "No streak history yet."
	}

	var b strings.Builder
	for i := 13; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i)
		key := dateKey(day)
		marker := "·"
		if streakDays[key] {
			marker = "█"
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", day.Format("Jan 02"), marker, statusForStreakDay(streakDays, day)))
	}
	return b.String()
}

func statusForStreakDay(days map[string]bool, day time.Time) string {
	key := dateKey(day)
	if days[key] {
		return "work logged"
	}
	return "no work"
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

func computeStreakState(entries []Entry) StreakState {
	days := make(map[string]bool)
	for _, e := range entries {
		if e.Type == "work" {
			days[dateKey(e.Start)] = true
		}
	}
	if len(days) == 0 {
		return StreakState{}
	}

	var list []string
	for d := range days {
		list = append(list, d)
	}
	sort.Strings(list)

	best, temp := 0, 0
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
			if temp > best {
				best = temp
			}
			temp = 1
		}
		last = date
	}
	if temp > best {
		best = temp
	}

	today := dateKey(time.Now())
	lastWorkDay := list[len(list)-1]
	current := 0
	recovery := false
	recoveryNeeded := false
	if days[today] {
		for i := 0; i < 365; i++ {
			if days[dateKey(time.Now().AddDate(0, 0, -i))] {
				current++
			} else if i > 0 {
				break
			}
		}
	} else {
		recoveryNeeded = true
		yesterday := dateKey(time.Now().AddDate(0, 0, -1))
		recovery = days[yesterday]
	}

	return StreakState{
		Current:           current,
		Best:              best,
		LastWorkDay:       lastWorkDay,
		RecoveryAvailable: recovery,
		RecoveryNeeded:    recoveryNeeded,
		RecoveryPrompt:    recoveryPrompt(days),
	}
}

func calculateStreaks(entries []Entry) (int, int) {
	streak := computeStreakState(entries)
	return streak.Current, streak.Best
}

func recoveryPrompt(days map[string]bool) string {
	today := dateKey(time.Now())
	if days[today] {
		return "Streak active today"
	}
	yesterday := dateKey(time.Now().AddDate(0, 0, -1))
	if days[yesterday] {
		return "Recovery mode: one session restores your streak"
	}
	return "Recovery mode: start today to rebuild momentum"
}

func recoveryLabel(streak StreakState) string {
	if streak.Current > 0 {
		return "Active today"
	}
	if streak.RecoveryAvailable {
		return "Recoverable"
	}
	if streak.RecoveryNeeded {
		return "Broken, recover"
	}
	return "No streak yet"
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
	backupPath := flag.String("backup", "", "Backup entries, templates, config, and notification queue to the provided file path")
	restorePath := flag.String("restore", "", "Restore entries, templates, config, and notification queue from the provided file path")
	flag.Parse()

	if *exportPath != "" && *importPath != "" {
		fmt.Println("Error: --export and --import cannot be used together.")
		os.Exit(1)
	}
	if *backupPath != "" && *restorePath != "" {
		fmt.Println("Error: --backup and --restore cannot be used together.")
		os.Exit(1)
	}
	if *backupPath != "" && (*exportPath != "" || *importPath != "") {
		fmt.Println("Error: --backup cannot be combined with --export or --import.")
		os.Exit(1)
	}
	if *restorePath != "" && (*exportPath != "" || *importPath != "") {
		fmt.Println("Error: --restore cannot be combined with --export or --import.")
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
	if *backupPath != "" {
		if err := backupProject(dataFile, "templates.json", "kairu.yaml", defaultOutboxFile(), *backupPath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Backup complete:", *backupPath)
		return
	}
	if *restorePath != "" {
		if err := restoreProject(dataFile, "templates.json", "kairu.yaml", defaultOutboxFile(), *restorePath); err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println("Restore complete:", *restorePath)
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

	petFile := "pet.json"
	pState, petErr := LoadPetState(petFile)
	petEnabled := true
	if petErr != nil {
		pState = DefaultPet("Neko", "kitty")
		if err := SavePetState(petFile, pState); err != nil {
			startupErrors = append(startupErrors, fmt.Sprintf("Failed to initialize default pet: %v", err))
		}
	} else {
		// Catch up on offline state decay
		pState.TickStateDecay(time.Now())
		if pState.IsDead {
			startupErrors = append(startupErrors, "COMPANION DIED OF NEGLECT! 🪦 Enter Tamagotchi screen (Ctrl+T) to rebirth Neko.")
		}
		if err := SavePetState(petFile, pState); err != nil {
			startupErrors = append(startupErrors, fmt.Sprintf("Failed to save caught-up pet state: %v", err))
		}
	}

	mode := "input"
	if fatalConfig {
		mode = "fatal"
	}
	streakState := computeStreakState(entryList)
	if streakState.RecoveryNeeded {
		startupErrors = append(startupErrors, streakState.RecoveryPrompt)
	}
	initialFocus := focusTask
	if len(templates) > 0 {
		initialFocus = focusTemplate
	}

	// Proactively initialize native audio speaker
	_ = initSpeaker()

	soundscapes, _ := loadSoundscapes(cfg.SoundscapesDir)
	fileTasks := loadTasksFromFile(cfg.TasksFile)

	m := model{
		mode:                  mode,
		textInput:             ti,
		durationInput:         di,
		noteInput:             ni,
		tagInput:              gi,
		focusedField:          initialFocus,
		entries:               entryList,
		templates:             templates,
		taskSuggestions:       buildTaskSuggestions(entryList, cfg.PinnedTasks, fileTasks),
		suggestionIndex:       -1,
		dataFile:              dataFile,
		templateFile:          "templates.json",
		configFile:            "kairu.yaml",
		config:                cfg,
		streakState:           streakState,
		appError:              strings.Join(startupErrors, " | "),
		outboxFile:            defaultOutboxFile(),
		deliveredNotifyIDs:    make(map[string]time.Time),
		soundscapes:           soundscapes,
		soundscapeIndex:       -1,
		internalLogs:          []string{},
		petState:              pState,
		petEnabled:            petEnabled,
		showPetSidebar:        true,
		showPetLevelUpOverlay: false,
		tamagotchiActiveMenu:  "",
		tamagotchiMenuSelect:  0,
		tamagotchiFeedback:    "",
	}
	m.logInternal("SYSTEM: Kairu TUI started")
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
