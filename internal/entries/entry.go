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

// FilterOption defines filter parameters for session history
type FilterOption struct {
	Query    string    // search task/note text
	DateFrom time.Time // start of date range
	DateTo   time.Time // end of date range  
	Type     string    // "all", "work", "break"
	Tags     []string  // match any of these tags
}

// FilterEntries filters a slice of Entry based on the given FilterOption
func FilterEntries(entries []Entry, opt FilterOption) []Entry {
	var filtered []Entry
	for _, entry := range entries {
		// Case-insensitive substring match against Task and Note
		if opt.Query != "" {
			q := strings.ToLower(opt.Query)
			taskMatch := strings.Contains(strings.ToLower(entry.Task), q)
			noteMatch := strings.Contains(strings.ToLower(entry.Note), q)
			if !taskMatch && !noteMatch {
				continue
			}
		}

		// DateFrom/DateTo range (inclusive)
		if !opt.DateFrom.IsZero() && entry.Start.Before(opt.DateFrom) {
			continue
		}
		if !opt.DateTo.IsZero() && entry.Start.After(opt.DateTo) {
			continue
		}

		// Type filtering
		if opt.Type != "" && opt.Type != "all" {
			if entry.Type != opt.Type {
				continue
			}
		}

		// Tags filtering (matches if any entry tag appears in the filter list)
		if len(opt.Tags) > 0 {
			match := false
			for _, et := range entry.Tags {
				for _, ft := range opt.Tags {
					if strings.EqualFold(et, ft) {
						match = true
						break
					}
				}
				if match {
					break
				}
			}
			if !match {
				continue
			}
		}

		filtered = append(filtered, entry)
	}
	return filtered
}

