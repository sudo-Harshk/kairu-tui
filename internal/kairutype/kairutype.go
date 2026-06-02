package kairutype

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// RecordsPath is the path to save/load typing records. Defaults to typing_records.json.
var RecordsPath = "typing_records.json"


// Curated dictionary of 200 coding, terminal, and productivity-oriented terms
var DeveloperDictionary = []string{
	"func", "struct", "interface", "package", "import", "return", "string", "int", "float64", "bool",
	"error", "nil", "err", "if", "else", "for", "range", "make", "new", "select",
	"chan", "go", "defer", "type", "var", "const", "map", "slice", "array", "pointer",
	"address", "compile", "runtime", "garbage", "collect", "goroutine", "mutex", "sync", "atomic", "context",
	"cancel", "deadline", "timeout", "channel", "buffer", "stream", "reader", "writer", "closer", "encoder",
	"decoder", "json", "yaml", "xml", "database", "query", "insert", "update", "delete", "index",
	"primary", "foreign", "table", "schema", "migration", "driver", "connection", "socket", "network", "protocol",
	"http", "tcp", "udp", "client", "server", "handler", "middleware", "router", "request", "response",
	"header", "cookie", "session", "auth", "token", "jwt", "oauth", "login", "logout", "register",
	"user", "admin", "role", "permission", "access", "deny", "allow", "secure", "cipher", "encrypt",
	"decrypt", "hash", "salt", "signature", "key", "public", "private", "cert", "tls", "ssl",
	"algorithm", "sort", "search", "binary", "tree", "graph", "node", "edge", "vertex", "path",
	"depth", "breadth", "stack", "queue", "heap", "hashmap", "trie", "bloom", "filter", "collision",
	"complexity", "time", "space", "omega", "theta", "recursion", "iteration", "loop", "condition", "branch",
	"jump", "goto", "break", "continue", "fallthrough", "switch", "case", "default", "panic", "recover",
	"trace", "debug", "test", "assert", "benchmark", "profile", "optimize", "refactor", "clean", "code",
	"smell", "legacy", "spaghetti", "architecture", "design", "pattern", "singleton", "factory", "builder", "prototype",
	"adapter", "decorator", "facade", "proxy", "flyweight", "composite", "bridge", "observer", "strategy", "template",
	"visitor", "state", "command", "mediator", "memento", "chain", "iterator", "interpreter", "git", "commit",
	"push", "pull", "merge", "rebase", "checkout", "branch", "stash", "clone", "fetch", "status",
}

// TypingRecords handles local storage of the high scores
type TypingRecords struct {
	BestTimeWPM map[int]int `json:"best_time_wpm"` // targetTime -> wpm
	BestWordWPM map[int]int `json:"best_word_wpm"` // targetWords -> wpm
}

// KairuTypeState holds the complete state of the typing test engine
type KairuTypeState struct {
	Active        bool
	Finished      bool
	Mode          string // "time" or "words"
	TargetTime    int    // 15, 30, 60
	TargetWords   int    // 10, 25, 50
	TimeRemaining int    // Counts down for Time Mode

	TargetText  string
	WordsList   []string
	TypedText   string

	StartTime   time.Time
	EndTime     time.Time
	Keystrokes  int
	ErrorCount  int
	WPM         int
	RawWPM      int
	Accuracy    float64
	WPMHistory  []int
	TimeSamples []int

	Records TypingRecords
}

// InitKairuType initializes the typing state
func InitKairuType(mode string, target int) KairuTypeState {
	targetTime := 30
	targetWords := 25

	if mode == "time" {
		targetTime = target
	} else {
		targetWords = target
	}

	state := KairuTypeState{
		Active:        false,
		Finished:      false,
		Mode:          mode,
		TargetTime:    targetTime,
		TargetWords:   targetWords,
		TimeRemaining: targetTime,
		WPMHistory:    make([]int, 0),
		TimeSamples:   make([]int, 0),
	}

	state.LoadRecords()
	state.ResetTest()
	return state
}

// ResetTest regenerates text and clears scores
func (s *KairuTypeState) ResetTest() {
	s.Finished = false
	s.Active = false
	s.TypedText = ""
	s.Keystrokes = 0
	s.ErrorCount = 0
	s.WPM = 0
	s.RawWPM = 0
	s.Accuracy = 100.0
	s.WPMHistory = make([]int, 0)
	s.TimeSamples = make([]int, 0)
	s.TimeRemaining = s.TargetTime

	// Generate words based on mode
	wordCount := s.TargetWords
	if s.Mode == "time" {
		wordCount = 120
	}

	var generated []string
	for i := 0; i < wordCount; i++ {
		generated = append(generated, DeveloperDictionary[rand.Intn(len(DeveloperDictionary))])
	}
	s.WordsList = generated
	s.TargetText = strings.Join(generated, " ")
}

// StartTest records start timestamp
func (s *KairuTypeState) StartTest() {
	s.Active = true
	s.StartTime = time.Now()
}

// AddChar appends typed key
func (s *KairuTypeState) AddChar(char string) {
	if s.Finished {
		return
	}
	if !s.Active {
		s.StartTest()
	}

	if len(s.TypedText) < len(s.TargetText) {
		s.Keystrokes++
		targetChar := s.TargetText[len(s.TypedText)]
		if char != string(targetChar) {
			s.ErrorCount++
		}
		s.TypedText += char
		s.UpdateStats()
		s.CheckCompletion()
	}
}

// HandleBackspace removes the last character
func (s *KairuTypeState) HandleBackspace() {
	if s.Finished || len(s.TypedText) == 0 {
		return
	}
	s.Keystrokes++
	s.TypedText = s.TypedText[:len(s.TypedText)-1]
	s.UpdateStats()
}

// UpdateStats calculates current WPM, Raw WPM, and Accuracy
func (s *KairuTypeState) UpdateStats() {
	if len(s.TypedText) == 0 {
		s.Accuracy = 100.0
		s.WPM = 0
		s.RawWPM = 0
		return
	}

	elapsed := time.Since(s.StartTime).Minutes()
	if elapsed <= 0 {
		elapsed = 0.01
	}

	correctChars := 0
	for i := 0; i < len(s.TypedText); i++ {
		if i < len(s.TargetText) && s.TypedText[i] == s.TargetText[i] {
			correctChars++
		}
	}

	s.WPM = int(float64(correctChars) / 5.0 / elapsed)
	s.RawWPM = int(float64(s.Keystrokes) / 5.0 / elapsed)

	s.Accuracy = (float64(correctChars) / float64(len(s.TypedText))) * 100.0
}

// SampleWPM records the WPM sample for this second
func (s *KairuTypeState) SampleWPM(elapsedSeconds int) {
	if !s.Active || s.Finished {
		return
	}
	s.UpdateStats()
	s.WPMHistory = append(s.WPMHistory, s.WPM)
	s.TimeSamples = append(s.TimeSamples, elapsedSeconds)
}

// CheckCompletion triggers completion check for Word Mode
func (s *KairuTypeState) CheckCompletion() {
	if s.Mode == "words" {
		targetWordsText := strings.Join(s.WordsList[:s.TargetWords], " ")
		if len(s.TypedText) >= len(targetWordsText) {
			s.FinishTest()
		}
	}
}

// FinishTest locks stats and persists records
func (s *KairuTypeState) FinishTest() {
	s.Finished = true
	s.EndTime = time.Now()
	s.UpdateStats()

	s.WPMHistory = append(s.WPMHistory, s.WPM)
	elapsedSecs := int(s.EndTime.Sub(s.StartTime).Seconds())
	s.TimeSamples = append(s.TimeSamples, elapsedSecs)

	s.SaveRecord()
}

// SaveRecord checks for high scores and saves to JSON
func (s *KairuTypeState) SaveRecord() {
	updated := false
	if s.Mode == "time" {
		if s.Records.BestTimeWPM == nil {
			s.Records.BestTimeWPM = make(map[int]int)
		}
		prev := s.Records.BestTimeWPM[s.TargetTime]
		if s.WPM > prev {
			s.Records.BestTimeWPM[s.TargetTime] = s.WPM
			updated = true
		}
	} else {
		if s.Records.BestWordWPM == nil {
			s.Records.BestWordWPM = make(map[int]int)
		}
		prev := s.Records.BestWordWPM[s.TargetWords]
		if s.WPM > prev {
			s.Records.BestWordWPM[s.TargetWords] = s.WPM
			updated = true
		}
	}

	if updated {
		data, err := json.MarshalIndent(s.Records, "", "  ")
		if err == nil {
			_ = os.WriteFile(RecordsPath, data, 0644)
		}
	}
}

// LoadRecords loads records from the path configured in RecordsPath
func (s *KairuTypeState) LoadRecords() {
	s.Records = TypingRecords{
		BestTimeWPM: make(map[int]int),
		BestWordWPM: make(map[int]int),
	}
	data, err := os.ReadFile(RecordsPath)
	if err == nil {
		_ = json.Unmarshal(data, &s.Records)
	}
}

// wrapTextPreservingSpaces wraps target text into lines while preserving spaces for exact character alignment
func wrapTextPreservingSpaces(text string, limit int) []string {
	words := strings.Split(text, " ")
	var lines []string
	currLine := ""
	for _, w := range words {
		if currLine == "" {
			currLine = w
		} else if len(currLine)+1+len(w) > limit {
			lines = append(lines, currLine+" ")
			currLine = w
		} else {
			currLine += " " + w
		}
	}
	if currLine != "" {
		lines = append(lines, currLine)
	}
	return lines
}

// RenderKairuTypeView compiles the typing arena TUI (standard borderless layout)
func RenderKairuTypeView(s KairuTypeState, width int, accent string, primary string, banner string) string {
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(accent))
	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Slate gray

	if s.Finished {
		return renderPostTestView(s, width, accent, primary, banner)
	}

	// 1. INFO HEADER
	var timerStr string
	if s.Mode == "time" {
		timerStr = fmt.Sprintf("⏱️ TIME REMAINING: %ds / %ds", s.TimeRemaining, s.TargetTime)
	} else {
		typedWords := len(strings.Fields(s.TypedText))
		timerStr = fmt.Sprintf("📝 WORDS TYPED: %d / %d", typedWords, s.TargetWords)
	}

	bestWPM := 0
	if s.Mode == "time" {
		bestWPM = s.Records.BestTimeWPM[s.TargetTime]
	} else {
		bestWPM = s.Records.BestWordWPM[s.TargetWords]
	}

	bestStr := ""
	if bestWPM > 0 {
		bestStr = fmt.Sprintf("  •  Best: %d WPM", bestWPM)
	}

	statsLine := fmt.Sprintf(
		"    %s  │  WPM: %d  │  Acc: %.1f%%%s",
		accentStyle.Bold(true).Render(timerStr),
		s.WPM,
		s.Accuracy,
		bestStr,
	)

	// 2. TYPING AREA
	targetText := s.TargetText
	if s.Mode == "words" {
		targetText = strings.Join(s.WordsList[:s.TargetWords], " ")
	}

	lines := wrapTextPreservingSpaces(targetText, 65)

	cursor := len(s.TypedText)
	activeLineIndex := len(lines) - 1 // Fallback to last line
	accumulatedLen := 0
	for idx, line := range lines {
		lineLen := len(line)
		if cursor >= accumulatedLen && cursor < accumulatedLen+lineLen {
			activeLineIndex = idx
			break
		}
		accumulatedLen += lineLen
	}

	// Centered rolling 3-line scroll window
	var windowLines []string
	startLine := activeLineIndex - 1
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + 2
	if endLine >= len(lines) {
		endLine = len(lines) - 1
		startLine = endLine - 2
		if startLine < 0 {
			startLine = 0
		}
	}

	// Calculate accumulated len up to startLine
	accumulatedLen = 0
	for idx := 0; idx < startLine; idx++ {
		accumulatedLen += len(lines[idx])
	}

	styleCorrect := lipgloss.NewStyle().Foreground(lipgloss.Color(primary))
	styleIncorrect := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Underline(true)
	styleUntyped := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleCaret := lipgloss.NewStyle().Background(lipgloss.Color(accent)).Foreground(lipgloss.Color("0"))

	for idx := startLine; idx <= endLine; idx++ {
		line := lines[idx]
		var builder strings.Builder
		builder.WriteString("    ") // Margin padding

		for i := 0; i < len(line); i++ {
			charIdx := accumulatedLen + i
			targetChar := line[i]

			if charIdx < cursor {
				if charIdx < len(s.TypedText) && s.TypedText[charIdx] == targetText[charIdx] {
					builder.WriteString(styleCorrect.Render(string(targetChar)))
				} else {
					builder.WriteString(styleIncorrect.Render(string(targetChar)))
				}
			} else if charIdx == cursor {
				builder.WriteString(styleCaret.Render(string(targetChar)))
			} else {
				builder.WriteString(styleUntyped.Render(string(targetChar)))
			}
		}
		windowLines = append(windowLines, builder.String())
		accumulatedLen += len(line)
	}

	if cursor >= len(targetText) {
		if len(windowLines) > 0 {
			windowLines[len(windowLines)-1] += styleCaret.Render(" ")
		}
	}

	typingAreaBlock := strings.Join(windowLines, "\n\n")

	// 3. FOOTER HINT
	footer := subtleStyle.Render("    [Esc] Exit Arena   [Ctrl+R] Restart Test   [Tab] Mode: " + strings.ToUpper(s.Mode))

	// Assemble final block using standard Kairu screen layout
	block := fmt.Sprintf(`%s
%s

%s

%s`, banner, statsLine, typingAreaBlock, footer)

	return fmt.Sprintf("\n%s\n", centerBlock(width, block))
}

// renderPostTestView compiles the Monkeytype speed metrics & line chart (standard borderless layout)
func renderPostTestView(s KairuTypeState, width int, accent string, primary string, banner string) string {
	accentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(accent))
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(primary))
	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	goldStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	// Stats line
	statsLine := fmt.Sprintf(
		"    %s  │  WPM: %s  │  Acc: %s  │  Keys: %d  │  Errors: %d",
		primaryStyle.Bold(true).Render("📊 RESULTS"),
		accentStyle.Bold(true).Render(fmt.Sprintf("%d", s.WPM)),
		accentStyle.Bold(true).Render(fmt.Sprintf("%.1f%%", s.Accuracy)),
		s.Keystrokes,
		s.ErrorCount,
	)

	isRecord := false
	if s.Mode == "time" {
		isRecord = s.WPM == s.Records.BestTimeWPM[s.TargetTime]
	} else {
		isRecord = s.WPM == s.Records.BestWordWPM[s.TargetWords]
	}

	recordBanner := ""
	if isRecord && s.WPM > 0 {
		recordBanner = goldStyle.Render("    🏆 NEW PERSONAL RECORD ACHIEVED!") + "\n"
	}

	// Render the WPM history graph
	graphBlock := renderASCIIChart(s.WPMHistory, 50, 6, accent)

	footer := subtleStyle.Render("    [Enter] Play Again   [Esc] Exit Arena   [Tab] Change Mode   [Ctrl+R] Restart")

	block := fmt.Sprintf(`%s
%s

%s%s

%s`, banner, statsLine, recordBanner, graphBlock, footer)

	return fmt.Sprintf("\n%s\n", centerBlock(width, block))
}

// renderASCIIChart constructs a 2D line graph from history data
func renderASCIIChart(history []int, width int, height int, color string) string {
	subtleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

	if len(history) == 0 {
		return subtleStyle.Render("    [Graph: Insufficient data samples]")
	}

	// 1. Find Min and Max values to scale
	minVal := history[0]
	maxVal := history[0]
	for _, v := range history {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Add margin to scale bounds
	if maxVal == minVal {
		maxVal = minVal + 10
		minVal = minVal - 10
		if minVal < 0 {
			minVal = 0
		}
	} else {
		span := maxVal - minVal
		maxVal += int(math.Ceil(float64(span) * 0.1))
		minVal -= int(math.Ceil(float64(span) * 0.1))
		if minVal < 0 {
			minVal = 0
		}
	}

	// 2. Initialize character canvas grid
	grid := make([][]string, height)
	for r := 0; r < height; r++ {
		grid[r] = make([]string, width)
		for c := 0; c < width; c++ {
			grid[r][c] = " "
		}
	}

	// 3. Project WPM coordinates and plot dots/connectors
	n := len(history)
	colMap := make([]int, n)
	rowMap := make([]int, n)

	for i := 0; i < n; i++ {
		col := 0
		if n > 1 {
			col = int(float64(i) / float64(n-1) * float64(width-1))
		}
		val := history[i]
		pct := float64(val-minVal) / float64(maxVal-minVal)
		row := int(pct * float64(height-1))
		row = height - 1 - row // Flip since row 0 is the top

		if row < 0 {
			row = 0
		} else if row >= height {
			row = height - 1
		}

		colMap[i] = col
		rowMap[i] = row
		grid[row][col] = "·" // Core sample plot dot
	}

	// Interpolate columns to fill visual gaps with line graphics
	for i := 0; i < n-1; i++ {
		c1, r1 := colMap[i], rowMap[i]
		c2, r2 := colMap[i+1], rowMap[i+1]

		if c2 == c1 {
			continue
		}

		for c := c1 + 1; c < c2; c++ {
			pct := float64(c-c1) / float64(c2-c1)
			r := int(float64(r1) + pct*float64(r2-r1))
			if r >= 0 && r < height {
				grid[r][c] = "·"
			}
		}
	}

	// Connect close neighbors with ASCII curves where possible
	for r := 0; r < height; r++ {
		for c := 0; c < width; c++ {
			if grid[r][c] == "·" {
				left := c > 0 && grid[r][c-1] == "·"
				right := c < width-1 && grid[r][c+1] == "·"
				up := r > 0 && grid[r-1][c] == "·"
				down := r < height-1 && grid[r+1][c] == "·"

				if left && right {
					grid[r][c] = "─"
				} else if up && down {
					grid[r][c] = "│"
				} else if left && up {
					grid[r][c] = "╯"
				} else if right && up {
					grid[r][c] = "╰"
				} else if left && down {
					grid[r][c] = "╮"
				} else if right && down {
					grid[r][c] = "╭"
				} else {
					grid[r][c] = "·"
				}
			}
		}
	}

	// 4. Print rows with axes
	var rows []string
	rows = append(rows, subtleStyle.Render(fmt.Sprintf("    [WPM progression over %ds]", n)))
	rows = append(rows, subtleStyle.Render("    ┌"+strings.Repeat("─", width+1)))

	for r := 0; r < height; r++ {
		var axisLabel string
		if r == 0 {
			axisLabel = fmt.Sprintf("%3d", maxVal)
		} else if r == height-1 {
			axisLabel = fmt.Sprintf("%3d", minVal)
		} else {
			axisLabel = "   "
		}

		lineContent := ""
		for c := 0; c < width; c++ {
			char := grid[r][c]
			if char != " " {
				lineContent += lineStyle.Render(char)
			} else {
				lineContent += " "
			}
		}
		rows = append(rows, fmt.Sprintf("%s │ %s", subtleStyle.Render(axisLabel), lineContent))
	}
	rows = append(rows, subtleStyle.Render("        "+strings.Repeat("─", width+1)))
	rows = append(rows, subtleStyle.Render(fmt.Sprintf("         0s%s%ds", strings.Repeat(" ", width-4), n)))

	return strings.Join(rows, "\n")
}

func centerBlock(width int, content string) string {
	if width <= 0 {
		return content
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(content)
}
