package tasks

import (
	"os"
	"strings"

	"kairu-tui/internal/entries"
)

// LoadTasksFromFile reads and parses tasks from a file.
func LoadTasksFromFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var tasks []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" {
			tasks = append(tasks, t)
		}
	}
	return tasks
}

// BuildTaskSuggestions constructs a list of unique task suggestions prioritized by pinned, file-based, and recent tasks.
func BuildTaskSuggestions(entryList []entries.Entry, pinned []string, fileTasks []string) []string {
	seen := make(map[string]struct{})
	var suggestions []string

	// 1. Pinned tasks (highest priority)
	for _, t := range pinned {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			suggestions = append(suggestions, t)
		}
	}

	// 2. File tasks
	for _, t := range fileTasks {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			suggestions = append(suggestions, t)
		}
	}

	// 3. Recent tasks from history
	for i := len(entryList) - 1; i >= 0; i-- {
		t := strings.TrimSpace(entryList[i].Task)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			suggestions = append(suggestions, t)
		}
	}
	return suggestions
}
