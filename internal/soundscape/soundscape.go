package soundscape

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadSoundscapes returns a list of available soundscapes, including native synthetic ones and any files found in the directory.
func LoadSoundscapes(dir string) ([]string, error) {
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
