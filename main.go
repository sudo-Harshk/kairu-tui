package main

import (
	"encoding/json"
	"errors"
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
	textInput          textinput.Model
	durationInput      textinput.Model
	focusedField       int
	inputError         string
	appError           string
	notificationStatus string
	taskName           string
	settingsCursor     int
	entries            []Entry
	dataFile           string
	configFile         string
	config             Config
	sessionStart       time.Time
	sessionCount       int
	totalWorkTime      int
	totalBreakTime     int
	notificationOutbox []notificationJob
	outboxFile         string
}

type notificationJob struct {
	ID          string    `json:"id"`
	SessionType string    `json:"session_type"`
	Task        string    `json:"task"`
	Duration    int       `json:"duration_seconds"`
	CreatedAt   time.Time `json:"created_at"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error,omitempty"`
}

type Entry struct {
	Task     string    `json:"task"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration int       `json:"duration_seconds"`
	Type     string    `json:"type"`
}

type tickTockMsg time.Time

const (
	focusTask = iota
	focusDuration
)

const (
	settingsDesktop = iota
	settingsWorkComplete
	settingsBreakComplete
	settingsSessionStart
	settingsSessionEnd
	settingsPauseResume
	settingsEndingSoon
	settingsQuietStart
	settingsQuietEnd
	settingsCount
)

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickTockMsg(t) })
}

func (m model) Init() tea.Cmd {
	if m.mode == "fatal" {
		return nil
	}
	if (m.mode == "timer" || m.mode == "break") && m.running {
		return tickCmd()
	}
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickTockMsg:
		if m.running && (m.mode == "timer" || m.mode == "break") {
			if m.seconds > 0 {
				m.seconds--
				m.sessionElapsed++
				if m.mode == "timer" {
					m.totalWorkTime++
				} else {
					m.totalBreakTime++
				}
			}
			if m.seconds == 0 {
				if m.mode == "timer" {
					m.notify("work_complete")
				} else {
					m.notify("break_complete")
				}
				return m.completeSession()
			}
			return m, tickCmd()
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
				if m.helpReturnMode == "timer" && m.seconds > 0 {
					m.saveSession()
				}
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
				m.mode = "timer"
				return m, nil
			case "q", "ctrl+c":
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
	if field == focusTask {
		m.textInput.Focus()
		m.durationInput.Blur()
	} else {
		m.durationInput.Focus()
		m.textInput.Blur()
	}
	return m
}

func (m model) openHelp() model {
	m.helpReturnMode = m.mode
	m.helpWasRunning = m.running
	if m.running && (m.mode == "timer" || m.mode == "break") {
		m.running = false
	}
	m.mode = "help"
	return m
}

func (m model) closeHelp(resume bool) (model, tea.Cmd) {
	m.mode = m.helpReturnMode
	if m.mode == "timer" || m.mode == "break" {
		if resume && m.helpWasRunning {
			m.running = true
			if m.seconds > 0 {
				return m, tickCmd()
			}
		} else {
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
		if m.mode == "timer" && m.seconds > 0 {
			m.saveSession()
		}
		return m, tea.Quit

	case "tab":
		if m.mode == "input" {
			if m.focusedField == focusTask {
				m = m.setInputFocus(focusDuration)
			} else {
				m = m.setInputFocus(focusTask)
			}
			return m, nil
		}
		if m.mode == "timer" || m.mode == "break" {
			m.mode = "settings"
			m.settingsCursor = settingsDesktop
			return m, nil
		}
		if m.mode == "stats" {
			m.mode = "settings"
			m.settingsCursor = settingsDesktop
			return m, nil
		}
		if m.mode == "settings" {
			m.mode = "stats"
			return m, nil
		}

	case "enter":
		if m.mode == "input" {
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
			m.sessionStart = time.Now()
			m.running = true
			m.sessionTarget = durationSeconds
			m.seconds = durationSeconds
			m.sessionElapsed = 0
			m.inputError = ""
			m.notify("session_start")
			return m, tickCmd()
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
			m.notify("session_end")
			return m.completeSession()
		}
	case " ", "space":
		if m.mode == "timer" || m.mode == "break" {
			m.running = !m.running
			m.notify("pause_resume")
			if m.running && m.seconds > 0 {
				return m, tickCmd()
			}
			return m, nil
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
	}

	if m.mode == "input" {
		if m.focusedField == focusTask {
			m.textInput, cmd = m.textInput.Update(msg)
		} else {
			m.durationInput, cmd = m.durationInput.Update(msg)
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
	m.saveSession()
	if m.mode == "timer" {
		m.sessionCount++
		if m.config.AutoBreak && m.sessionCount%m.config.SessionsBeforeBreak == 0 {
			m.mode = "break"
			m.sessionStart = time.Now()
			m.sessionTarget = m.config.BreakDuration * 60
			m.seconds = m.sessionTarget
			m.sessionElapsed = 0
			m.running = true
			return m, tickCmd()
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
	m = m.setInputFocus(focusTask)
	return m, nil
}

func (m *model) toggleSetting() {
	switch m.settingsCursor {
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
	}
	if err := saveConfigFile(m.configFile, m.config); err != nil {
		m.setAppError(err, "Failed to save config")
	}
}

func (m *model) adjustSetting(delta int) {
	switch m.settingsCursor {
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

func newNotificationJob(sessionType, task string, duration int) notificationJob {
	return notificationJob{
		ID:          fmt.Sprintf("%d-%s", time.Now().UnixNano(), sessionType),
		SessionType: sessionType,
		Task:        task,
		Duration:    duration,
		CreatedAt:   time.Now(),
	}
}

func (j notificationJob) message() string {
	return fmt.Sprintf("Session completed: %s (%s)", j.Task, formatDuration(j.Duration))
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

func (m model) notify(event string) {
	if !m.config.Notifications || !m.eventEnabled(event) {
		return
	}
	if m.quietHoursActive(time.Now()) {
		m.setNotificationStatus("Notification suppressed by quiet hours")
		return
	}

	title := m.notificationTitle(event)
	body := m.notificationBody(event)
	if body == "" {
		return
	}

	if err := m.sendNotificationWithFallback(title, body); err != nil {
		m.setAppError(err, "Notification failed")
	}
}

func (m model) sendNotificationWithFallback(title, body string) error {
	if m.config.DesktopNotifications {
		if err := sendDesktopNotification(title, body); err == nil {
			m.setNotificationStatus("Desktop notification delivered")
			return nil
		}
	}

	if m.config.SoundCommand != "" {
		if err := exec.Command("sh", "-c", m.config.SoundCommand).Run(); err == nil {
			m.setNotificationStatus("Sound fallback delivered")
			return nil
		}
	}

	if token := strings.TrimSpace(m.config.TelegramBotToken); token != "" && strings.TrimSpace(m.config.TelegramChatID) != "" {
		if err := sendTelegramMessage(token, strings.TrimSpace(m.config.TelegramChatID), body); err == nil {
			m.setNotificationStatus("Telegram fallback delivered")
			return nil
		}
	}

	return fmt.Errorf("all notification channels failed")
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

func (m *model) flushNotificationOutbox() {
	if !m.config.Notifications || len(m.notificationOutbox) == 0 {
		return
	}

	remaining := make([]notificationJob, 0, len(m.notificationOutbox))
	for _, job := range m.notificationOutbox {
		if err := m.sendNotificationJob(job); err != nil {
			job.Attempts++
			job.LastError = err.Error()
			remaining = append(remaining, job)
			continue
		}
		m.setNotificationStatus("Notification queue delivered")
	}
	m.notificationOutbox = remaining
	if err := saveNotificationOutbox(m.outboxFile, m.notificationOutbox); err != nil {
		m.setAppError(err, "Failed to save notification queue")
	}
}

func (m model) sendNotificationJob(job notificationJob) error {
	if job.SessionType != "work" {
		return nil
	}
	token := strings.TrimSpace(m.config.TelegramBotToken)
	chatID := strings.TrimSpace(m.config.TelegramChatID)
	if token == "" || chatID == "" {
		return fmt.Errorf("telegram notifications require %s and %s", envTelegramBotToken, envTelegramChatID)
	}

	var lastErr error
	backoffs := []time.Duration{0, 2 * time.Second, 5 * time.Second}
	for attempt := 0; attempt < len(backoffs); attempt++ {
		if backoffs[attempt] > 0 {
			time.Sleep(backoffs[attempt])
		}
		if err := sendTelegramMessage(token, chatID, job.message()); err != nil {
			lastErr = err
			continue
		}
		if m.config.SoundCommand != "" {
			if err := exec.Command("sh", "-c", m.config.SoundCommand).Run(); err != nil {
				return fmt.Errorf("sound command failed: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("telegram send failed after retries: %w", lastErr)
}

func (m *model) saveSession() {
	duration := m.sessionElapsed
	sessionType := "work"
	if m.mode == "break" {
		sessionType = "break"
	}

	entry := Entry{
		Task:     m.taskName,
		Start:    m.sessionStart,
		End:      time.Now(),
		Duration: duration,
		Type:     sessionType,
	}

	var entries []Entry
	if data, err := os.ReadFile(m.dataFile); err == nil {
		if err := json.Unmarshal(data, &entries); err != nil {
			m.setAppError(err, "Failed to parse entries")
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		m.setAppError(err, "Failed to read entries")
		return
	}
	entries = append(entries, entry)
	fileData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		m.setAppError(err, "Failed to encode entries")
		return
	}
	if err := os.WriteFile(m.dataFile, fileData, 0644); err != nil {
		m.setAppError(err, "Failed to write entries")
		return
	}
	m.entries = entries

	if m.config.Notifications {
		job := newNotificationJob(sessionType, m.taskName, duration)
		m.notificationOutbox = append(m.notificationOutbox, job)
		if err := saveNotificationOutbox(m.outboxFile, m.notificationOutbox); err != nil {
			m.setAppError(err, "Failed to save notification queue")
			return
		}
		m.flushNotificationOutbox()
		if len(m.notificationOutbox) > 0 {
			m.setAppError(fmt.Errorf("%s", m.notificationOutbox[0].LastError), "Notification queued for retry")
		}
	}
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
	return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.appError)
}

func renderNotificationStatus(m model) string {
	if strings.TrimSpace(m.notificationStatus) == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(m.notificationStatus)
}

func renderInputView(m model) string {
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	errorBlock := joinNonEmptyLines(errorLine, renderAppError(m))
	return fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📝  What are you working on?      │
╰─────────────────────────────────────╯

%s

%s

%s

[Tab] Switch Field   [Enter] Start   [?] Help   [q] Quit
`, m.textInput.View(), m.durationInput.View(), errorBlock)
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

[Enter] Apply   [Esc] Cancel   [?] Help   [q] Quit`, renderBanner(), m.taskName, elapsed, m.durationInput.View(), errorBlock)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func renderTimerView(m model) string {
	timeStr := formatClock(m.seconds)

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

	hint := "[Space] Pause  [E] Edit  [Enter] End  [Tab] Stats  [?] Help  [q] Quit"
	if !m.running {
		hint = "[Space] Resume  [E] Edit  [Enter] End  [Tab] Stats  [?] Help  [q] Quit"
	}

	header := fmt.Sprintf("%s • %s", modeStr, m.taskName)
	ascii := renderASCIITimer(timeStr)
	innerWidth := lipgloss.Width(progress)
	if asciiWidth := lipgloss.Width(ascii); asciiWidth > innerWidth {
		innerWidth = asciiWidth
	}
	ascii = lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(ascii)
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

func renderASCIITimer(timeStr string) string {
	chars := map[rune][]string{
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
	}

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

	workRatio := 0
	total := m.totalWorkTime + m.totalBreakTime
	if total > 0 {
		workRatio = m.totalWorkTime * 100 / total
	}
	footer := "[Tab] Back   [?] Help   [q] Quit"
	errorLine := renderAppError(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
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
`, daily, current, longest, workRatio, 100-workRatio, barChart, footer)
}

func renderSettingsView(m model) string {
	footer := "[Tab] Switch   [Space] Toggle   [Left/Right] Quiet hours   [Esc] Back   [q] Quit"
	errorLine := renderAppError(m)
	statusLine := renderNotificationStatus(m)
	if errorLine != "" {
		footer = fmt.Sprintf("%s\n%s", errorLine, footer)
	}
	if statusLine != "" {
		footer = fmt.Sprintf("%s\n%s", statusLine, footer)
	}

	items := []string{
		renderSettingLine(m.settingsCursor == settingsDesktop, "Desktop notifications", boolLabel(m.config.DesktopNotifications)),
		renderSettingLine(m.settingsCursor == settingsWorkComplete, "Work complete", boolLabel(m.config.NotifyWorkComplete)),
		renderSettingLine(m.settingsCursor == settingsBreakComplete, "Break complete", boolLabel(m.config.NotifyBreakComplete)),
		renderSettingLine(m.settingsCursor == settingsSessionStart, "Session start", boolLabel(m.config.NotifySessionStart)),
		renderSettingLine(m.settingsCursor == settingsSessionEnd, "Session end", boolLabel(m.config.NotifySessionEnd)),
		renderSettingLine(m.settingsCursor == settingsPauseResume, "Pause/resume", boolLabel(m.config.NotifyPauseResume)),
		renderSettingLine(m.settingsCursor == settingsEndingSoon, "Ending soon", boolLabel(m.config.NotifyEndingSoon)),
		renderSettingLine(m.settingsCursor == settingsQuietStart, "Quiet start", hourLabel(m.config.QuietHoursStart)),
		renderSettingLine(m.settingsCursor == settingsQuietEnd, "Quiet end", hourLabel(m.config.QuietHoursEnd)),
	}

	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  Notification Settings              │
╰─────────────────────────────────────╯

%s

%s`, renderBanner(), strings.Join(items, "\n"), footer)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
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
		"Timer pauses while help is open.",
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
		"",
		"Edit:",
		formatHelpLine("Enter", "Apply"),
		formatHelpLine("Esc", "Cancel"),
		"",
		"Stats:",
		formatHelpLine("Tab", "Back"),
		"",
	}
	body := lipgloss.NewStyle().Width(35).Render(strings.Join(lines, "\n"))
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  Help                               │
╰─────────────────────────────────────╯

%s

%s`, renderBanner(), body, footer)
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

[q] Quit`, renderBanner(), message)
	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func getWeeklyData(entries []Entry) map[string]int {
	weekly := make(map[string]int)
	today := time.Now()
	for i := 0; i < 7; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		weekly[date] = 0
	}
	for _, e := range entries {
		date := e.Start.Format("2006-01-02")
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
		maxMinutes = 1
	}

	var b strings.Builder
	for i := 6; i >= 0; i-- {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		dayName := days[today.AddDate(0, 0, -i).Weekday()]
		minutes := weeklyData[date] / 60

		barLen := minutes * 40 / maxMinutes
		bar := strings.Repeat("█", barLen) + strings.Repeat("░", 40-barLen)

		b.WriteString(fmt.Sprintf("%s │%s│ %dm\n", dayName, bar, minutes))
	}

	return b.String()
}

func getDailyTotal(entries []Entry, sessionType string) int {
	today := time.Now().Format("2006-01-02")
	total := 0
	for _, e := range entries {
		if e.Start.Format("2006-01-02") == today && e.Type == sessionType {
			total += e.Duration
		}
	}
	return total
}

func calculateStreaks(entries []Entry) (int, int) {
	days := make(map[string]bool)
	for _, e := range entries {
		if e.Type == "work" {
			days[e.Start.Format("2006-01-02")] = true
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
		date, err := time.Parse("2006-01-02", d)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Kairu: invalid entry date:", d)
			continue
		}
		if last.IsZero() {
			temp = 1
		} else if int(date.Sub(last).Hours()/24) == 1 {
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
	if !days[today.Format("2006-01-02")] {
		return 0, longest
	}
	current := 0
	for i := 0; i < 365; i++ {
		if days[today.AddDate(0, 0, -i).Format("2006-01-02")] {
			current++
		} else if i > 0 {
			break
		}
	}

	return current, longest
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

func renderBanner() string {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Padding(0, 2).Render("🌳  KAIRU  •  Grow Your Focus  🌳")
}

func main() {
	dataFile := "entries.json"
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
	ti := textinput.New()
	ti.Placeholder = "Task name"
	ti.Focus()
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

	m := model{
		mode:          mode,
		textInput:     ti,
		durationInput: di,
		focusedField:  focusTask,
		entries:       entryList,
		dataFile:      dataFile,
		configFile:    "kairu.yaml",
		config:        cfg,
		appError:      strings.Join(startupErrors, " | "),
		outboxFile:    defaultOutboxFile(),
	}

	if jobs, err := loadNotificationOutbox(m.outboxFile); err == nil {
		m.notificationOutbox = jobs
		m.flushNotificationOutbox()
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
