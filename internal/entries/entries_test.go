package entries

import (
	"testing"
	"time"
)

func TestValidateEntries(t *testing.T) {
	t.Parallel()

	now := time.Now()
	valid := []Entry{
		{Task: "focus", Start: now, End: now, Duration: 0, Type: "work"},
		{Task: "break", Start: now, End: now.Add(2 * time.Minute), Duration: 120, Type: "break"},
	}
	if err := ValidateEntries(valid); err != nil {
		t.Fatalf("expected valid entries, got error: %v", err)
	}

	cases := []struct {
		name   string
		entry  Entry
		hasErr bool
	}{
		{name: "missingTask", entry: Entry{Task: " ", Start: now, End: now, Duration: 10, Type: "work"}, hasErr: true},
		{name: "invalidType", entry: Entry{Task: "x", Start: now, End: now, Duration: 10, Type: "other"}, hasErr: true},
		{name: "missingStart", entry: Entry{Task: "x", End: now, Duration: 10, Type: "work"}, hasErr: true},
		{name: "missingEnd", entry: Entry{Task: "x", Start: now, Duration: 10, Type: "work"}, hasErr: true},
		{name: "endBeforeStart", entry: Entry{Task: "x", Start: now, End: now.Add(-time.Minute), Duration: 10, Type: "work"}, hasErr: true},
		{name: "negativeDuration", entry: Entry{Task: "x", Start: now, End: now, Duration: -1, Type: "work"}, hasErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateEntries([]Entry{tc.entry}); (err != nil) != tc.hasErr {
				t.Fatalf("expected error=%t, got %v", tc.hasErr, err)
			}
		})
	}
}

func TestMergeEntriesDedup(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 6, 1, 9, 0, 0, 0, time.Local)
	entryA := Entry{Task: "a", Start: start, End: start.Add(25 * time.Minute), Duration: 1500, Type: "work"}
	entryB := Entry{Task: "b", Start: start.Add(2 * time.Hour), End: start.Add(2*time.Hour + 5*time.Minute), Duration: 300, Type: "break"}
	entryC := Entry{Task: "c", Start: start.Add(3 * time.Hour), End: start.Add(3*time.Hour + 10*time.Minute), Duration: 600, Type: "work"}

	existing := []Entry{entryA, entryB}
	incoming := []Entry{entryA, entryC}
	merged := MergeEntries(existing, incoming)

	if len(merged) != 3 {
		t.Fatalf("expected 3 merged entries, got %d", len(merged))
	}

	seen := map[string]bool{}
	for _, entry := range merged {
		seen[EntryKey(entry)] = true
	}
	if !seen[EntryKey(entryA)] || !seen[EntryKey(entryB)] || !seen[EntryKey(entryC)] {
		t.Fatalf("merged entries missing expected items")
	}
}

func TestFilterEntries(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	testEntries := []Entry{
		{
			Task:     "Quarterly review",
			Note:     "Reviewing Q1 performance",
			Tags:     []string{"work", "meeting"},
			Start:    baseTime,
			End:      baseTime.Add(25 * time.Minute),
			Duration: 1500,
			Type:     "work",
		},
		{
			Task:     "Quick Coffee Break",
			Note:     "Relaxing",
			Tags:     []string{"break", "coffee"},
			Start:    baseTime.Add(1 * time.Hour),
			End:      baseTime.Add(1*time.Hour + 5*time.Minute),
			Duration: 300,
			Type:     "break",
		},
		{
			Task:     "Refactoring paths",
			Note:     "Adding XDG paths support",
			Tags:     []string{"dev", "go"},
			Start:    baseTime.Add(2 * time.Hour),
			End:      baseTime.Add(2*time.Hour + 45*time.Minute),
			Duration: 2700,
			Type:     "work",
		},
	}

	tests := []struct {
		name     string
		opt      FilterOption
		expected int
	}{
		{
			name:     "no filter (empty opt)",
			opt:      FilterOption{},
			expected: 3,
		},
		{
			name:     "query case-insensitive task match",
			opt:      FilterOption{Query: "quarterly"},
			expected: 1,
		},
		{
			name:     "query case-insensitive note match",
			opt:      FilterOption{Query: "xdg"},
			expected: 1,
		},
		{
			name:     "query no match",
			opt:      FilterOption{Query: "nonexistent"},
			expected: 0,
		},
		{
			name:     "date range inclusive from start",
			opt:      FilterOption{DateFrom: baseTime.Add(30 * time.Minute)},
			expected: 2,
		},
		{
			name:     "date range inclusive to end",
			opt:      FilterOption{DateTo: baseTime.Add(90 * time.Minute)},
			expected: 2,
		},
		{
			name:     "type filter work",
			opt:      FilterOption{Type: "work"},
			expected: 2,
		},
		{
			name:     "type filter break",
			opt:      FilterOption{Type: "break"},
			expected: 1,
		},
		{
			name:     "type filter all",
			opt:      FilterOption{Type: "all"},
			expected: 3,
		},
		{
			name:     "tags filter match single tag",
			opt:      FilterOption{Tags: []string{"coffee"}},
			expected: 1,
		},
		{
			name:     "tags filter match multi tag",
			opt:      FilterOption{Tags: []string{"meeting", "go"}},
			expected: 2,
		},
		{
			name:     "tags filter no match",
			opt:      FilterOption{Tags: []string{"rust"}},
			expected: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := FilterEntries(testEntries, tc.opt)
			if len(got) != tc.expected {
				t.Errorf("expected %d entries, got %d", tc.expected, len(got))
			}
		})
	}
}

