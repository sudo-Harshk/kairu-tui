package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
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
	running        bool
	mode           string
	textInput      textinput.Model
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
			m.seconds++
			if m.mode == "timer" {
				m.totalWorkTime++
			} else {
				m.totalBreakTime++
			}
		}
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.mode == "timer" && m.seconds > 0 {
				m.saveSession()
			}
			return m, tea.Quit

		case "tab":
			if m.mode == "timer" || m.mode == "break" {
				m.mode = "stats"
			} else if m.mode == "stats" {
				m.mode = "timer"
			}
			return m, nil

		case "up", "k":
			if m.mode == "timer" || m.mode == "break" {
				m.seconds += 60
			}
			return m, nil

		case "down", "j":
			if (m.mode == "timer" || m.mode == "break") && m.seconds >= 60 {
				m.seconds -= 60
			}
			return m, nil

		case "enter":
			if m.mode == "input" {
				if strings.TrimSpace(m.textInput.Value()) == "" {
					return m, nil
				}
				m.mode = "timer"
				m.taskName = m.textInput.Value()
				m.textInput.Blur()
				m.sessionStart = time.Now()
				m.running = true
				m.seconds = m.config.WorkDuration * 60
				return m, tickCmd()
			}

			if m.mode == "timer" {
				m.saveSession()
				m.sessionCount++

				if m.config.AutoBreak && m.sessionCount%m.config.SessionsBeforeBreak == 0 {
					m.mode = "break"
					m.seconds = m.config.BreakDuration * 60
					m.running = true
					return m, tickCmd()
				}

				m.mode = "input"
				m.taskName = ""
				m.seconds = 0
				m.running = false
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, nil
			}

			if m.mode == "break" {
				m.saveSession()
				m.mode = "input"
				m.seconds = 0
				m.running = false
				m.textInput.SetValue("")
				m.textInput.Focus()
				return m, nil
			}
		}

		if m.mode == "input" {
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) saveSession() {
	duration := m.seconds
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
	case "stats":
		return renderStatsView(m)
	default:
		return renderInputView(m)
	}
}

func renderInputView(m model) string {
	return fmt.Sprintf(`
╭─────────────────────────────────────╮
│  📝  What are you working on?      │
╰─────────────────────────────────────╯

%s

[Enter] Start   [Tab] Stats   [q] Quit
`, m.textInput.View())
}

func renderTimerView(m model) string {
	h, mnt, s := m.seconds/3600, (m.seconds%3600)/60, m.seconds%60
	timeStr := fmt.Sprintf("%02d:%02d:%02d", h, mnt, s)

	status := "▶ RUNNING"
	if !m.running {
		status = "⏸ PAUSED"
	}

	modeStr := "🎯 WORK"
	if m.mode == "break" {
		modeStr = "☕ BREAK"
	}

	// Progress bar
	targetSeconds := m.config.WorkDuration * 60
	if m.mode == "break" {
		targetSeconds = m.config.BreakDuration * 60
	}
	progressPct := float64(m.seconds) / float64(targetSeconds) * 100
	if progressPct > 100 {
		progressPct = 100
	}
	barWidth := 40
	filled := int(progressPct / 100 * float64(barWidth))
	empty := barWidth - filled
	progress := fmt.Sprintf("[%s%s] %.0f%%\n", strings.Repeat("█", filled), strings.Repeat("░", empty), progressPct)

	hint := "[Space] Pause  [↑/↓] Adjust  [Enter] End  [Tab] Stats  [q] Quit"
	if !m.running {
		hint = "[Space] Resume  [↑/↓] Adjust  [Enter] End  [Tab] Stats  [q] Quit"
	}

	return fmt.Sprintf(`
%s

╭─────────────────────────────────────╮
│  %s  %-20s  │
╰─────────────────────────────────────╯

%s

   ⏱  %s  •  %s

Session #%d

%s
%s
`, renderBanner(), modeStr, m.taskName, renderASCIITimer(timeStr), timeStr, status, m.sessionCount+1, progress, hint)
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
	ti.Placeholder = "What are you working on?"
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	var entryList []Entry
	if data, err := os.ReadFile(dataFile); err == nil {
		json.Unmarshal(data, &entryList)
	}

	m := model{
		mode:      "input",
		textInput: ti,
		entries:   entryList,
		dataFile:  dataFile,
		config:    defaultConfig,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
