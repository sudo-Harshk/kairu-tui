package entries

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func LoadEntries(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}
	var entriesList []Entry
	if err := json.Unmarshal(data, &entriesList); err != nil {
		return nil, err
	}
	if err := ValidateEntries(entriesList); err != nil {
		return nil, err
	}
	return entriesList, nil
}

func ExportEntries(dataFile, exportPath string) error {
	entriesList, err := LoadEntries(dataFile)
	if err != nil {
		return fmt.Errorf("failed to read entries: %w", err)
	}
	data, err := json.MarshalIndent(entriesList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode entries: %w", err)
	}
	if err := os.WriteFile(exportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	return nil
}

func ImportEntries(dataFile, importPath string) error {
	incomingData, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}
	var incoming []Entry
	if err := json.Unmarshal(incomingData, &incoming); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}
	if err := ValidateEntries(incoming); err != nil {
		return fmt.Errorf("import validation failed: %w", err)
	}
	existing, err := LoadEntries(dataFile)
	if err != nil {
		return fmt.Errorf("failed to read existing entries: %w", err)
	}
	merged := MergeEntries(existing, incoming)
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode merged entries: %w", err)
	}
	if err := os.WriteFile(dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write entries: %w", err)
	}
	return nil
}
