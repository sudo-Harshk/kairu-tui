package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"kairu-tui/internal/soundscape"
	"kairu-tui/internal/ui"
)

func renderSoundscapeMenuView(m model) string {
	theme := activeTheme(m.config)
	var items []string
	if m.soundscapeIndex == -1 {
		items = append(items, "[*] None")
	} else {
		items = append(items, "[ ] None")
	}
	for i, track := range m.soundscapes {
		if i == m.soundscapeIndex {
			items = append(items, "[*] "+track)
		} else {
			items = append(items, "[ ] "+track)
		}
	}

	menuIndex := m.soundscapeIndex + 1

	var content string
	if len(m.soundscapes) == 0 {
		content = ui.Menu(items, menuIndex, theme) + "\n\n  (No audio files found in " + m.config.SoundscapesDir + ")"
	} else {
		content = ui.Menu(items, menuIndex, theme)
	}

	formWidth := 46
	soundscapeCard := ui.Panel("🎵 Soundscapes Selector", content, theme, formWidth, lipgloss.RoundedBorder(), theme.Primary)
	statusBar := ui.StatusBar([]string{"[Enter] Select", "[Esc/Ctrl+B] Cancel"}, "", theme, formWidth)

	block := renderBanner(m.config) + "\n\n" + soundscapeCard + "\n\n" + statusBar

	return fmt.Sprintf("\n%s\n", centerBlock(m.width, block))
}

func (m *model) startSoundscape() {
	if m.soundscapeIndex < 0 || m.soundscapeIndex >= len(m.soundscapes) {
		return
	}
	m.stopSoundscape()

	track := m.soundscapes[m.soundscapeIndex]
	m.logInternal("AUDIO: Starting %s", track)

	if soundscape.IsSyntheticTrack(track) {
		if err := soundscape.StartNativeSynth(track, m.config); err != nil {
			m.setAppError(err, "Failed to start native synthesizer")
		}
		soundscape.FadeNativeVolume(m.config.SynthVolume, time.Duration(m.config.FadeInDuration)*time.Millisecond)
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
	soundscape.StopNativeSynth()

	if m.activeSoundscapeCmd != nil && m.activeSoundscapeCmd.Process != nil {
		m.logInternal("AUDIO: Stopping playback")
		_ = m.activeSoundscapeCmd.Process.Kill()
		_ = m.activeSoundscapeCmd.Wait()
		m.activeSoundscapeCmd = nil
	}
}
