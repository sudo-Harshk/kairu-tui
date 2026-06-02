package entries

import (
	"fmt"
	"strings"
	"time"
)

type Entry struct {
	Task     string    `json:"task"`
	Note     string    `json:"note,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration int       `json:"duration_seconds"`
	Type     string    `json:"type"`
}

func ValidateEntries(entries []Entry) error {
	for i, entry := range entries {
		if strings.TrimSpace(entry.Task) == "" {
			return fmt.Errorf("entry %d: task is required", i)
		}
		if entry.Type != "work" && entry.Type != "break" {
			return fmt.Errorf("entry %d: type must be work or break", i)
		}
		if entry.Start.IsZero() || entry.End.IsZero() {
			return fmt.Errorf("entry %d: start and end must be set", i)
		}
		if entry.End.Before(entry.Start) {
			return fmt.Errorf("entry %d: end is before start", i)
		}
		if entry.Duration < 0 {
			return fmt.Errorf("entry %d: duration must be 0 or greater", i)
		}
	}
	return nil
}

func MergeEntries(existing, incoming []Entry) []Entry {
	seen := make(map[string]struct{}, len(existing))
	for _, entry := range existing {
		seen[EntryKey(entry)] = struct{}{}
	}
	merged := append([]Entry{}, existing...)
	for _, entry := range incoming {
		key := EntryKey(entry)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, entry)
	}
	return merged
}

func EntryKey(entry Entry) string {
	return fmt.Sprintf("%s|%s|%s|%s",
		strings.TrimSpace(entry.Task),
		entry.Start.Format(time.RFC3339Nano),
		entry.End.Format(time.RFC3339Nano),
		entry.Type,
	)
}
