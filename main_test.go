package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseDurationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		wantSeconds int
		wantErr     bool
	}{
		{name: "minutes", input: "25", wantSeconds: 25 * 60},
		{name: "hoursMinutes", input: "1:00", wantSeconds: 60 * 60},
		{name: "zeroHoursMinutes", input: "0:30", wantSeconds: 30 * 60},
		{name: "trimmed", input: "  5  ", wantSeconds: 5 * 60},
		{name: "empty", input: "", wantErr: true},
		{name: "zeroMinutes", input: "0", wantErr: true},
		{name: "negativeMinutes", input: "-5", wantErr: true},
		{name: "invalidMinutes", input: "1:60", wantErr: true},
		{name: "invalidFormat", input: "1:2:3", wantErr: true},
		{name: "notNumber", input: "abc", wantErr: true},
		{name: "negativePart", input: "1:-1", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDurationInput(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (input=%q)", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error (input=%q): %v", tc.input, err)
			}
			if got != tc.wantSeconds {
				t.Fatalf("got %d seconds, want %d (input=%q)", got, tc.wantSeconds, tc.input)
			}
		})
	}
}

func TestFormatDurationInput(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seconds int
		want    string
	}{
		{seconds: 0, want: "0"},
		{seconds: -10, want: "0"},
		{seconds: 60, want: "1"},
		{seconds: 600, want: "10"},
		{seconds: 3600, want: "1:00"},
		{seconds: 3660, want: "1:01"},
		{seconds: 7320, want: "2:02"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			got := formatDurationInput(tc.seconds)
			if got != tc.want {
				t.Fatalf("got %q, want %q (seconds=%d)", got, tc.want, tc.seconds)
			}
		})
	}
}

func TestGetDailyTotal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, loc)
	yesterday := today.AddDate(0, 0, -1)

	entries := []Entry{
		{Task: "a", Start: today, Duration: 600, Type: "work"},
		{Task: "b", Start: today.Add(2 * time.Hour), Duration: 120, Type: "work"},
		{Task: "c", Start: today.Add(3 * time.Hour), Duration: 300, Type: "break"},
		{Task: "d", Start: yesterday, Duration: 999, Type: "work"},
	}

	if got := getDailyTotal(entries, "work"); got != 720 {
		t.Fatalf("work total got %d, want %d", got, 720)
	}
	if got := getDailyTotal(entries, "break"); got != 300 {
		t.Fatalf("break total got %d, want %d", got, 300)
	}
}

func TestGetWeeklyData(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	twoDaysAgo := today.AddDate(0, 0, -2)
	eightDaysAgo := today.AddDate(0, 0, -8)

	entries := []Entry{
		{Task: "today", Start: today, Duration: 600, Type: "work"},
		{Task: "today-break", Start: today.Add(1 * time.Hour), Duration: 300, Type: "break"},
		{Task: "two-days", Start: twoDaysAgo, Duration: 120, Type: "work"},
		{Task: "old", Start: eightDaysAgo, Duration: 999, Type: "work"},
	}

	weekly := getWeeklyData(entries)
	if len(weekly) != 7 {
		t.Fatalf("weekly data size got %d, want 7", len(weekly))
	}

	if got := weekly[dateKey(today)]; got != 600 {
		t.Fatalf("today total got %d, want %d", got, 600)
	}
	if got := weekly[dateKey(twoDaysAgo)]; got != 120 {
		t.Fatalf("two-days-ago total got %d, want %d", got, 120)
	}

	if _, ok := weekly[dateKey(eightDaysAgo)]; ok {
		t.Fatalf("expected date %s to be out of range", dateKey(eightDaysAgo))
	}
}

func TestCalculateStreaks(t *testing.T) {
	t.Parallel()

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)

	makeEntry := func(date time.Time, sessionType string) Entry {
		return Entry{Task: "t", Start: date, End: date.Add(30 * time.Minute), Duration: 1800, Type: sessionType}
	}

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		current, longest := calculateStreaks(nil)
		if current != 0 || longest != 0 {
			t.Fatalf("got current=%d longest=%d, want 0,0", current, longest)
		}
	})

	t.Run("consecutiveIncludingToday", func(t *testing.T) {
		t.Parallel()

		entries := []Entry{
			makeEntry(today.AddDate(0, 0, -2), "work"),
			makeEntry(today.AddDate(0, 0, -1), "work"),
			makeEntry(today, "work"),
			makeEntry(today, "break"),
		}
		current, longest := calculateStreaks(entries)
		if current != 3 || longest != 3 {
			t.Fatalf("got current=%d longest=%d, want 3,3", current, longest)
		}
	})

	t.Run("gapBeforeToday", func(t *testing.T) {
		t.Parallel()

		entries := []Entry{
			makeEntry(today.AddDate(0, 0, -2), "work"),
			makeEntry(today, "work"),
		}
		current, longest := calculateStreaks(entries)
		if current != 1 || longest != 1 {
			t.Fatalf("got current=%d longest=%d, want 1,1", current, longest)
		}
	})

	t.Run("longestWithoutToday", func(t *testing.T) {
		t.Parallel()

		entries := []Entry{
			makeEntry(today.AddDate(0, 0, -5), "work"),
			makeEntry(today.AddDate(0, 0, -4), "work"),
			makeEntry(today.AddDate(0, 0, -3), "work"),
		}
		current, longest := calculateStreaks(entries)
		if current != 0 || longest != 3 {
			t.Fatalf("got current=%d longest=%d, want 0,3", current, longest)
		}
	})
}

func TestDateKeyUsesLocal(t *testing.T) {
	t.Parallel()

	now := time.Now()
	got := dateKey(now)
	want := now.In(time.Local).Format("2006-01-02")
	if !strings.EqualFold(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
