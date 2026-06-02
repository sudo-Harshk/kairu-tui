package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kairu-tui/internal/soundscape"
)

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

	footer := "[Enter] Select   [Esc/Ctrl+B] Cancel"
	block := renderBanner(m.config) + "\n\n" +
		"╭─────────────────────────────────────╮\n" +
		"│  🎵  Soundscapes                   │\n" +
		"╰─────────────────────────────────────╯\n\n" +
		strings.Join(lines, "\n") + "\n\n" +
		footer

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
