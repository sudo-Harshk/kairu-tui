package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Configuration (defaults only - expand with YAML later)
type Config struct {
	WorkDuration        int
	BreakDuration       int
	Font                string
	Notifications       bool
	SoundCommand        string
	AutoBreak           bool
	SessionsBeforeBreak int
}

var defaultConfig = Config{
	WorkDuration:        25,
	BreakDuration:       5,
	Font:                "ansi",
	Notifications:       true,
	SoundCommand:        "",
	AutoBreak:           false,
	SessionsBeforeBreak: 4,
}

type model struct {
	seconds        int
	sessionTarget  int
	sessionElapsed int
	width          int
	running        bool
	mode           string
	editReturnMode string
	editWasRunning bool
	textInput      textinput.Model
	durationInput  textinput.Model
	focusedField   int
	inputError     string
	taskName       string
	entries        []Entry
	dataFile       string
	config         Config
	sessionStart   time.Time
	sessionCount   int
	totalWorkTime  int
	totalBreakTime int
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

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickTockMsg(t) })
}

func (m model) Init() tea.Cmd {
	if (m.mode == "timer" || m.mode == "break") && m.running {
		return tickCmd()
	}
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

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
				return m.completeSession()
			}
			return m, tickCmd()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
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
				m.mode = "stats"
				return m, nil
			}
			if m.mode == "stats" {
				m.mode = "timer"
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
				return m.completeSession()
			}
		case " ", "space":
			if m.mode == "timer" || m.mode == "break" {
				m.running = !m.running
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

func (m model) saveSession() {
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
		json.Unmarshal(data, &entries)
	}
	entries = append(entries, entry)
	fileData, _ := json.MarshalIndent(entries, "", "  ")
	os.WriteFile(m.dataFile, fileData, 0644)
	m.entries = entries

	if m.config.Notifications {
		m.sendNotification(sessionType)
	}
}

func (m model) sendNotification(sessionType string) {
	msg := fmt.Sprintf("Session completed: %s", m.taskName)
	if sessionType == "break" {
		msg = "Break time!"
	}

	// Cross-platform notifications
	if exec.Command("which", "notify-send").Run() == nil {
		exec.Command("notify-send", "Kairu", msg).Run()
	} else if exec.Command("which", "osascript").Run() == nil {
		exec.Command("osascript", "-e", fmt.Sprintf(`display notification "%s" with title "Kairu"`, msg)).Run()
	} else {
		exec.Command("powershell", "-Command", fmt.Sprintf(`[System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms"); [System.Windows.Forms.MessageBox]::Show("%s", "Kairu")`, msg)).Run()
	}

	if m.config.SoundCommand != "" {
		exec.Command("sh", "-c", m.config.SoundCommand).Run()
	}
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
	default:
		return renderInputView(m)
	}
}

func renderInputView(m model) string {
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	return fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📝  What are you working on?      │
╰─────────────────────────────────────╯

%s

%s

%s

[Tab] Switch Field   [Enter] Start   [q] Quit
`, m.textInput.View(), m.durationInput.View(), errorLine)
}

func renderEditView(m model) string {
	errorLine := ""
	if m.inputError != "" {
		errorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.inputError)
	}
	elapsed := formatClock(m.sessionElapsed)
	block := fmt.Sprintf(`%s

╭─────────────────────────────────────╮
│  ✏️  Adjust Session Time           │
╰─────────────────────────────────────╯

Task: %s
Elapsed: %s

%s

%s

[Enter] Apply   [Esc] Cancel   [q] Quit`, renderBanner(), m.taskName, elapsed, m.durationInput.View(), errorLine)
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

	hint := "[Space] Pause  [E] Edit  [Enter] End  [Tab] Stats  [q] Quit"
	if !m.running {
		hint = "[Space] Resume  [E] Edit  [Enter] End  [Tab] Stats  [q] Quit"
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

	block := fmt.Sprintf(`%s

%s

%s`, header, timerFrame, hint)
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

[Tab] Back   [q] Quit
`, daily, current, longest, workRatio, 100-workRatio, barChart)
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
	for i, d := range list {
		date, _ := time.Parse("2006-01-02", d)
		if i == 0 {
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

	current := 0
	today := time.Now()
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
	di.SetValue(fmt.Sprintf("%d", defaultConfig.WorkDuration))
	di.Blur()

	var entryList []Entry
	if data, err := os.ReadFile(dataFile); err == nil {
		json.Unmarshal(data, &entryList)
	}

	m := model{
		mode:          "input",
		textInput:     ti,
		durationInput: di,
		focusedField:  focusTask,
		entries:       entryList,
		dataFile:      dataFile,
		config:        defaultConfig,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
