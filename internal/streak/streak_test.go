package streak

import (
	"testing"
	"time"

	"kairu-tui/internal/entries"
)

func TestCalculateStreaks(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)

	makeEntry := func(date time.Time, sessionType string) entries.Entry {
		return entries.Entry{Task: "t", Start: date, End: date.Add(30 * time.Minute), Duration: 1800, Type: sessionType}
	}

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		current, longest := CalculateStreaks(nil)
		if current != 0 || longest != 0 {
			t.Fatalf("got current=%d longest=%d, want 0,0", current, longest)
		}
	})

	t.Run("consecutiveIncludingToday", func(t *testing.T) {
		t.Parallel()

		entryList := []entries.Entry{
			makeEntry(today.AddDate(0, 0, -2), "work"),
			makeEntry(today.AddDate(0, 0, -1), "work"),
			makeEntry(today, "work"),
			makeEntry(today, "break"),
		}
		current, longest := CalculateStreaks(entryList)
		if current != 3 || longest != 3 {
			t.Fatalf("got current=%d longest=%d, want 3,3", current, longest)
		}
	})

	t.Run("gapBeforeToday", func(t *testing.T) {
		t.Parallel()

		entryList := []entries.Entry{
			makeEntry(today.AddDate(0, 0, -2), "work"),
			makeEntry(today, "work"),
		}
		current, longest := CalculateStreaks(entryList)
		if current != 1 || longest != 1 {
			t.Fatalf("got current=%d longest=%d, want 1,1", current, longest)
		}
	})

	t.Run("longestWithoutToday", func(t *testing.T) {
		t.Parallel()

		entryList := []entries.Entry{
			makeEntry(today.AddDate(0, 0, -5), "work"),
			makeEntry(today.AddDate(0, 0, -4), "work"),
			makeEntry(today.AddDate(0, 0, -3), "work"),
		}
		current, longest := CalculateStreaks(entryList)
		if current != 0 || longest != 3 {
			t.Fatalf("got current=%d longest=%d, want 0,3", current, longest)
		}
	})
}

func TestComputeStreakState(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	entryList := []entries.Entry{
		{Task: "a", Start: twoDaysAgo, End: twoDaysAgo.Add(30 * time.Minute), Duration: 1800, Type: "work"},
		{Task: "b", Start: yesterday, End: yesterday.Add(30 * time.Minute), Duration: 1800, Type: "work"},
	}

	streak := ComputeStreakState(entryList)
	if streak.Current != 0 {
		t.Fatalf("expected current streak 0, got %d", streak.Current)
	}
	if streak.Best != 2 {
		t.Fatalf("expected best streak 2, got %d", streak.Best)
	}
	if !streak.RecoveryAvailable {
		t.Fatalf("expected recovery to be available")
	}
	if !streak.RecoveryNeeded {
		t.Fatalf("expected recovery to be needed")
	}
}

func TestDateKeyUsesLocal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	got := DateKey(now)
	want := now.In(time.Local).Format("2006-01-02")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
