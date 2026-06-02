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
