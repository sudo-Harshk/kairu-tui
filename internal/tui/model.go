package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"kairu-tui/internal/config"
	"kairu-tui/internal/entries"
	"kairu-tui/internal/kairutype"
	"kairu-tui/internal/notification"
	"kairu-tui/internal/pet"
	"kairu-tui/internal/soundscape"
	"kairu-tui/internal/streak"
	"kairu-tui/internal/tasks"
	"kairu-tui/internal/templates"
)

type tickTockMsg time.Time
type petAnimTickMsg time.Time

type notifResultMsg struct {
	id     string
	status string
	err    error
}

type outboxFlushedMsg struct {
	remaining    []notification.NotificationJob
	deliveredIDs []string
	status       string
	err          error
}

type deletedTemplateState struct {
	template  templates.SessionTemplate
	index     int
	expiresAt time.Time
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
	notificationCounter int
	lastDeletedTemplate *deletedTemplateState
	taskName            string
	taskSuggestions     []string
	suggestionIndex     int
	showRecentOverlay   bool
	settingsCursor      int
	entries             []entries.Entry
	templates           []templates.SessionTemplate
	dataFile            string
	templateFile        string
	configFile          string
	petFile             string
	config              config.Config
	sessionStart        time.Time
	sessionCount        int
	totalWorkTime       int
	totalBreakTime      int
	streakState         streak.StreakState
	notificationOutbox  []notification.NotificationJob
	deliveredNotifyIDs  map[string]time.Time
	outboxFile          string
	soundscapeReturnMode string
	soundscapes         []string
	soundscapeIndex     int
	activeSoundscapeCmd *exec.Cmd
	internalLogs        []string
	guardianLocked      bool
	abortConfirmation   bool
	petState            pet.PetState
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
	typingGame             pet.TypingGameState
	binaryGame             pet.BinaryGameState
	kairuType              kairutype.KairuTypeState
}

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
	if m.config.QuietHoursStart >= 0 {
		cmds = append(cmds, tickCmd())
	} else if m.running {
		cmds = append(cmds, tickCmd())
	}
	if m.petEnabled {
		cmds = append(cmds, petAnimTick())
	}
	return tea.Batch(cmds...)
}

// New creates and initializes a new Bubbletea model with paths.
func New(paths config.Paths) tea.Model {
	kairutype.RecordsPath = paths.TypingRecordsFile
	startupErrors := []string{}
	if err := paths.EnsureDirsExist(); err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to create configuration directories: %v", err))
	}
	if err := config.LoadEnvFile(".env"); err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to load .env: %v", err))
	}
	cfg, err := config.LoadConfig(paths.ConfigFile)
	fatalConfig := false
	if err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to load config: %v", err))
		fatalConfig = true
	}
	templatesList, err := templates.LoadSessionTemplates(paths.TemplateFile)
	if err != nil {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to load templates: %v", err))
	}

	ti := textinput.New()
	ti.Placeholder = "Task name"
	ti.CharLimit = 50
	ti.Width = 40
	ti.Prompt = "> "

	di := textinput.New()
	di.Placeholder = "25"
	di.CharLimit = 8
	di.Width = 16
	di.Prompt = "> "
	di.SetValue(fmt.Sprintf("%d", cfg.WorkDuration))
	di.Blur()

	ni := textinput.New()
	ni.Placeholder = "Optional note"
	ni.CharLimit = 120
	ni.Width = 40
	ni.Prompt = "> "
	ni.Blur()

	gi := textinput.New()
	gi.Placeholder = "Optional tags, comma separated"
	gi.CharLimit = 120
	gi.Width = 40
	gi.Prompt = "> "
	gi.Blur()

	var entryList []entries.Entry
	if data, err := os.ReadFile(paths.DataFile); err == nil {
		if err := json.Unmarshal(data, &entryList); err != nil {
			startupErrors = append(startupErrors, fmt.Sprintf("Failed to parse entries: %v", err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to read entries: %v", err))
	}

	pState, petErr := pet.LoadPetState(paths.PetFile)
	petEnabled := true
	if petErr != nil {
		pState = pet.DefaultPet("Neko", "kitty")
		if err := pet.SavePetState(paths.PetFile, pState); err != nil {
			startupErrors = append(startupErrors, fmt.Sprintf("Failed to initialize default pet: %v", err))
		}
	} else {
		pState.TickStateDecay(time.Now())
		if pState.IsDead {
			startupErrors = append(startupErrors, "COMPANION DIED OF NEGLECT! 🪦 Enter Tamagotchi screen (Ctrl+T) to rebirth Neko.")
		}
		if err := pet.SavePetState(paths.PetFile, pState); err != nil {
			startupErrors = append(startupErrors, fmt.Sprintf("Failed to save caught-up pet state: %v", err))
		}
	}

	mode := "input"
	if fatalConfig {
		mode = "fatal"
	}
	streakState := streak.ComputeStreakState(entryList)
	if streakState.RecoveryNeeded {
		startupErrors = append(startupErrors, streakState.RecoveryPrompt)
	}
	initialFocus := focusTask
	if len(templatesList) > 0 {
		initialFocus = focusTemplate
	}

	_ = soundscape.InitSpeaker()

	soundscapesList, _ := soundscape.LoadSoundscapes(cfg.SoundscapesDir)
	fileTasks := tasks.LoadTasksFromFile(cfg.TasksFile)

	m := model{
		mode:                  mode,
		textInput:             ti,
		durationInput:         di,
		noteInput:             ni,
		tagInput:              gi,
		focusedField:          initialFocus,
		entries:               entryList,
		templates:             templatesList,
		taskSuggestions:       tasks.BuildTaskSuggestions(entryList, cfg.PinnedTasks, fileTasks),
		suggestionIndex:       -1,
		dataFile:              paths.DataFile,
		templateFile:          paths.TemplateFile,
		configFile:            paths.ConfigFile,
		petFile:               paths.PetFile,
		config:                cfg,
		streakState:           streakState,
		appError:              strings.Join(startupErrors, " | "),
		outboxFile:            paths.OutboxFile,
		deliveredNotifyIDs:    make(map[string]time.Time),
		soundscapes:           soundscapesList,
		soundscapeIndex:       -1,
		internalLogs:          []string{},
		petState:              pState,
		petEnabled:            petEnabled,
		showPetSidebar:        true,
		showPetLevelUpOverlay: false,
		tamagotchiActiveMenu:  "",
		tamagotchiMenuSelect:  0,
		tamagotchiFeedback:    "",
		kairuType:             kairutype.InitKairuType("time", 30),
	}
	m.logInternal("SYSTEM: Kairu TUI started")
	m = m.setInputFocus(initialFocus)

	if jobs, err := notification.LoadNotificationOutbox(m.outboxFile); err == nil {
		m.notificationOutbox = jobs
	} else {
		startupErrors = append(startupErrors, fmt.Sprintf("Failed to read notification queue: %v", err))
		m.appError = strings.Join(startupErrors, " | ")
	}

	return m
}

func (m model) View() string {
	m.petState.TimerRunning = m.running
	m.petState.TimerMode = m.mode
	m.petState.SessionElapsed = m.sessionElapsed

	switch m.mode {
	case "input":
		return renderInputView(m)
	case "timer", "break":
		return renderTimerView(m)
	case "kairu_type":
		theme := activeTheme(m.config)
		return kairutype.RenderKairuTypeView(m.kairuType, m.width, theme.Accent, theme.Primary, renderBanner(m.config))
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
	case "tamagotchi":
		return renderTamagotchiView(m)
	default:
		return renderFatalView(m)
	}
}
