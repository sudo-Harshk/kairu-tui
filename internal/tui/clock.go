package tui

import (
	"strings"

	"kairu-tui/internal/config"
	"kairu-tui/internal/timer"
)

func renderClock(seconds int, font config.TimerFont) string {
	timeStr := timer.FormatClock(seconds)
	chars := font.Digits

	var height int
	for _, art := range chars {
		height = len(art)
		break
	}
	if height == 0 {
		height = 5
	}

	lines := make([]string, height)
	for _, ch := range timeStr {
		if art, ok := chars[ch]; ok {
			for i := 0; i < height; i++ {
				lines[i] += art[i] + " "
			}
		}
	}

	return strings.Join(lines, "\n")
}
