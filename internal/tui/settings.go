package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/config"
	"kairu-tui/internal/soundscape"
)

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

func renderSettingsSection(cfg config.Config, title string, lines []string) string {
	theme := activeTheme(cfg)
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Accent)).Render(title)
	return fmt.Sprintf(`╭─────────────────────────────────────╮
│  %s
╰─────────────────────────────────────╯

%s`, header, strings.Join(lines, "\n"))
}

func renderSettingsPreview(m model) string {
	theme := activeTheme(m.config)
	font := activeFont(m.config)
	timerStr := renderASCIITimer("08:45", m.config)
	lines := strings.Split(timerStr, "\n")
	if len(lines) > 5 {
		lines = lines[:5]
	}

	themeLine := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Render("Theme: " + themeLabel(m.config.Theme))
	fontLine := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Render("Font: " + font.Label)
	layoutLine := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Notice)).Render("Layout: " + layoutLabel(m.config.Layout))
	timerBlock := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Render(strings.Join(lines, "\n"))

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

func (m *model) toggleSetting() {
	themeOrder := config.ThemeOrder
	fontOrder := config.FontOrder
	layoutOrder := config.LayoutOrder
	binauralPresetsOrder := config.BinauralPresetsOrder

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
		soundscape.SetNativeVolume(m.config.SynthVolume)
	case settingsBinauralPreset:
		m.config.BinauralPreset = nextValue(binauralPresetsOrder, m.config.BinauralPreset, 1)
		soundscape.UpdateActiveBinauralFrequencies(m.config)
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
		soundscape.UpdateActiveBinauralFrequencies(m.config)
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
		soundscape.UpdateActiveBinauralFrequencies(m.config)
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
	if err := config.SaveConfigFile(m.configFile, m.config); err != nil {
		m.setAppError(err, "Failed to save config")
	}
}

func (m *model) adjustSetting(delta int) {
	themeOrder := config.ThemeOrder
	fontOrder := config.FontOrder
	layoutOrder := config.LayoutOrder
	binauralPresetsOrder := config.BinauralPresetsOrder

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
		soundscape.SetNativeVolume(m.config.SynthVolume)
	case settingsBinauralPreset:
		m.config.BinauralPreset = nextValue(binauralPresetsOrder, m.config.BinauralPreset, delta)
		soundscape.UpdateActiveBinauralFrequencies(m.config)
	case settingsBinauralCarrier:
		m.config.BinauralCarrier += float64(delta) * 5.0
		if m.config.BinauralCarrier < 20.0 {
			m.config.BinauralCarrier = 20.0
		} else if m.config.BinauralCarrier > 20000.0 {
			m.config.BinauralCarrier = 20000.0
		}
		soundscape.UpdateActiveBinauralFrequencies(m.config)
	case settingsBinauralBeat:
		m.config.BinauralBeat += float64(delta) * 0.5
		if m.config.BinauralBeat < 0.1 {
			m.config.BinauralBeat = 0.1
		} else if m.config.BinauralBeat > 100.0 {
			m.config.BinauralBeat = 100.0
		}
		soundscape.UpdateActiveBinauralFrequencies(m.config)
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
	if err := config.SaveConfigFile(m.configFile, m.config); err != nil {
		m.setAppError(err, "Failed to save config")
	}
}
