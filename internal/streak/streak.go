package streak

import (
	"fmt"
	"os"
	"sort"
	"time"

	"kairu-tui/internal/entries"
)

type StreakState struct {
	Current           int
	Best              int
	LastWorkDay       string
	RecoveryAvailable bool
	RecoveryNeeded    bool
	RecoveryPrompt    string
}

func ComputeStreakState(entryList []entries.Entry) StreakState {
	days := make(map[string]bool)
	for _, e := range entryList {
		if e.Type == "work" {
			days[DateKey(e.Start)] = true
		}
	}
	if len(days) == 0 {
		return StreakState{}
	}

	var list []string
	for d := range days {
		list = append(list, d)
	}
	sort.Strings(list)

	best, temp := 0, 0
	var last time.Time
	for _, d := range list {
		date, err := time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Kairu: invalid entry date:", d)
			continue
		}
		if last.IsZero() {
			temp = 1
		} else if last.AddDate(0, 0, 1).Equal(date) {
			temp++
		} else {
			if temp > best {
				best = temp
			}
			temp = 1
		}
		last = date
	}
	if temp > best {
		best = temp
	}

	today := DateKey(time.Now())
	lastWorkDay := list[len(list)-1]
	current := 0
	recovery := false
	recoveryNeeded := false
	if days[today] {
		for i := 0; i < 365; i++ {
			if days[DateKey(time.Now().AddDate(0, 0, -i))] {
				current++
			} else if i > 0 {
				break
			}
		}
	} else {
		recoveryNeeded = true
		yesterday := DateKey(time.Now().AddDate(0, 0, -1))
		recovery = days[yesterday]
	}

	return StreakState{
		Current:           current,
		Best:              best,
		LastWorkDay:       lastWorkDay,
		RecoveryAvailable: recovery,
		RecoveryNeeded:    recoveryNeeded,
		RecoveryPrompt:    recoveryPrompt(days),
	}
}

func CalculateStreaks(entryList []entries.Entry) (int, int) {
	s := ComputeStreakState(entryList)
	return s.Current, s.Best
}

func recoveryPrompt(days map[string]bool) string {
	today := DateKey(time.Now())
	if days[today] {
		return "Streak active today"
	}
	yesterday := DateKey(time.Now().AddDate(0, 0, -1))
	if days[yesterday] {
		return "Recovery mode: one session restores your streak"
	}
	return "Recovery mode: start today to rebuild momentum"
}

func RecoveryLabel(streak StreakState) string {
	if streak.Current > 0 {
		return "Active today"
	}
	if streak.RecoveryAvailable {
		return "Recoverable"
	}
	if streak.RecoveryNeeded {
		return "Broken, recover"
	}
	return "No streak yet"
}

func DateKey(value time.Time) string {
	return value.In(time.Local).Format("2006-01-02")
}

func FormatDuration(seconds int) string {
	h, m := seconds/3600, (seconds%3600)/60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return "0m"
}
