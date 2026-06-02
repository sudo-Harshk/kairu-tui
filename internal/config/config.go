package config

import (
	"errors"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

type Config struct {
	WorkDuration         int      `yaml:"work_duration"`
	BreakDuration        int      `yaml:"break_duration"`
	Font                 string   `yaml:"font"`
	Theme                string   `yaml:"theme"`
	Notifications        bool     `yaml:"notifications"`
	DesktopNotifications bool     `yaml:"desktop_notifications"`
	NotifyWorkComplete   bool     `yaml:"notify_work_complete"`
	NotifyBreakComplete  bool     `yaml:"notify_break_complete"`
	NotifySessionStart   bool     `yaml:"notify_session_start"`
	NotifySessionEnd     bool     `yaml:"notify_session_end"`
	NotifyPauseResume    bool     `yaml:"notify_pause_resume"`
	NotifyEndingSoon     bool     `yaml:"notify_ending_soon"`
	QuietHoursStart      int      `yaml:"quiet_hours_start"`
	QuietHoursEnd        int      `yaml:"quiet_hours_end"`
	SoundCommand         string   `yaml:"sound_command"`
	AutoBreak            bool     `yaml:"auto_break"`
	SessionsBeforeBreak  int      `yaml:"sessions_before_break"`
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

var DefaultConfig = Config{
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

func LoadEnvFile(path string) error {
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig
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

func SaveConfigFile(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func applyEnvOverrides(cfg *Config) {
	if val := strings.TrimSpace(os.Getenv(envTelegramBotToken)); val != "" {
		cfg.TelegramBotToken = val
	}

	if val := strings.TrimSpace(os.Getenv(envTelegramChatID)); val != "" {
		cfg.TelegramChatID = val
	}
}

func NormalizeTheme(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range ThemeOrder {
		if candidate == name {
			return candidate
		}
	}
	return DefaultConfig.Theme
}

func NormalizeFont(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range FontOrder {
		if candidate == name {
			return candidate
		}
	}
	return DefaultConfig.Font
}

func NormalizeLayout(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, candidate := range LayoutOrder {
		if candidate == name {
			return candidate
		}
	}
	return DefaultConfig.Layout
}

func ActiveTheme(cfg Config) ThemeStyle {
	if theme, ok := ThemeStyles[NormalizeTheme(cfg.Theme)]; ok {
		return theme
	}
	return ThemeStyles[DefaultConfig.Theme]
}

func ActiveFont(cfg Config) TimerFont {
	if font, ok := TimerFonts[NormalizeFont(cfg.Font)]; ok {
		return font
	}
	return TimerFonts[DefaultConfig.Font]
}

func ThemedStyle(cfg Config, color string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func NextValue(order []string, current string, delta int) string {
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

func ThemeLabel(name string) string {
	return cases.Title(language.English).String(NormalizeTheme(name))
}

func FontLabel(name string) string {
	font := ActiveFont(Config{Font: NormalizeFont(name)})
	return font.Label
}

func LayoutLabel(name string) string {
	return cases.Title(language.English).String(NormalizeLayout(name))
}
